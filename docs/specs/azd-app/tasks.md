<!-- NEXT: 1 -->

## TODO

### 1. Convert tests to loopback bindings
- Description: Find all tests that bind TCP listeners to all interfaces (e.g., ":0", ":1234") and update them to bind to the loopback interface (`127.0.0.1` or `localhost`). For tests that intentionally require all-interface binds, mark them to skip on Windows with a runtime check or use build tags.
- Acceptance criteria: No test in CI or local Windows runs binds to all interfaces by default; unit tests pass locally after change.

### 2. Add Windows test networking doc
- Description: Add `cli/docs/troubleshooting/windows-tests.md` explaining why tests bind to loopback, how to run tests that require all-interface binds, and example PowerShell commands to add firewall exceptions.
- Acceptance criteria: Doc added and referenced from troubleshooting index.

### 3. Add test helper for loopback binds (optional)
- Description: Add a small test-only helper (e.g., `testutil.BindLoopback(port)`) to standardize loopback listener creation in tests.
- Acceptance criteria: Helper added under test packages that need it and used by at least 1 test file.

### 4. Provide an optional PowerShell helper script
- Description: Add `scripts/windows/add-firewall-rule.ps1` (optional) that safely adds/removes firewall rules for test ports with checks for elevation.
- Acceptance criteria: Script added under `scripts/windows/` with usage instructions in `cli/docs/troubleshooting/windows-tests.md`.

### 5. Run tests and verify
- Description: Run `go test ./...` in `cli` on Windows (or a Windows CI runner) and confirm no firewall prompts or failing tests. Report results.
- Acceptance criteria: Unit tests pass; any integration tests that require network access are documented and skipped on Windows or isolated.

## IN PROGRESS

- (none)

## DONE

- (none)
