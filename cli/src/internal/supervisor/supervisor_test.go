package supervisor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldRestart(t *testing.T) {
	tests := []struct {
		name             string
		policy           service.RestartPolicy
		exitedUnexpected bool
		attempt          int
		expected         bool
	}{
		{
			name: "never policy never restarts",
			policy: service.RestartPolicy{
				Policy:     service.RestartPolicyNever,
				MaxRetries: 3,
			},
			exitedUnexpected: true,
			attempt:          1,
			expected:         false,
		},
		{
			name: "on-failure restarts within retry cap",
			policy: service.RestartPolicy{
				Policy:     service.RestartPolicyOnFailure,
				MaxRetries: 2,
			},
			exitedUnexpected: true,
			attempt:          2,
			expected:         true,
		},
		{
			name: "always policy stops after retry cap",
			policy: service.RestartPolicy{
				Policy:     service.RestartPolicyAlways,
				MaxRetries: 2,
			},
			exitedUnexpected: true,
			attempt:          3,
			expected:         false,
		},
		{
			name: "unexpected exit required for restart",
			policy: service.RestartPolicy{
				Policy:     service.RestartPolicyAlways,
				MaxRetries: 3,
			},
			exitedUnexpected: false,
			attempt:          1,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ShouldRestart(tt.policy, tt.exitedUnexpected, tt.attempt)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBackoffDuration(t *testing.T) {
	t.Run("increases exponentially", func(t *testing.T) {
		base := 100 * time.Millisecond
		assert.Equal(t, 100*time.Millisecond, BackoffDuration(1, base))
		assert.Equal(t, 200*time.Millisecond, BackoffDuration(2, base))
		assert.Equal(t, 400*time.Millisecond, BackoffDuration(3, base))
	})

	t.Run("caps at maximum duration", func(t *testing.T) {
		base := 10 * time.Second
		assert.Equal(t, service.MaxRestartBackoffBase, BackoffDuration(3, base))
		assert.Equal(t, service.MaxRestartBackoffBase, BackoffDuration(10, base))
	})
}

func TestSupervisorOnStateTransition_RestartsUntilMaxRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	policies := map[string]service.RestartPolicy{
		"api": {
			Policy:      service.RestartPolicyOnFailure,
			MaxRetries:  2,
			BackoffBase: time.Millisecond,
		},
	}

	var mu sync.Mutex
	restartCalls := 0
	supervisor := New(ctx, policies, func(serviceName string) error {
		mu.Lock()
		defer mu.Unlock()
		restartCalls++
		assert.Equal(t, "api", serviceName)
		return nil
	})

	crash := newCrashTransition("api", "running")
	supervisor.OnStateTransition(crash)
	supervisor.OnStateTransition(crash)
	supervisor.OnStateTransition(crash)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return restartCalls == 2
	}, time.Second, 10*time.Millisecond)

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	assert.Equal(t, 2, restartCalls)
	mu.Unlock()
}

func TestSupervisorOnStateTransition_PolicyNeverDoesNotRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	supervisor := New(ctx, map[string]service.RestartPolicy{
		"api": {
			Policy:      service.RestartPolicyNever,
			MaxRetries:  5,
			BackoffBase: time.Millisecond,
		},
	}, func(string) error {
		t.Fatal("restart callback should not be called for never policy")
		return nil
	})

	supervisor.OnStateTransition(newCrashTransition("api", "running"))

	time.Sleep(50 * time.Millisecond)
}

func TestSupervisorOnStateTransition_DoesNotRestartExpectedStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	supervisor := New(ctx, map[string]service.RestartPolicy{
		"api": {
			Policy:      service.RestartPolicyOnFailure,
			MaxRetries:  3,
			BackoffBase: time.Millisecond,
		},
	}, func(string) error {
		t.Fatal("restart callback should not be called for stopped service transitions")
		return nil
	})

	supervisor.OnStateTransition(newCrashTransition("api", "stopped"))

	time.Sleep(50 * time.Millisecond)
}

func TestSupervisorOnStateTransition_DoesNotRestartAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	supervisor := New(ctx, map[string]service.RestartPolicy{
		"api": {
			Policy:      service.RestartPolicyOnFailure,
			MaxRetries:  3,
			BackoffBase: time.Millisecond,
		},
	}, func(string) error {
		t.Fatal("restart callback should not run after cancellation")
		return nil
	})

	supervisor.OnStateTransition(newCrashTransition("api", "running"))

	time.Sleep(50 * time.Millisecond)
}

func TestSupervisorOnStateTransition_ResetsAttemptsAfterRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	policies := map[string]service.RestartPolicy{
		"worker": {
			Policy:      service.RestartPolicyOnFailure,
			MaxRetries:  1,
			BackoffBase: time.Millisecond,
		},
	}

	var mu sync.Mutex
	restartCalls := 0
	supervisor := New(ctx, policies, func(serviceName string) error {
		mu.Lock()
		defer mu.Unlock()
		restartCalls++
		assert.Equal(t, "worker", serviceName)
		return nil
	})

	crash := newCrashTransition("worker", "running")
	supervisor.OnStateTransition(crash)
	supervisor.OnStateTransition(crash)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return restartCalls == 1
	}, time.Second, 10*time.Millisecond)

	supervisor.OnStateTransition(monitor.StateTransition{
		ServiceName: "worker",
		Severity:    monitor.SeverityInfo,
		Description: "Service started successfully",
		ToState: &monitor.ServiceState{
			Name:   "worker",
			Status: "running",
			Health: "healthy",
		},
	})

	supervisor.OnStateTransition(crash)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return restartCalls == 2
	}, time.Second, 10*time.Millisecond)
}

func newCrashTransition(serviceName, status string) monitor.StateTransition {
	return monitor.StateTransition{
		ServiceName: serviceName,
		Severity:    monitor.SeverityCritical,
		Description: "Process crashed - PID 4242 no longer exists",
		ToState: &monitor.ServiceState{
			Name:   serviceName,
			Status: status,
		},
	}
}
