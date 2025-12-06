package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// ProjectInstallTask represents a single project installation task.
type ProjectInstallTask struct {
	ID          string
	Description string
	Type        string
	Dir         string
	Path        string
	Manager     string
	Project     interface{} // Store the actual project for installation
}

// ParallelInstaller handles parallel installation of multiple projects with progress tracking.
type ParallelInstaller struct {
	tasks       []ProjectInstallTask
	multiProg   *output.MultiProgress
	mu          sync.Mutex
	results     []ProjectInstallResult
	statusLines []output.StatusLine
	Verbose     bool            // Show full installation output
	ctx         context.Context // Context for cancellation
}

// ProjectInstallResult represents the result of a project installation.
type ProjectInstallResult struct {
	Task    ProjectInstallTask
	Success bool
	Error   error
}

// NewParallelInstaller creates a new parallel installer.
func NewParallelInstaller() *ParallelInstaller {
	return &ParallelInstaller{
		tasks:       []ProjectInstallTask{},
		results:     []ProjectInstallResult{},
		statusLines: []output.StatusLine{},
		ctx:         context.Background(),
	}
}

// NewParallelInstallerWithContext creates a new parallel installer with cancellation support.
func NewParallelInstallerWithContext(ctx context.Context) *ParallelInstaller {
	return &ParallelInstaller{
		tasks:       []ProjectInstallTask{},
		results:     []ProjectInstallResult{},
		statusLines: []output.StatusLine{},
		ctx:         ctx,
	}
}

// AddTask adds a new installation task.
func (pi *ParallelInstaller) AddTask(task ProjectInstallTask) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.tasks = append(pi.tasks, task)
}

// AddNodeProject adds a Node.js project installation task.
func (pi *ParallelInstaller) AddNodeProject(project types.NodeProject) {
	projectName := getProjectName(project.Dir)
	task := ProjectInstallTask{
		ID:          project.Dir,
		Description: projectName + " (" + project.PackageManager + ")",
		Type:        "node",
		Dir:         project.Dir,
		Manager:     project.PackageManager,
		Project:     project,
	}
	pi.AddTask(task)
}

// AddPythonProject adds a Python project installation task.
func (pi *ParallelInstaller) AddPythonProject(project types.PythonProject) {
	projectName := getProjectName(project.Dir)
	task := ProjectInstallTask{
		ID:          project.Dir,
		Description: projectName + " (" + project.PackageManager + ")",
		Type:        "python",
		Dir:         project.Dir,
		Manager:     project.PackageManager,
		Project:     project,
	}
	pi.AddTask(task)
}

// AddDotnetProject adds a .NET project installation task.
func (pi *ParallelInstaller) AddDotnetProject(project types.DotnetProject) {
	projectName := getProjectName(project.Path)
	task := ProjectInstallTask{
		ID:          project.Path,
		Description: projectName + " (dotnet)",
		Type:        "dotnet",
		Path:        project.Path,
		Manager:     "dotnet",
		Project:     project,
	}
	pi.AddTask(task)
}

// executeTask is the unified task execution logic.
// It handles all project types and writes output to the provided writer.
func (pi *ParallelInstaller) executeTask(task ProjectInstallTask, writer io.Writer) error {
	// Check for cancellation before starting
	select {
	case <-pi.ctx.Done():
		return pi.ctx.Err()
	default:
	}

	switch task.Type {
	case "node":
		if project, ok := task.Project.(types.NodeProject); ok {
			return installNodeDependenciesWithWriter(project, writer)
		}
	case "python":
		if project, ok := task.Project.(types.PythonProject); ok {
			return setupPythonVirtualEnvWithWriter(project, writer)
		}
	case "dotnet":
		if project, ok := task.Project.(types.DotnetProject); ok {
			return restoreDotnetProjectWithWriter(project, writer)
		}
	}
	return fmt.Errorf("unknown task type: %s", task.Type)
}

// addResult safely adds a result to the results slice.
func (pi *ParallelInstaller) addResult(result ProjectInstallResult) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.results = append(pi.results, result)

	// Build status line
	statusLine := output.StatusLine{
		Description: result.Task.Description,
		Success:     result.Success,
	}
	if result.Error != nil {
		statusLine.Error = result.Error.Error()
	}
	pi.statusLines = append(pi.statusLines, statusLine)
}

