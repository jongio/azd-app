package rpc

import (
	"sync"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
)

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
	head    int // index of the oldest entry when ring is full
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
		// Ring full: overwrite oldest via index, O(1).
		r.buf[r.head] = e
		r.head = (r.head + 1) % r.cap
		r.dropped++
	} else {
		r.buf = append(r.buf, e)
	}
	r.mu.Unlock()

	select {
	case r.notify <- struct{}{}:
	default:
		// Already a pending wakeup; the consumer's next drain picks
		// this entry up too. Coalescing.
	}
}

// drain returns all entries currently in the ring in insertion order plus
// the cumulative dropped count. Resets the ring; the caller compares
// dropped against its previously-observed value to compute the delta.
func (r *localLogRing) drain() ([]*v1.LogEntry, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.buf) == 0 {
		return nil, r.dropped
	}
	n := len(r.buf)
	out := make([]*v1.LogEntry, n)
	if r.head == 0 || n < r.cap {
		// Not wrapped yet or head is at start - simple copy
		copy(out, r.buf)
	} else {
		// Ring has wrapped: linearize from head
		copy(out, r.buf[r.head:])
		copy(out[n-r.head:], r.buf[:r.head])
	}
	r.buf = r.buf[:0]
	r.head = 0
	return out, r.dropped
}
