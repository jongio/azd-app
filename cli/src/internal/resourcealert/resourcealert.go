// Package resourcealert evaluates per-service CPU and memory samples against
// optional thresholds and reports the ones that are exceeded. It backs the
// per-service resource warnings raised during a run.
//
// Threshold comparison is a pure function (Threshold.Exceeds), so callers can
// surface an over-threshold indicator without side effects. Engine adds
// per-service, per-dimension debounce so a service that hovers just over a
// limit does not flood the console with repeated warnings.
package resourcealert

import (
	"sync"
	"time"
)

const bytesPerMB = 1024 * 1024

// DefaultDebounce is the minimum time between warnings for the same service and
// resource dimension.
const DefaultDebounce = 30 * time.Second

// Kind identifies which resource dimension a breach is about.
type Kind string

// Resource dimensions that can be exceeded.
const (
	KindCPU    Kind = "cpu"
	KindMemory Kind = "memory"
)

// Threshold defines optional per-service resource limits. A zero field means
// that dimension is not checked.
type Threshold struct {
	// CPUPercent is the maximum CPU usage as a percentage of a single core.
	CPUPercent float64
	// MemoryMB is the maximum resident memory in megabytes.
	MemoryMB uint64
}

// Configured reports whether the threshold checks at least one dimension.
func (t Threshold) Configured() bool {
	return t.CPUPercent > 0 || t.MemoryMB > 0
}

// Breach describes a single threshold that was exceeded. Value and Limit use
// percent for CPU and megabytes for memory.
type Breach struct {
	Service string
	Kind    Kind
	Value   float64
	Limit   float64
	Time    time.Time
}

// Exceeds returns the breaches for one sample, ignoring debounce. It is a pure
// comparison suitable for computing an over-threshold indicator.
func (t Threshold) Exceeds(service string, cpuPercent float64, memoryBytes uint64, now time.Time) []Breach {
	var breaches []Breach
	if t.CPUPercent > 0 && cpuPercent > t.CPUPercent {
		breaches = append(breaches, Breach{Service: service, Kind: KindCPU, Value: cpuPercent, Limit: t.CPUPercent, Time: now})
	}
	if t.MemoryMB > 0 {
		memMB := float64(memoryBytes) / bytesPerMB
		if memMB > float64(t.MemoryMB) {
			breaches = append(breaches, Breach{Service: service, Kind: KindMemory, Value: memMB, Limit: float64(t.MemoryMB), Time: now})
		}
	}
	return breaches
}

// Engine adds per-service, per-dimension debounce on top of Threshold.Exceeds.
// It is safe for concurrent use.
type Engine struct {
	mu        sync.Mutex
	debounce  time.Duration
	lastFired map[string]time.Time
}

// NewEngine creates an engine with the given debounce window. A non-positive
// debounce falls back to DefaultDebounce.
func NewEngine(debounce time.Duration) *Engine {
	if debounce <= 0 {
		debounce = DefaultDebounce
	}
	return &Engine{debounce: debounce, lastFired: make(map[string]time.Time)}
}

// Evaluate returns the breaches that should fire at time now, suppressing
// repeats within the debounce window for the same service and dimension. It
// returns nil when the threshold is unconfigured or nothing is exceeded.
func (e *Engine) Evaluate(service string, cpuPercent float64, memoryBytes uint64, threshold Threshold, now time.Time) []Breach {
	if e == nil || !threshold.Configured() {
		return nil
	}
	candidate := threshold.Exceeds(service, cpuPercent, memoryBytes, now)
	if len(candidate) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	var fired []Breach
	for _, b := range candidate {
		key := string(b.Kind) + "\x00" + b.Service
		if last, ok := e.lastFired[key]; ok && now.Sub(last) < e.debounce {
			continue
		}
		e.lastFired[key] = now
		fired = append(fired, b)
	}
	return fired
}
