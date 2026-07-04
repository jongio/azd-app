package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// AllocatePortFunc allocates a port for a scaled service instance.
type AllocatePortFunc func(instanceName string, avoid map[int]bool) (int, error)

// ExpandScaledRuntimes expands runtimes and services for services listed in scale.
func ExpandScaledRuntimes(
	runtimes []*ServiceRuntime,
	services map[string]Service,
	scale map[string]int,
	alloc AllocatePortFunc,
) ([]*ServiceRuntime, map[string]Service, error) {
	if len(scale) == 0 {
		return runtimes, services, nil
	}

	runtimeByName := make(map[string]*ServiceRuntime, len(runtimes))
	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}
		runtimeByName[runtime.Name] = runtime
	}

	validNames := sortedRunnableServiceNames(runtimeByName, services)
	unknownServices := make([]string, 0)
	scaleNames := make([]string, 0, len(scale))

	for serviceName := range scale {
		scaleNames = append(scaleNames, serviceName)

		runtime, hasRuntime := runtimeByName[serviceName]
		_, hasService := services[serviceName]
		if !hasRuntime || !hasService {
			unknownServices = append(unknownServices, serviceName)
			continue
		}

		if runtime.Type == ServiceTypeContainer {
			return nil, nil, fmt.Errorf(
				"service %q cannot be scaled, container services binding fixed host ports are not supported",
				serviceName,
			)
		}
	}

	if len(unknownServices) > 0 {
		sort.Strings(unknownServices)
		return nil, nil, fmt.Errorf(
			"unknown service in --scale: %s (valid services: %s)",
			strings.Join(unknownServices, ", "),
			formatValidServiceNames(validNames),
		)
	}

	sort.Strings(scaleNames)

	expandedServices := make(map[string]Service, len(services))
	for name, svc := range services {
		expandedServices[name] = svc
	}

	avoidPorts := make(map[int]bool, len(runtimes))
	for _, runtime := range runtimes {
		if runtime == nil || runtime.Port <= 0 {
			continue
		}
		avoidPorts[runtime.Port] = true
	}

	expandedRuntimes := make([]*ServiceRuntime, 0, len(runtimes))
	baseRuntimeCopies := make(map[string]*ServiceRuntime, len(scale))

	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}

		runtimeCopy := cloneServiceRuntime(runtime)
		if _, shouldScale := scale[runtime.Name]; shouldScale {
			if runtimeCopy.Env == nil {
				runtimeCopy.Env = make(map[string]string)
			}
			runtimeCopy.Env["AZD_APP_INSTANCE"] = "1"
			baseRuntimeCopies[runtime.Name] = runtimeCopy
		}

		expandedRuntimes = append(expandedRuntimes, runtimeCopy)
	}

	for _, serviceName := range scaleNames {
		count := scale[serviceName]
		if count <= 1 {
			continue
		}

		baseRuntime, exists := baseRuntimeCopies[serviceName]
		if !exists {
			return nil, nil, fmt.Errorf("failed to prepare scaled runtime for service %q", serviceName)
		}

		baseService := services[serviceName]
		for i := 2; i <= count; i++ {
			instanceName := fmt.Sprintf("%s-%d", serviceName, i)
			instanceRuntime := cloneServiceRuntime(baseRuntime)
			instanceRuntime.Name = instanceName

			if instanceRuntime.Env == nil {
				instanceRuntime.Env = make(map[string]string)
			}
			instanceRuntime.Env["AZD_APP_INSTANCE"] = strconv.Itoa(i)

			if baseRuntime.Port > 0 {
				if alloc == nil {
					return nil, nil, fmt.Errorf("failed to allocate port for %s: no allocator provided", instanceName)
				}

				portsBefore := copyPortSet(avoidPorts)
				newPort, err := alloc(instanceName, avoidPorts)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to allocate port for %s: %w", instanceName, err)
				}
				if newPort <= 0 {
					return nil, nil, fmt.Errorf("failed to allocate port for %s: allocator returned %d", instanceName, newPort)
				}
				if portsBefore[newPort] {
					return nil, nil, fmt.Errorf("failed to allocate port for %s: port %d is already in use", instanceName, newPort)
				}

				instanceRuntime.Port = newPort
				updateScaledHealthCheckPort(instanceRuntime, baseRuntime.Port, newPort)
				avoidPorts[newPort] = true
			}

			expandedRuntimes = append(expandedRuntimes, instanceRuntime)
			expandedServices[instanceName] = baseService
		}
	}

	return expandedRuntimes, expandedServices, nil
}

func cloneServiceRuntime(runtime *ServiceRuntime) *ServiceRuntime {
	if runtime == nil {
		return nil
	}

	clone := *runtime
	if runtime.Args != nil {
		clone.Args = append([]string(nil), runtime.Args...)
	}
	clone.Env = cloneStringMap(runtime.Env)

	return &clone
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}

	return cloned
}

func copyPortSet(ports map[int]bool) map[int]bool {
	snapshot := make(map[int]bool, len(ports))
	for port, used := range ports {
		snapshot[port] = used
	}
	return snapshot
}

func updateScaledHealthCheckPort(runtime *ServiceRuntime, basePort, newPort int) {
	if runtime == nil || basePort <= 0 || newPort <= 0 {
		return
	}

	healthType := strings.ToLower(strings.TrimSpace(runtime.HealthCheck.Type))
	if runtime.HealthCheck.Port == basePort || (runtime.HealthCheck.Port == 0 && isPortBasedHealthType(healthType)) {
		runtime.HealthCheck.Port = newPort
	}
}

func isPortBasedHealthType(healthType string) bool {
	switch healthType {
	case "", ServiceTypeHTTP, ServiceTypeTCP, "port":
		return true
	default:
		return false
	}
}

func sortedRunnableServiceNames(runtimes map[string]*ServiceRuntime, services map[string]Service) []string {
	names := make([]string, 0, len(runtimes))
	for name := range runtimes {
		if _, exists := services[name]; exists {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func formatValidServiceNames(validNames []string) string {
	if len(validNames) == 0 {
		return "<none>"
	}

	return strings.Join(validNames, ", ")
}
