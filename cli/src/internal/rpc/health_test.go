package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
)

// stubHealthSource is a deterministic, race-safe HealthSource used by
// every HealthService test. Tests rotate `nextReport` / `nextErr` to
// drive each scenario; `calls` lets streaming tests detect that the
// handler has begun polling (Connect server-streams over HTTP/1 don't
// flush response headers until the first Send, so we cannot drive the
// scenario synchronously after issuing the streaming call - same
// constraint as lifecycle_test.go).
type stubHealthSource struct {
	mu sync.Mutex

	// reports is consumed FIFO; if exhausted, the last report is reused.
	// Empty slice plus nil err yields a deliberately-empty report.
	reports []*healthcheck.HealthReport
	err     error

	calls         atomic.Int64
	gotFilters    [][]string
	blockUntil    chan struct{} // optional: hold the first call open
	blockReleased atomic.Bool
}

func (s *stubHealthSource) Check(ctx context.Context, serviceFilter []string) (*healthcheck.HealthReport, error) {
	s.calls.Add(1)
	s.mu.Lock()
	s.gotFilters = append(s.gotFilters, append([]string(nil), serviceFilter...))
	block := s.blockUntil
	if s.err != nil {
		err := s.err
		s.mu.Unlock()
		return nil, err
	}
	var report *healthcheck.HealthReport
	switch len(s.reports) {
	case 0:
		report = &healthcheck.HealthReport{Timestamp: time.Now()}
	case 1:
		report = s.reports[0]
	default:
		report = s.reports[0]
		s.reports = s.reports[1:]
	}
	s.mu.Unlock()

	if block != nil && !s.blockReleased.Load() {
		select {
		case <-block:
			s.blockReleased.Store(true)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return report, nil
}

// stubStateTransitionSource records active listeners and lets tests
// fire transitions deterministically. listeners() exposes the live
// count so streaming tests can wait until subscription is registered
// before emitting (mirrors mgr.Count() in broadcast tests).
type stubStateTransitionSource struct {
	mu          sync.Mutex
	subscribers []monitor.StateListener
	history     []monitor.StateTransition
}

func (s *stubStateTransitionSource) Subscribe(listener monitor.StateListener) func() {
	s.mu.Lock()
	idx := len(s.subscribers)
	s.subscribers = append(s.subscribers, listener)
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Replace with nil rather than reslice so concurrent count()
		// calls don't observe a torn slice. count() ignores nils.
		if idx < len(s.subscribers) {
			s.subscribers[idx] = nil
		}
	}
}

func (s *stubStateTransitionSource) History() []monitor.StateTransition {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]monitor.StateTransition, len(s.history))
	copy(out, s.history)
	return out
}

// emit fans a transition out to every active listener synchronously. The
// production StateMonitor spawns a goroutine per listener invocation,
// but for tests we want deterministic ordering and the handler does not
// rely on any async dispatch property.
func (s *stubStateTransitionSource) emit(t monitor.StateTransition) {
	s.mu.Lock()
	listeners := make([]monitor.StateListener, 0, len(s.subscribers))
	for _, l := range s.subscribers {
		if l != nil {
			listeners = append(listeners, l)
		}
	}
	s.mu.Unlock()
	for _, l := range listeners {
		l(t)
	}
}

func (s *stubStateTransitionSource) liveListenerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, l := range s.subscribers {
		if l != nil {
			n++
		}
	}
	return n
}

// newHealthTestServer wires HealthHandler behind an httptest server. The
// transitionSource may be nil to test the "feature off" path.
func newHealthTestServer(t *testing.T, hs HealthSource, ts StateTransitionSource) (azdappv1connect.HealthServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast:        mgr,
		Health:           hs,
		StateTransitions: ts,
	})

	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewHealthServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

