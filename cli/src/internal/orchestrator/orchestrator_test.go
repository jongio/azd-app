package orchestrator

import (
	"errors"
	"testing"
)

func TestNewOrchestrator(t *testing.T) {
	o := NewOrchestrator()
	if o == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	if o.commands == nil {
		t.Error("commands map is nil")
	}
	if o.executed == nil {
		t.Error("executed map is nil")
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name      string
		command   *Command
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid command",
			command: &Command{
				Name:    "test",
				Execute: func() error { return nil },
			},
			wantError: false,
		},
		{
			name: "empty name",
			command: &Command{
				Name:    "",
				Execute: func() error { return nil },
			},
			wantError: true,
			errorMsg:  "command name cannot be empty",
		},
		{
			name: "nil execute function",
			command: &Command{
				Name:    "test",
				Execute: nil,
			},
			wantError: true,
			errorMsg:  "command test must have an Execute function",
		},
		{
			name: "command with dependencies",
			command: &Command{
				Name:         "test-with-deps",
				Execute:      func() error { return nil },
				Dependencies: []string{"dep1", "dep2"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewOrchestrator()
			err := o.Register(tt.command)

			if tt.wantError {
				if err == nil {
					t.Errorf("Register() expected error but got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Register() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Register() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRegisterDuplicate(t *testing.T) {
	o := NewOrchestrator()
	cmd := &Command{
		Name:    "test",
		Execute: func() error { return nil },
	}

	// First registration should succeed
	if err := o.Register(cmd); err != nil {
		t.Fatalf("First Register() failed: %v", err)
	}

	// Second registration should fail
	err := o.Register(cmd)
	if err == nil {
		t.Error("Register() duplicate command should fail")
	}
	if err != nil && err.Error() != "command test is already registered" {
		t.Errorf("Register() error = %v, want 'command test is already registered'", err)
	}
}

func TestRunSimpleCommand(t *testing.T) {
	o := NewOrchestrator()
	executed := false

	cmd := &Command{
		Name: "simple",
		Execute: func() error {
			executed = true
			return nil
		},
	}

	if err := o.Register(cmd); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if err := o.Run("simple"); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if !executed {
		t.Error("Command was not executed")
	}

	if !o.IsExecuted("simple") {
		t.Error("Command not marked as executed")
	}
}

func TestRunWithDependencies(t *testing.T) {
	o := NewOrchestrator()
	executionOrder := []string{}

	// Create commands with dependencies
	// reqs (no deps) -> deps (depends on reqs) -> run (depends on deps)
	reqs := &Command{
		Name: "reqs",
		Execute: func() error {
			executionOrder = append(executionOrder, "reqs")
			return nil
		},
	}

	deps := &Command{
		Name:         "deps",
		Dependencies: []string{"reqs"},
		Execute: func() error {
			executionOrder = append(executionOrder, "deps")
			return nil
		},
	}

	run := &Command{
		Name:         "run",
		Dependencies: []string{"deps"},
		Execute: func() error {
			executionOrder = append(executionOrder, "run")
			return nil
		},
	}

	// Register all commands
	if err := o.Register(reqs); err != nil {
		t.Fatalf("Register(reqs) failed: %v", err)
	}
	if err := o.Register(deps); err != nil {
		t.Fatalf("Register(deps) failed: %v", err)
	}
	if err := o.Register(run); err != nil {
		t.Fatalf("Register(run) failed: %v", err)
	}

	// Run the top-level command
	if err := o.Run("run"); err != nil {
		t.Fatalf("Run(run) failed: %v", err)
	}

	// Verify execution order
	expectedOrder := []string{"reqs", "deps", "run"}
	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Execution order length = %d, want %d", len(executionOrder), len(expectedOrder))
	}

	for i, cmd := range expectedOrder {
		if executionOrder[i] != cmd {
			t.Errorf("Execution order[%d] = %s, want %s", i, executionOrder[i], cmd)
		}
	}

	// Verify all marked as executed
	for _, cmd := range expectedOrder {
		if !o.IsExecuted(cmd) {
			t.Errorf("Command %s not marked as executed", cmd)
		}
	}
}

func TestRunMemoization(t *testing.T) {
	o := NewOrchestrator()
	execCount := 0

	// Create a shared dependency
	shared := &Command{
		Name: "shared",
		Execute: func() error {
			execCount++
			return nil
		},
	}

	// Create two commands that both depend on shared
	cmd1 := &Command{
		Name:         "cmd1",
		Dependencies: []string{"shared"},
		Execute:      func() error { return nil },
	}

	cmd2 := &Command{
		Name:         "cmd2",
		Dependencies: []string{"shared"},
		Execute:      func() error { return nil },
	}

	// Register all
	if err := o.Register(shared); err != nil {
		t.Fatalf("Register(shared) failed: %v", err)
	}
	if err := o.Register(cmd1); err != nil {
		t.Fatalf("Register(cmd1) failed: %v", err)
	}
	if err := o.Register(cmd2); err != nil {
		t.Fatalf("Register(cmd2) failed: %v", err)
	}

	// Run both commands
	if err := o.Run("cmd1"); err != nil {
		t.Fatalf("Run(cmd1) failed: %v", err)
	}
	if err := o.Run("cmd2"); err != nil {
		t.Fatalf("Run(cmd2) failed: %v", err)
	}

	// Shared should only execute once
	if execCount != 1 {
		t.Errorf("Shared command executed %d times, want 1", execCount)
	}
}

func TestRunCircularDependency(t *testing.T) {
	o := NewOrchestrator()

	// Create circular dependency: cmd1 -> cmd2 -> cmd1
	cmd1 := &Command{
		Name:         "cmd1",
		Dependencies: []string{"cmd2"},
		Execute:      func() error { return nil },
	}

	cmd2 := &Command{
		Name:         "cmd2",
		Dependencies: []string{"cmd1"},
		Execute:      func() error { return nil },
	}

	if err := o.Register(cmd1); err != nil {
		t.Fatalf("Register(cmd1) failed: %v", err)
	}
	if err := o.Register(cmd2); err != nil {
		t.Fatalf("Register(cmd2) failed: %v", err)
	}

	// Run should detect the cycle
	err := o.Run("cmd1")
	if err == nil {
		t.Error("Run() should detect circular dependency")
	}
	if err != nil && err.Error() != "dependency cmd2 failed for cmd1: circular dependency detected for command cmd1" {
		t.Logf("Got error: %v", err)
	}
}

func TestRunCommandNotRegistered(t *testing.T) {
	o := NewOrchestrator()

	err := o.Run("nonexistent")
	if err == nil {
		t.Error("Run() should fail for unregistered command")
	}
	if err != nil && err.Error() != "command nonexistent is not registered" {
		t.Errorf("Run() error = %v, want 'command nonexistent is not registered'", err)
	}
}

func TestRunDependencyNotRegistered(t *testing.T) {
	o := NewOrchestrator()

	cmd := &Command{
		Name:         "cmd",
		Dependencies: []string{"missing"},
		Execute:      func() error { return nil },
	}

	if err := o.Register(cmd); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	err := o.Run("cmd")
	if err == nil {
		t.Error("Run() should fail when dependency is not registered")
	}
}

func TestRunCommandError(t *testing.T) {
	o := NewOrchestrator()
	testErr := errors.New("command execution failed")

	cmd := &Command{
		Name: "failing",
		Execute: func() error {
			return testErr
		},
	}

	if err := o.Register(cmd); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	err := o.Run("failing")
	if err == nil {
		t.Error("Run() should propagate command error")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("Run() error chain should contain original error")
	}
}

func TestRunDependencyError(t *testing.T) {
	o := NewOrchestrator()
	testErr := errors.New("dependency failed")

	dep := &Command{
		Name: "dep",
		Execute: func() error {
			return testErr
		},
	}

	cmd := &Command{
		Name:         "cmd",
		Dependencies: []string{"dep"},
		Execute:      func() error { return nil },
	}

	if err := o.Register(dep); err != nil {
		t.Fatalf("Register(dep) failed: %v", err)
	}
	if err := o.Register(cmd); err != nil {
		t.Fatalf("Register(cmd) failed: %v", err)
	}

	err := o.Run("cmd")
	if err == nil {
		t.Error("Run() should propagate dependency error")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("Run() error chain should contain original error")
	}

	// cmd should not be marked as executed since dependency failed
	if o.IsExecuted("cmd") {
		t.Error("Command should not be marked executed when dependency fails")
	}
}

func TestReset(t *testing.T) {
	o := NewOrchestrator()
	execCount := 0

	cmd := &Command{
		Name: "test",
		Execute: func() error {
			execCount++
			return nil
		},
	}

	if err := o.Register(cmd); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// First run
	if err := o.Run("test"); err != nil {
		t.Fatalf("First Run() failed: %v", err)
	}
	if execCount != 1 {
		t.Errorf("First run: execCount = %d, want 1", execCount)
	}

	// Second run without reset (should not execute again)
	if err := o.Run("test"); err != nil {
		t.Fatalf("Second Run() failed: %v", err)
	}
	if execCount != 1 {
		t.Errorf("Second run: execCount = %d, want 1 (memoized)", execCount)
	}

	// Reset and run again
	o.Reset()
	if err := o.Run("test"); err != nil {
		t.Fatalf("Third Run() after reset failed: %v", err)
	}
	if execCount != 2 {
		t.Errorf("After reset: execCount = %d, want 2", execCount)
	}
}

func TestGetRegisteredCommands(t *testing.T) {
	o := NewOrchestrator()

	// Empty orchestrator
	commands := o.GetRegisteredCommands()
	if len(commands) != 0 {
		t.Errorf("Empty orchestrator should have 0 commands, got %d", len(commands))
	}

	// Register some commands
	names := []string{"cmd1", "cmd2", "cmd3"}
	for _, name := range names {
		cmd := &Command{
			Name:    name,
			Execute: func() error { return nil },
		}
		if err := o.Register(cmd); err != nil {
			t.Fatalf("Register(%s) failed: %v", name, err)
		}
	}

	// Get registered commands
	registered := o.GetRegisteredCommands()
	if len(registered) != len(names) {
		t.Errorf("GetRegisteredCommands() returned %d commands, want %d", len(registered), len(names))
	}

	// Check all names are present
	nameSet := make(map[string]bool)
	for _, name := range registered {
		nameSet[name] = true
	}
	for _, name := range names {
		if !nameSet[name] {
			t.Errorf("GetRegisteredCommands() missing %s", name)
		}
	}
}

func TestComplexDependencyGraph(t *testing.T) {
	o := NewOrchestrator()
	executionOrder := []string{}

	// Create a diamond dependency graph:
	//       run
	//      /   \
	//    deps  deploy
	//      \   /
	//       reqs

	reqs := &Command{
		Name: "reqs",
		Execute: func() error {
			executionOrder = append(executionOrder, "reqs")
			return nil
		},
	}

	deps := &Command{
		Name:         "deps",
		Dependencies: []string{"reqs"},
		Execute: func() error {
			executionOrder = append(executionOrder, "deps")
			return nil
		},
	}

	deploy := &Command{
		Name:         "deploy",
		Dependencies: []string{"reqs"},
		Execute: func() error {
			executionOrder = append(executionOrder, "deploy")
			return nil
		},
	}

	run := &Command{
		Name:         "run",
		Dependencies: []string{"deps", "deploy"},
		Execute: func() error {
			executionOrder = append(executionOrder, "run")
			return nil
		},
	}

	// Register all
	for _, cmd := range []*Command{reqs, deps, deploy, run} {
		if err := o.Register(cmd); err != nil {
			t.Fatalf("Register(%s) failed: %v", cmd.Name, err)
		}
	}

	// Run top command
	if err := o.Run("run"); err != nil {
		t.Fatalf("Run(run) failed: %v", err)
	}

	// Verify reqs only executed once
	reqsCount := 0
	for _, cmd := range executionOrder {
		if cmd == "reqs" {
			reqsCount++
		}
	}
	if reqsCount != 1 {
		t.Errorf("reqs executed %d times, want 1", reqsCount)
	}

	// Verify all commands executed
	expectedCommands := map[string]bool{
		"reqs":   false,
		"deps":   false,
		"deploy": false,
		"run":    false,
	}
	for _, cmd := range executionOrder {
		expectedCommands[cmd] = true
	}
	for cmd, executed := range expectedCommands {
		if !executed {
			t.Errorf("Command %s was not executed", cmd)
		}
	}

	// Verify reqs executed before deps and deploy
	reqsIdx := -1
	depsIdx := -1
	deployIdx := -1
	for i, cmd := range executionOrder {
		switch cmd {
		case "reqs":
			reqsIdx = i
		case "deps":
			depsIdx = i
		case "deploy":
			deployIdx = i
		}
	}

	if reqsIdx >= depsIdx {
		t.Errorf("reqs should execute before deps")
	}
	if reqsIdx >= deployIdx {
		t.Errorf("reqs should execute before deploy")
	}
}