// Run executes all tasks with progress tracking.
// Tasks are grouped by package manager to avoid race conditions:
// - pnpm tasks run sequentially (shared global store causes conflicts)
// - All other tasks (npm, yarn, pip, dotnet) run in parallel
func (pi *ParallelInstaller) Run() error {
	if len(pi.tasks) == 0 {
		return nil
	}

	// Check for cancellation
	select {
	case <-pi.ctx.Done():
		return pi.ctx.Err()
	default:
	}

	// In verbose mode, skip progress bars and show full output
	if pi.Verbose {
		return pi.runVerbose()
	}

	// Initialize multi-progress
	pi.multiProg = output.NewMultiProgress()

	// Add all tasks to the progress display first
	for _, task := range pi.tasks {
		pi.multiProg.AddBar(task.ID, task.Description)
	}

	// Start rendering progress bars (mpb handles space automatically)
	pi.multiProg.Start()

	// Separate pnpm tasks from others
	pnpmTasks, parallelTasks := pi.separateTasksByManager()

	// Run all tasks
	var wg sync.WaitGroup

	// Run non-pnpm tasks in parallel
	for _, task := range parallelTasks {
		wg.Add(1)
		go func(t ProjectInstallTask) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					pi.addResult(ProjectInstallResult{
						Task:    t,
						Success: false,
						Error:   fmt.Errorf("panic during installation: %v", r),
					})
				}
			}()
			pi.runTaskWithProgress(t)
		}(task)
	}

	// Run pnpm tasks sequentially in a separate goroutine
	if len(pnpmTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, task := range pnpmTasks {
				// Check for cancellation between tasks
				select {
				case <-pi.ctx.Done():
					pi.addResult(ProjectInstallResult{
						Task:    task,
						Success: false,
						Error:   pi.ctx.Err(),
					})
					return
				default:
				}
				pi.runTaskWithProgress(task)
			}
		}()
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Stop progress display
	pi.multiProg.Stop()

	// Print summary
	pi.printSummary()

	return nil
}

// separateTasksByManager separates pnpm tasks from other tasks.
func (pi *ParallelInstaller) separateTasksByManager() (pnpmTasks, parallelTasks []ProjectInstallTask) {
	for _, task := range pi.tasks {
		if task.Manager == "pnpm" {
			pnpmTasks = append(pnpmTasks, task)
		} else {
			parallelTasks = append(parallelTasks, task)
		}
	}
	return
}

// runTaskWithProgress executes a task with progress bar tracking.
func (pi *ParallelInstaller) runTaskWithProgress(task ProjectInstallTask) {
	bar := pi.multiProg.GetBar(task.ID)
	bar.Start()

	spinnerWriter := output.NewSpinnerWriter(bar)

	var writer io.Writer = spinnerWriter
	if pi.Verbose {
		writer = os.Stdout
	}

	err := pi.executeTask(task, writer)

	if err != nil {
		bar.Fail(err.Error())
	} else {
		bar.Complete()
	}

	pi.addResult(ProjectInstallResult{
		Task:    task,
		Success: err == nil,
		Error:   err,
	})
}

// runVerbose runs installations with full output instead of progress bars.
func (pi *ParallelInstaller) runVerbose() error {
	pnpmTasks, parallelTasks := pi.separateTasksByManager()

	var wg sync.WaitGroup

	// Run non-pnpm tasks in parallel
	for _, task := range parallelTasks {
		wg.Add(1)
		go func(t ProjectInstallTask) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					pi.addResult(ProjectInstallResult{
						Task:    t,
						Success: false,
						Error:   fmt.Errorf("panic during installation: %v", r),
					})
				}
			}()
			pi.runTaskVerbose(t)
		}(task)
	}

	// Run pnpm tasks sequentially
	if len(pnpmTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, task := range pnpmTasks {
				select {
				case <-pi.ctx.Done():
					pi.addResult(ProjectInstallResult{
						Task:    task,
						Success: false,
						Error:   pi.ctx.Err(),
					})
					return
				default:
				}
				// Print task header for clarity
				fmt.Fprintf(os.Stdout, "\n=== Installing: %s ===\n", task.Description)
				pi.runTaskVerbose(task)
			}
		}()
	}

	wg.Wait()
	pi.printSummary()

	return nil
}

// runTaskVerbose executes a single task with verbose output.
func (pi *ParallelInstaller) runTaskVerbose(task ProjectInstallTask) {
	err := pi.executeTask(task, os.Stdout)
	pi.addResult(ProjectInstallResult{
		Task:    task,
		Success: err == nil,
		Error:   err,
	})
}

// printSummary prints the overall installation summary.
func (pi *ParallelInstaller) printSummary() {
	totalCount := len(pi.results)
	successCount := 0
	var failedTasks []string

	for _, result := range pi.results {
		if result.Success {
			successCount++
		} else {
			errMsg := result.Task.Description
			if result.Error != nil {
				errMsg += ": " + result.Error.Error()
			}
			failedTasks = append(failedTasks, errMsg)
		}
	}

	output.Newline()
	output.PrintSummary(totalCount, successCount, failedTasks)
}

// GetResults returns all installation results.
func (pi *ParallelInstaller) GetResults() []ProjectInstallResult {
	return pi.results
}

// HasFailures returns true if any installation failed.
func (pi *ParallelInstaller) HasFailures() bool {
	for _, result := range pi.results {
		if !result.Success {
			return true
		}
	}
	return false
}

// FailedProjects returns a list of project descriptions that failed installation.
func (pi *ParallelInstaller) FailedProjects() []string {
	var failed []string
	for _, result := range pi.results {
		if !result.Success {
			failed = append(failed, result.Task.Description)
		}
	}
	return failed
}

// TotalProjects returns the total number of projects that were processed.
func (pi *ParallelInstaller) TotalProjects() int {
	return len(pi.results)
}

// getProjectName extracts the project name from a full path.
// For example: "C:\\code\\project\\api" -> "api"
func getProjectName(path string) string {
	cleanPath := filepath.Clean(path)
	baseName := filepath.Base(cleanPath)
	// Handle edge cases where Base returns "." or path separator
	if baseName == "." || baseName == string(filepath.Separator) || baseName == "" {
		return path // Fallback to original path
	}
	return baseName
}