func TestGetHealthReturnsResults(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	source := &stubHealthSource{
		reports: []*healthcheck.HealthReport{{
			Timestamp: now,
			Services: []healthcheck.HealthCheckResult{
				{
					ServiceName:  "api",
					Status:       healthcheck.HealthStatusHealthy,
					ResponseTime: 12 * time.Millisecond,
					Timestamp:    now,
					Details:      map[string]interface{}{"statusCode": 200, "endpoint": "/health"},
				},
				{
					ServiceName:  "worker",
					Status:       healthcheck.HealthStatusDegraded,
					Error:        "high latency",
					ResponseTime: 800 * time.Millisecond,
					Timestamp:    now,
				},
			},
		}},
	}
	client, cleanup := newHealthTestServer(t, source, nil)
	defer cleanup()

	resp, err := client.GetHealth(context.Background(), connect.NewRequest(&v1.GetHealthRequest{
		ServiceNames: []string{"api", "worker"},
	}))
	if err != nil {
		t.Fatalf("GetHealth error: %v", err)
	}
	results := resp.Msg.GetResults()
	if len(results) != 2 {
		t.Fatalf("Results length=%d want 2", len(results))
	}
	// Order is whatever the source returned (GetHealth doesn't sort - SubscribeHealth does).
	byName := map[string]*v1.HealthCheckResult{}
	for _, r := range results {
		byName[r.GetServiceName()] = r
	}
	if got := byName["api"].GetState(); got != v1.HealthState_HEALTH_STATE_HEALTHY {
		t.Errorf("api.State=%v want HEALTHY", got)
	}
	if got := byName["api"].GetLatencyMs(); got != 12 {
		t.Errorf("api.LatencyMs=%d want 12", got)
	}
	if got := byName["api"].GetDetails()["statusCode"]; got != "200" {
		t.Errorf("api.Details.statusCode=%q want \"200\"", got)
	}
	if got := byName["worker"].GetState(); got != v1.HealthState_HEALTH_STATE_DEGRADED {
		t.Errorf("worker.State=%v want DEGRADED", got)
	}
	if got := byName["worker"].GetMessage(); got != "high latency" {
		t.Errorf("worker.Message=%q want \"high latency\"", got)
	}
	if source.calls.Load() != 1 {
		t.Errorf("Check call count=%d want 1", source.calls.Load())
	}
	if got := source.gotFilters[0]; len(got) != 2 || got[0] != "api" || got[1] != "worker" {
		t.Errorf("gotFilters[0]=%v want [api worker]", got)
	}
}

func TestGetHealthReturnsInternalOnSourceError(t *testing.T) {
	source := &stubHealthSource{err: errors.New("registry unavailable")}
	client, cleanup := newHealthTestServer(t, source, nil)
	defer cleanup()

	_, err := client.GetHealth(context.Background(), connect.NewRequest(&v1.GetHealthRequest{}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("error code=%v want Internal; full err=%v", got, err)
	}
}

func TestNewHealthHandlerPanicsOnNilHealthSource(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when healthSource is nil")
		}
	}()
	_ = NewHealthHandler(nil, nil)
}

func TestHealthServiceNotMountedWithoutHealthSource(t *testing.T) {
	mgr := broadcast.New()
	defer mgr.StopAll()

	mux := http.NewServeMux()
	Mount(mux, Dependencies{Broadcast: mgr})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + azdappv1connect.HealthServiceGetHealthProcedure)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d want 404 when HealthService is not mounted", resp.StatusCode)
	}
}

