package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
)

// Tunables for HealthService streaming. The legacy /api/health/stream uses
// the same numbers; keeping them constant here means the Connect surface
// behaves identically to the SSE surface during the parallel-stack window.
const (
	defaultHealthInterval = 5 * time.Second
	minHealthInterval     = 1 * time.Second
	maxHealthInterval     = 60 * time.Second
	heartbeatInterval     = 30 * time.Second

	// stateTransitionsBufferSize bounds the per-subscriber backlog for
	// StreamStateTransitions. The number is taken from ADR-0001's
	// back-pressure table. monitor.StateMonitor already rate-limits
	// non-CRITICAL transitions at the source (default 5min window), so a
	// 256-deep buffer absorbs any realistic burst (e.g. all services
	// crashing simultaneously) without forcing the producer to block.
	stateTransitionsBufferSize = 256

	// maxStateTransitionsBackfill caps client-requested backfill so a
	// well-meaning UI cannot scrape the entire history per reconnect.
	// Mirrors the proto comment ("capped server-side at 100").
	maxStateTransitionsBackfill = 100
)

// HealthSource is the narrow interface HealthHandler uses to run a probe
// and obtain a HealthReport. Production wires it to a per-call
// HealthStreamManager (matching the legacy /api/health/stream behaviour
// of constructing one monitor per stream); tests inject a deterministic
// stub. Returning the full HealthReport keeps the surface narrow without
// forcing this package to know how the report is computed.
type HealthSource interface {
	Check(ctx context.Context, serviceFilter []string) (*healthcheck.HealthReport, error)
}

// HealthSourceFunc adapts a plain function to HealthSource so production
// code can wire HealthStreamManager.PerformHealthCheck directly without
// an extra wrapper struct (matches ProjectSourceFunc / ServiceListerFunc).
type HealthSourceFunc func(ctx context.Context, serviceFilter []string) (*healthcheck.HealthReport, error)

// Check implements HealthSource by delegating to the func.
func (f HealthSourceFunc) Check(ctx context.Context, serviceFilter []string) (*healthcheck.HealthReport, error) {
	return f(ctx, serviceFilter)
}

// StateTransitionSource is the narrow interface StreamStateTransitions
// uses to subscribe to monitor.StateMonitor events. Subscribe MUST return
// a cancel function that removes the listener; otherwise the handler
// would leak a listener per stream. History returns a snapshot of the
// most-recent transitions (capped by the source) used for backfill.
//
// Production wires this to monitor.StateMonitor via StateTransitionSourceAdapter;
// tests inject a stub. The interface intentionally hides any monitor
// internals beyond what's needed to satisfy the proto contract.
type StateTransitionSource interface {
	Subscribe(listener monitor.StateListener) (cancel func())
	History() []monitor.StateTransition
}

// HealthHandler implements azdappv1connect.HealthServiceHandler.
//
// The handler is split across two sources because the proto's three RPCs
// have two distinct origins: GetHealth and StreamHealth both probe via
// HealthSource (the existing healthcheck package), while
// StreamStateTransitions taps monitor.StateMonitor. transitionSource is
// optional - dashboards built without a StateMonitor (the current
// production wiring) can leave it nil, in which case
// StreamStateTransitions returns Unimplemented so well-behaved clients
// can detect "feature off" rather than hanging.
type HealthHandler struct {
	healthSource     HealthSource
	transitionSource StateTransitionSource // optional
}

// Compile-time interface conformance.
var _ azdappv1connect.HealthServiceHandler = (*HealthHandler)(nil)

// NewHealthHandler constructs a HealthHandler. healthSource is required
// (a nil source means GetHealth/StreamHealth can't function);
// transitionSource is optional. We panic on nil healthSource at
// construction time so the dashboard fails fast at startup instead of
// surfacing nil-pointer panics on the first request.
func NewHealthHandler(healthSource HealthSource, transitionSource StateTransitionSource) *HealthHandler {
	if healthSource == nil {
		panic("rpc: NewHealthHandler called with nil HealthSource")
	}
	return &HealthHandler{healthSource: healthSource, transitionSource: transitionSource}
}

