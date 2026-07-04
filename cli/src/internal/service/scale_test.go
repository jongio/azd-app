package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandScaledRuntimes_NoScaleNoOp(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{
			Name: "worker",
			Port: 8080,
			Env:  map[string]string{"EXISTING": "value"},
		},
	}
	services := map[string]Service{
		"worker": {Uses: []string{"redis"}},
	}

	expandedRuntimes, expandedServices, err := ExpandScaledRuntimes(runtimes, services, map[string]int{}, nil)
	require.NoError(t, err)

	assert.Equal(t, runtimes, expandedRuntimes)
	assert.Equal(t, services, expandedServices)
	require.Len(t, expandedRuntimes, 1)
	assert.Equal(t, runtimes[0], expandedRuntimes[0])
}

func TestExpandScaledRuntimes_ScaleOneSetsBaseInstanceOnly(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{
			Name: "worker",
			Port: 8080,
			Env:  map[string]string{"EXISTING": "value"},
		},
	}
	services := map[string]Service{
		"worker": {Uses: []string{"redis"}},
	}

	allocCalls := 0
	expandedRuntimes, expandedServices, err := ExpandScaledRuntimes(
		runtimes,
		services,
		map[string]int{"worker": 1},
		func(_ string, _ map[int]bool) (int, error) {
			allocCalls++
			return 0, nil
		},
	)
	require.NoError(t, err)
	require.Len(t, expandedRuntimes, 1)
	require.Len(t, expandedServices, 1)

	assert.Equal(t, 0, allocCalls)
	assert.Equal(t, "1", expandedRuntimes[0].Env["AZD_APP_INSTANCE"])
	assert.Equal(t, "value", expandedRuntimes[0].Env["EXISTING"])
	assert.Equal(t, 8080, expandedRuntimes[0].Port)
	assert.NotContains(t, runtimes[0].Env, "AZD_APP_INSTANCE")
}

func TestExpandScaledRuntimes_ScaleToThreeAddsInstances(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{
			Name:  "worker",
			Port:  8080,
			Args:  []string{"run", "--flag"},
			Env:   map[string]string{"EXISTING": "value"},
			Type:  ServiceTypeHTTP,
			Image: "",
			HealthCheck: HealthCheckConfig{
				Type: ServiceTypeTCP,
				Port: 8080,
			},
		},
	}
	services := map[string]Service{
		"worker": {
			Uses: []string{"redis"},
		},
	}

	nextPort := 9001
	allocCalls := 0
	expandedRuntimes, expandedServices, err := ExpandScaledRuntimes(
		runtimes,
		services,
		map[string]int{"worker": 3},
		func(instanceName string, avoid map[int]bool) (int, error) {
			allocCalls++
			require.True(t, avoid[8080], "allocator should see existing base runtime port")
			port := nextPort
			nextPort++
			avoid[port] = true
			return port, nil
		},
	)
	require.NoError(t, err)
	require.Len(t, expandedRuntimes, 3)
	require.Len(t, expandedServices, 3)
	assert.Equal(t, 2, allocCalls)

	baseRuntime := runtimeByName(t, expandedRuntimes, "worker")
	instance2 := runtimeByName(t, expandedRuntimes, "worker-2")
	instance3 := runtimeByName(t, expandedRuntimes, "worker-3")

	assert.Equal(t, "1", baseRuntime.Env["AZD_APP_INSTANCE"])
	assert.Equal(t, "2", instance2.Env["AZD_APP_INSTANCE"])
	assert.Equal(t, "3", instance3.Env["AZD_APP_INSTANCE"])
	assert.Equal(t, "value", instance2.Env["EXISTING"])
	assert.Equal(t, 8080, baseRuntime.Port)
	assert.Equal(t, 9001, instance2.Port)
	assert.Equal(t, 9002, instance3.Port)
	assert.Equal(t, 9001, instance2.HealthCheck.Port)
	assert.Equal(t, 9002, instance3.HealthCheck.Port)
	assert.Equal(t, []string{"run", "--flag"}, instance2.Args)

	require.Contains(t, expandedServices, "worker-2")
	require.Contains(t, expandedServices, "worker-3")
	assert.Equal(t, services["worker"].Uses, expandedServices["worker-2"].Uses)
	assert.Equal(t, services["worker"].Uses, expandedServices["worker-3"].Uses)

	assert.NotContains(t, runtimes[0].Env, "AZD_APP_INSTANCE")
	assert.Equal(t, 1, len(services))
	assert.NotContains(t, services, "worker-2")
}

