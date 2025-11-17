package installer

import (
	"fmt"
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
	Verbose     bool // Show full installation output
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

// Run executes all tasks in parallel with progress tracking.
func (pi *ParallelInstaller) Run() error {
	if len(pi.tasks) == 0 {
		return nil
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

	// Run all tasks in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan ProjectInstallResult, len(pi.tasks))

	for _, task := range pi.tasks {
		wg.Add(1)
		go func(t ProjectInstallTask) {
			// Recover from panics to prevent crash
			defer func() {
				if r := recover(); r != nil {
					resultsChan <- ProjectInstallResult{
						Task:    t,
						Success: false,
						Error:   fmt.Errorf("panic during installation: %v", r),
					}
					wg.Done()
				}
			}()
			pi.runTask(t, &wg, resultsChan)
		}(task)
	}

	// Wait for all tasks to complete
	wg.Wait()
	close(resultsChan)

	// Stop progress display
	pi.multiProg.Stop()

	// Collect results
	for result := range resultsChan {
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

	// Print summary
	pi.printSummary()

	return nil
}

// runTask executes a single installation task with progress tracking.
func (pi *ParallelInstaller) runTask(task ProjectInstallTask, wg *sync.WaitGroup, resultsChan chan<- ProjectInstallResult) {
	defer wg.Done()

	// Get the progress bar for this task
	bar := pi.multiProg.GetBar(task.ID)

	// Mark as started
	bar.Start()

	// Create a spinner writer to track progress
	spinnerWriter := output.NewSpinnerWriter(bar)

	// Execute the installation based on type
	var err error
	switch task.Type {
	case "node":
		if project, ok := task.Project.(types.NodeProject); ok {
			if pi.Verbose {
				err = installNodeDependenciesWithWriter(project, os.Stdout)
			} else {
				err = installNodeDependenciesWithWriter(project, spinnerWriter)
			}
		}
	case "python":
		if project, ok := task.Project.(types.PythonProject); ok {
			if pi.Verbose {
				err = setupPythonVirtualEnvWithWriter(project, os.Stdout)
			} else {
				err = setupPythonVirtualEnvWithWriter(project, spinnerWriter)
			}
		}
	case "dotnet":
		if project, ok := task.Project.(types.DotnetProject); ok {
			if pi.Verbose {
				err = restoreDotnetProjectWithWriter(project, os.Stdout)
			} else {
				err = restoreDotnetProjectWithWriter(project, spinnerWriter)
			}
		}
	}

	// Mark as completed or failed
	if err != nil {
		bar.Fail(err.Error())
	} else {
		bar.Complete()
	}

	// Send result
	resultsChan <- ProjectInstallResult{
		Task:    task,
		Success: err == nil,
		Error:   err,
	}
}

// runVerbose runs installations in parallel with full output instead of progress bars.
func (pi *ParallelInstaller) runVerbose() error {
	// Run all tasks in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan ProjectInstallResult, len(pi.tasks))

	for _, task := range pi.tasks {
		wg.Add(1)
		go pi.runTaskVerbose(task, &wg, resultsChan)
	}

	// Wait for all tasks to complete
	wg.Wait()
	close(resultsChan)

	// Collect results
	for result := range resultsChan {
		pi.results = append(pi.results, result)
	}

	// Print summary
	pi.printSummary()

	return nil
}

// runTaskVerbose executes a single installation task with full output.
func (pi *ParallelInstaller) runTaskVerbose(task ProjectInstallTask, wg *sync.WaitGroup, resultsChan chan<- ProjectInstallResult) {
	defer wg.Done()

	// Execute the installation based on type with full output to stdout
	var err error
	switch task.Type {
	case "node":
		if project, ok := task.Project.(types.NodeProject); ok {
			err = installNodeDependenciesWithWriter(project, os.Stdout)
		}
	case "python":
		if project, ok := task.Project.(types.PythonProject); ok {
			err = setupPythonVirtualEnvWithWriter(project, os.Stdout)
		}
	case "dotnet":
		if project, ok := task.Project.(types.DotnetProject); ok {
			err = restoreDotnetProjectWithWriter(project, os.Stdout)
		}
	}

	// Send result
	resultsChan <- ProjectInstallResult{
		Task:    task,
		Success: err == nil,
		Error:   err,
	}
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
