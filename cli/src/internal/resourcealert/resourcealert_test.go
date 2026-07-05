package resourcealert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThresholdConfigured(t *testing.T) {
	assert.False(t, Threshold{}.Configured())
	assert.True(t, Threshold{CPUPercent: 80}.Configured())
	assert.True(t, Threshold{MemoryMB: 512}.Configured())
}

func TestThresholdExceeds(t *testing.T) {
	now := time.Now()
	const mb = 1024 * 1024

	tests := []struct {
		name      string
		threshold Threshold
		cpu       float64
		memBytes  uint64
		wantKinds []Kind
	}{
		{"nothing configured", Threshold{}, 999, 999 * mb, nil},
		{"cpu under", Threshold{CPUPercent: 80}, 50, 0, nil},
		{"cpu over", Threshold{CPUPercent: 80}, 90, 0, []Kind{KindCPU}},
		{"cpu exactly at limit not over", Threshold{CPUPercent: 80}, 80, 0, nil},
		{"memory under", Threshold{MemoryMB: 512}, 0, 400 * mb, nil},
		{"memory over", Threshold{MemoryMB: 512}, 0, 600 * mb, []Kind{KindMemory}},
		{"both over", Threshold{CPUPercent: 80, MemoryMB: 512}, 95, 600 * mb, []Kind{KindCPU, KindMemory}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breaches := tt.threshold.Exceeds("api", tt.cpu, tt.memBytes, now)
			var kinds []Kind
			for _, b := range breaches {
				kinds = append(kinds, b.Kind)
				assert.Equal(t, "api", b.Service)
			}
			assert.Equal(t, tt.wantKinds, kinds)
		})
	}
}

func TestThresholdExceedsReportsValues(t *testing.T) {
	now := time.Now()
	breaches := Threshold{CPUPercent: 80, MemoryMB: 100}.Exceeds("api", 90, 200*1024*1024, now)
	assert.Len(t, breaches, 2)
	for _, b := range breaches {
		switch b.Kind {
		case KindCPU:
			assert.Equal(t, 90.0, b.Value)
			assert.Equal(t, 80.0, b.Limit)
		case KindMemory:
			assert.Equal(t, 200.0, b.Value)
			assert.Equal(t, 100.0, b.Limit)
		}
	}
}

func TestEngineEvaluateThrottles(t *testing.T) {
	e := NewEngine(30 * time.Second)
	threshold := Threshold{CPUPercent: 80}
	base := time.Now()

	first := e.Evaluate("api", 90, 0, threshold, base)
	assert.Len(t, first, 1, "first breach should fire")

	within := e.Evaluate("api", 95, 0, threshold, base.Add(10*time.Second))
	assert.Empty(t, within, "repeat within debounce should be suppressed")

	after := e.Evaluate("api", 95, 0, threshold, base.Add(31*time.Second))
	assert.Len(t, after, 1, "breach after debounce window should fire again")
}

func TestEngineEvaluateSeparateServicesAndKinds(t *testing.T) {
	e := NewEngine(30 * time.Second)
	threshold := Threshold{CPUPercent: 80, MemoryMB: 100}
	now := time.Now()

	api := e.Evaluate("api", 90, 200*1024*1024, threshold, now)
	assert.Len(t, api, 2, "cpu and memory are throttled independently")

	worker := e.Evaluate("worker", 90, 0, threshold, now)
	assert.Len(t, worker, 1, "a different service has its own throttle state")
}

func TestEngineEvaluateUnconfigured(t *testing.T) {
	e := NewEngine(0)
	assert.Empty(t, e.Evaluate("api", 9999, 9999, Threshold{}, time.Now()))
}

func TestNilEngineSafe(t *testing.T) {
	var e *Engine
	assert.Empty(t, e.Evaluate("api", 90, 0, Threshold{CPUPercent: 80}, time.Now()))
}