func TestExpandScaledRuntimes_ContainerServiceReturnsError(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{
			Name: "redis",
			Type: ServiceTypeContainer,
			Port: 6379,
		},
	}
	services := map[string]Service{
		"redis": {Image: "redis:latest"},
	}

	_, _, err := ExpandScaledRuntimes(runtimes, services, map[string]int{"redis": 2}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be scaled")
	assert.Contains(t, err.Error(), "container services")
}

func TestExpandScaledRuntimes_UnknownServiceReturnsError(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{Name: "api", Port: 8080},
	}
	services := map[string]Service{
		"api": {},
	}

	_, _, err := ExpandScaledRuntimes(runtimes, services, map[string]int{"worker": 2}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service in --scale")
	assert.Contains(t, err.Error(), "worker")
	assert.Contains(t, err.Error(), "api")
}

func TestExpandScaledRuntimes_PortZeroDoesNotAllocate(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{
			Name: "worker",
			Port: 0,
			Type: ServiceTypeProcess,
			HealthCheck: HealthCheckConfig{
				Type: "process",
			},
		},
	}
	services := map[string]Service{
		"worker": {},
	}

	allocCalls := 0
	expandedRuntimes, expandedServices, err := ExpandScaledRuntimes(
		runtimes,
		services,
		map[string]int{"worker": 3},
		func(_ string, _ map[int]bool) (int, error) {
			allocCalls++
			return 0, nil
		},
	)
	require.NoError(t, err)
	require.Len(t, expandedRuntimes, 3)
	require.Len(t, expandedServices, 3)

	assert.Equal(t, 0, allocCalls)
	for _, runtime := range expandedRuntimes {
		assert.Equal(t, 0, runtime.Port)
	}
}

func TestExpandScaledRuntimes_AllocatorErrorPropagates(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{
			Name: "worker",
			Port: 8080,
		},
	}
	services := map[string]Service{
		"worker": {},
	}

	expectedErr := errors.New("allocator failed")
	_, _, err := ExpandScaledRuntimes(
		runtimes,
		services,
		map[string]int{"worker": 2},
		func(_ string, _ map[int]bool) (int, error) {
			return 0, expectedErr
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to allocate port for worker-2")
	assert.ErrorIs(t, err, expectedErr)
}

func TestExpandScaledRuntimes_DeterministicScaledOrdering(t *testing.T) {
	runtimes := []*ServiceRuntime{
		{Name: "worker", Port: 8081},
		{Name: "api", Port: 8080},
	}
	services := map[string]Service{
		"worker": {},
		"api":    {},
	}

	nextPort := 9000
	expandedRuntimes, _, err := ExpandScaledRuntimes(
		runtimes,
		services,
		map[string]int{"worker": 2, "api": 2},
		func(instanceName string, avoid map[int]bool) (int, error) {
			nextPort++
			avoid[nextPort] = true
			return nextPort, nil
		},
	)
	require.NoError(t, err)

	names := make([]string, 0, len(expandedRuntimes))
	for _, runtime := range expandedRuntimes {
		names = append(names, runtime.Name)
	}
	assert.Equal(t, []string{"worker", "api", "api-2", "worker-2"}, names)
}

func runtimeByName(t *testing.T, runtimes []*ServiceRuntime, name string) *ServiceRuntime {
	t.Helper()

	for _, runtime := range runtimes {
		if runtime.Name == name {
			return runtime
		}
	}

	require.FailNow(t, fmt.Sprintf("runtime %q not found", name))
	return nil
}