// GetHealth runs a one-shot health probe across the requested services
// (or all services if the filter is empty) and returns the full report.
// Mirrors GET /api/health byte-for-byte (modulo proto3 wire encoding).
func (h *HealthHandler) GetHealth(
	ctx context.Context,
	req *connect.Request[v1.GetHealthRequest],
) (*connect.Response[v1.GetHealthResponse], error) {
	report, err := h.healthSource.Check(ctx, req.Msg.GetServiceNames())
	if err != nil {
		// Healthcheck failures here mean the dashboard cannot probe
		// services at all (e.g. azure.yaml missing, registry corrupt).
		// Surface as Internal to match the legacy 500; a more granular
		// classification would be a behaviour change vs. /api/health.
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.GetHealthResponse{
		Results:     toProtoResults(report.Services),
		GeneratedAt: timestamppb.New(report.Timestamp),
	}), nil
}

// StreamHealth runs a periodic health probe and emits one HealthEvent per
// tick (a HealthChange per state transition followed by a HealthReport),
// plus a Heartbeat every 30s during idle periods. Replaces the SSE
// /api/health/stream endpoint.
//
// Back-pressure: last-value-wins. The producer (this handler) and the
// consumer (Connect runtime → HTTP/2 stream) are the same goroutine, so
// there is no separate queue to overflow. If the network is slow,
// stream.Send blocks and the next ticker fire is naturally postponed -
// the result is exactly the "drop intermediate, keep latest" semantic
// the ADR specifies, without an explicit coalescing buffer.
func (h *HealthHandler) StreamHealth(
	ctx context.Context,
	req *connect.Request[v1.StreamHealthRequest],
	stream *connect.ServerStream[v1.StreamHealthResponse],
) error {
	interval := clampHealthInterval(time.Duration(req.Msg.GetIntervalSeconds()) * time.Second)
	serviceFilter := req.Msg.GetServiceNames()

	// Per-stream change detection state. Mirrors HealthStreamManager's
	// previousStates map, but scoped to this subscription so concurrent
	// streams don't leak state into each other.
	previousStates := make(map[string]healthcheck.HealthStatus)

	sendUpdate := func() error {
		report, err := h.healthSource.Check(ctx, serviceFilter)
		if err != nil {
			// A probe failure does NOT terminate the stream in the
			// legacy SSE implementation - it logs and skips - because a
			// transient registry hiccup shouldn't drop every connected
			// dashboard. We mirror that: surface as Internal only when
			// the context is dead, otherwise log and continue.
			if ctx.Err() != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
			slog.Warn(
				"StreamHealth: probe failed; skipping tick",
				"err", err.Error(),
				"services", serviceFilter,
			)
			return nil
		}

		// Emit per-service change events first so consumers see the
		// transition before the snapshot that already reflects it.
		// Deterministic order keeps tests stable.
		results := report.Services
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].ServiceName < results[j].ServiceName
		})
		for _, result := range results {
			prev, exists := previousStates[result.ServiceName]
			if exists && prev != result.Status {
				changeMsg := &v1.StreamHealthResponse{
					Event: &v1.HealthEvent{
						Event: &v1.HealthEvent_Change{
							Change: &v1.HealthChange{
								ServiceName:   result.ServiceName,
								PreviousState: toProtoHealthState(prev),
								CurrentState:  toProtoHealthState(result.Status),
								Message:       result.Error,
								ChangedAt:     timestamppb.New(result.Timestamp),
							},
						},
					},
				}
				if err := stream.Send(changeMsg); err != nil {
					return err
				}
			}
			previousStates[result.ServiceName] = result.Status
		}

		// Snapshot.
		reportMsg := &v1.StreamHealthResponse{
			Event: &v1.HealthEvent{
				Event: &v1.HealthEvent_Report{
					Report: &v1.HealthReport{
						Results:     toProtoResults(results),
						GeneratedAt: timestamppb.New(report.Timestamp),
					},
				},
			},
		}
		return stream.Send(reportMsg)
	}

	// Initial probe matches legacy SSE behaviour ("send initial health
	// check immediately") so the client doesn't wait `interval` for the
	// first useful payload.
	if err := sendUpdate(); err != nil {
		return err
	}

	healthTicker := time.NewTicker(interval)
	defer healthTicker.Stop()
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnect or deadline. nil is the clean-close
			// signal; the framework propagates ctx.Err to the peer.
			return nil

		case <-healthTicker.C:
			if err := sendUpdate(); err != nil {
				return err
			}

		case <-heartbeatTicker.C:
			heartbeatMsg := &v1.StreamHealthResponse{
				Event: &v1.HealthEvent{
					Event: &v1.HealthEvent_Heartbeat{
						Heartbeat: &v1.Heartbeat{ServerTime: timestamppb.Now()},
					},
				},
			}
			if err := stream.Send(heartbeatMsg); err != nil {
				return err
			}
		}
	}
}

