package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// Tunables for LogsService streaming. The numbers match the back-pressure
// table in ADR-0001 and the per-buffer sizing already used by the legacy
// WebSocket handler in dashboard/server_websocket.go.
const (
	// localLogRingBufferSize bounds the per-stream Connect ring used by
	// StreamLocalLogs. The underlying *service.LogBuffer subscriber
	// channel is 100-buffered (logbuffer.go); this 1024-entry ring sits
	// behind it and provides drop-OLDEST + accurate drop-counting so a
	// dashboard that briefly stalls (e.g. tab backgrounded) sees the
	// most-recent state instead of stale history when it catches up.
	localLogRingBufferSize = 1024

	// maxLocalLogBackfill caps client-requested initial backfill so a
	// reconnecting tab cannot scrape an entire 10k-entry buffer per
	// retry. The value matches the GET /api/logs upper bound in
	// dashboard/server_handlers.go (handleGetLogs clamps tail to 10k).
	maxLocalLogBackfill = 10_000

	// minGetLogsTail / maxGetLogsTail mirror handleGetLogs:
	// zero-or-negative defaults to 500, anything above 10k clamps to
	// 10k. tailClamped is set on the response when adjustment happens.
	defaultGetLogsTail = 500
	maxGetLogsTail     = 10_000
)

// LogsHandler implements azdappv1connect.LogsServiceHandler.
//
// The handler owns no state of its own. All side effects flow through
// the injected LogsStore, keeping the Connect surface trivially testable
// and the dashboard package's storage helpers (loadAzureYaml,
// getOrCreateConfigClient, getLogManager) the single source of truth for
// each underlying resource.
type LogsHandler struct {
	store LogsStore
}

// Compile-time interface conformance.
var _ azdappv1connect.LogsServiceHandler = (*LogsHandler)(nil)

// NewLogsHandler constructs a LogsHandler. A nil store is a programming
// error - panic at construction time so misconfigured dashboards fail
// fast at startup instead of surfacing nil-pointer panics on the first
// request.
func NewLogsHandler(store LogsStore) *LogsHandler {
	if store == nil {
		panic("rpc: NewLogsHandler called with nil LogsStore")
	}
	return &LogsHandler{store: store}
}

// GetLogs returns recent buffered logs. Mirrors handleGetLogs query
// parameter handling: empty service_name -> merged tail; tail<=0 -> 500;
// tail>10000 -> 10000 (with tail_clamped=true on the response so the
// client knows it asked for more than it got).
func (h *LogsHandler) GetLogs(
	_ context.Context,
	req *connect.Request[v1.GetLogsRequest],
) (*connect.Response[v1.GetLogsResponse], error) {
	serviceName := req.Msg.GetServiceName()
	tail, clamped := clampGetLogsTail(req.Msg.GetTail())

	var entries []service.LogEntry
	if serviceName == "" {
		entries = h.store.GetAll(tail)
	} else {
		got, exists := h.store.GetRecent(serviceName, tail)
		if !exists {
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf("service %q not found", serviceName),
			)
		}
		entries = got
	}

	return connect.NewResponse(&v1.GetLogsResponse{
		Entries:     toProtoLogEntries(entries),
		TailClamped: clamped,
	}), nil
}

