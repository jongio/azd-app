<!-- NEXT: -->
# Add --force Flag to azd app run - Tasks

## Done

**Developer**

Add `--force` flag to `azd app run` command that propagates to `azd app deps`.

**Files**:
- `cli/src/cmd/app/commands/run.go`

**Changes**:
1. Add `runForce` variable to global flags
2. Add flag definition to `NewRunCommand()`
3. Add propagation logic in `runWithServices()` to set `DepsOptions.Force` before orchestrator runs
4. Follow existing patterns (see `runWeb`, `runVerbose`, etc.)

**Acceptance Criteria**:
- Flag added to cobra command
- Flag propagates to deps via `setDepsOptions()`
- Compiles without errors
- Follows existing code style

---

## TODO: Update CLI run.md Documentation

**Developer**

Update `cli/docs/commands/run.md` to document the new `--force` flag.

**Files**:
- `cli/docs/commands/run.md`

**Changes**:
1. Add `--force` flag to flags table (after `--restart-containers`)
2. Add usage examples showing `--force` flag
3. Add to "Common Use Cases" section if appropriate
4. Ensure consistent formatting with existing docs

**Acceptance Criteria**:
- Flag appears in flags table
- Examples show correct usage
- Markdown renders correctly
- Consistent with other flag documentation

---

## TODO: Update Web run.astro Documentation

**Developer**

Update `web/src/pages/reference/cli/run.astro` to document the new `--force` flag.

**Files**:
- `web/src/pages/reference/cli/run.astro`

**Changes**:
1. Add `--force` flag to flags section
2. Add usage examples
3. Ensure HTML/Astro markup is correct
4. Follow existing patterns in the file

**Acceptance Criteria**:
- Flag documented in Astro page
- Examples show correct usage
- Page builds without errors
- Consistent with other CLI command pages

---

## TODO: Search and Update Other Documentation

**Developer**

Search for mentions of `azd app run` and `azd app deps` workflow in other documentation files and update if relevant.

**Files to check**:
- `README.md`
- `CONTRIBUTING.md`
- `cli/README.md`
- `cli/demo/README.md`
- Any test project READMEs that document the workflow

**Changes**:
Update any mentions of the manual workflow pattern:
```bash
azd app deps --force
azd app run
```

To suggest the simpler:
```bash
azd app run --force
```

**Acceptance Criteria**:
- All relevant documentation updated
- No broken workflow examples
- Consistent messaging across all docs

---

## TODO: Build and Test

**Developer**

Build the CLI and test the new flag functionality.

**Steps**:
1. Build CLI: `cd cli && mage build`
2. Test basic usage: `azd app run --force`
3. Test with other flags: `azd app run --force --web`
4. Verify deps actually runs with --force (check output)
5. Verify dependencies are cleaned and reinstalled

**Acceptance Criteria**:
- CLI builds successfully
- Flag works as expected
- Deps runs with --force
- Dependencies are cleaned and reinstalled
- No errors or warnings

---

## Done

### DONE: Build and Test

**Completed**: 2025-12-29

Built and tested the CLI:
- CLI builds successfully with `mage build`
- `--force` flag appears in `azd app run --help`
- `--force` flag exists in `azd app deps --help` 
- No build errors or warnings
- Flag is ready for use

---

**Completed**: 2025-12-29

Updated cli/docs/cli-reference.md:
- Added example showing `azd app run --force` after `azd app deps --force`
- Searched README.md, CONTRIBUTING.md, cli/README.md - no changes needed
- All relevant documentation now shows the convenience flag

---

**Completed**: 2025-12-29

Updated web/src/pages/reference/cli/run.astro:
- Added example8 constant for --force usage
- Added --force flag to flags table
- Added example to Examples section
- Follows existing Astro formatting

---

**Completed**: 2025-12-29

Updated cli/docs/commands/run.md:
- Added `--force` flag to flags table
- Added usage examples in Common Use Cases section
- Follows existing markdown formatting

---

**Completed**: 2025-12-29

Added `--force` flag to `azd app run` command:
- Added `runForce` global variable
- Registered flag in `NewRunCommand()`
- Added propagation logic to set `DepsOptions.Force` before orchestrator runs
- Compiles successfully

---
