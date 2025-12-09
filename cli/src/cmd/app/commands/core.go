// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
	"github.com/jongio/azd-app/cli/src/internal/output"
)

// Global orchestrator instance shared across all commands.
var cmdOrchestrator *orchestrator.Orchestrator

// ExecutionContext holds runtime configuration for command execution.
type ExecutionContext struct {
	CacheEnabled bool
}

// ReqsResult represents the JSON output structure for reqs command.
type ReqsResult struct {
	Satisfied bool        `json:"satisfied"`
	Reqs      []ReqResult `json:"reqs"`
}

// DepsResult represents the JSON output structure for deps command.
type DepsResult struct {
	Success  bool            `json:"success"`
	Projects []InstallResult `json:"projects"`
	Message  string          `json:"message,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// CleanDependenciesError represents an error during dependency cleaning with details.
type CleanDependenciesError struct {
	Count   int
	Details []string
}

// Error implements the error interface.
func (e *CleanDependenciesError) Error() string {
	if len(e.Details) == 0 {
		return fmt.Sprintf("encountered %d error(s) while cleaning dependencies", e.Count)
	}
	if len(e.Details) == 1 {
		return fmt.Sprintf("failed to clean dependencies: %s", e.Details[0])
	}
	return fmt.Sprintf("encountered %d error(s) while cleaning dependencies:\n  - %s",
		e.Count, strings.Join(e.Details, "\n  - "))
}

const (
	// msgNoProjectsDetected is used when no projects are found for dependency installation.
	msgNoProjectsDetected = "No projects detected"
)

// Global execution context (temporary until proper context passing is implemented)
var execContext = &ExecutionContext{
	CacheEnabled: true, // Default: cache is enabled
}

// SetCacheEnabled configures whether caching should be enabled.
func SetCacheEnabled(enabled bool) {
	execContext.CacheEnabled = enabled
}

// init initializes the command orchestrator and registers all commands.
func init() {
	cmdOrchestrator = orchestrator.NewOrchestrator()

	// Register commands with their dependencies
	// reqs has no dependencies
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:    "reqs",
		Execute: executeReqs,
	}); err != nil {
		// Log error but don't exit - let the app handle it gracefully
		fmt.Fprintf(os.Stderr, "Warning: Failed to register reqs command: %v\n", err)
	}

	// deps depends on reqs
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "deps",
		Dependencies: []string{"reqs"},
		Execute:      executeDeps,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register deps command: %v\n", err)
	}

	// run depends on deps (which depends on reqs)
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "run",
		Dependencies: []string{"deps"},
		Execute:      executeRun,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register run command: %v\n", err)
	}

	// test depends on deps (test tools like jest/vitest must be installed first)
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "test",
		Dependencies: []string{"deps"},
		Execute:      executeTest,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register test command: %v\n", err)
	}
}

// executeReqs is the core logic for the reqs command.
func executeReqs() error {
	output.CommandHeader("reqs", "Check required prerequisites")
	// Load azure.yaml
	azureYamlPath, azureYaml, err := loadAzureYaml()
	if err != nil {
		return err
	}

	// Build effective requirements list
	effectiveReqs := azureYaml.Reqs

	// Auto-inject Docker requirement if container services are detected
	if azureYaml.hasContainerServices() && !azureYaml.hasDockerReq() {
		dockerReq := Prerequisite{
			Name:         "docker",
			MinVersion:   "20.0.0",
			CheckRunning: true,
		}
		effectiveReqs = append(effectiveReqs, dockerReq)
	}

	// If no reqs section exists, skip checks gracefully
	if len(effectiveReqs) == 0 {
		if output.IsJSON() {
			return output.PrintJSON(ReqsResult{
				Satisfied: true,
				Reqs:      []ReqResult{},
			})
		}
		return nil
	}

	// Initialize cache manager
	cacheManager := createCacheManager(execContext.CacheEnabled)

	// Check requirements (with caching)
	results, allSatisfied := checkRequirementsWithCache(effectiveReqs, azureYamlPath, cacheManager)

	// JSON output
	if output.IsJSON() {
		return output.PrintJSON(ReqsResult{
			Satisfied: allSatisfied,
			Reqs:      results,
		})
	}

	// Default output
	output.Newline()
	if !allSatisfied {
		output.Info("%s If you recently installed any missing tools, run 'azd app reqs --fix' to refresh PATH", output.IconBulb)
		return fmt.Errorf("requirement check failed")
	}

	output.Success("All reqs satisfied!")
	return nil
}

// executeDeps is the core logic for the deps command.
func executeDeps() error {
	// Get options set by the command
	opts := GetDepsOptions()

	// Create executor with production dependencies and execute
	executor := newDepsExecutor(opts)
	return executor.execute()
}

// executeRun is the function executed by the orchestrator for the run command.
// This ensures deps (and transitively reqs) are run before starting services.
func executeRun() error {
	// The actual run logic is handled by the run command's RunE function
	// This is just a marker to ensure the dependency chain is executed
	return nil
}

// executeTest is the function executed by the orchestrator for the test command.
// This ensures reqs are run before executing tests.
func executeTest() error {
	// The actual test logic is handled by the test command's RunE function
	// This is just a marker to ensure the dependency chain is executed
	return nil
}
