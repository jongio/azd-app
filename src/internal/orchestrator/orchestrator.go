package orchestrator

import (
	"fmt"
	"sync"
)

// CommandFunc represents a command execution function.
type CommandFunc func() error

// Command represents a command with its dependencies.
type Command struct {
	Name         string
	Execute      CommandFunc
	Dependencies []string
}

// Orchestrator manages command execution with dependency resolution.
type Orchestrator struct {
	commands map[string]*Command
	executed map[string]bool
	mu       sync.Mutex
}

// NewOrchestrator creates a new command orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		commands: make(map[string]*Command),
		executed: make(map[string]bool),
	}
}

// Register registers a command with the orchestrator.
func (o *Orchestrator) Register(cmd *Command) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if cmd.Name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if cmd.Execute == nil {
		return fmt.Errorf("command %s must have an Execute function", cmd.Name)
	}

	if _, exists := o.commands[cmd.Name]; exists {
		return fmt.Errorf("command %s is already registered", cmd.Name)
	}

	o.commands[cmd.Name] = cmd
	return nil
}

// Run executes a command and all its dependencies in the correct order.
// It uses memoization to avoid running the same command multiple times.
func (o *Orchestrator) Run(commandName string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.runLocked(commandName, make(map[string]bool))
}

// runLocked executes a command with cycle detection.
// Must be called with o.mu held.
func (o *Orchestrator) runLocked(commandName string, visiting map[string]bool) error {
	// Check if already executed
	if o.executed[commandName] {
		return nil
	}

	// Check if command exists
	cmd, exists := o.commands[commandName]
	if !exists {
		return fmt.Errorf("command %s is not registered", commandName)
	}

	// Detect cycles
	if visiting[commandName] {
		return fmt.Errorf("circular dependency detected for command %s", commandName)
	}

	// Mark as visiting
	visiting[commandName] = true

	// Execute dependencies first
	for _, depName := range cmd.Dependencies {
		if err := o.runLocked(depName, visiting); err != nil {
			return fmt.Errorf("dependency %s failed for %s: %w", depName, commandName, err)
		}
	}

	// Unmark visiting
	delete(visiting, commandName)

	// Execute the command
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("command %s failed: %w", commandName, err)
	}

	// Mark as executed
	o.executed[commandName] = true
	return nil
}

// Reset clears the execution state, allowing commands to be run again.
func (o *Orchestrator) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.executed = make(map[string]bool)
}

// IsExecuted checks if a command has been executed.
func (o *Orchestrator) IsExecuted(commandName string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.executed[commandName]
}

// GetRegisteredCommands returns a list of all registered command names.
func (o *Orchestrator) GetRegisteredCommands() []string {
	o.mu.Lock()
	defer o.mu.Unlock()

	names := make([]string, 0, len(o.commands))
	for name := range o.commands {
		names = append(names, name)
	}
	return names
}
