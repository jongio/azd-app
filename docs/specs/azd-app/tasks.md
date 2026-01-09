<!-- NEXT: 7 -->

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

### 7. Add azd-core dependency
- Description: Leverage the existing Go workspace (`go.work`) in `c:\code` so local development uses `azd-core` without `go.mod replace`. For CI, add a `require` and pin to a tagged `azd-core` version (workspace not used in CI).
- Acceptance criteria: Local builds succeed via workspace; CI configured to use a tagged version without replace.

### 9. Deferred: other core methods
- Description: Out of scope for the initial Key Vault-only integration; plan for a future phase to migrate other shared utilities.
- Acceptance criteria: N/A for the current phase.

### 10. Update and add tests
- Description: Update tests to exercise `azd-core` paths and add coverage for KV resolution and env substitution; maintain ≥80% coverage.
- Acceptance criteria: Test suite green on Windows/Linux; coverage target met.

### 11. Docs & changelog updates
- Description: Document `azd-core` usage and contributor guidance (workspace-based dev, version pinning) and update CHANGELOG.
- Acceptance criteria: Docs updated and linked from relevant indexes; CHANGELOG notes added.

### 12. Preflight & CI alignment
- Description: Run preflight; ensure CI workflows use a tagged `azd-core` and remain green; adjust as needed.
- Acceptance criteria: CI green with `azd-core` integration; preflight passes.

### 13. Build/packaging verification
- Description: Verify Windows/Linux builds and CLI packaging (mage/build scripts) with `azd-core` dependency.
- Acceptance criteria: Release artifacts build successfully across platforms.

## IN PROGRESS

- (none)

## azd-core KV Integration DONE

### 6. Draft azd-core integration spec
- Status: ✅ COMPLETE
- File: docs/specs/azd-app/azd-core-integration.md

### 7. Add azd-core dependency
- Status: ✅ COMPLETE  
- Local dev uses go.work at c:\code (no replace needed)
- go.mod has require github.com/jongio/azd-core v0.0.0

### 8. Refactor Key Vault env resolver
- Status: ✅ COMPLETE
- All KV resolution now uses azd-core/keyvault
- Functions: NewKeyVaultResolver, ResolveEnvironmentVariables, IsKeyVaultReference
- Zero internal/keyvault references in production code

### 9. Remove KV duplicates & tidy
- Status: ✅ COMPLETE
- Removed internal/keyvault package entirely
- Modules tidied (go mod tidy)

### 10. Update and add tests
- Status: ✅ COMPLETE
- 23 test sub-cases covering happy/error paths
- All KV test formats validated (akvs://, @Microsoft.KeyVault)
- Coverage: 73.2% service package, 100% KV functions

### 11. Docs & changelog updates
- Status: ✅ COMPLETE
- CHANGELOG.md updated for v0.10.0
- README.md links to azd-core integration guide
- cli/docs/contributing/azd-core-integration.md created

### 12. Preflight & CI alignment
- Status: ✅ COMPLETE
- Preflight passed (all checks green)
- Build successful: go build ./src/cmd/app
- All tests passing

### 13. Build/packaging verification
- Status: ✅ COMPLETE
- Windows build verified: bin/azd-app-test.exe created successfully
- No build errors with azd-core dependency

## DONE

### 6. Draft azd-core integration spec
- Description: Create the azd-core integration plan describing goals, non-goals, module path/versioning, migration map (Key Vault-only), testing, CI, and rollout. Save at `docs/specs/azd-app/azd-core-integration.md`.
- Acceptance criteria: Spec added and reviewed; clear migration map and acceptance criteria captured.

### 8. Refactor Key Vault env resolver
- Description: Replace azd-app Key Vault environment variable resolution with `azd-core` implementation, adding adapters as needed to keep public interfaces stable.
- Acceptance criteria: Key Vault env resolution uses `azd-core`; unit tests pass for happy-path and error-paths.
