// Package broadcast implements an in-process publish/subscribe fanout used
// by the dashboard server to deliver coarse-grained UI events to multiple
// transports (Connect StreamBroadcast, future SSE, future test harness)
// from a single producer call.
//
// It lives in a neutral subpackage so both the dashboard package (the
// producer) and the rpc package (a consumer that turns Manager events into
// Connect stream messages) can import it without an import cycle.
//
// Back-pressure policy: drop-oldest with slow-consumer disconnect. See
// Manager.Emit for the full semantics. The thresholds intentionally mirror
// the WebSocket fanout's recordWriteFailure / WebSocketMaxWriteFailures
// behavior so the new transport is no more or less permissive than the
// existing /api/ws path.
package broadcast

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Tuning constants for the per-subscriber bounded channel.
const (
	// SubBufferSize is the per-subscriber channel buffer size. Sized to
	// absorb a small burst without requiring sample-for-sample consumer
	// keepup. Picked at 64 to match the order of magnitude of
	// WebSocketLogChannelBuffer while staying memory-cheap when many
	// subscribers exist.
	SubBufferSize = 64

	// SubMaxDrops is the cumulative dropped-event count after which a slow
	// consumer is forcibly disconnected. A consumer that cannot keep up
	// with even a 64-event burst over its lifetime is genuinely broken and
	// must shed load to protect other subscribers and the producer.
	SubMaxDrops = 32
)

// Event is the in-process representation of a broadcast event. It mirrors
// the on-the-wire azdapp.v1.BroadcastEvent shape (type/timestamp/payload)
// but uses a Go-native payload (map[string]any) so this package and its
// producers do not depend on the generated proto types. Connect handlers
// translate Payload to structpb.Struct at the wire boundary.
type Event struct {
	Type      string
	Timestamp time.Time
	Payload   map[string]any
}

// Subscriber represents a single consumer of the broadcast fanout. It owns
// a bounded channel and a context that the Manager cancels when the
// consumer falls too far behind.
type Subscriber struct {
	ch           chan Event
	eventFilter  map[string]struct{} // nil/empty = subscribe to all
	droppedCount atomic.Int64
	ctx          context.Context
	cancel       context.CancelFunc
}

// Events returns the receive-only event channel. The channel is closed
// when Unsubscribe is called, when StopAll is called on the parent
// Manager, or when the subscriber's context is cancelled (slow-consumer
// disconnect).
func (s *Subscriber) Events() <-chan Event { return s.ch }

// Context returns the subscriber's context. It is cancelled when the
// Manager forcibly disconnects this subscriber (slow consumer), when
// StopAll runs, or when the parent context the subscription was created
// with is cancelled.
func (s *Subscriber) Context() context.Context { return s.ctx }

// DroppedCount returns the running tally of events dropped for this
// subscriber. Useful for observability and for tests that exercise the
// slow-consumer path.
func (s *Subscriber) DroppedCount() int64 { return s.droppedCount.Load() }

// matches reports whether evt should be delivered to this subscriber given
// its event-type filter. An empty filter matches every event.
func (s *Subscriber) matches(evt Event) bool {
	if len(s.eventFilter) == 0 {
		return true
	}
	_, ok := s.eventFilter[evt.Type]
	return ok
}

// Manager owns the set of active subscribers and the fanout machinery. The
// zero value is ready to use; New is provided for symmetry and to make
// future construction-time options painless to introduce.
type Manager struct {
	mu   sync.Mutex
	subs map[*Subscriber]struct{}
}

// New returns a ready-to-use Manager. Equivalent to &Manager{} today, but
// callers should prefer this constructor so we can add option args without
// breaking them later.
func New() *Manager { return &Manager{} }

