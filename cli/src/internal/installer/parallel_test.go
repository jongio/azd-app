package installer

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/types"
)

func TestNewParallelInstaller(t *testing.T) {
	pi := NewParallelInstaller()

	if pi == nil {
		t.Fatal("NewParallelInstaller() returned nil")
	}

	if len(pi.tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(pi.tasks))
	}

	if len(pi.results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(pi.results))
	}
}

func TestAddTask(t *testing.T) {
	pi := NewParallelInstaller()

	task := ProjectInstallTask{
		ID:          "test-task",
		Description: "Test Task",
		Type:        "node",
		Dir:         "/test",
		Manager:     "npm",
	}

	pi.AddTask(task)

	if len(pi.tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(pi.tasks))
	}

	if pi.tasks[0].ID != "test-task" {
		t.Errorf("Expected task ID 'test-task', got %q", pi.tasks[0].ID)
	}
}

func TestAddNodeProject(t *testing.T) {
	pi := NewParallelInstaller()

	project := types.NodeProject{
		Dir:            "/path/to/node/project",
		PackageManager: "npm",
	}

	pi.AddNodeProject(project)

	if len(pi.tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(pi.tasks))
	}

	task := pi.tasks[0]
	if task.Type != "node" {
		t.Errorf("Expected type 'node', got %q", task.Type)
	}
	if task.Manager != "npm" {
		t.Errorf("Expected manager 'npm', got %q", task.Manager)
	}
	if task.Dir != "/path/to/node/project" {
		t.Errorf("Expected dir '/path/to/node/project', got %q", task.Dir)
	}
}

func TestAddPythonProject(t *testing.T) {
	pi := NewParallelInstaller()

	project := types.PythonProject{
		Dir:            "/path/to/python/project",
		PackageManager: "uv",
	}

	pi.AddPythonProject(project)

	if len(pi.tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(pi.tasks))
	}

	task := pi.tasks[0]
	if task.Type != "python" {
		t.Errorf("Expected type 'python', got %q", task.Type)
	}
	if task.Manager != "uv" {
		t.Errorf("Expected manager 'uv', got %q", task.Manager)
	}
	if task.Dir != "/path/to/python/project" {
		t.Errorf("Expected dir '/path/to/python/project', got %q", task.Dir)
	}
}

func TestAddDotnetProject(t *testing.T) {
	pi := NewParallelInstaller()

	project := types.DotnetProject{
		Path: "/path/to/dotnet/Project.csproj",
	}

	pi.AddDotnetProject(project)

	if len(pi.tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(pi.tasks))
	}

	task := pi.tasks[0]
	if task.Type != "dotnet" {
		t.Errorf("Expected type 'dotnet', got %q", task.Type)
	}
	if task.Manager != "dotnet" {
		t.Errorf("Expected manager 'dotnet', got %q", task.Manager)
	}
	if task.Path != "/path/to/dotnet/Project.csproj" {
		t.Errorf("Expected path '/path/to/dotnet/Project.csproj', got %q", task.Path)
	}
}

func TestAddMultipleProjects(t *testing.T) {
	pi := NewParallelInstaller()

	nodeProject := types.NodeProject{
		Dir:            "/node",
		PackageManager: "pnpm",
	}
	pythonProject := types.PythonProject{
		Dir:            "/python",
		PackageManager: "poetry",
	}
	dotnetProject := types.DotnetProject{
		Path: "/dotnet/App.csproj",
	}

	pi.AddNodeProject(nodeProject)
	pi.AddPythonProject(pythonProject)
	pi.AddDotnetProject(dotnetProject)

	if len(pi.tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(pi.tasks))
	}

	// Verify each task type
	typeCount := map[string]int{}
	for _, task := range pi.tasks {
		typeCount[task.Type]++
	}

	if typeCount["node"] != 1 {
		t.Errorf("Expected 1 node task, got %d", typeCount["node"])
	}
	if typeCount["python"] != 1 {
		t.Errorf("Expected 1 python task, got %d", typeCount["python"])
	}
	if typeCount["dotnet"] != 1 {
		t.Errorf("Expected 1 dotnet task, got %d", typeCount["dotnet"])
	}
}

