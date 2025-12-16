package orchestrator

import (
	"errors"
	"testing"
)

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