// StreamStateTransitions emits severity-classified state transitions
// from monitor.StateMonitor as they happen, optionally preceded by a
// bounded backfill of recent history.
//
// Back-pressure: block-producer with a 256-event per-subscriber buffer.
// monitor.StateMonitor.notifyListeners spawns a goroutine per listener
// invocation, so blocking on the buffer does NOT propagate to the
// underlying StateMonitor's check loop - it stacks up listener
// goroutines at the source. Combined with monitor's source-side rate
// limiting (5 min window for non-CRITICAL), the goroutine pile-up is
// bounded by typical operational pace. We deliberately do not drop:
// CRITICAL transitions ("service crashed") are operational events the
// user MUST see, so when both the buffer and the stream are healthy,
// every transition is delivered. The only drop path is when the stream
// itself is going away (ctx.Done), which is correct behaviour - there is
// no consumer left to deliver to.
func (h *HealthHandler) StreamStateTransitions(
	ctx context.Context,
	req *connect.Request[v1.StreamStateTransitionsRequest],
	stream *connect.ServerStream[v1.StreamStateTransitionsResponse],
) error {
	if h.transitionSource == nil {
		// Fail loud: clients can detect "feature off" via Unimplemented
		// instead of waiting on a stream that will never produce.
		return connect.NewError(
			connect.CodeUnimplemented,
			errors.New("StreamStateTransitions requires a StateTransitionSource; none configured"),
		)
	}

	minSeverity := req.Msg.GetMinSeverity()
	serviceFilter := stringSet(req.Msg.GetServiceNames())

	// Backfill: clamp to the proto-documented cap. Negative values from
	// a misbehaving client default to zero (no backfill).
	backfill := req.Msg.GetBackfill()
	if backfill < 0 {
		backfill = 0
	}
	if backfill > maxStateTransitionsBackfill {
		backfill = maxStateTransitionsBackfill
	}

	if backfill > 0 {
		history := h.transitionSource.History()
		matching := make([]monitor.StateTransition, 0, len(history))
		for _, t := range history {
			if matchesTransitionFilter(t, minSeverity, serviceFilter) {
				matching = append(matching, t)
			}
		}
		// Take the tail, then send oldest-first so consumers see
		// transitions in temporal order.
		if int32(len(matching)) > backfill {
			matching = matching[len(matching)-int(backfill):]
		}
		for i := range matching {
			if err := stream.Send(toStateTransitionResponse(&matching[i])); err != nil {
				return err
			}
		}
	}

	// Per-stream context: cancelled on return so any listener-goroutine
	// already in flight unblocks instead of dangling. Defer ordering is
	// LIFO, so unsubscribeFn runs FIRST (stops new invocations), then
	// streamCancel runs (releases any in-flight ones blocked on `buf`).
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	buf := make(chan monitor.StateTransition, stateTransitionsBufferSize)
	var droppedOnShutdown atomic.Int64

	unsubscribeFn := h.transitionSource.Subscribe(func(t monitor.StateTransition) {
		if !matchesTransitionFilter(t, minSeverity, serviceFilter) {
			return
		}
		// Fast path: buffer has room. The compile-time bound (256) plus
		// monitor's source-side rate limit make this the common case.
		select {
		case buf <- t:
			return
		default:
		}
		// Slow path: buffer full. Block until either the consumer
		// drains it or the stream is going away. This is the
		// "block-producer" half of the policy.
		select {
		case buf <- t:
		case <-streamCtx.Done():
			// Stream is closing; drop and account for it. Logged at the
			// outer return so we get one summary per stream rather than
			// per-event spam.
			droppedOnShutdown.Add(1)
		}
	})
	defer unsubscribeFn()

	for {
		select {
		case <-ctx.Done():
			if d := droppedOnShutdown.Load(); d > 0 {
				slog.Warn(
					"StreamStateTransitions: dropped transitions during shutdown",
					"count", d,
				)
			}
			return nil
		case t := <-buf:
			if err := stream.Send(toStateTransitionResponse(&t)); err != nil {
				return err
			}
		}
	}
}

