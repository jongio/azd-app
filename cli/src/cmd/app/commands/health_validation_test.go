package commands

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// setHealthFlagState installs a valid baseline for every package-level health
// flag and restores the previous values when the test ends. Each case then
// perturbs exactly the one variable it is asserting on, so a failure names the
// branch that produced it.
func setHealthFlagState(t *testing.T) {
	t.Helper()

	prev := struct {
		stream          bool
		interval        time.Duration
		timeout         time.Duration
		output          string
		metrics         bool
		metricsPort     int
		breaker         bool
		breakCount      int
		breakTime       time.Duration
		restoreRequired bool
	}{
		stream:          healthStream,
		interval:        healthInterval,
		timeout:         healthTimeout,
		output:          healthOutput,
		metrics:         healthEnableMetrics,
		metricsPort:     healthMetricsPort,
		breaker:         healthCircuitBreaker,
		breakCount:      healthCircuitBreakCount,
		breakTime:       healthCircuitBreakTime,
		restoreRequired: true,
	}

	t.Cleanup(func() {
		if !prev.restoreRequired {
			return
		}
		healthStream = prev.stream
		healthInterval = prev.interval
		healthTimeout = prev.timeout
		healthOutput = prev.output
		healthEnableMetrics = prev.metrics
		healthMetricsPort = prev.metricsPort
		healthCircuitBreaker = prev.breaker
		healthCircuitBreakCount = prev.breakCount
		healthCircuitBreakTime = prev.breakTime
	})

	healthStream = false
	healthInterval = defaultHealthInterval
	healthTimeout = defaultHealthTimeout
	healthOutput = outputFormatText
	healthEnableMetrics = false
	healthMetricsPort = 9090
	healthCircuitBreaker = false
	healthCircuitBreakCount = 5
	healthCircuitBreakTime = 60 * time.Second
}

// TestValidateHealthFlagsUsesTypedErrors pins that every rejection branch of
// validateHealthFlags returns an *azdext.LocalError carrying a machine-readable
// code and an actionable suggestion, not a bare fmt.Errorf. Without this, a
// branch can silently regress to an untyped error that the host cannot classify
// and that gives the user nothing to act on.
func TestValidateHealthFlagsUsesTypedErrors(t *testing.T) {
	cases := []struct {
		name        string
		perturb     func()
		wantMessage string
	}{
		{
			name:        "interval below floor",
			perturb:     func() { healthInterval = 500 * time.Millisecond },
			wantMessage: "interval must be at least",
		},
		{
			name:        "timeout below floor",
			perturb:     func() { healthTimeout = 500 * time.Millisecond },
			wantMessage: "timeout must be between",
		},
		{
			name:        "timeout above ceiling",
			perturb:     func() { healthTimeout = maxHealthTimeout + time.Second },
			wantMessage: "timeout must be between",
		},
		{
			name: "streaming interval not greater than timeout",
			perturb: func() {
				healthStream = true
				healthInterval = 5 * time.Second
				healthTimeout = 5 * time.Second
			},
			wantMessage: "must be greater than timeout",
		},
		{
			name:        "unknown output format",
			perturb:     func() { healthOutput = "xml" },
			wantMessage: `invalid --output value "xml"`,
		},
		{
			name: "metrics port below range",
			perturb: func() {
				healthEnableMetrics = true
				healthMetricsPort = 0
			},
			wantMessage: "metrics port must be between 1 and 65535",
		},
		{
			name: "metrics port above range",
			perturb: func() {
				healthEnableMetrics = true
				healthMetricsPort = 65536
			},
			wantMessage: "metrics port must be between 1 and 65535",
		},
		{
			name: "circuit breaker count below floor",
			perturb: func() {
				healthCircuitBreaker = true
				healthCircuitBreakCount = 0
			},
			wantMessage: "circuit breaker count must be at least 1",
		},
		{
			name: "circuit breaker timeout below floor",
			perturb: func() {
				healthCircuitBreaker = true
				healthCircuitBreakTime = 500 * time.Millisecond
			},
			wantMessage: "circuit breaker timeout must be at least 1s",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setHealthFlagState(t)
			tc.perturb()

			err := validateHealthFlags()
			if err == nil {
				t.Fatal("expected validateHealthFlags to reject this state, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantMessage)
			}

			var localErr *azdext.LocalError
			if !errors.As(err, &localErr) {
				t.Fatalf("error is %T, want *azdext.LocalError so the host can classify it", err)
			}
			if localErr.Code != ErrCodeInvalidFlagUsage {
				t.Errorf("code = %q, want %q", localErr.Code, ErrCodeInvalidFlagUsage)
			}
			if localErr.Category != azdext.LocalErrorCategoryValidation {
				t.Errorf("category = %q, want %q", localErr.Category, azdext.LocalErrorCategoryValidation)
			}
			if strings.TrimSpace(localErr.Suggestion) == "" {
				t.Error("suggestion is empty; every rejection must tell the user what to do instead")
			}
		})
	}
}

// TestValidateHealthFlagsAcceptsDefaults guards the baseline the table above
// perturbs. If the defaults ever drift out of their own valid range, every case
// would still fail for the wrong reason and the table would stop proving
// anything about the branch it names.
func TestValidateHealthFlagsAcceptsDefaults(t *testing.T) {
	setHealthFlagState(t)

	if err := validateHealthFlags(); err != nil {
		t.Fatalf("default health flag state is rejected: %v", err)
	}
}

// TestHealthSuggestionsNameRealFlags keeps the suggestion text honest. Each
// suggestion tells the user to pass a flag; if that flag is not registered on
// the command, the advice is wrong.
func TestHealthSuggestionsNameRealFlags(t *testing.T) {
	cmd := NewHealthCommand()

	for _, name := range []string{
		"interval",
		"timeout",
		"metrics-port",
		"circuit-breaker",
		"circuit-break-count",
		"circuit-break-timeout",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("suggestion text references --%s, which the command does not register", name)
		}
	}
}
