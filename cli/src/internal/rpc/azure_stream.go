package rpc

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// Stream tunables (see ADR-0001 + commit-B-1 plan):
//
//   - Polling: per-subscriber 256-buffer with backpressureDelay growing from
//     50ms to 2s on consecutive blocked sends, ON TOP of azure.PollingState's
//     own 5s..60s adaptive interval.
//   - Realtime: drop-OLDEST ring with the same 256 capacity. Drops emit an
//     AzureDroppedNotice carrying count + reason="realtime_buffer_overflow".
//   - StreamStatus emitted on first connect, mode flip, retry/recovery.
const (
	azureStreamBufferSize       = 256
	azureBackpressureDelayInit  = 50 * time.Millisecond
	azureBackpressureDelayMax   = 2 * time.Second
	azureBackpressureDelayMul   = 2
	azureRealtimeFlushInterval  = 100 * time.Millisecond
	azureDefaultBackfillSeconds = int64(30 * 60)
)

// StreamAzureLogs implements the streaming RPC. Polling is the lossless
// baseline; realtime falls back to polling when the streamer cannot be
// constructed (mirrors the legacy WebSocket behaviour). Polling's loop
// extends nextInterval to max(PollingState.NextDelay, backpressureDelay)
// so a slow consumer slows the polling cadence (Log Analytics queries are
// metered - skipping a poll is the right answer).
func (h *AzureHandler) StreamAzureLogs(
	ctx context.Context,
	req *connect.Request[v1.StreamAzureLogsRequest],
	stream *connect.ServerStream[v1.StreamAzureLogsResponse],
) error {
	if req.Msg.Service == "" {
		return connect.NewError(connect.CodeInvalidArgument,
			errors.New("service is required"))
	}

	backfill := time.Duration(azureDefaultBackfillSeconds) * time.Second
	if req.Msg.BackfillSeconds > 0 {
		backfill = time.Duration(req.Msg.BackfillSeconds) * time.Second
	}

	// Send initial StreamStatus so the dashboard's polling-health UI
	// renders before the first entry arrives (Connect-over-HTTP/1
	// doesn't flush headers until first Send).
	mode := "polling"
	if req.Msg.Realtime {
		mode = "realtime"
	}
	if err := sendAzureStatus(stream, "connected", mode, 0, "", nil); err != nil {
		return err
	}

	if req.Msg.Realtime {
		err := h.streamRealtime(ctx, req.Msg.Service, backfill, stream)
		if err == nil || ctx.Err() != nil {
			return err
		}
		// Realtime setup failed - emit a status flip and fall through
		// to polling (parity with handleAzureLogsStream).
		slog.Warn("azure realtime streamer failed; falling back to polling",
			"service", req.Msg.Service, "error", err)
		_ = sendAzureStatus(stream, "degraded", "polling", 0, err.Error(), nil)
	}

	return h.streamPolling(ctx, req.Msg.Service, backfill, stream)
}

// streamPolling runs the adaptive-interval polling loop. PollingState owns
// the base cadence (5s..60s); backpressureDelay extends it whenever the
// per-subscriber send blocks.
func (h *AzureHandler) streamPolling(
	ctx context.Context,
	serviceName string,
	backfill time.Duration,
	stream *connect.ServerStream[v1.StreamAzureLogsResponse],
) error {
	state := azure.NewPollingState(0)
	since := backfill
	bp := time.Duration(0)
	lastSentTimestamp := time.Time{}

	// First poll uses the requested backfill window. Subsequent polls
	// shorten to PollingState's recommended interval to avoid overlap.
	for {
		cfg := azure.StandaloneLogsConfig{
			Services: []string{serviceName},
			Since:    since,
			Limit:    azureStreamBufferSize,
		}

		logs, err := h.store.FetchLogs(ctx, cfg)
		if err != nil && !isNoResults(err) {
			state.RecordFailure(err)
			hh := state.GetHealth()
			_ = sendAzureStatus(stream, hh.Status, "polling", int32(hh.ConsecutiveFails), err.Error(), &hh.NextRetry)
		} else {
			state.RecordSuccess()
		}

		// Drain new entries (skip ones we already sent).
		blocked := false
		for _, l := range logs {
			if !l.Timestamp.After(lastSentTimestamp) {
				continue
			}
			err := sendWithBackpressure(stream, &v1.StreamAzureLogsResponse{
				Event: &v1.StreamAzureLogsResponse_Entry{Entry: toProtoAzureLogEntry(l)},
			})
			if err != nil {
				if errors.Is(err, errStreamBlocked) {
					blocked = true
					break
				}
				return err
			}
			lastSentTimestamp = l.Timestamp
		}

		if blocked {
			if bp == 0 {
				bp = azureBackpressureDelayInit
			} else if bp < azureBackpressureDelayMax {
				bp = bp * azureBackpressureDelayMul
				if bp > azureBackpressureDelayMax {
					bp = azureBackpressureDelayMax
				}
			}
		} else {
			bp = 0
		}

		next := state.NextDelay()
		if bp > next {
			next = bp
		}
		// Next poll only needs to cover the gap since the last fetch.
		since = next + 5*time.Second

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(next):
		}
	}
}

