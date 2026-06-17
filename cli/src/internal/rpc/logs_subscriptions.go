// logs_subscriptions.go manages per-stream subscription lifecycle for
// StreamLocalLogs, including pump goroutines and dynamic service addition.
package rpc

import (
	"context"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// streamSubscriptions manages the per-stream set of LogBuffer subscriptions
// and their pump goroutines for StreamLocalLogs. Each subscribed service's
// channel is drained by a dedicated pump goroutine that pushes entries into
// the shared ring. Concurrent add (from the dynamic new-service path) and
// close are safe under mu.
//
// Lifecycle: construct with newStreamSubscriptions, add services via add,
// then call close once on stream exit. close cancels every pump, waits for
// them to drain in-flight pushes, then unsubscribes each source channel.
type streamSubscriptions struct {
	store LogSource
	ring  *localLogRing

	ctx    context.Context
	cancel context.CancelFunc

	mu    sync.Mutex
	subs  map[string]chan service.LogEntry
	pumps sync.WaitGroup
}

// newStreamSubscriptions creates a subscription set bound to parent's
// lifetime. The internal context is cancelled by close or when parent is
// cancelled, whichever comes first.
func newStreamSubscriptions(parent context.Context, store LogSource, ring *localLogRing) *streamSubscriptions {
	ctx, cancel := context.WithCancel(parent)
	return &streamSubscriptions{
		store:  store,
		ring:   ring,
		ctx:    ctx,
		cancel: cancel,
		subs:   make(map[string]chan service.LogEntry),
	}
}

// add subscribes to serviceName and starts its pump if not already
// subscribed. Returns true if a new subscription was created, false if the
// service was already subscribed or does not exist in the store.
func (s *streamSubscriptions) add(serviceName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, already := s.subs[serviceName]; already {
		return false
	}
	ch, exists := s.store.Subscribe(serviceName)
	if !exists {
		return false
	}
	s.subs[serviceName] = ch
	s.startPump(ch)
	return true
}

// startPump launches a pump goroutine that drains src into the ring until
// the channel closes or the stream context is cancelled. Drop-oldest ring
// semantics mean push never blocks, so the pump always observes ctx.Done.
// Caller must hold mu.
func (s *streamSubscriptions) startPump(src chan service.LogEntry) {
	s.pumps.Go(func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case entry, ok := <-src:
				if !ok {
					return
				}
				s.ring.push(toProtoLogEntry(entry))
			}
		}
	})
}

// close tears down every subscription. Ordering matters: cancel first so
// pumps stop reading, wait so any in-flight push completes, then unsubscribe
// to close the source channels (which a now-stopped pump won't read from).
func (s *streamSubscriptions) close() {
	s.cancel()
	s.pumps.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, ch := range s.subs {
		s.store.Unsubscribe(name, ch)
	}
}
