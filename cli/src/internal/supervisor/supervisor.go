// Package supervisor provides automatic restart supervision for crashed services.
package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

const (
	crashTransitionPrefix      = "Process crashed - PID "
	recoveryStartedDescription = "Service started successfully"
	recoveryHealthyDescription = "Service became healthy"
)

// RestartFunc performs a service restart operation.
type RestartFunc func(serviceName string) error

// Supervisor manages per-service restart attempts and triggers automatic restarts.
type Supervisor struct {
	ctx      context.Context
	restart  RestartFunc
	mu       sync.Mutex
	policies map[string]service.RestartPolicy
	attempts map[string]int
}

// New creates a restart supervisor bound to a context and service policy map.
func New(ctx context.Context, policies map[string]service.RestartPolicy, restart RestartFunc) *Supervisor {
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedPolicies := make(map[string]service.RestartPolicy, len(policies))
	for serviceName, policy := range policies {
		normalizedPolicies[serviceName] = normalizePolicy(policy)
	}

	return &Supervisor{
		ctx:      ctx,
		restart:  restart,
		policies: normalizedPolicies,
		attempts: make(map[string]int),
	}
}

// ShouldRestart decides if a restart should be attempted for a transition.
func ShouldRestart(policy service.RestartPolicy, exitedUnexpectedly bool, attempt int) bool {
	if !exitedUnexpectedly || attempt <= 0 {
		return false
	}

	normalized := normalizePolicy(policy)
	if normalized.Policy == service.RestartPolicyNever {
		return false
	}

	return attempt <= normalized.MaxRetries
}

// BackoffDuration returns the exponential backoff delay for an attempt.
func BackoffDuration(attempt int, base time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	if base <= 0 {
		base = service.DefaultRestartBackoffBase
	}

	if base > service.MaxRestartBackoffBase {
		return service.MaxRestartBackoffBase
	}

	delay := base
	for i := 1; i < attempt; i++ {
		if delay >= service.MaxRestartBackoffBase/2 {
			return service.MaxRestartBackoffBase
		}
		delay *= 2
	}

	if delay > service.MaxRestartBackoffBase {
		return service.MaxRestartBackoffBase
	}

	return delay
}

// OnStateTransition handles state transitions and schedules restarts when needed.
func (s *Supervisor) OnStateTransition(transition monitor.StateTransition) {
	if s == nil || s.restart == nil || transition.ServiceName == "" {
		return
	}

	select {
	case <-s.ctx.Done():
		return
	default:
	}

	if isRecoveryTransition(transition) {
		s.resetAttempts(transition.ServiceName)
		return
	}

	if !isCrashTransition(transition) {
		return
	}

	policy, exists := s.getPolicy(transition.ServiceName)
	if !exists {
		return
	}

	unexpectedExit := isUnexpectedExit(transition)
	attempt := s.peekNextAttempt(transition.ServiceName)
	if !ShouldRestart(policy, unexpectedExit, attempt) {
		if unexpectedExit && policy.Policy != service.RestartPolicyNever && attempt > policy.MaxRetries {
			slog.Warn("auto-restart retry limit reached",
				"service", transition.ServiceName,
				"maxRetries", policy.MaxRetries)
		}
		return
	}

	s.setAttempt(transition.ServiceName, attempt)
	backoff := BackoffDuration(attempt, policy.BackoffBase)

	slog.Warn("service crash detected, scheduling auto-restart",
		"service", transition.ServiceName,
		"attempt", attempt,
		"maxRetries", policy.MaxRetries,
		"backoff", backoff)

	go s.restartAfterBackoff(transition.ServiceName, attempt, backoff)
}

func (s *Supervisor) restartAfterBackoff(serviceName string, attempt int, backoff time.Duration) {
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-s.ctx.Done():
		return
	case <-timer.C:
	}

	if err := s.restart(serviceName); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		slog.Error("auto-restart failed",
			"service", serviceName,
			"attempt", attempt,
			"error", err)
		return
	}

	slog.Info("service restarted automatically",
		"service", serviceName,
		"attempt", attempt)
}

func (s *Supervisor) getPolicy(serviceName string) (service.RestartPolicy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, exists := s.policies[serviceName]
	return policy, exists
}

func (s *Supervisor) peekNextAttempt(serviceName string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.attempts[serviceName] + 1
}

func (s *Supervisor) setAttempt(serviceName string, attempt int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts[serviceName] = attempt
}

func (s *Supervisor) resetAttempts(serviceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.attempts, serviceName)
}

func normalizePolicy(policy service.RestartPolicy) service.RestartPolicy {
	switch normalized := strings.ToLower(strings.TrimSpace(policy.Policy)); normalized {
	case service.RestartPolicyOnFailure, service.RestartPolicyAlways:
		policy.Policy = normalized
	default:
		policy.Policy = service.RestartPolicyNever
	}

	if policy.MaxRetries <= 0 {
		policy.MaxRetries = service.DefaultRestartMaxRetries
	}

	if policy.BackoffBase <= 0 {
		policy.BackoffBase = service.DefaultRestartBackoffBase
	}

	if policy.BackoffBase > service.MaxRestartBackoffBase {
		policy.BackoffBase = service.MaxRestartBackoffBase
	}

	return policy
}

func isCrashTransition(transition monitor.StateTransition) bool {
	return transition.Severity == monitor.SeverityCritical &&
		strings.HasPrefix(transition.Description, crashTransitionPrefix)
}

func isUnexpectedExit(transition monitor.StateTransition) bool {
	if transition.ToState == nil {
		return true
	}

	switch strings.ToLower(strings.TrimSpace(transition.ToState.Status)) {
	case "stopping", "stopped", "not-running", "completed", "built":
		return false
	default:
		return true
	}
}

func isRecoveryTransition(transition monitor.StateTransition) bool {
	if transition.Severity != monitor.SeverityInfo {
		return false
	}

	if transition.Description == recoveryStartedDescription || transition.Description == recoveryHealthyDescription {
		return true
	}

	if transition.ToState == nil {
		return false
	}

	status := strings.ToLower(strings.TrimSpace(transition.ToState.Status))
	if status == "running" || status == "ready" {
		return true
	}

	return strings.ToLower(strings.TrimSpace(transition.ToState.Health)) == "healthy"
}
