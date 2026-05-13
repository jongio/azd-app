package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
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

// LogSource is the narrow read/subscribe slice of service.LogManager that
// LogsHandler needs. Production wires it to a per-call LogManager
// (matching how dashboard/server_handlers.go and dashboard/server_websocket.go
// resolve the manager); tests inject an in-memory stub.
//
// Subscribe returns the buffered channel that *service.LogBuffer.Subscribe
// produces (capacity 100, broadcast drop-newest with timeout). Unsubscribe
// MUST close the channel; the implementation's existing semantics
// (logbuffer.go) already do this.
type LogSource interface {
	// GetRecent returns the last `tail` entries for one service. The
	// bool reports whether the service exists in the manager (false ->
	// NotFound on the wire).
	GetRecent(serviceName string, tail int) ([]service.LogEntry, bool)
	// GetAll returns the merged tail across all services (limit per
	// service applied internally, matching LogManager.GetAllLogs).
	GetAll(tail int) []service.LogEntry
	// ServiceNames lists the buffer keys currently registered. Used
	// for the all-services subscribe path so the handler can fan in
	// across every buffer without poking LogManager internals.
	ServiceNames() []string
	// Subscribe creates a per-service subscription channel. Returns
	// false if the service does not exist (NotFound on the wire).
	Subscribe(serviceName string) (chan service.LogEntry, bool)
	// Unsubscribe releases the channel and frees server-side resources.
	// MUST be safe to call with a channel from a now-removed service
	// (idempotent / no panic on missing buffer).
	Unsubscribe(serviceName string, ch chan service.LogEntry)
}

// ClassificationStore is the read/write slice of azure.yaml-stored log
// classifications LogsHandler needs. Production wires it to dashboard's
// loadAzureYaml/saveAzureYaml pair (closing over the package-level
// classificationsMu so REST and Connect handlers share the same lock);
// tests inject an in-memory stub.
//
// SaveClassifications receives the FULL replacement slice; the handler
// performs read-modify-write under the same mutex the legacy REST
// handler holds, so concurrent Add/Delete operations do not race.
type ClassificationStore interface {
	// LoadClassifications returns the current rule list from azure.yaml.
	// An empty slice is valid (no rules). Error means the YAML could not
	// be read or parsed; the handler maps that to Internal.
	LoadClassifications() ([]service.LogClassification, error)
	// SaveClassifications persists the full replacement slice, atomically
	// from the dashboard side (fileutil.AtomicWriteFile).
	SaveClassifications(classifications []service.LogClassification) error
}

// PreferenceStore is the read/write slice of the user-preferences blob.
// Production wires it to azdconfig.ConfigClient.{Get,Set}PreferenceSection
// keyed by "logs"; tests inject an in-memory stub.
//
// The blob is opaque JSON; the handler decodes/encodes via protojson with
// DiscardUnknown=true so older or future-schema keys round-trip without
// breaking GetPreferences. See the proto comment on Preferences for why
// theme + ui.grid_auto_fit live here despite being absent from the Go
// dashboard.UserPreferences struct.
type PreferenceStore interface {
	// LoadPreferences returns the raw JSON blob, or nil/empty if no
	// preferences have been stored yet. nil and zero-length both signal
	// "use defaults".
	LoadPreferences() ([]byte, error)
	// SavePreferences persists the raw JSON blob.
	SavePreferences(data []byte) error
}

// LogsStore bundles the three narrow interfaces LogsHandler needs so the
// Mount() conditional stays a single nil-check. Production wires it via
// LogsStoreFuncs (below) closing over dashboard.Server methods; tests
// satisfy it with in-memory stubs that implement all three slices.
type LogsStore interface {
	LogSource
	ClassificationStore
	PreferenceStore
}

// LogsStoreFuncs adapts plain function values to LogsStore. Lets the
// dashboard wire its private loadAzureYaml/saveAzureYaml/getOrCreateConfigClient
// helpers as method values without exporting them or smuggling the
// classifications mutex through a wrapper struct.
//
// Field semantics mirror the LogSource/ClassificationStore/PreferenceStore
// interface methods one-for-one. Every field MUST be set; nil function
// values cause LogsHandler RPCs to panic at call time, which is the
// fail-loud signal we want for a misconfigured wiring.
type LogsStoreFuncs struct {
	GetRecentFn           func(serviceName string, tail int) ([]service.LogEntry, bool)
	GetAllFn              func(tail int) []service.LogEntry
	ServiceNamesFn        func() []string
	SubscribeFn           func(serviceName string) (chan service.LogEntry, bool)
	UnsubscribeFn         func(serviceName string, ch chan service.LogEntry)
	LoadClassificationsFn func() ([]service.LogClassification, error)
	SaveClassificationsFn func(classifications []service.LogClassification) error
	LoadPreferencesFn     func() ([]byte, error)
	SavePreferencesFn     func(data []byte) error
}

// GetRecent implements LogSource.
func (f LogsStoreFuncs) GetRecent(serviceName string, tail int) ([]service.LogEntry, bool) {
	return f.GetRecentFn(serviceName, tail)
}