// Subscribe registers a new subscriber. The returned Subscriber's channel
// will receive every event whose type matches eventTypes (empty filter =
// all events).
//
// The subscriber's context is a child of parentCtx; cancelling parentCtx
// (e.g., when the Connect stream client disconnects) cleanly tears down
// the subscription. The Manager may also cancel the subscriber's context
// if the consumer falls behind by more than SubMaxDrops events.
//
// Callers MUST call Unsubscribe when done to release resources.
func (m *Manager) Subscribe(parentCtx context.Context, eventTypes []string) *Subscriber {
	ctx, cancel := context.WithCancel(parentCtx)
	sub := &Subscriber{
		ch:     make(chan Event, SubBufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
	if len(eventTypes) > 0 {
		sub.eventFilter = make(map[string]struct{}, len(eventTypes))
		for _, t := range eventTypes {
			if t != "" {
				sub.eventFilter[t] = struct{}{}
			}
		}
	}

	m.mu.Lock()
	if m.subs == nil {
		m.subs = make(map[*Subscriber]struct{})
	}
	m.subs[sub] = struct{}{}
	m.mu.Unlock()

	return sub
}

// Unsubscribe removes a subscriber from the fanout and closes its event
// channel. Safe to call multiple times; idempotent.
func (m *Manager) Unsubscribe(sub *Subscriber) {
	if sub == nil {
		return
	}
	m.mu.Lock()
	_, present := m.subs[sub]
	if present {
		delete(m.subs, sub)
	}
	m.mu.Unlock()

	if !present {
		// Already unsubscribed; do not double-close the channel.
		return
	}

	// Cancel context first so any in-flight consumer wakes up, then close
	// the channel. Order matters: closing first could cause a consumer in
	// the middle of select to read a zero-value event after channel close.
	sub.cancel()
	close(sub.ch)
}

// Emit fans an event out to every matching subscriber using a drop-oldest,
// disconnect-slow-consumer policy:
//
//   - Non-blocking send first. If the buffer has room, deliver immediately.
//   - If full, drop one oldest event from the buffer to make room and
//     retry. Increment the per-subscriber dropped counter.
//   - Once a subscriber has dropped SubMaxDrops events cumulatively, cancel
//     its context. Its stream handler will observe ctx.Done and tear down.
//     Unsubscribe will be called by the handler's defer; we do not call it
//     here to avoid holding the manager mutex across channel close.
//
// This mirrors the existing WebSocket broadcast policy in dashboard's
// websocket.go (recordWriteFailure / WebSocketMaxWriteFailures) so the
// Connect transport is no more permissive about slow consumers than the
// /api/ws path is today, and no less.
func (m *Manager) Emit(evt Event) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}

	// Snapshot subscribers under the lock; deliver outside the lock so a
	// slow consumer cannot block other subscribers or the producer.
	m.mu.Lock()
	if len(m.subs) == 0 {
		m.mu.Unlock()
		return
	}
	subs := make([]*Subscriber, 0, len(m.subs))
	for sub := range m.subs {
		subs = append(subs, sub)
	}
	m.mu.Unlock()

	for _, sub := range subs {
		if !sub.matches(evt) {
			continue
		}
		// Skip subscribers whose context is already cancelled; their
		// handler will Unsubscribe shortly.
		if sub.ctx.Err() != nil {
			continue
		}
		deliver(sub, evt)
	}
}

// deliver implements the per-subscriber drop-oldest send. Extracted so the
// test suite can exercise the drop path deterministically.
func deliver(sub *Subscriber, evt Event) {
	// Fast path: room in the buffer.
	select {
	case sub.ch <- evt:
		return
	default:
	}

	// Slow path: buffer is full. Drop oldest, then deliver new event.
	// Bounded by a single drain-then-send so we cannot spin if a concurrent
	// consumer is also draining; if the second send still cannot proceed,
	// we count it as a drop of the new event itself.
	select {
	case <-sub.ch:
		dropped := sub.droppedCount.Add(1)
		select {
		case sub.ch <- evt:
		default:
			dropped = sub.droppedCount.Add(1)
		}
		if dropped >= SubMaxDrops {
			slog.Warn("broadcast subscriber exceeded max drops, disconnecting",
				"dropped", dropped, "event_type", evt.Type)
			sub.cancel()
		}
	default:
		// Buffer drained between the two selects (concurrent consumer).
		// Try one final non-blocking send.
		select {
		case sub.ch <- evt:
		default:
			dropped := sub.droppedCount.Add(1)
			if dropped >= SubMaxDrops {
				slog.Warn("broadcast subscriber exceeded max drops, disconnecting",
					"dropped", dropped, "event_type", evt.Type)
				sub.cancel()
			}
		}
	}
}

// StopAll closes every active subscriber's channel so stream handlers
// observe end-of-stream and exit. Called from Server.Stop during shutdown.
//
// Note: this does NOT cancel each subscriber's context. The context is
// reserved for the slow-consumer disconnect signal (sub.cancel() in
// deliver), so handlers can distinguish "server shut down" (channel
// closed → CodeUnavailable) from "you were too slow"
// (ctx canceled → CodeResourceExhausted). Cancelling both here would
// race the two paths, surfacing the wrong error code on shutdown.
//
// Race-freedom: the map is cleared inside the critical section so no
// concurrent Unsubscribe can also try to close the channel (Unsubscribe
// no-ops when the subscriber is not in the map). After clearing, this
// goroutine is the sole owner of every subscriber's channel.
func (m *Manager) StopAll() {
	m.mu.Lock()
	subs := m.subs
	m.subs = nil
	m.mu.Unlock()

	for sub := range subs {
		close(sub.ch)
	}
}

// Count returns the current number of active subscribers. Exposed for
// tests and observability; not used on hot paths.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.subs)
}