// clampHealthInterval mirrors parseHealthStreamParams in the legacy SSE
// handler: zero/negative requests default to 5s, and the value is bounded
// to [1s, 60s]. We clamp silently (no error) to match the legacy
// behaviour where a too-aggressive ?interval=100ms simply got bumped up.
func clampHealthInterval(d time.Duration) time.Duration {
	if d <= 0 {
		return defaultHealthInterval
	}
	if d < minHealthInterval {
		return minHealthInterval
	}
	if d > maxHealthInterval {
		return maxHealthInterval
	}
	return d
}

// toProtoHealthState maps the string-typed healthcheck.HealthStatus to
// the proto enum. Unknown strings collapse to UNSPECIFIED so a future
// healthcheck status doesn't silently masquerade as HEALTHY.
func toProtoHealthState(s healthcheck.HealthStatus) v1.HealthState {
	switch s {
	case healthcheck.HealthStatusHealthy:
		return v1.HealthState_HEALTH_STATE_HEALTHY
	case healthcheck.HealthStatusDegraded:
		return v1.HealthState_HEALTH_STATE_DEGRADED
	case healthcheck.HealthStatusUnhealthy:
		return v1.HealthState_HEALTH_STATE_UNHEALTHY
	case healthcheck.HealthStatusStarting:
		return v1.HealthState_HEALTH_STATE_STARTING
	case healthcheck.HealthStatusUnknown:
		return v1.HealthState_HEALTH_STATE_UNKNOWN
	default:
		return v1.HealthState_HEALTH_STATE_UNSPECIFIED
	}
}

// toProtoSeverity maps monitor.Severity to the proto enum.
func toProtoSeverity(s monitor.Severity) v1.Severity {
	switch s {
	case monitor.SeverityInfo:
		return v1.Severity_SEVERITY_INFO
	case monitor.SeverityWarning:
		return v1.Severity_SEVERITY_WARNING
	case monitor.SeverityCritical:
		return v1.Severity_SEVERITY_CRITICAL
	default:
		return v1.Severity_SEVERITY_UNSPECIFIED
	}
}

// toProtoServiceStatus maps the textual ServiceState.Status (matching the
// legacy /api/services strings) to the proto enum. Unknown strings
// collapse to UNSPECIFIED for the same reason as toProtoHealthState.
func toProtoServiceStatus(s string) v1.ServiceStatus {
	switch s {
	case "stopped", "not-running":
		return v1.ServiceStatus_SERVICE_STATUS_STOPPED
	case "starting":
		return v1.ServiceStatus_SERVICE_STATUS_STARTING
	case "ready", "running":
		return v1.ServiceStatus_SERVICE_STATUS_READY
	case "degraded":
		return v1.ServiceStatus_SERVICE_STATUS_DEGRADED
	case "error":
		return v1.ServiceStatus_SERVICE_STATUS_ERROR
	case "stopping":
		return v1.ServiceStatus_SERVICE_STATUS_STOPPING
	default:
		return v1.ServiceStatus_SERVICE_STATUS_UNSPECIFIED
	}
}

// toProtoResults converts a slice of healthcheck results to their proto
// representation. Allocates one slice; per-element conversion is O(1)
// outside of the details map round-trip.
func toProtoResults(results []healthcheck.HealthCheckResult) []*v1.HealthCheckResult {
	out := make([]*v1.HealthCheckResult, len(results))
	for i, r := range results {
		out[i] = &v1.HealthCheckResult{
			ServiceName: r.ServiceName,
			State:       toProtoHealthState(r.Status),
			Message:     r.Error,
			CheckedAt:   timestamppb.New(r.Timestamp),
			LatencyMs:   r.ResponseTime.Milliseconds(),
			Details:     detailsToStringMap(r.Details),
		}
	}
	return out
}

// detailsToStringMap flattens healthcheck.HealthCheckResult.Details
// (map[string]interface{}) into the proto's map<string,string>. Strings
// pass through; other primitives stringify via strconv; complex values
// JSON-marshal (so a nested map shows up as the JSON literal). This is
// lossy by design: the Details map exists for human-readable diagnostics,
// not structured consumption, and proto's map<string,string> matches that
// intent. Switching to google.protobuf.Struct here would force every
// dashboard consumer through a Value-tree decoder for negligible benefit.
func detailsToStringMap(in map[string]interface{}) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = stringifyDetailValue(v)
	}
	return out
}

