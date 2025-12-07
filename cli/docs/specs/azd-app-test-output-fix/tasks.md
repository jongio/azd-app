# azd app test Output Fix - Tasks

## Tasks

### Task 1: Fix double checkmark in test.go
- **Status**: DONE
- **Assignee**: Developer
- **Files**: `cli/src/cmd/app/commands/test.go`
- **Action**: Remove `✓` from format strings in `output.Success()` calls on lines 246, 249, 259

### Task 2: Fix executor to capture stderr
- **Status**: DONE
- **Assignee**: Developer
- **Files**: `cli/src/internal/executor/executor.go`
- **Action**: Change `cmd.Output()` to `cmd.CombinedOutput()` in `RunCommandWithOutput` function

### Task 3: Fix additional double icons
- **Status**: DONE
- **Assignee**: Developer
- **Files**: 
  - `cli/src/internal/executor/hooks.go` - Remove `✓` from Success call
  - `cli/src/cmd/app/commands/run.go` - Remove `✗` from Error calls

### Task 4: Verify fix
- **Status**: DONE
- **Assignee**: Tester
- **Action**: Run `azd app test` in demo folder and verify:
  - Single checkmark per line
  - Test counts match actual Jest results (10 passed, 10 total)
  - All existing tests still pass