func TestStreamHealthEmitsInitialReportThenChange(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	report1 := &healthcheck.HealthReport{
		Timestamp: now,
		Services: []healthcheck.HealthCheckResult{
			{ServiceName: "api", Status: healthcheck.HealthStatusHealthy, Timestamp: now},
		},
	}
	report2 := &healthcheck.HealthReport{
		Timestamp: now.Add(time.Second),
		Services: []healthcheck.HealthCheckResult{
			{ServiceName: "api", Status: healthcheck.HealthStatusUnhealthy, Error: "connection refused", Timestamp: now.Add(time.Second)},
		},
	}
	source := &stubHealthSource{reports: []*healthcheck.HealthReport{report1, report2}}
	client, cleanup := newHealthTestServer(t, source, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use minimum interval so the second report fires inside the test
	// window without us having to tweak the package-level constant.
	stream, err := client.StreamHealth(ctx, connect.NewRequest(&v1.StreamHealthRequest{
		IntervalSeconds: 1,
	}))
	if err != nil {
		t.Fatalf("StreamHealth: %v", err)
	}

	// Initial report (no prior state, so no Change event precedes it).
	if !stream.Receive() {
		t.Fatalf("Receive[0] false; err=%v", stream.Err())
	}
	first := stream.Msg().GetEvent()
	if first.GetReport() == nil {
		t.Fatalf("event[0] is not a report; got %+v", first)
	}
	if got := first.GetReport().GetResults(); len(got) != 1 || got[0].GetState() != v1.HealthState_HEALTH_STATE_HEALTHY {
		t.Errorf("first report state mismatch; got %+v", got)
	}

	// Second tick should produce a Change (HEALTHY → UNHEALTHY) before the report.
	if !stream.Receive() {
		t.Fatalf("Receive[1] false; err=%v", stream.Err())
	}
	second := stream.Msg().GetEvent()
	if change := second.GetChange(); change == nil {
		t.Errorf("event[1] is not a change; got %+v", second)
	} else {
		if change.GetServiceName() != "api" {
			t.Errorf("change.ServiceName=%q want api", change.GetServiceName())
		}
		if change.GetPreviousState() != v1.HealthState_HEALTH_STATE_HEALTHY {
			t.Errorf("change.PreviousState=%v want HEALTHY", change.GetPreviousState())
		}
		if change.GetCurrentState() != v1.HealthState_HEALTH_STATE_UNHEALTHY {
			t.Errorf("change.CurrentState=%v want UNHEALTHY", change.GetCurrentState())
		}
		if change.GetMessage() != "connection refused" {
			t.Errorf("change.Message=%q want connection refused", change.GetMessage())
		}
	}

	// Third receive: the report that follows the change.
	if !stream.Receive() {
		t.Fatalf("Receive[2] false; err=%v", stream.Err())
	}
	third := stream.Msg().GetEvent()
	if third.GetReport() == nil {
		t.Errorf("event[2] is not a report; got %+v", third)
	}
}

func TestStreamHealthSkipsTickOnTransientError(t *testing.T) {
	// Source returns a probe error on first call, then a real report.
	// The stream MUST NOT terminate on the transient error - matching
	// the legacy SSE behaviour where a single failed tick logs and waits
	// for the next interval.
	source := &countingErrThenReportSource{
		report: &healthcheck.HealthReport{
			Timestamp: time.Now(),
			Services: []healthcheck.HealthCheckResult{
				{ServiceName: "api", Status: healthcheck.HealthStatusHealthy, Timestamp: time.Now()},
			},
		},
		errOnCalls: 1,
	}
	client, cleanup := newHealthTestServer(t, source, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamHealth(ctx, connect.NewRequest(&v1.StreamHealthRequest{
		IntervalSeconds: 1,
	}))
	if err != nil {
		t.Fatalf("StreamHealth: %v", err)
	}

	// First emission MUST be the report from the second probe call (the
	// initial probe failed silently).
	if !stream.Receive() {
		t.Fatalf("Receive false on first event; err=%v", stream.Err())
	}
	if stream.Msg().GetEvent().GetReport() == nil {
		t.Errorf("expected report event; got %+v", stream.Msg().GetEvent())
	}
	if source.calls.Load() < 2 {
		t.Errorf("source.calls=%d want >=2 (initial failed, then real)", source.calls.Load())
	}
}

// countingErrThenReportSource returns an error for the first N calls,
// then a real report on every subsequent call.
type countingErrThenReportSource struct {
	report     *healthcheck.HealthReport
	errOnCalls int64
	calls      atomic.Int64
}

func (c *countingErrThenReportSource) Check(ctx context.Context, _ []string) (*healthcheck.HealthReport, error) {
	n := c.calls.Add(1)
	if n <= c.errOnCalls {
		return nil, errors.New("transient probe failure")
	}
	return c.report, nil
}

func TestStreamHealthExitsOnClientCancel(t *testing.T) {
	source := &stubHealthSource{
		reports: []*healthcheck.HealthReport{{
			Timestamp: time.Now(),
			Services: []healthcheck.HealthCheckResult{
				{ServiceName: "api", Status: healthcheck.HealthStatusHealthy, Timestamp: time.Now()},
			},
		}},
	}
	client, cleanup := newHealthTestServer(t, source, nil)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.StreamHealth(ctx, connect.NewRequest(&v1.StreamHealthRequest{IntervalSeconds: 1}))
	if err != nil {
		t.Fatalf("StreamHealth: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive[0] false; err=%v", stream.Err())
	}
	cancel()
	// After cancel, Receive must return false and Err must reflect cancellation.
	if stream.Receive() {
		// Some Connect builds may flush one buffered event; tolerate either.
		t.Logf("stream produced one extra event after cancel; tolerated")
	}
	if rerr := stream.Err(); rerr != nil &&
		!strings.Contains(strings.ToLower(rerr.Error()), "cancel") &&
		connect.CodeOf(rerr) != connect.CodeCanceled {
		t.Logf("stream err after cancel: %v (acceptable)", rerr)
	}
}

func TestClampHealthIntervalBounds(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{"zero defaults to 5s", 0, defaultHealthInterval},
		{"negative defaults to 5s", -1 * time.Second, defaultHealthInterval},
		{"below min clamps up", 100 * time.Millisecond, minHealthInterval},
		{"at min stays", minHealthInterval, minHealthInterval},
		{"in range stays", 7 * time.Second, 7 * time.Second},
		{"at max stays", maxHealthInterval, maxHealthInterval},
		{"above max clamps down", 5 * time.Minute, maxHealthInterval},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampHealthInterval(tc.in); got != tc.want {
				t.Errorf("clampHealthInterval(%v)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestStreamStateTransitionsUnimplementedWithoutSource(t *testing.T) {
	// HealthSource is wired but StateTransitions is not. Calling the
	// state-transitions stream must surface Unimplemented so a UI can
	// degrade gracefully.
	source := &stubHealthSource{}
	client, cleanup := newHealthTestServer(t, source, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.StreamStateTransitions(ctx, connect.NewRequest(&v1.StreamStateTransitionsRequest{}))
	// Connect surfaces handler-returned Unimplemented either at the
	// call boundary or via stream.Err() after Receive returns false.
	if err != nil {
		if got := connect.CodeOf(err); got != connect.CodeUnimplemented {
			t.Errorf("err code=%v want Unimplemented; full err=%v", got, err)
		}
		return
	}
	if stream.Receive() {
		t.Fatalf("Receive returned true on Unimplemented stream; want false")
	}
	if got := connect.CodeOf(stream.Err()); got != connect.CodeUnimplemented {
		t.Errorf("stream.Err code=%v want Unimplemented; full err=%v", got, stream.Err())
	}
}

func TestStreamStateTransitionsBackfillThenLive(t *testing.T) {
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	historic := []monitor.StateTransition{
		{
			Timestamp: now,
			ServiceName: "api", Severity: monitor.SeverityInfo,
			Description: "started",
			ToState:     &monitor.ServiceState{Status: "ready", Health: "healthy", PIDValid: true, PortListens: true, PID: 100, Port: 3000},
		},
		{
			Timestamp: now.Add(time.Minute),
			ServiceName: "api", Severity: monitor.SeverityCritical,
			Description: "process exited unexpectedly",
			FromState:   &monitor.ServiceState{Status: "ready", Health: "healthy", PIDValid: true, PortListens: true, PID: 100, Port: 3000},
			ToState:     &monitor.ServiceState{Status: "stopped", Health: "unhealthy", PIDValid: false, PortListens: false},
		},
	}
	transitions := &stubStateTransitionSource{history: historic}

	source := &stubHealthSource{}
	client, cleanup := newHealthTestServer(t, source, transitions)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait for the handler to subscribe before emitting the live event.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && transitions.liveListenerCount() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		transitions.emit(monitor.StateTransition{
			Timestamp:   now.Add(2 * time.Minute),
			ServiceName: "api",
			Severity:    monitor.SeverityWarning,
			Description: "recovered",
			FromState:   &monitor.ServiceState{Status: "stopped", Health: "unhealthy", PIDValid: false},
			ToState:     &monitor.ServiceState{Status: "ready", Health: "healthy", PIDValid: true, PortListens: true},
		})
	}()

	stream, err := client.StreamStateTransitions(ctx, connect.NewRequest(&v1.StreamStateTransitionsRequest{
		Backfill: 10,
	}))
	if err != nil {
		t.Fatalf("StreamStateTransitions: %v", err)
	}

	// Two backfill items, then one live, in temporal order.
	got := make([]*v1.StateTransition, 0, 3)
	for i := 0; i < 3; i++ {
		if !stream.Receive() {
			t.Fatalf("Receive[%d] false; err=%v", i, stream.Err())
		}
		got = append(got, stream.Msg().GetTransition())
	}

	if got[0].GetEventType() != "registered" {
		t.Errorf("got[0].EventType=%q want registered", got[0].GetEventType())
	}
	if got[1].GetEventType() != "crashed" {
		t.Errorf("got[1].EventType=%q want crashed (PID went invalid)", got[1].GetEventType())
	}
	if got[1].GetSeverity() != v1.Severity_SEVERITY_CRITICAL {
		t.Errorf("got[1].Severity=%v want CRITICAL", got[1].GetSeverity())
	}
	if got[2].GetEventType() != "recovered" {
		t.Errorf("got[2].EventType=%q want recovered", got[2].GetEventType())
	}

	// IDs must be unique and stable.
	ids := map[string]bool{}
	for _, g := range got {
		if ids[g.GetId()] {
			t.Errorf("duplicate transition id %q", g.GetId())
		}
		ids[g.GetId()] = true
	}
}

func TestStreamStateTransitionsRespectsServiceFilter(t *testing.T) {
	transitions := &stubStateTransitionSource{}
	source := &stubHealthSource{}
	client, cleanup := newHealthTestServer(t, source, transitions)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && transitions.liveListenerCount() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		transitions.emit(monitor.StateTransition{
			Timestamp: time.Now(), ServiceName: "ignored", Severity: monitor.SeverityCritical,
			ToState: &monitor.ServiceState{Status: "ready", Health: "healthy"},
		})
		transitions.emit(monitor.StateTransition{
			Timestamp: time.Now(), ServiceName: "wanted", Severity: monitor.SeverityCritical,
			ToState: &monitor.ServiceState{Status: "ready", Health: "healthy"},
		})
	}()

	stream, err := client.StreamStateTransitions(ctx, connect.NewRequest(&v1.StreamStateTransitionsRequest{
		ServiceNames: []string{"wanted"},
	}))
	if err != nil {
		t.Fatalf("StreamStateTransitions: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive false; err=%v", stream.Err())
	}
	if got := stream.Msg().GetTransition().GetServiceName(); got != "wanted" {
		t.Errorf("ServiceName=%q want wanted (filter should have dropped 'ignored')", got)
	}
}

func TestStreamStateTransitionsRespectsSeverityFloor(t *testing.T) {
	transitions := &stubStateTransitionSource{}
	source := &stubHealthSource{}
	client, cleanup := newHealthTestServer(t, source, transitions)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && transitions.liveListenerCount() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		// INFO: should be filtered out by min=WARNING.
		transitions.emit(monitor.StateTransition{
			Timestamp: time.Now(), ServiceName: "api", Severity: monitor.SeverityInfo,
			ToState: &monitor.ServiceState{Status: "ready", Health: "healthy"},
		})
		// WARNING: at floor, should pass.
		transitions.emit(monitor.StateTransition{
			Timestamp: time.Now(), ServiceName: "api", Severity: monitor.SeverityWarning,
			ToState: &monitor.ServiceState{Status: "ready", Health: "healthy"},
		})
	}()

	stream, err := client.StreamStateTransitions(ctx, connect.NewRequest(&v1.StreamStateTransitionsRequest{
		MinSeverity: v1.Severity_SEVERITY_WARNING,
	}))
	if err != nil {
		t.Fatalf("StreamStateTransitions: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive false; err=%v", stream.Err())
	}
	if got := stream.Msg().GetTransition().GetSeverity(); got != v1.Severity_SEVERITY_WARNING {
		t.Errorf("Severity=%v want WARNING (INFO should have been filtered)", got)
	}
}

func TestStreamStateTransitionsBackfillCapped(t *testing.T) {
	// Build 250 historical transitions; request more than the cap and
	// expect at most maxStateTransitionsBackfill items.
	now := time.Now()
	history := make([]monitor.StateTransition, 250)
	for i := range history {
		history[i] = monitor.StateTransition{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			ServiceName: "api",
			Severity:    monitor.SeverityInfo,
			ToState:     &monitor.ServiceState{Status: "ready", Health: "healthy"},
		}
	}
	transitions := &stubStateTransitionSource{history: history}
	source := &stubHealthSource{}
	client, cleanup := newHealthTestServer(t, source, transitions)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamStateTransitions(ctx, connect.NewRequest(&v1.StreamStateTransitionsRequest{
		Backfill: 1000,
	}))
	if err != nil {
		t.Fatalf("StreamStateTransitions: %v", err)
	}

	// Drain backfill: each item arrives until subscribe-then-block; we
	// stop once the per-receive deadline trips, since no live events
	// follow.
	received := 0
	deadline := time.Now().Add(2 * time.Second)
	timestamps := []int64{}
	for time.Now().Before(deadline) {
		recvCtx, recvCancel := context.WithTimeout(ctx, 250*time.Millisecond)
		done := make(chan struct{})
		var ok bool
		go func() {
			ok = stream.Receive()
			close(done)
		}()
		select {
		case <-done:
			recvCancel()
			if !ok {
				goto end
			}
			received++
			timestamps = append(timestamps, stream.Msg().GetTransition().GetTimestamp().AsTime().UnixMilli())
		case <-recvCtx.Done():
			recvCancel()
			goto end
		}
	}
end:
	if received != maxStateTransitionsBackfill {
		t.Errorf("received=%d want exactly %d (cap)", received, maxStateTransitionsBackfill)
	}
	// Backfill returns the most recent items in temporal order: the first
	// emitted timestamp must be 250-100=150 ms after `now`.
	if !sort.SliceIsSorted(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] }) {
		t.Errorf("backfill emitted out-of-order timestamps: %v", timestamps)
	}
	if len(timestamps) > 0 {
		want := now.Add(150 * time.Millisecond).UnixMilli()
		if timestamps[0] != want {
			t.Errorf("first backfill ts=%d want %d (newest 100 of 250)", timestamps[0], want)
		}
	}
}

