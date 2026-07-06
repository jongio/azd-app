package serviceinfo

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/resourcesampler"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withStubbedSampler(t *testing.T, usage resourcesampler.Usage) {
	t.Helper()
	original := sampleResourceUsage
	sampleResourceUsage = func(int) resourcesampler.Usage { return usage }
	t.Cleanup(func() { sampleResourceUsage = original })
}

func findService(result []*ServiceInfo, name string) *ServiceInfo {
	for _, svc := range result {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}

func TestMergeServiceInfoResourceOverThreshold(t *testing.T) {
	const mb = 1024 * 1024
	withStubbedSampler(t, resourcesampler.Usage{CPUPercent: 150, MemoryBytes: 800 * mb})

	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"api": {
				Host: "local",
				Resources: &service.ResourceThresholds{
					CPUPercent: 90,
					MemoryMB:   512,
				},
			},
		},
	}
	running := []*registry.ServiceRegistryEntry{
		{Name: "api", Status: "running", PID: 4321},
	}

	result := mergeServiceInfo(azureYaml, running, nil, nil)
	api := findService(result, "api")
	require.NotNil(t, api)
	require.NotNil(t, api.Local)

	assert.Equal(t, 90.0, api.Local.CPUThresholdPercent)
	assert.Equal(t, uint64(512), api.Local.MemoryThresholdMB)
	assert.True(t, api.Local.CPUOverThreshold, "cpu 150 over 90 should flag")
	assert.True(t, api.Local.MemoryOverThreshold, "memory 800MB over 512 should flag")
}

func TestMergeServiceInfoResourceUnderThreshold(t *testing.T) {
	const mb = 1024 * 1024
	withStubbedSampler(t, resourcesampler.Usage{CPUPercent: 20, MemoryBytes: 100 * mb})

	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"api": {
				Host: "local",
				Resources: &service.ResourceThresholds{
					CPUPercent: 90,
					MemoryMB:   512,
				},
			},
		},
	}
	running := []*registry.ServiceRegistryEntry{
		{Name: "api", Status: "running", PID: 4321},
	}

	api := findService(mergeServiceInfo(azureYaml, running, nil, nil), "api")
	require.NotNil(t, api)
	require.NotNil(t, api.Local)

	assert.False(t, api.Local.CPUOverThreshold)
	assert.False(t, api.Local.MemoryOverThreshold)
	// Thresholds are still echoed so the dashboard can show the configured limits.
	assert.Equal(t, 90.0, api.Local.CPUThresholdPercent)
	assert.Equal(t, uint64(512), api.Local.MemoryThresholdMB)
}

func TestMergeServiceInfoNoThresholdsConfigured(t *testing.T) {
	const mb = 1024 * 1024
	withStubbedSampler(t, resourcesampler.Usage{CPUPercent: 999, MemoryBytes: 9999 * mb})

	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"api": {Host: "local"},
		},
	}
	running := []*registry.ServiceRegistryEntry{
		{Name: "api", Status: "running", PID: 4321},
	}

	api := findService(mergeServiceInfo(azureYaml, running, nil, nil), "api")
	require.NotNil(t, api)
	require.NotNil(t, api.Local)

	assert.False(t, api.Local.CPUOverThreshold)
	assert.False(t, api.Local.MemoryOverThreshold)
	assert.Zero(t, api.Local.CPUThresholdPercent)
	assert.Zero(t, api.Local.MemoryThresholdMB)
}