// StreamLocalLogs tails one (or all) services. See the proto comment for
// back-pressure semantics; the per-stream ring is implemented in the
// localLogRing type below.
//
// Lifecycle:
//  1. Resolve subscriptions (one channel per service).
//  2. Spawn one pump goroutine per source channel; each pump pushes
//     incoming entries into the per-stream ring (drop-OLDEST if full).
//  3. Optionally backfill the most-recent N buffered entries before
//     live tail begins (so reconnects don't blank the dashboard).
//  4. Main loop selects on ctx.Done and the ring's notify channel;
//     drains entries per notify, emitting a DroppedNotice ahead of the
//     next entry whenever the ring's drop counter advanced.
//  5. On return, deferred Unsubscribe closes each source channel; pump
//     goroutines see the closed channel and exit.
func (h *LogsHandler) StreamLocalLogs(
	ctx context.Context,
	req *connect.Request[v1.StreamLocalLogsRequest],
	stream *connect.ServerStream[v1.StreamLocalLogsResponse],
) error {
	serviceName := req.Msg.GetServiceName()
	backfill := req.Msg.GetBackfill()
	if backfill < 0 {
		backfill = 0
	}
	if backfill > maxLocalLogBackfill {
		backfill = maxLocalLogBackfill
	}

	// Resolve subscriptions. Single-service: one channel; all-services:
	// one channel per registered buffer at the moment of subscribe.
	// New buffers created mid-stream will not appear (matches the legacy
	// WebSocket behaviour - reconnect to pick up new services).
	subs := make(map[string]chan service.LogEntry)
	if serviceName != "" {
		ch, exists := h.store.Subscribe(serviceName)
		if !exists {
			return connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf("service %q not found", serviceName),
			)
		}
		subs[serviceName] = ch
	} else {
		for _, name := range h.store.ServiceNames() {
			if ch, exists := h.store.Subscribe(name); exists {
				subs[name] = ch
			}
		}
	}

	// Always release every subscription even on early-return paths.
	defer func() {
		for name, ch := range subs {
			h.store.Unsubscribe(name, ch)
		}
	}()

	// Per-stream ring: bounded, drop-oldest, drop-counter wakes main loop.
	ring := newLocalLogRing(localLogRingBufferSize)

	// Per-stream context so pump goroutines exit cleanly when the
	// outer ctx is cancelled OR when this function returns for any
	// other reason (e.g. send error). Defer ordering: streamCancel
	// fires LAST (after Unsubscribe closes the source channels), so a
	// pump goroutine that's blocked on ring.push gets woken either by
	// the ring draining or by the cancel.
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	// Optional backfill: the most-recent N buffered entries before live
	// tail begins. Resolved once at stream start so we don't race with
	// concurrent buffer rotation.
	if backfill > 0 {
		var seed []service.LogEntry
		if serviceName != "" {
			if got, exists := h.store.GetRecent(serviceName, int(backfill)); exists {
				seed = got
			}
		} else {
			seed = h.store.GetAll(int(backfill))
		}
		for i := range seed {
			if err := stream.Send(&v1.StreamLocalLogsResponse{
				Event: &v1.StreamLocalLogsResponse_Entry{
					Entry: toProtoLogEntry(seed[i]),
				},
			}); err != nil {
				return err
			}
		}
	}

	// Spawn one pump goroutine per subscription. Pumps exit when their
	// source channel closes (Unsubscribe path) or the stream context is
	// cancelled. They never block indefinitely on ring.push because
	// drop-oldest semantics mean ring.push always returns immediately.
	var pumpWg sync.WaitGroup
	for _, ch := range subs {
		pumpWg.Add(1)
		go func(src chan service.LogEntry) {
			defer pumpWg.Done()
			for {
				select {
				case <-streamCtx.Done():
					return
				case entry, ok := <-src:
					if !ok {
						return
					}
					ring.push(toProtoLogEntry(entry))
				}
			}
		}(ch)
	}

	// Main loop: drain ring on notify, emit dropped-notice when the
	// drop counter advanced since the last drain.
	var lastDropped int64
	for {
		select {
		case <-ctx.Done():
			// Wait for pumps to wind down so any in-flight ring.push
			// completes before the function returns. Bounded by the
			// per-pump select on streamCtx so this can't hang.
			streamCancel()
			pumpWg.Wait()
			return nil

		case <-ring.notify:
			entries, dropped := ring.drain()

			// Emit drop notice before the next entry whenever the
			// counter advanced. A single notice covers all entries
			// dropped since the last drain - clients see one banner
			// per stall, not one per dropped line.
			if dropped > lastDropped {
				delta := dropped - lastDropped
				lastDropped = dropped
				if err := stream.Send(&v1.StreamLocalLogsResponse{
					Event: &v1.StreamLocalLogsResponse_Dropped{
						Dropped: &v1.DroppedNotice{
							Count: delta,
							At:    timestamppb.Now(),
						},
					},
				}); err != nil {
					return err
				}
			}

			for _, msg := range entries {
				if err := stream.Send(&v1.StreamLocalLogsResponse{
					Event: &v1.StreamLocalLogsResponse_Entry{Entry: msg},
				}); err != nil {
					return err
				}
			}
		}
	}
}

// ListClassifications returns the current rule list. An azure.yaml that
// doesn't exist or fails to parse is surfaced as Internal - the legacy
// REST handler "logs and returns empty" but for a typed RPC clients
// benefit from knowing the read failed (so they don't display a stale-
// looking empty list as ground truth).
func (h *LogsHandler) ListClassifications(
	_ context.Context,
	_ *connect.Request[v1.ListClassificationsRequest],
) (*connect.Response[v1.ListClassificationsResponse], error) {
	classifications, err := h.store.LoadClassifications()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("load classifications: %w", err))
	}
	return connect.NewResponse(&v1.ListClassificationsResponse{
		Classifications: toProtoClassifications(classifications),
	}), nil
}

