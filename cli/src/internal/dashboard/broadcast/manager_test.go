package broadcast

import (
	"context"
	"testing"
	"time"
)

// drainOnce reads at most n events from ch with a per-event timeout. Returns
// the events actually received. Used so tests do not deadlock if the
// implementation is wrong.
func drainOnce(t *testing.T, ch <-chan Event, n int, perEventTimeout time.Duration) []Event {
	t.Helper()
	out := make([]Event, 0, n)
	for i := 0; i < n; i++ {
		select {
		case evt, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, evt)
		case <-time.After(perEventTimeout):
			return out
		}
	}
	return out
}

func TestSubscribeDeliversAllEvents(t *testing.T) {
	m := New()
	sub := m.Subscribe(context.Background(), nil)
	defer m.Unsubscribe(sub)

	for i := 0; i < 5; i++ {
		m.Emit(Event{Type: "x", Payload: map[string]any{"i": i}})
	}

	got := drainOnce(t, sub.Events(), 5, time.Second)
	if len(got) != 5 {
		t.Fatalf("got %d events, want 5", len(got))
	}
	for i, evt := range got {
		if evt.Type != "x" {
			t.Errorf("event[%d] type=%q want %q", i, evt.Type, "x")
		}
		if evt.Timestamp.IsZero() {
			t.Errorf("event[%d] timestamp not stamped", i)
		}
		if evt.Payload["i"] != i {
			t.Errorf("event[%d] payload[i]=%v want %d", i, evt.Payload["i"], i)
		}
	}
}

func TestSubscribeFilterIgnoresNonMatchingEvents(t *testing.T) {
	m := New()
	sub := m.Subscribe(context.Background(), []string{"want"})
	defer m.Unsubscribe(sub)

	m.Emit(Event{Type: "skip"})
	m.Emit(Event{Type: "want"})
	m.Emit(Event{Type: "skip"})
	m.Emit(Event{Type: "want"})

	got := drainOnce(t, sub.Events(), 4, 200*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 matching events", len(got))
	}
	for i, evt := range got {
		if evt.Type != "want" {
			t.Errorf("event[%d] type=%q want %q", i, evt.Type, "want")
		}
	}
}

func TestEmitFanOutToMultipleSubscribers(t *testing.T) {
	m := New()
	sub1 := m.Subscribe(context.Background(), nil)
	defer m.Unsubscribe(sub1)
	sub2 := m.Subscribe(context.Background(), nil)
	defer m.Unsubscribe(sub2)

	if c := m.Count(); c != 2 {
		t.Fatalf("Count=%d want 2", c)
	}

	m.Emit(Event{Type: "broadcast"})

	for i, sub := range []*Subscriber{sub1, sub2} {
		got := drainOnce(t, sub.Events(), 1, time.Second)
		if len(got) != 1 || got[0].Type != "broadcast" {
			t.Errorf("sub%d got %+v want one broadcast event", i, got)
		}
	}
}

func TestUnsubscribeIsIdempotent(t *testing.T) {
	m := New()
	sub := m.Subscribe(context.Background(), nil)

	m.Unsubscribe(sub)
	m.Unsubscribe(sub) // would panic on double-close if not guarded
	m.Unsubscribe(nil) // nil-safe

	if c := m.Count(); c != 0 {
		t.Fatalf("Count=%d want 0 after unsubscribe", c)
	}
}