// GetAll implements LogSource.
func (f LogsStoreFuncs) GetAll(tail int) []service.LogEntry { return f.GetAllFn(tail) }

// ServiceNames implements LogSource.
func (f LogsStoreFuncs) ServiceNames() []string { return f.ServiceNamesFn() }

// Subscribe implements LogSource.
func (f LogsStoreFuncs) Subscribe(serviceName string) (chan service.LogEntry, bool) {
	return f.SubscribeFn(serviceName)
}

// Unsubscribe implements LogSource.
func (f LogsStoreFuncs) Unsubscribe(serviceName string, ch chan service.LogEntry) {
	f.UnsubscribeFn(serviceName, ch)
}

// LoadClassifications implements ClassificationStore.
func (f LogsStoreFuncs) LoadClassifications() ([]service.LogClassification, error) {
	return f.LoadClassificationsFn()
}

// SaveClassifications implements ClassificationStore.
func (f LogsStoreFuncs) SaveClassifications(classifications []service.LogClassification) error {
	return f.SaveClassificationsFn(classifications)
}

// LoadPreferences implements PreferenceStore.
func (f LogsStoreFuncs) LoadPreferences() ([]byte, error) { return f.LoadPreferencesFn() }

// SavePreferences implements PreferenceStore.
func (f LogsStoreFuncs) SavePreferences(data []byte) error { return f.SavePreferencesFn(data) }

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
			slog.Warn("LogsService.GetPreferences: stored blob did not decode; returning defaults",
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

// localLogRing is a bounded drop-oldest ring buffer with a coalescing
// notify channel. Pumps push entries; the main loop drains on notify.
// Concurrent push from N pump goroutines + concurrent drain from one
// consumer is safe under the mutex.
//
// Notify channel semantics: capacity 1, non-blocking send. A single
// notify wakes the consumer to drain everything, including any pushes
// that landed AFTER the consumer started its current drain. The "drain
// everything available" pattern means we never need more than one
// outstanding notify regardless of producer rate.
type localLogRing struct {
	mu      sync.Mutex
	buf     []*v1.LogEntry
	cap     int
	dropped int64
	notify  chan struct{}
}

func newLocalLogRing(capacity int) *localLogRing {
	if capacity <= 0 {
		capacity = 1
	}
	return &localLogRing{
		buf:    make([]*v1.LogEntry, 0, capacity),
		cap:    capacity,
		notify: make(chan struct{}, 1),
	}
}

// push appends an entry, dropping the oldest one if the ring is full.
// Always non-blocking. Coalesces notifications so a burst of pushes
// produces at most one wakeup per drain cycle.
func (r *localLogRing) push(e *v1.LogEntry) {
	r.mu.Lock()
	if len(r.buf) >= r.cap {
		// Drop oldest. Allocation is O(cap) once; subsequent pushes
		// reuse the trimmed backing array up to cap.
		r.buf = append(r.buf[:0], r.buf[1:]...)
		r.dropped++
	}
	r.buf = append(r.buf, e)
	r.mu.Unlock()

	select {
	case r.notify <- struct{}{}:
	default:
		// Already a pending wakeup; the consumer's next drain picks
		// this entry up too. Coalescing.
	}
}

// drain returns all entries currently in the ring plus the cumulative
// dropped count. Resets the ring; the caller compares dropped against
// its previously-observed value to compute the delta.
func (r *localLogRing) drain() ([]*v1.LogEntry, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.buf) == 0 {
		return nil, r.dropped
	}
	out := make([]*v1.LogEntry, len(r.buf))
	copy(out, r.buf)
	r.buf = r.buf[:0]
	return out, r.dropped
}

// droppedCount exposes the cumulative drop counter for tests.
//
//nolint:unused // referenced from logs_test.go
func (r *localLogRing) droppedCount() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dropped
}

// clampGetLogsTail mirrors handleGetLogs' tail-parameter handling:
// zero/negative -> 500, above 10k -> 10k. Returns the adjusted value
// and a flag indicating whether adjustment happened.
func clampGetLogsTail(requested int32) (int, bool) {
	if requested <= 0 {
		// Zero is the proto default - the caller didn't ask for
		// anything specific, so substituting the default is not a
		// clamp. Negative values are nonsensical but we treat them as
		// "use default" rather than erroring; flagging them as clamped
		// would be technically true but useless to clients (they can't
		// retry with a smaller positive value to "uncramp").
		return defaultGetLogsTail, false
	}
	if requested > maxGetLogsTail {
		return maxGetLogsTail, true
	}
	return int(requested), false
}

// fromProtoLogLevel maps the wire enum to the string ValidateClassificationLevel
// expects. Unknown / unspecified collapses to "" so validation rejects it
// rather than silently coercing to a default.
func fromProtoLogLevel(l v1.LogLevel) string {
	switch l {
	case v1.LogLevel_LOG_LEVEL_INFO:
		return "info"
	case v1.LogLevel_LOG_LEVEL_WARN:
		return "warning"
	case v1.LogLevel_LOG_LEVEL_ERROR:
		return "error"
	case v1.LogLevel_LOG_LEVEL_DEBUG:
		// Debug is a valid LogLevel for log entries but NOT a valid
		// classification level (ValidateClassificationLevel only
		// accepts info/warning/error). Return "" so AddClassification
		// rejects it with InvalidArgument instead of silently saving
		// an unparseable rule.
		return ""
	default:
		return ""
	}
}

