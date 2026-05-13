package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/envinfo"
)

// BroadcastSource is the narrow interface LifecycleHandler.StreamBroadcast
// uses to subscribe to in-process events. dashboard.Server satisfies this
// interface via its embedded *broadcast.Manager; the indirection lets unit
// tests inject a stub manager without spinning up a full dashboard.
type BroadcastSource interface {
	Subscribe(parentCtx context.Context, eventTypes []string) *broadcast.Subscriber
	Unsubscribe(*broadcast.Subscriber)
}

// LifecycleHandler implements azdappv1connect.LifecycleServiceHandler.
//
// Each method is a thin adapter over an existing dashboard primitive:
//   - Ping          → static "ok" + server time + injected version.
//   - GetEnvironment→ dashboard.DetectEnvironment (shared with /api/environment).
//   - StreamBroadcast→ broadcast.Manager (shared with /api/ws fanout).
//
// The handler holds no mutable state and is safe to share across requests.
type LifecycleHandler struct {
	deps Dependencies
}

// Compile-time interface conformance check. Catches signature drift the
// instant the generated interface changes shape.
var _ azdappv1connect.LifecycleServiceHandler = (*LifecycleHandler)(nil)

// NewLifecycleHandler constructs a LifecycleHandler. deps.Broadcast must
// be non-nil; constructing a handler without it is a programming error
// because StreamBroadcast cannot function. We panic on misconfiguration
// at construction time so the dashboard fails fast at startup rather than
// surfacing nil-pointer panics on the first stream request.
func NewLifecycleHandler(deps Dependencies) *LifecycleHandler {
	if deps.Broadcast == nil {
		panic("rpc: NewLifecycleHandler called with nil Dependencies.Broadcast")
	}
	return &LifecycleHandler{deps: deps}
}

// Ping responds with a static "ok", the current server time, and the
// build version. Mirrors GET /api/ping. Side-effect free, no auth.
func (h *LifecycleHandler) Ping(
	ctx context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{
		Status:     "ok",
		ServerTime: timestamppb.Now(),
		Version:    h.deps.Version,
	}), nil
}

// GetEnvironment returns the same Codespace + AZD environment metadata
// the legacy /api/environment endpoint produces, sourced from the shared
// dashboard.DetectEnvironment helper.
func (h *LifecycleHandler) GetEnvironment(
	ctx context.Context,
	_ *connect.Request[v1.GetEnvironmentRequest],
) (*connect.Response[v1.GetEnvironmentResponse], error) {
	env := envinfo.Detect(ctx)

	return connect.NewResponse(&v1.GetEnvironmentResponse{
		Codespace: &v1.CodespaceInfo{
			Enabled:         env.Codespace.Enabled,
			Name:            env.Codespace.Name,
			Domain:          env.Codespace.Domain,
			IsVsCodeDesktop: env.Codespace.IsVsCodeDesktop,
		},
		EnvironmentName: env.EnvironmentName,
	}), nil
}

// StreamBroadcast subscribes the caller to the in-process broadcast fanout
// and forwards every matching event as a StreamBroadcastResponse until one
// of the following happens:
//
//   - The client cancels the request (ctx.Done).
//   - The subscriber's context is cancelled by the broadcast manager
//     (slow-consumer disconnect after broadcast.SubMaxDrops drops).
//   - The dashboard server is shutting down (Manager.StopAll closes the
//     subscriber's channel).
//
// We Unsubscribe in a defer to release the subscriber slot and close the
// channel; the manager's Unsubscribe is idempotent so this is safe even
// when StopAll has already cleared the map.
func (h *LifecycleHandler) StreamBroadcast(
	ctx context.Context,
	req *connect.Request[v1.StreamBroadcastRequest],
	stream *connect.ServerStream[v1.StreamBroadcastResponse],
) error {
	sub := h.deps.Broadcast.Subscribe(ctx, req.Msg.GetEventTypes())
	defer h.deps.Broadcast.Unsubscribe(sub)

	for {
		select {
		case <-ctx.Done():
			// Client disconnect or deadline. Returning nil signals a
			// clean stream close to the Connect runtime; the underlying
			// ctx.Err is surfaced to the client by the framework.
			return nil

		case <-sub.Context().Done():
			// Slow-consumer disconnect. Surface as ResourceExhausted so
			// the client can distinguish back-pressure shed from a normal
			// cancel and decide whether to reconnect with backoff.
			return connect.NewError(
				connect.CodeResourceExhausted,
				fmt.Errorf("broadcast subscriber disconnected after %d dropped events", sub.DroppedCount()),
			)

		case evt, ok := <-sub.Events():
			if !ok {
				// Channel closed by Manager.StopAll (server shutdown).
				// Use Unavailable so well-behaved clients reconnect.
				return connect.NewError(
					connect.CodeUnavailable,
					errors.New("broadcast stream closed: server shutting down"),
				)
			}

			payload, err := payloadToStruct(evt.Payload)
			if err != nil {
				// Producer-side bug: payload must be JSON-serializable.
				// Surface as Internal so it bubbles up to ops without
				// taking the whole stream down silently.
				return connect.NewError(
					connect.CodeInternal,
					fmt.Errorf("encode broadcast payload for %q: %w", evt.Type, err),
				)
			}

			msg := &v1.StreamBroadcastResponse{
				Event: &v1.BroadcastEvent{
					Type:      evt.Type,
					Timestamp: timestamppb.New(evt.Timestamp),
					Payload:   payload,
				},
			}
			if err := stream.Send(msg); err != nil {
				// Send error means the underlying transport is gone; the
				// Connect runtime will translate the error code for the
				// peer. We just propagate.
				return err
			}
		}
	}
}

// payloadToStruct converts a broadcast Event payload into a structpb.Struct
// suitable for the wire. structpb.NewStruct only accepts a closed set of
// JSON-compatible primitives, but our payloads frequently embed Go structs
// (e.g. []*registry.ServiceRegistryEntry) that don't satisfy that contract
// directly.
//
// We round-trip through JSON to guarantee the resulting tree is composed
// entirely of map[string]any / []any / primitive values, exactly the shape
// the legacy /api/ws fanout already serializes today. Cost: one extra
// marshal+unmarshal per broadcast event. Acceptable on the dashboard's
// traffic profile (low QPS, interactive UI) and worth it for the strong
// guarantee that wire output matches /api/ws byte-for-byte semantically.
func payloadToStruct(payload map[string]any) (*structpb.Struct, error) {
	if len(payload) == 0 {
		// Avoid the round-trip for the common no-payload case.
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}, nil
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	out, err := structpb.NewStruct(normalized)
	if err != nil {
		return nil, fmt.Errorf("build structpb: %w", err)
	}
	return out, nil
}
