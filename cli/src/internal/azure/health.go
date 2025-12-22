package azure

import (
	"math"
	"time"
)

// ConnectionHealth represents the health status of an Azure connection.
type ConnectionHealth struct {
	Status           string    `json:"status"`           // "connected", "degraded", "disconnected"
	LastSuccess      time.Time `json:"lastSuccess"`      // Last successful operation
	ConsecutiveFails int       `json:"consecutiveFails"` // Number of consecutive failures
	LastError        string    `json:"lastError"`        // Last error message
	NextRetry        time.Time `json:"nextRetry"`        // Next retry time (if disconnected)
}

// PollingState manages exponential backoff for Azure log polling.
type PollingState struct {
	baseInterval     time.Duration
	maxBackoff       time.Duration
	failureCount     int
	lastSuccess      time.Time
	lastError        string
	consecutiveFails int
}

// NewPollingState creates a new polling state with default settings.
func NewPollingState(baseInterval time.Duration) *PollingState {
	if baseInterval == 0 {
		baseInterval = 5 * time.Second
	}
	return &PollingState{
		baseInterval: baseInterval,
		maxBackoff:   60 * time.Second, // Max 1 minute backoff
		lastSuccess:  time.Now(),
	}
}

// NextDelay calculates the next polling delay based on failure count.
// Uses exponential backoff: baseInterval * 2^failureCount, capped at maxBackoff.
func (p *PollingState) NextDelay() time.Duration {
	if p.failureCount == 0 {
		return p.baseInterval
	}

	// Exponential backoff: 2^n * baseInterval
	delay := time.Duration(float64(p.baseInterval) * math.Pow(2, float64(p.failureCount)))
	if delay > p.maxBackoff {
		delay = p.maxBackoff
	}
	return delay
}

// RecordSuccess resets the failure count after a successful operation.
func (p *PollingState) RecordSuccess() {
	p.failureCount = 0
	p.consecutiveFails = 0
	p.lastSuccess = time.Now()
	p.lastError = ""
}

// RecordFailure increments the failure count and tracks the error.
func (p *PollingState) RecordFailure(err error) {
	p.failureCount++
	p.consecutiveFails++
	if err != nil {
		p.lastError = err.Error()
	}
}

// GetHealth returns the current connection health status.
func (p *PollingState) GetHealth() ConnectionHealth {
	status := "connected"
	if p.consecutiveFails > 0 {
		if p.consecutiveFails >= 3 {
			status = "disconnected"
		} else {
			status = "degraded"
		}
	}

	nextRetry := time.Time{}
	if p.consecutiveFails > 0 {
		nextRetry = time.Now().Add(p.NextDelay())
	}

	return ConnectionHealth{
		Status:           status,
		LastSuccess:      p.lastSuccess,
		ConsecutiveFails: p.consecutiveFails,
		LastError:        p.lastError,
		NextRetry:        nextRetry,
	}
}

// ShouldRetry determines if we should retry now based on backoff delay.
func (p *PollingState) ShouldRetry(lastAttempt time.Time) bool {
	if p.failureCount == 0 {
		return true // No failures, proceed normally
	}
	return time.Since(lastAttempt) >= p.NextDelay()
}