// toProtoLogLevel maps service.LogLevel (int) to the wire enum.
func toProtoLogLevel(l service.LogLevel) v1.LogLevel {
	switch l {
	case service.LogLevelInfo:
		return v1.LogLevel_LOG_LEVEL_INFO
	case service.LogLevelWarn:
		return v1.LogLevel_LOG_LEVEL_WARN
	case service.LogLevelError:
		return v1.LogLevel_LOG_LEVEL_ERROR
	case service.LogLevelDebug:
		return v1.LogLevel_LOG_LEVEL_DEBUG
	default:
		return v1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// classificationLevelToProto maps the string-typed classification level
// to the wire enum. Unknown strings collapse to UNSPECIFIED so a future
// stored value doesn't masquerade as info.
func classificationLevelToProto(level string) v1.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "info":
		return v1.LogLevel_LOG_LEVEL_INFO
	case "warning":
		return v1.LogLevel_LOG_LEVEL_WARN
	case "error":
		return v1.LogLevel_LOG_LEVEL_ERROR
	default:
		return v1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// toProtoLogEntry converts a single service.LogEntry to its proto
// representation. Maps IsStderr to LogStream STDOUT/STDERR; sets
// LogSource based on service.LogEntry.Source ("local"/"azure"/empty);
// includes timestamp, service, level, and message.
//
// service.LogEntry.Sequence and AzureMetadata are intentionally NOT
// surfaced here: this RPC is for LOCAL logs only (StreamLocalLogs /
// GetLogs scope), so Azure-only fields would be permanently zero on the
// wire and confuse consumers. The Azure surface gets its own RPC.
func toProtoLogEntry(e service.LogEntry) *v1.LogEntry {
	stream := v1.LogStream_LOG_STREAM_STDOUT
	if e.IsStderr {
		stream = v1.LogStream_LOG_STREAM_STDERR
	}
	src := v1.LogSource_LOG_SOURCE_UNSPECIFIED
	switch e.Source {
	case service.LogSourceLocal, "":
		src = v1.LogSource_LOG_SOURCE_LOCAL
	case service.LogSourceAzure:
		src = v1.LogSource_LOG_SOURCE_AZURE
	}
	return &v1.LogEntry{
		Service:   e.Service,
		Message:   e.Message,
		Level:     toProtoLogLevel(e.Level),
		Timestamp: timestamppb.New(e.Timestamp),
		Stream:    stream,
		Source:    src,
	}
}

// toProtoLogEntries converts a slice of LogEntry. Sorted by timestamp
// for determinism (LogManager.GetAllLogs already sorts; per-buffer
// GetRecent is naturally ordered, but the merged path benefits from a
// belt-and-braces sort here).
func toProtoLogEntries(entries []service.LogEntry) []*v1.LogEntry {
	out := make([]*v1.LogEntry, len(entries))
	// Stable sort to preserve relative order of entries with identical
	// timestamps (e.g. burst-logged events from the same service).
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	for i := range entries {
		out[i] = toProtoLogEntry(entries[i])
	}
	return out
}

// toProtoClassification maps a single service.LogClassification.
func toProtoClassification(c service.LogClassification) *v1.Classification {
	return &v1.Classification{
		Text:  c.Text,
		Level: classificationLevelToProto(c.Level),
	}
}

// toProtoClassifications maps a slice of service.LogClassification.
func toProtoClassifications(in []service.LogClassification) []*v1.Classification {
	out := make([]*v1.Classification, len(in))
	for i := range in {
		out[i] = toProtoClassification(in[i])
	}
	return out
}

// defaultProtoPreferences returns the default Preferences value, mirroring
// dashboard.getDefaultPreferences (logs_config.go) exactly. Drift between
// these two is a bug; the test harness asserts equality of the relevant
// fields.
func defaultProtoPreferences() *v1.Preferences {
	return &v1.Preferences{
		Version: "1.0",
		// theme: empty string ("system") is the documented default in
		// the dashboard hook; absence of a stored value falls back to
		// the user-agent's preferred colour scheme. Leaving it empty
		// here matches that behaviour without forcing a value the
		// legacy code never wrote.
		Theme: "",
		Ui: &v1.UIPreferences{
			GridColumns:      2,
			GridAutoFit:      false,
			ViewMode:         "grid",
			SelectedServices: []string{},
		},
		Behavior: &v1.BehaviorPreferences{
			AutoScroll:      true,
			PauseOnScroll:   true,
			TimestampFormat: "hh:mm:ss.sss",
		},
		Copy: &v1.CopyPreferences{
			DefaultFormat:    "plaintext",
			IncludeTimestamp: true,
			IncludeService:   true,
		},
	}
}
