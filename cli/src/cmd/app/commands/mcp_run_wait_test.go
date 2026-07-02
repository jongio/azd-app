package commands

import (
	"context"
	"testing"
	"time"

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
