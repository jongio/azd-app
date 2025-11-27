package installer

import (
	"testing"

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