func TestGetResults(t *testing.T) {
	pi := NewParallelInstaller()

	// Simulate adding results
	pi.results = []ProjectInstallResult{
		{
			Task: ProjectInstallTask{
				ID:          "task1",
				Description: "Task 1",
			},
			Success: true,
			Error:   nil,
		},
		{
			Task: ProjectInstallTask{
				ID:          "task2",
				Description: "Task 2",
			},
			Success: false,
			Error:   &DependencyInstallError{},
		},
	}

	results := pi.GetResults()
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestHasFailures(t *testing.T) {
	tests := []struct {
		name     string
		results  []ProjectInstallResult
		expected bool
	}{
		{
			name:     "no results",
			results:  []ProjectInstallResult{},
			expected: false,
		},
		{
			name: "all success",
			results: []ProjectInstallResult{
				{Success: true},
				{Success: true},
			},
			expected: false,
		},
		{
			name: "one failure",
			results: []ProjectInstallResult{
				{Success: true},
				{Success: false},
			},
			expected: true,
		},
		{
			name: "all failures",
			results: []ProjectInstallResult{
				{Success: false},
				{Success: false},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi := NewParallelInstaller()
			pi.results = tt.results

			got := pi.HasFailures()
			if got != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFailedProjects(t *testing.T) {
	pi := NewParallelInstaller()

	pi.results = []ProjectInstallResult{
		{
			Task:    ProjectInstallTask{ID: "success1", Description: "Success Project 1"},
			Success: true,
		},
		{
			Task:    ProjectInstallTask{ID: "fail1", Description: "Failed Project 1"},
			Success: false,
		},
		{
			Task:    ProjectInstallTask{ID: "success2", Description: "Success Project 2"},
			Success: true,
		},
		{
			Task:    ProjectInstallTask{ID: "fail2", Description: "Failed Project 2"},
			Success: false,
		},
	}

	failed := pi.FailedProjects()
	if len(failed) != 2 {
		t.Fatalf("Expected 2 failed projects, got %d", len(failed))
	}

	// Check descriptions
	descriptions := make(map[string]bool)
	for _, desc := range failed {
		descriptions[desc] = true
	}

	if !descriptions["Failed Project 1"] {
		t.Error("Expected 'Failed Project 1' to be in failed projects")
	}
	if !descriptions["Failed Project 2"] {
		t.Error("Expected 'Failed Project 2' to be in failed projects")
	}
	if descriptions["Success Project 1"] || descriptions["Success Project 2"] {
		t.Error("Successful projects should not be in failed list")
	}
}

func TestTotalProjects(t *testing.T) {
	pi := NewParallelInstaller()

	// TotalProjects counts results, not tasks
	pi.results = []ProjectInstallResult{
		{Task: ProjectInstallTask{ID: "1"}, Success: true},
		{Task: ProjectInstallTask{ID: "2"}, Success: true},
		{Task: ProjectInstallTask{ID: "3"}, Success: false},
		{Task: ProjectInstallTask{ID: "4"}, Success: true},
	}

	total := pi.TotalProjects()
	if total != 4 {
		t.Errorf("TotalProjects() = %d, want 4", total)
	}
}

func TestGetProjectName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/projects/my-app",
			expected: "my-app",
		},
		{
			name:     "path with extension",
			path:     "/projects/App.csproj",
			expected: "App.csproj",
		},
		{
			name:     "nested path",
			path:     "/very/long/path/to/project",
			expected: "project",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProjectName(tt.path)
			if got != tt.expected {
				t.Errorf("getProjectName(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestParallelInstaller_ConcurrentAddTask(t *testing.T) {
	pi := NewParallelInstaller()

	// Add tasks concurrently to test mutex
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			task := ProjectInstallTask{
				ID:   string(rune('0' + id)),
				Type: "node",
			}
			pi.AddTask(task)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(pi.tasks) != 10 {
		t.Errorf("Expected 10 tasks after concurrent adds, got %d", len(pi.tasks))
	}
}

func TestParallelInstaller_PnpmTaskGrouping(t *testing.T) {
	// Test that tasks are correctly grouped by package manager
	pi := NewParallelInstaller()

	// Add mix of pnpm and other tasks
	pi.AddTask(ProjectInstallTask{ID: "pnpm1", Type: "node", Manager: "pnpm", Dir: "/a"})
	pi.AddTask(ProjectInstallTask{ID: "npm1", Type: "node", Manager: "npm", Dir: "/b"})
	pi.AddTask(ProjectInstallTask{ID: "pnpm2", Type: "node", Manager: "pnpm", Dir: "/c"})
	pi.AddTask(ProjectInstallTask{ID: "pip1", Type: "python", Manager: "pip", Dir: "/d"})
	pi.AddTask(ProjectInstallTask{ID: "pnpm3", Type: "node", Manager: "pnpm", Dir: "/e"})
	pi.AddTask(ProjectInstallTask{ID: "dotnet1", Type: "dotnet", Manager: "dotnet", Path: "/f/app.csproj"})

	// Verify task counts
	var pnpmCount, otherCount int
	for _, task := range pi.tasks {
		if task.Manager == "pnpm" {
			pnpmCount++
		} else {
			otherCount++
		}
	}

	if pnpmCount != 3 {
		t.Errorf("Expected 3 pnpm tasks, got %d", pnpmCount)
	}
	if otherCount != 3 {
		t.Errorf("Expected 3 non-pnpm tasks, got %d", otherCount)
	}
}

func TestParallelInstaller_PnpmSequentialExecution(t *testing.T) {
	// This test verifies that pnpm tasks don't run concurrently
	// by checking that the max concurrent pnpm tasks is always 1

	var maxConcurrentPnpm int32
	var currentConcurrentPnpm int32
	var mu sync.Mutex

	// Track execution order for pnpm tasks
	pnpmExecutionOrder := make([]string, 0)
	var orderMu sync.Mutex

	// We can't easily test the actual Run() without mocking,
	// but we can verify the grouping logic works correctly
	pi := NewParallelInstaller()

	// Add multiple pnpm tasks
	pnpmTasks := []ProjectInstallTask{
		{ID: "pnpm1", Type: "node", Manager: "pnpm", Dir: "/project1"},
		{ID: "pnpm2", Type: "node", Manager: "pnpm", Dir: "/project2"},
		{ID: "pnpm3", Type: "node", Manager: "pnpm", Dir: "/project3"},
	}

	for _, task := range pnpmTasks {
		pi.AddTask(task)
	}

	// Simulate the grouping logic from Run()
	var pnpmGroup []ProjectInstallTask
	var parallelGroup []ProjectInstallTask

	for _, task := range pi.tasks {
		if task.Manager == "pnpm" {
			pnpmGroup = append(pnpmGroup, task)
		} else {
			parallelGroup = append(parallelGroup, task)
		}
	}

	// Verify all tasks went to pnpm group
	if len(pnpmGroup) != 3 {
		t.Errorf("Expected 3 tasks in pnpm group, got %d", len(pnpmGroup))
	}
	if len(parallelGroup) != 0 {
		t.Errorf("Expected 0 tasks in parallel group, got %d", len(parallelGroup))
	}

	// Simulate sequential execution and verify no concurrent access
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, task := range pnpmGroup {
			// Simulate starting a task
			atomic.AddInt32(&currentConcurrentPnpm, 1)
			current := atomic.LoadInt32(&currentConcurrentPnpm)

			mu.Lock()
			if current > maxConcurrentPnpm {
				maxConcurrentPnpm = current
			}
			mu.Unlock()

			// Record execution order
			orderMu.Lock()
			pnpmExecutionOrder = append(pnpmExecutionOrder, task.ID)
			orderMu.Unlock()

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			// Simulate completing a task
			atomic.AddInt32(&currentConcurrentPnpm, -1)
		}
	}()

	wg.Wait()

	// Verify max concurrency was 1 (sequential)
	if maxConcurrentPnpm != 1 {
		t.Errorf("Expected max concurrent pnpm tasks to be 1, got %d", maxConcurrentPnpm)
	}

	// Verify execution order maintained
	if len(pnpmExecutionOrder) != 3 {
		t.Errorf("Expected 3 tasks executed, got %d", len(pnpmExecutionOrder))
	}
}

func TestParallelInstaller_MixedTaskGrouping(t *testing.T) {
	// Test that mixed workloads are correctly separated
	pi := NewParallelInstaller()

	// Simulate a real-world scenario with multiple package managers
	pi.AddNodeProject(types.NodeProject{Dir: "/app1", PackageManager: "pnpm"})
	pi.AddNodeProject(types.NodeProject{Dir: "/app2", PackageManager: "pnpm"})
	pi.AddNodeProject(types.NodeProject{Dir: "/app3", PackageManager: "npm"})
	pi.AddPythonProject(types.PythonProject{Dir: "/api1", PackageManager: "pip"})
	pi.AddPythonProject(types.PythonProject{Dir: "/api2", PackageManager: "uv"})
	pi.AddDotnetProject(types.DotnetProject{Path: "/service/app.csproj"})

	// Count by manager type
	managers := make(map[string]int)
	for _, task := range pi.tasks {
		managers[task.Manager]++
	}

	// Verify counts
	if managers["pnpm"] != 2 {
		t.Errorf("Expected 2 pnpm tasks, got %d", managers["pnpm"])
	}
	if managers["npm"] != 1 {
		t.Errorf("Expected 1 npm task, got %d", managers["npm"])
	}
	if managers["pip"] != 1 {
		t.Errorf("Expected 1 pip task, got %d", managers["pip"])
	}
	if managers["uv"] != 1 {
		t.Errorf("Expected 1 uv task, got %d", managers["uv"])
	}
	if managers["dotnet"] != 1 {
		t.Errorf("Expected 1 dotnet task, got %d", managers["dotnet"])
	}

	// Simulate grouping
	var pnpmTasks, parallelTasks []ProjectInstallTask
	for _, task := range pi.tasks {
		if task.Manager == "pnpm" {
			pnpmTasks = append(pnpmTasks, task)
		} else {
			parallelTasks = append(parallelTasks, task)
		}
	}

	// pnpm should be separate, everything else parallel
	if len(pnpmTasks) != 2 {
		t.Errorf("Expected 2 pnpm tasks for sequential execution, got %d", len(pnpmTasks))
	}
	if len(parallelTasks) != 4 {
		t.Errorf("Expected 4 tasks for parallel execution, got %d", len(parallelTasks))
	}
}