// AddClassification appends or updates a classification. Mirrors
// handleCreateClassification's add-or-update semantic exactly: text
// matched case-insensitively, level overwritten in place if found,
// otherwise appended.
func (h *LogsHandler) AddClassification(
	_ context.Context,
	req *connect.Request[v1.AddClassificationRequest],
) (*connect.Response[v1.AddClassificationResponse], error) {
	in := req.Msg.GetClassification()
	if in == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("classification is required"))
	}
	text := strings.TrimSpace(in.GetText())
	if text == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("text is required"))
	}
	level := fromProtoLogLevel(in.GetLevel())
	if !service.ValidateClassificationLevel(level) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("level must be info, warning, or error; got %q", level))
	}

	current, err := h.store.LoadClassifications()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("load classifications: %w", err))
	}

	// Normalise to the post-write shape so the response and the
	// persisted value agree even when the input had leading/trailing
	// whitespace on text.
	normalised := service.LogClassification{Text: text, Level: level}

	// Case-insensitive update-in-place; matches handleCreateClassification.
	updated := false
	for i := range current {
		if strings.EqualFold(current[i].Text, text) {
			current[i].Level = level
			normalised = current[i]
			updated = true
			break
		}
	}
	if !updated {
		current = append(current, normalised)
	}

	if err := h.store.SaveClassifications(current); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("save classifications: %w", err))
	}

	return connect.NewResponse(&v1.AddClassificationResponse{
		Classification: toProtoClassification(normalised),
	}), nil
}

// DeleteClassification removes by positional index. See proto comment
// for the index contract; CodeInvalidArgument for negative, CodeNotFound
// for out-of-range.
func (h *LogsHandler) DeleteClassification(
	_ context.Context,
	req *connect.Request[v1.DeleteClassificationRequest],
) (*connect.Response[v1.DeleteClassificationResponse], error) {
	idx := int(req.Msg.GetIndex())
	if idx < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("index must be non-negative; got %d", idx))
	}

	current, err := h.store.LoadClassifications()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("load classifications: %w", err))
	}
	if idx >= len(current) {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("index %d out of range; have %d classifications", idx, len(current)))
	}

	current = append(current[:idx], current[idx+1:]...)
	if err := h.store.SaveClassifications(current); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("save classifications: %w", err))
	}

	return connect.NewResponse(&v1.DeleteClassificationResponse{}), nil
}

// GetPreferences returns saved preferences, or proto defaults when no
// blob has been persisted yet. The handler decodes via protojson with
// DiscardUnknown=true so a JSON blob persisted from another schema
// version (older or newer) does not cause GetPreferences to fail.
func (h *LogsHandler) GetPreferences(
	_ context.Context,
	_ *connect.Request[v1.GetPreferencesRequest],
) (*connect.Response[v1.GetPreferencesResponse], error) {
	data, err := h.store.LoadPreferences()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("load preferences: %w", err))
	}

	prefs := defaultProtoPreferences()
	if len(data) > 0 {
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(data, prefs); err != nil {
			// Match the legacy REST handler's permissiveness: a corrupt
			// blob falls back to defaults rather than failing the read,
			// so a single bad write doesn't lock the user out of their
			// preferences UI. Log so the operator can debug.
			slog.Warn(
				"LogsService.GetPreferences: stored blob did not decode; returning defaults",
				"err", err.Error(),
			)
			prefs = defaultProtoPreferences()
		}
	}

	return connect.NewResponse(&v1.GetPreferencesResponse{Preferences: prefs}), nil
}

// SavePreferences persists the supplied preferences. The handler encodes
// via protojson (NOT through dashboard.UserPreferences) so theme and
// ui.grid_auto_fit round-trip end-to-end without a Go-struct strip step.
func (h *LogsHandler) SavePreferences(
	_ context.Context,
	req *connect.Request[v1.SavePreferencesRequest],
) (*connect.Response[v1.SavePreferencesResponse], error) {
	prefs := req.Msg.GetPreferences()
	if prefs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("preferences is required"))
	}

	// Marshal with deterministic output so the persisted blob is stable
	// across calls with identical input (helps tests and external diff
	// tools). UseProtoNames=false keeps camelCase on the wire, matching
	// the legacy REST handler's Go-json output.
	mopts := protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}
	data, err := mopts.Marshal(prefs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("marshal preferences: %w", err))
	}

	if err := h.store.SavePreferences(data); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("save preferences: %w", err))
	}

	return connect.NewResponse(&v1.SavePreferencesResponse{Preferences: prefs}), nil
}