// streamRealtime runs the per-resource event-hub streamer with a drop-
// oldest local ring. AzureDroppedNotice is emitted whenever the ring
// drops one or more entries since the last drain.
func (h *AzureHandler) streamRealtime(
	ctx context.Context,
	serviceName string,
	backfill time.Duration,
	stream *connect.ServerStream[v1.StreamAzureLogsResponse],
) error {
	resource, err := h.store.ResolveResource(ctx, serviceName)
	if err != nil {
		return err
	}
	if resource == nil {
		return errors.New("resource not found")
	}

	cred, err := h.store.NewLogAnalyticsCredential()
	if err != nil {
		return err
	}

	streamer, err := h.store.NewRealtimeStreamer(resource.ResourceType, azure.StreamerConfig{
		SubscriptionID: resource.SubscriptionID,
		ResourceGroup:  resource.ResourceGroup,
		ResourceName:   resource.Name,
		ServiceName:    resource.ServiceName,
		Credential:     cred,
	})
	if err != nil {
		return err
	}

	logsCh := make(chan azure.LogEntry, azureStreamBufferSize)
	ring := newAzureLogRing(azureStreamBufferSize)

	// Pump source channel into the drop-oldest ring.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var pumpWG sync.WaitGroup
	pumpWG.Go(func() {
		_ = streamer.Start(streamCtx, logsCh)
	})
	pumpWG.Go(func() {
		for {
			select {
			case <-streamCtx.Done():
				return
			case e, ok := <-logsCh:
				if !ok {
					return
				}
				ring.push(toProtoAzureLogEntry(e))
			}
		}
	})

	defer func() {
		_ = streamer.Stop()
		cancel()
		pumpWG.Wait()
	}()

	// Best-effort backfill before realtime begins.
	if backfill > 0 {
		fetched, ferr := h.store.FetchLogs(ctx, azure.StandaloneLogsConfig{
			Services: []string{serviceName},
			Since:    backfill,
			Limit:    azureStreamBufferSize,
		})
		if ferr == nil {
			for _, l := range fetched {
				if err := stream.Send(&v1.StreamAzureLogsResponse{
					Event: &v1.StreamAzureLogsResponse_Entry{Entry: toProtoAzureLogEntry(l)},
				}); err != nil {
					return err
				}
			}
		}
	}

	ticker := time.NewTicker(azureRealtimeFlushInterval)
	defer ticker.Stop()

	var lastDropped int64
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ring.notify:
		case <-ticker.C:
		}

		entries, dropped := ring.drain()
		for _, e := range entries {
			if err := stream.Send(&v1.StreamAzureLogsResponse{
				Event: &v1.StreamAzureLogsResponse_Entry{Entry: e},
			}); err != nil {
				return err
			}
		}

		if delta := dropped - lastDropped; delta > 0 {
			lastDropped = dropped
			if err := stream.Send(&v1.StreamAzureLogsResponse{
				Event: &v1.StreamAzureLogsResponse_Dropped{
					Dropped: &v1.AzureDroppedNotice{
						Count:  delta,
						At:     timestamppb.Now(),
						Reason: "realtime_buffer_overflow",
					},
				},
			}); err != nil {
				return err
			}
		}
	}
}

// sendAzureStatus emits a StreamStatus event. Returns the underlying
// stream.Send error so the caller can unwind the RPC.
func sendAzureStatus(
	stream *connect.ServerStream[v1.StreamAzureLogsResponse],
	status, mode string,
	consecutiveFails int32,
	errMsg string,
	nextRetry *time.Time,
) error {
	s := &v1.StreamStatus{
		Status:           status,
		Mode:             mode,
		ConsecutiveFails: consecutiveFails,
		Error:            errMsg,
	}
	if nextRetry != nil && !nextRetry.IsZero() {
		s.NextRetry = timestamppb.New(*nextRetry)
	}
	return stream.Send(&v1.StreamAzureLogsResponse{
		Event: &v1.StreamAzureLogsResponse_Status{Status: s},
	})
}

// errStreamBlocked is the sentinel returned by sendWithBackpressure when
// the polling-mode send cannot complete within the per-attempt budget,
// signalling that the caller should slow its production rate.
var errStreamBlocked = errors.New("stream blocked")

// sendWithBackpressure sends one frame with a short timeout. On timeout
// the caller backs off; on transport error it propagates.
//
// connect-go's ServerStream.Send is synchronous, so we approximate
// "non-blocking" by running it in a goroutine and racing it against a
// ticker. The 100ms budget is small enough not to dominate the polling
// loop's cadence and large enough to amortise the goroutine cost.
func sendWithBackpressure(
	stream *connect.ServerStream[v1.StreamAzureLogsResponse],
	msg *v1.StreamAzureLogsResponse,
) error {
	done := make(chan error, 1)
	go func() { done <- stream.Send(msg) }()
	select {
	case err := <-done:
		return err
	case <-time.After(100 * time.Millisecond):
		return errStreamBlocked
	}
}

// =============================================================================
// localAzureLogRing - drop-OLDEST ring used by the realtime path.
// =============================================================================

// localAzureLogRing mirrors localLogRing in logs.go: bounded, drop-oldest,
// coalescing notify channel. Kept separate so the two streams' failure
// modes can evolve independently.
type localAzureLogRing struct {
	mu      sync.Mutex
	buf     []*v1.LogEntry
	cap     int
	dropped int64
	notify  chan struct{}
}

func newAzureLogRing(capacity int) *localAzureLogRing {
	if capacity <= 0 {
		capacity = 1
	}
	return &localAzureLogRing{
		buf:    make([]*v1.LogEntry, 0, capacity),
		cap:    capacity,
		notify: make(chan struct{}, 1),
	}
}

func (r *localAzureLogRing) push(e *v1.LogEntry) {
	r.mu.Lock()
	if len(r.buf) >= r.cap {
		r.buf = append(r.buf[:0], r.buf[1:]...)
		r.dropped++
	}
	r.buf = append(r.buf, e)
	r.mu.Unlock()
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

func (r *localAzureLogRing) drain() ([]*v1.LogEntry, int64) {
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

//nolint:unused // referenced from azure_test.go
func (r *localAzureLogRing) droppedCount() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dropped
}