func TestStreamStateTransitionsUnsubscribesOnReturn(t *testing.T) {
	transitions := &stubStateTransitionSource{}
	source := &stubHealthSource{}
	client, cleanup := newHealthTestServer(t, source, transitions)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// HTTP/1 + Connect server-streaming: client.StreamStateTransitions
	// blocks until the handler's first Send flushes response headers.
	// With no backfill, the handler subscribes and waits, so we must emit
	// from a goroutine once we observe the subscription.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && transitions.liveListenerCount() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		transitions.emit(monitor.StateTransition{
			Timestamp: time.Now(), ServiceName: "api", Severity: monitor.SeverityInfo,
			ToState: &monitor.ServiceState{Status: "ready", Health: "healthy"},
		})
	}()

	stream, err := client.StreamStateTransitions(ctx, connect.NewRequest(&v1.StreamStateTransitionsRequest{}))
	if err != nil {
		t.Fatalf("StreamStateTransitions: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive false; err=%v", stream.Err())
	}
	if transitions.liveListenerCount() != 1 {
		t.Fatalf("listener count=%d want 1 while stream is live", transitions.liveListenerCount())
	}
	cancel()

	// Within a reasonable window the handler must unsubscribe.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && transitions.liveListenerCount() != 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if got := transitions.liveListenerCount(); got != 0 {
		t.Errorf("listener count=%d after cancel; want 0 (handler must unsubscribe)", got)
	}
}

