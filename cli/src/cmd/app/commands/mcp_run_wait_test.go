package commands

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllServicesReady(t *testing.T) {
	tests := []struct {
		name string
		in   []serviceReadiness
		want bool
	}{
		{"empty is not ready", nil, false},
		{
			"all ready",
			[]serviceReadiness{{Name: "api", Ready: true}, {Name: "web", Ready: true}},
			true,
		},
		{
			"one pending",
			[]serviceReadiness{{Name: "api", Ready: true}, {Name: "web", Ready: false}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, allServicesReady(tt.in))
		})
	}
}

func TestWaitForServicesReadyAllReady(t *testing.T) {
	orig := pollServicesReadiness
	defer func() { pollServicesReadiness = orig }()

	pollServicesReadiness = func(_ context.Context, _ string) ([]serviceReadiness, error) {
		return []serviceReadiness{
			{Name: "api", Status: "running", Health: "healthy", Ready: true},
			{Name: "web", Status: "running", Health: "unknown", Ready: true},
		}, nil
	}

	res := waitForServicesReady(context.Background(), "", time.Second, time.Millisecond)
	assert.True(t, res.Ready)
	assert.False(t, res.TimedOut)
	require.Len(t, res.Services, 2)
}

func TestWaitForServicesReadyBecomesReadyAfterPolling(t *testing.T) {
	orig := pollServicesReadiness
	defer func() { pollServicesReadiness = orig }()

	calls := 0
	pollServicesReadiness = func(_ context.Context, _ string) ([]serviceReadiness, error) {
		calls++
		if calls < 3 {
			return []serviceReadiness{
				{Name: "api", Status: "starting", Ready: false},
			}, nil
		}
		return []serviceReadiness{
			{Name: "api", Status: "running", Health: "healthy", Ready: true},
		}, nil
	}

	res := waitForServicesReady(context.Background(), "", 2*time.Second, time.Millisecond)
	assert.True(t, res.Ready)
	assert.False(t, res.TimedOut)
	assert.GreaterOrEqual(t, calls, 3)
}

func TestWaitForServicesReadyTimeoutReportsPending(t *testing.T) {
	orig := pollServicesReadiness
	defer func() { pollServicesReadiness = orig }()

	pollServicesReadiness = func(_ context.Context, _ string) ([]serviceReadiness, error) {
		return []serviceReadiness{
			{Name: "api", Status: "running", Health: "healthy", Ready: true},
			{Name: "web", Status: "not-running", Ready: false},
		}, nil
	}

	res := waitForServicesReady(context.Background(), "", 20*time.Millisecond, 5*time.Millisecond)
	assert.False(t, res.Ready)
	assert.True(t, res.TimedOut)
	require.Len(t, res.Services, 2)

	pending := map[string]bool{}
	for _, s := range res.Services {
		pending[s.Name] = s.Ready
	}
	assert.True(t, pending["api"], "api should be reported ready")
	assert.False(t, pending["web"], "web should be reported pending")
}

func TestWaitForServicesReadyContextCancel(t *testing.T) {
	orig := pollServicesReadiness
	defer func() { pollServicesReadiness = orig }()

	pollServicesReadiness = func(_ context.Context, _ string) ([]serviceReadiness, error) {
		return []serviceReadiness{{Name: "api", Ready: false}}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := waitForServicesReady(ctx, "", time.Hour, 10*time.Millisecond)
	assert.False(t, res.Ready)
	assert.True(t, res.TimedOut)
}

// TestHandleRunServicesNonBlockingSkipsPolling verifies the default path
// (wait omitted) never polls readiness, preserving the fire-and-forget
// behavior.
func TestHandleRunServicesNonBlockingSkipsPolling(t *testing.T) {
	orig := pollServicesReadiness
	defer func() { pollServicesReadiness = orig }()

	polled := false
	pollServicesReadiness = func(_ context.Context, _ string) ([]serviceReadiness, error) {
		polled = true
		return nil, nil
	}

	// azd app run has no azure.yaml in the test working directory, so the
	// process (if azd is installed) exits immediately. Either way, the
	// non-blocking path must not poll readiness.
	res, err := handleRunServices(context.Background(), testToolArgs(map[string]any{"wait": false}))
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, polled, "wait=false must not poll service readiness")
}

