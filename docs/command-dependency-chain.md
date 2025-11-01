# Command Dependency Chain Design

## Overview

The App CLI implements a robust command dependency chain using an idiomatic Go orchestrator pattern. This ensures commands are executed in the correct order with proper dependency resolution, cycle detection, and memoization.

## Dependency Chain

```
run command
  ↓ depends on
deps command  
  ↓ depends on
reqs command
```

When you run `azd app run`, the orchestrator automatically:
1. Checks prerequisites (reqs)
2. Installs dependencies (deps)
3. Starts the development environment (run)

## Architecture

### Core Components

#### 1. Orchestrator (`internal/orchestrator/orchestrator.go`)

The orchestrator manages command execution with the following features:

- **Dependency Resolution**: Commands declare dependencies, and the orchestrator executes them in the correct order
- **Memoization**: Each command runs only once, even if multiple commands depend on it
- **Cycle Detection**: Prevents infinite loops by detecting circular dependencies
- **Error Propagation**: Failures in dependencies prevent dependent commands from running
- **Thread Safety**: Uses mutex locks for concurrent safety

```go
type Command struct {
    Name         string
    Execute      CommandFunc
    Dependencies []string
}

type Orchestrator struct {
    commands map[string]*Command
    executed map[string]bool
    mu       sync.Mutex
}
```

#### 2. Command Registration (`cmd/app/commands/core.go`)

Commands are registered in the `init()` function with their dependencies:

```go
func init() {
    cmdOrchestrator = orchestrator.NewOrchestrator()

    // reqs has no dependencies
    cmdOrchestrator.Register(&orchestrator.Command{
        Name:    "reqs",
        Execute: executeReqs,
    })

    // deps depends on reqs
    cmdOrchestrator.Register(&orchestrator.Command{
        Name:         "deps",
        Dependencies: []string{"reqs"},
        Execute:      executeDeps,
    })

    // run depends on deps (which transitively depends on reqs)
    cmdOrchestrator.Register(&orchestrator.Command{
        Name:         "run",
        Dependencies: []string{"deps"},
        Execute:      executeRun,
    })
}
```

#### 3. Command Implementation

Each command's logic is implemented as a standalone function:

- `executeReqs()`: Checks prerequisites from azure.yaml
- `executeDeps()`: Installs project dependencies
- `executeRun()`: Starts the development environment

#### 4. Cobra Integration

Cobra commands delegate to the orchestrator:

```go
func NewRunCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "run",
        Short: "Run the development environment",
        RunE: func(cmd *cobra.Command, args []string) error {
            return cmdOrchestrator.Run("run")
        },
    }
}
```

## Execution Flow

### Example: Running `azd app run`

1. User executes: `azd app run`
2. Cobra invokes `NewRunCommand().RunE`
3. RunE calls `cmdOrchestrator.Run("run")`
4. Orchestrator checks if "run" has dependencies → finds "deps"
5. Orchestrator recursively runs "deps"
6. Orchestrator checks if "deps" has dependencies → finds "reqs"
7. Orchestrator recursively runs "reqs"
8. "reqs" has no dependencies, so `executeReqs()` runs
9. "reqs" is marked as executed
10. Control returns to "deps", `executeDeps()` runs
11. "deps" is marked as executed
12. Control returns to "run", `executeRun()` runs
13. "run" is marked as executed

### Diamond Dependency Example

The orchestrator handles complex dependency graphs efficiently:

```
       run
      /   \
    deps  deploy
      \   /
       reqs
```

In this scenario, `reqs` would run only once, even though both `deps` and `deploy` depend on it.

## Key Features

### 1. Memoization

Commands are executed only once per orchestrator instance:

```go
// First call executes the command
cmdOrchestrator.Run("reqs")  // ✓ Executes

// Second call is a no-op
cmdOrchestrator.Run("reqs")  // ✗ Skipped (already executed)

// Reset allows re-execution
cmdOrchestrator.Reset()
cmdOrchestrator.Run("reqs")  // ✓ Executes again
```

### 2. Cycle Detection

The orchestrator detects and prevents circular dependencies:

```go
// cmd1 depends on cmd2, cmd2 depends on cmd1
// This will return an error: "circular dependency detected"
```

### 3. Error Handling

Errors in dependencies propagate up the chain:

```go
// If reqs fails, deps and run will not execute
// Error message: "dependency reqs failed for deps: prerequisite check failed"
```

### 4. Graceful Degradation

Commands can gracefully skip when there's nothing to do:

```go
func executeReqs() error {
    if !fileExists("azure.yaml") {
        fmt.Println("ℹ️  No azure.yaml found - skipping prerequisite check")
        return nil  // Not an error, just nothing to do
    }
    // ... check prerequisites
}
```

## Testing

The orchestrator includes comprehensive tests covering:

- Simple command execution
- Dependency chain execution
- Memoization behavior
- Circular dependency detection
- Error propagation
- Complex dependency graphs (diamond dependencies)
- Reset functionality

Run tests:
```bash
go test ./internal/orchestrator/...
```

## Benefits

### For Developers

- **Separation of Concerns**: Command logic is separate from dependency management
- **Testability**: Each command function can be tested independently
- **Extensibility**: New commands can be added easily with clear dependency declarations
- **Type Safety**: Compile-time checks for command registration

### For Users

- **Consistency**: Commands always run in the correct order
- **Efficiency**: Repeated operations are skipped automatically
- **Reliability**: Dependency failures prevent execution of dependent commands
- **Transparency**: Clear error messages when dependencies fail

## Adding New Commands

To add a new command with dependencies:

1. Create the execution function in `core.go`:
   ```go
   func executeMyCommand() error {
       // Implementation
       return nil
   }
   ```

2. Register it in the `init()` function:
   ```go
   cmdOrchestrator.Register(&orchestrator.Command{
       Name:         "mycommand",
       Dependencies: []string{"deps", "reqs"},
       Execute:      executeMyCommand,
   })
   ```

3. Create the Cobra command in a new file:
   ```go
   func NewMyCommand() *cobra.Command {
       return &cobra.Command{
           Use:   "mycommand",
           Short: "My command description",
           RunE: func(cmd *cobra.Command, args []string) error {
               return cmdOrchestrator.Run("mycommand")
           },
       }
   }
   ```

4. Register it in `main.go`:
   ```go
   rootCmd.AddCommand(
       commands.NewMyCommand(),
   )
   ```

## Implementation Notes

- The orchestrator uses a mutex for thread safety, though currently commands are called sequentially
- Dependencies are specified by name (string), allowing for late binding
- The `executed` map provides O(1) lookup for memoization
- Cycle detection uses a separate `visiting` map to track the current execution path
- Error wrapping with `%w` maintains the error chain for proper error handling

## Future Enhancements

Potential improvements for the orchestrator:

1. **Parallel Execution**: Run independent commands in parallel
2. **Progress Reporting**: Show progress for long-running dependency chains
3. **Dry Run Mode**: Show what would execute without actually running
4. **Dependency Visualization**: Generate a graph of command dependencies
5. **Conditional Dependencies**: Support optional dependencies based on runtime conditions
6. **Retry Logic**: Automatic retry for transient failures
7. **Hooks**: Pre/post execution hooks for commands

## Conclusion

The command orchestrator provides a robust, idiomatic Go solution for managing command dependencies. It ensures correct execution order, prevents common pitfalls like circular dependencies, and provides a clean API for both command implementers and users.