func TestUnsubscribeClosesChannelAndCancelsContext(t *testing.T) {
	m := New()
	sub := m.Subscribe(context.Background(), nil)

	m.Unsubscribe(sub)

	select {
	case _, ok := <-sub.Events():
		if ok {
			t.Fatal("channel returned a value but should be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed within timeout")
	}

	if sub.Context().Err() == nil {
		t.Fatal("subscriber context not cancelled after unsubscribe")
	}
}

func TestParentContextCancellationLeavesSubscriberCancelled(t *testing.T) {
	m := New()
	parent, cancel := context.WithCancel(context.Background())
	sub := m.Subscribe(parent, nil)
	defer m.Unsubscribe(sub)

	cancel()

	select {
	case <-sub.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("subscriber context not cancelled after parent cancel")
	}
}

func TestEmitDropsOldestWhenBufferFull(t *testing.T) {
	m := New()
	// Subscribe with a context the test owns so the consumer never reads.
	sub := m.Subscribe(context.Background(), nil)
	defer m.Unsubscribe(sub)

	// Fill the buffer to exactly SubBufferSize without anyone reading.
	for i := 0; i < SubBufferSize; i++ {
		m.Emit(Event{Type: "fill", Payload: map[string]any{"i": i}})
	}
	if d := sub.DroppedCount(); d != 0 {
		t.Fatalf("dropped=%d before overflow, want 0", d)
	}

	// One more event must trigger drop-oldest.
	m.Emit(Event{Type: "overflow"})
	if d := sub.DroppedCount(); d != 1 {
		t.Fatalf("dropped=%d after one overflow, want 1", d)
	}

	// First event in the buffer should now be the second emit (i=1), not i=0.
	got := drainOnce(t, sub.Events(), 1, time.Second)
	if len(got) != 1 {
		t.Fatalf("no event drained")
	}
	if first := got[0].Payload["i"]; first != 1 {
		t.Errorf("first remaining event payload[i]=%v want 1 (i=0 should have been dropped)", first)
	}
}

func TestSlowConsumerDisconnectAfterMaxDrops(t *testing.T) {
	m := New()
	sub := m.Subscribe(context.Background(), nil)
	defer m.Unsubscribe(sub)

	// Fill buffer.
	for i := 0; i < SubBufferSize; i++ {
		m.Emit(Event{Type: "fill"})
	}

	// Trigger SubMaxDrops drops by sending more events without anyone reading.
	for i := 0; i < SubMaxDrops; i++ {
		m.Emit(Event{Type: "drop"})
	}

	if d := sub.DroppedCount(); d < SubMaxDrops {
		t.Fatalf("dropped=%d want >= %d", d, SubMaxDrops)
	}

	select {
	case <-sub.Context().Done():
	case <-time.After(time.Second):
		t.Fatalf("subscriber not disconnected after %d drops", SubMaxDrops)
	}
}

func TestStopAllClosesEverySubscriberAndClears(t *testing.T) {
	m := New()
	subs := []*Subscriber{
		m.Subscribe(context.Background(), nil),
		m.Subscribe(context.Background(), nil),
		m.Subscribe(context.Background(), nil),
	}

	m.StopAll()

	if c := m.Count(); c != 0 {
		t.Fatalf("Count=%d after StopAll, want 0", c)
	}

	for i, sub := range subs {
		select {
		case _, ok := <-sub.Events():
			if ok {
				t.Errorf("sub%d channel still open after StopAll", i)
			}
		case <-time.After(time.Second):
			t.Errorf("sub%d channel not closed within timeout", i)
		}
		// Note: StopAll does NOT cancel sub.Context(). The context is
		// reserved for the slow-consumer disconnect signal so handlers
		// can distinguish shutdown (channel closed) from
		// slow-consumer (ctx canceled). See Manager.StopAll.
	}

	// Subsequent Unsubscribe must remain safe (handler defers run after StopAll).
	for _, sub := range subs {
		m.Unsubscribe(sub)
	}
}

func TestEmitWithNoSubscribersIsNoop(t *testing.T) {
	m := New()
	// Should not panic, allocate, or block.
	for i := 0; i < 100; i++ {
		m.Emit(Event{Type: "x"})
	}
}

func TestEmitTimestampPreservedWhenSet(t *testing.T) {
	m := New()
	sub := m.Subscribe(context.Background(), nil)
	defer m.Unsubscribe(sub)

	want := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	m.Emit(Event{Type: "ts", Timestamp: want})

	got := drainOnce(t, sub.Events(), 1, time.Second)
	if len(got) != 1 {
		t.Fatal("no event delivered")
	}
	if !got[0].Timestamp.Equal(want) {
		t.Errorf("timestamp=%v want %v", got[0].Timestamp, want)
	}
}