// TestHandleRunServicesWaitInvalidProjectDir verifies that requesting wait with
// an invalid projectDir returns an error result and never starts the process
// or polls readiness.
func TestHandleRunServicesWaitInvalidProjectDir(t *testing.T) {
	orig := pollServicesReadiness
	defer func() { pollServicesReadiness = orig }()

	polled := false
	pollServicesReadiness = func(_ context.Context, _ string) ([]serviceReadiness, error) {
		polled = true
		return nil, nil
	}

	missing := filepath.Join(t.TempDir(), "does-not-exist")
	res, err := handleRunServices(context.Background(), testToolArgs(map[string]any{
		"wait":       true,
		"projectDir": missing,
	}))
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.IsError, "invalid project dir should return an error result")
	assert.False(t, polled, "invalid project dir must short-circuit before polling")
}

func TestClampRunWaitTimeout(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero uses default", 0, defaultRunWaitTimeoutSeconds},
		{"negative uses default", -10, defaultRunWaitTimeoutSeconds},
		{"in range preserved", 30, 30},
		{"max preserved", maxRunWaitTimeoutSeconds, maxRunWaitTimeoutSeconds},
		{"over max is capped", maxRunWaitTimeoutSeconds + 1, maxRunWaitTimeoutSeconds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clampRunWaitTimeout(tt.in))
		})
	}
}

func TestToServiceReadiness(t *testing.T) {
	running := func(health string) *serviceinfo.LocalServiceInfo {
		return &serviceinfo.LocalServiceInfo{Status: serviceStatusRunning, Health: health}
	}
	services := []*serviceinfo.ServiceInfo{
		nil, // must be skipped
		{Name: "healthy", Local: running("healthy")},
		{Name: "unknown-health", Local: running("")},
		{Name: "unhealthy", Local: running(serviceHealthUnhealthy)},
		{Name: "stopped", Local: &serviceinfo.LocalServiceInfo{Status: "not-running", Health: "unknown"}},
		{Name: "no-local"},
	}

	got := toServiceReadiness(services)
	require.Len(t, got, 5, "nil service entries must be skipped")

	byName := map[string]serviceReadiness{}
	for _, r := range got {
		byName[r.Name] = r
	}

	assert.True(t, byName["healthy"].Ready)
	assert.Equal(t, serviceStatusRunning, byName["healthy"].Status)
	assert.Equal(t, "healthy", byName["healthy"].Health)

	assert.True(t, byName["unknown-health"].Ready, "running service without a health probe counts as ready")
	assert.False(t, byName["unhealthy"].Ready, "explicitly unhealthy service is not ready")
	assert.False(t, byName["stopped"].Ready, "non-running service is not ready")

	assert.False(t, byName["no-local"].Ready, "service without local info is not ready")
	assert.Empty(t, byName["no-local"].Status)
}

func TestToServiceReadinessEmpty(t *testing.T) {
	assert.Empty(t, toServiceReadiness(nil))
}

func TestBuildRunWaitResultReady(t *testing.T) {
	res := runWaitResult{
		Ready:    true,
		Services: []serviceReadiness{{Name: "api", Ready: true}},
	}
	out := buildRunWaitResult(res, 4321, 120)
	assert.Equal(t, "ready", out["status"])
	assert.Equal(t, true, out["ready"])
	assert.Equal(t, "All services are ready.", out["message"])
	assert.Equal(t, 4321, out["pid"])
	assert.NotNil(t, out["services"])
}

func TestBuildRunWaitResultTimeout(t *testing.T) {
	res := runWaitResult{
		Ready:    false,
		TimedOut: true,
		Services: []serviceReadiness{{Name: "web", Ready: false}},
	}
	out := buildRunWaitResult(res, 0, 45)
	assert.Equal(t, "timeout", out["status"])
	assert.Equal(t, false, out["ready"])
	assert.Contains(t, out["message"], "45s")
	_, hasPID := out["pid"]
	assert.False(t, hasPID, "pid must be omitted when the process was not started")
}