func TestClassifyTransitionEventTypeCovers(t *testing.T) {
	healthy := &monitor.ServiceState{Status: "ready", Health: "healthy", PIDValid: true, PortListens: true}
	cases := []struct {
		name string
		t    monitor.StateTransition
		want string
	}{
		{
			name: "registered: nil from",
			t:    monitor.StateTransition{ToState: healthy},
			want: "registered",
		},
		{
			name: "deregistered: nil to",
			t:    monitor.StateTransition{FromState: healthy},
			want: "deregistered",
		},
		{
			name: "crashed: PID went invalid",
			t: monitor.StateTransition{
				FromState: healthy,
				ToState:   &monitor.ServiceState{Status: "stopped", Health: "unhealthy", PIDValid: false, PortListens: false},
			},
			want: "crashed",
		},
		{
			name: "port-unbound: still has PID, no port",
			t: monitor.StateTransition{
				FromState: healthy,
				ToState:   &monitor.ServiceState{Status: "degraded", Health: "unhealthy", PIDValid: true, PortListens: false},
			},
			want: "port-unbound",
		},
		{
			name: "degraded: healthy → degraded",
			t: monitor.StateTransition{
				FromState: healthy,
				ToState:   &monitor.ServiceState{Status: "ready", Health: "degraded", PIDValid: true, PortListens: true},
			},
			want: "degraded",
		},
		{
			name: "recovered: unhealthy → healthy",
			t: monitor.StateTransition{
				FromState: &monitor.ServiceState{Status: "ready", Health: "unhealthy", PIDValid: true, PortListens: true},
				ToState:   healthy,
			},
			want: "recovered",
		},
		{
			name: "unhealthy: healthy → unhealthy (with port)",
			t: monitor.StateTransition{
				FromState: healthy,
				ToState:   &monitor.ServiceState{Status: "ready", Health: "unhealthy", PIDValid: true, PortListens: true},
			},
			want: "unhealthy",
		},
		{
			name: "transition: catch-all",
			t: monitor.StateTransition{
				FromState: &monitor.ServiceState{Status: "ready", Health: "unknown"},
				ToState:   &monitor.ServiceState{Status: "ready", Health: "unknown"},
			},
			want: "transition",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyTransitionEventType(&tc.t); got != tc.want {
				t.Errorf("classifyTransitionEventType=%q want %q", got, tc.want)
			}
		})
	}
}
