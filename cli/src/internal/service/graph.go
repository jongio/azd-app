package service

import (
	"fmt"
	"sort"
)

// BuildDependencyGraph creates a dependency graph from services and resources.
func BuildDependencyGraph(services map[string]Service, resources map[string]Resource) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// Add service nodes
	for name, svc := range services {
		node := &DependencyNode{
			Name:         name,
			Service:      &svc,
			IsResource:   false,
			Dependencies: svc.Uses,
		}
		graph.Nodes[name] = node
		graph.Edges[name] = svc.Uses
	}

	// Add resource nodes (for dependency tracking, but won't be started)
	for name, res := range resources {
		node := &DependencyNode{
			Name:         name,
			IsResource:   true,
			Dependencies: res.Uses,
		}
		graph.Nodes[name] = node
		graph.Edges[name] = res.Uses
	}

	// Detect cycles
	if err := DetectCycles(graph); err != nil {
		return nil, err
	}

	// Calculate topological levels
	if err := calculateLevels(graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// DetectCycles checks for circular dependencies in the graph.
func DetectCycles(graph *DependencyGraph) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range graph.Nodes {
		if !visited[node] {
			if hasCycle(node, graph, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving: %s", node)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles.
func hasCycle(node string, graph *DependencyGraph, visited map[string]bool, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, dep := range graph.Edges[node] {
		if !visited[dep] {
			if hasCycle(dep, graph, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// calculateLevels assigns topological levels to nodes.
// Level 0 = no dependencies, Level 1 = depends on level 0, etc.
func calculateLevels(graph *DependencyGraph) error {
	// Initialize all levels to 0
	for _, node := range graph.Nodes {
		node.Level = 0
	}

	// Calculate levels iteratively
	changed := true
	iterations := 0
	maxIterations := len(graph.Nodes) * 2 // More iterations for complex graphs

	for changed && iterations < maxIterations {
		changed = false
		iterations++

		for name, node := range graph.Nodes {
			deps := graph.Edges[name]

			// If no dependencies, level stays 0
			if len(deps) == 0 {
				continue
			}

			maxDepLevel := -1

			// Find the maximum level among dependencies
			for _, depName := range deps {
				if depNode, exists := graph.Nodes[depName]; exists {
					if depNode.Level > maxDepLevel {
						maxDepLevel = depNode.Level
					}
				} else {
					// Dependency not found in graph - treat as external (level -1)
					maxDepLevel = max(maxDepLevel, -1)
				}
			}

			// Set level to one more than max dependency level
			newLevel := maxDepLevel + 1
			if newLevel != node.Level {
				node.Level = newLevel
				changed = true
			}
		}
	}

	if iterations >= maxIterations {
		return fmt.Errorf("failed to calculate dependency levels (possible cycle)")
	}

	return nil
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TopologicalSort returns services grouped by dependency level.
// Each slice contains services that can be started in parallel.
func TopologicalSort(graph *DependencyGraph) [][]string {
	// Group services by level
	levelMap := make(map[int][]string)
	maxLevel := 0

	for name, node := range graph.Nodes {
		// Skip resources - we only want to start services
		if node.IsResource {
			continue
		}

		level := node.Level
		levelMap[level] = append(levelMap[level], name)

		if level > maxLevel {
			maxLevel = level
		}
	}

	// Convert to sorted slice of slices
	result := make([][]string, 0, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		if services, exists := levelMap[level]; exists {
			// Sort services within level for deterministic ordering
			sort.Strings(services)
			result = append(result, services)
		}
	}

	return result
}

// GetServiceDependencies returns the direct dependencies of a service.
func GetServiceDependencies(serviceName string, graph *DependencyGraph) []string {
	if edges, exists := graph.Edges[serviceName]; exists {
		return edges
	}
	return []string{}
}

// GetDependents returns services that depend on the given service.
func GetDependents(serviceName string, graph *DependencyGraph) []string {
	dependents := []string{}

	for name, edges := range graph.Edges {
		for _, dep := range edges {
			if dep == serviceName {
				dependents = append(dependents, name)
				break
			}
		}
	}

	sort.Strings(dependents)
	return dependents
}

// FilterGraphByServices returns a subgraph containing only the specified services and their dependencies.
func FilterGraphByServices(graph *DependencyGraph, serviceNames []string) (*DependencyGraph, error) {
	filtered := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// Recursively add services and their dependencies
	visited := make(map[string]bool)
	for _, name := range serviceNames {
		if err := addServiceAndDeps(name, graph, filtered, visited); err != nil {
			return nil, err
		}
	}

	// Recalculate levels for filtered graph
	if err := calculateLevels(filtered); err != nil {
		return nil, err
	}

	return filtered, nil
}

// addServiceAndDeps recursively adds a service and its dependencies to the filtered graph.
func addServiceAndDeps(name string, source *DependencyGraph, dest *DependencyGraph, visited map[string]bool) error {
	if visited[name] {
		return nil
	}

	node, exists := source.Nodes[name]
	if !exists {
		return fmt.Errorf("service or resource '%s' not found", name)
	}

	// Add node to filtered graph
	dest.Nodes[name] = node
	dest.Edges[name] = source.Edges[name]
	visited[name] = true

	// Recursively add dependencies
	for _, dep := range source.Edges[name] {
		if err := addServiceAndDeps(dep, source, dest, visited); err != nil {
			return err
		}
	}

	return nil
}