func stringifyDetailValue(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case fmt.Stringer:
		return x.String()
	default:
		// Fallback: JSON-encode. We swallow the error because the only
		// way json.Marshal fails on a value that already round-tripped
		// through HealthCheckResult.Details is an unsupported type
		// (channels, funcs), and surfacing that mid-stream would be
		// hostile to operators trying to debug their probes. Empty
		// string is the legible failure mode.
		raw, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

// stringSet builds a lookup set from a slice. An empty slice yields a
// nil map, which the matcher treats as "no filter" (allow all).
func stringSet(in []string) map[string]struct{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}

// matchesTransitionFilter applies the proto request's severity floor and
// service filter to a candidate transition. A nil serviceFilter (empty
// repeated field on the wire) means "all services". minSeverity of
// SEVERITY_UNSPECIFIED is treated as no floor (matches everything),
// matching proto3 zero-value semantics so a client that forgets to set
// the field doesn't get an empty stream.
func matchesTransitionFilter(t monitor.StateTransition, minSeverity v1.Severity, serviceFilter map[string]struct{}) bool {
	if serviceFilter != nil {
		if _, ok := serviceFilter[t.ServiceName]; !ok {
			return false
		}
	}
	if minSeverity > v1.Severity_SEVERITY_UNSPECIFIED {
		if toProtoSeverity(t.Severity) < minSeverity {
			return false
		}
	}
	return true
}

// toStateTransitionResponse builds the wire response for a single
// transition. The proto carries an `id` field that monitor.StateTransition
// does NOT have; we synthesise one from timestamp + service + severity so
// clients can dedupe across reconnects (backfill replays may overlap
// with live events). The format is documented in the proto comment.
func toStateTransitionResponse(t *monitor.StateTransition) *v1.StreamStateTransitionsResponse {
	return &v1.StreamStateTransitionsResponse{
		Transition: &v1.StateTransition{
			Id:          synthesiseTransitionID(t),
			Timestamp:   timestamppb.New(t.Timestamp),
			ServiceName: t.ServiceName,
			Severity:    toProtoSeverity(t.Severity),
			EventType:   classifyTransitionEventType(t),
			Message:     t.Description,
			Previous:    toServiceStateSnapshot(t.FromState),
			Current:     toServiceStateSnapshot(t.ToState),
		},
	}
}

// synthesiseTransitionID gives every emitted transition a stable,
// dedupe-friendly identifier. monitor.StateTransition has no native ID
// (the package wasn't designed for fan-out); rather than pollute the
// monitor API for one consumer, we derive an ID from the fields the
// transition already carries. UnixNano + service + severity is unique in
// practice because the source rate-limits non-CRITICAL transitions per
// service to one per 5min, so collisions would require two CRITICAL
// transitions for the same service in the same nanosecond.
func synthesiseTransitionID(t *monitor.StateTransition) string {
	return fmt.Sprintf(
		"%d-%s-%d",
		t.Timestamp.UnixNano(),
		t.ServiceName,
		int(t.Severity),
	)
}

// classifyTransitionEventType folds the StateMonitor's free-form
// Description into the small set of event_type strings the proto comment
// enumerates ("crashed", "port-unbound", "degraded", ...). We classify
// from the FROM/TO state pair because Description is operator-facing
// English; classifying from structured fields keeps the mapping stable
// across description rewording. Unknown shapes return "transition" so
// the field is never empty.
func classifyTransitionEventType(t *monitor.StateTransition) string {
	to := t.ToState
	from := t.FromState

	// New service appearing in monitor.
	if from == nil && to != nil {
		return "registered"
	}
	// Service disappearing.
	if to == nil {
		return "deregistered"
	}

	// Process died.
	if from.PIDValid && !to.PIDValid {
		return "crashed"
	}
	// Port stopped listening.
	if from.PortListens && !to.PortListens {
		return "port-unbound"
	}
	// Health degraded specifically (vs full unhealthy).
	if from.Health == "healthy" && to.Health == "degraded" {
		return "degraded"
	}
	// Health recovered.
	if from.Health != "healthy" && to.Health == "healthy" {
		return "recovered"
	}
	// Health failed.
	if from.Health == "healthy" && to.Health == "unhealthy" {
		return "unhealthy"
	}
	return "transition"
}

func toServiceStateSnapshot(s *monitor.ServiceState) *v1.ServiceStateSnapshot {
	if s == nil {
		return nil
	}
	return &v1.ServiceStateSnapshot{
		Status: toProtoServiceStatus(s.Status),
		Health: toProtoHealthState(healthcheck.HealthStatus(s.Health)),
		Pid:    int32(s.PID),
		Port:   int32(s.Port),
	}
}
