# Port Kill Improvements Tasks

## Overview
Fix port killing to actually kill processes and add "always kill" preference.

Spec: [spec.md](spec.md)

---

## Tasks

### 1. Fix process tree killing on Windows
- **Status**: DONE
- **Assignee**: Developer
- **File**: `src/internal/portmanager/process.go`

**Acceptance Criteria**:
- [x] Update `buildKillProcessCommand` to kill child processes first on Windows
- [x] Use `Get-CimInstance Win32_Process` to find child processes
- [x] Kill children recursively (deepest first)
- [x] Add recursive PowerShell function for clarity
- [x] Unit tests for process tree kill command generation

---

### 2. Fix process tree killing on Unix
- **Status**: DONE
- **Assignee**: Developer
- **File**: `src/internal/portmanager/process.go`

**Acceptance Criteria**:
- [x] Update kill command to use `pkill -P` for child processes first
- [x] Then kill parent with `kill -9`
- [x] Unit tests for Unix kill command generation

---

### 3. Add "always kill" preference support
- **Status**: DONE  
- **Assignee**: Developer
- **Files**: `src/internal/portmanager/portmanager.go`

**Acceptance Criteria**:
- [x] Add `getAlwaysKillPreference()` method using `ConfigClient.GetPreference`
- [x] Add `setAlwaysKillPreference(bool)` method using `ConfigClient.SetPreference`
- [x] Preference key: `alwaysKillPortConflicts`
- [x] Unit tests for preference get/set

---

### 4. Update AssignPort prompts with "always kill" option
- **Status**: DONE
- **Assignee**: Developer
- **File**: `src/internal/portmanager/portmanager.go`

**Acceptance Criteria**:
- [x] Check `alwaysKillPortConflicts` preference before showing prompt
- [x] If preference is `true`, skip prompt and kill automatically
- [x] Add option 4: "Always kill processes (don't ask again)"
- [x] When user chooses 4, set preference and kill process
- [x] Update all three prompt locations (explicit, assigned, preferred ports)
- [x] Unit tests for prompt bypass when preference set

---

### 5. Documentation updates
- **Status**: DONE
- **Assignee**: Developer
- **Files**: `docs/features/ports.md`

**Acceptance Criteria**:
- [x] Document the `alwaysKillPortConflicts` preference
- [x] Document how to reset the preference
- [x] Document process tree killing behavior

---

### 6. Integration testing
- **Status**: TODO
- **Assignee**: Tester
- **Files**: Test projects

**Acceptance Criteria**:
- [ ] Test killing Node.js process with workers
- [ ] Test killing Python process with workers
- [ ] Test preference persistence across runs
- [ ] Test on Windows, macOS, Linux

---

## Dependencies
- ~~Task 3 depends on Tasks 1-2 (kill logic must work before adding auto-kill)~~
- ~~Task 4 depends on Task 3 (need preference support)~~
- ~~Task 5-6 can run in parallel after Task 4~~

All implementation tasks complete. Integration testing remains.
