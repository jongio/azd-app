# Windows Test Networking and Firewall

Some tests create temporary TCP listeners. On Windows, binding a listener to all interfaces (for example `:0` or `0.0.0.0:1234`) will often trigger a Windows Firewall "allow access" prompt. To avoid interactive prompts and reduce accidental exposure during automated test runs, tests in this project bind to the loopback interface (`127.0.0.1` or `localhost`) by default.

What we changed
- Tests that create listeners now bind explicitly to `127.0.0.1` or `localhost` when they only need local connectivity. This preserves ephemeral-port semantics and avoids firewall prompts.

Running tests that require all-interface binds
- If an integration test intentionally needs to listen on all interfaces, do one of the following:
  - Run the test on a non-Windows runner (Linux, macOS, or WSL).
  - Mark the test to skip on Windows using a runtime guard:

```go
if runtime.GOOS == "windows" {
    t.Skip("skipping all-interface bind test on Windows")
}
```

Adding a firewall exception (manual)
- If you must run all-interface tests on Windows and want to avoid interactive prompts, add a persistent firewall rule as Administrator. Example PowerShell commands:

```powershell
# Allow inbound TCP to a single port
New-NetFirewallRule -DisplayName "Allow azd-app test port 12345" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 12345

# Allow a range of ports (e.g. 45000-46000)
New-NetFirewallRule -DisplayName "Allow azd-app test ports 45000-46000" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 45000-46000

# Remove a rule by name
Remove-NetFirewallRule -DisplayName "Allow azd-app test port 12345"
```

Helper script
- A helper script is provided at `scripts/windows/add-firewall-rule.ps1` to add/remove firewall rules from an elevated PowerShell session. Example usage:

```powershell
# Add single port
.\scripts\windows\add-firewall-rule.ps1 -Action Add -Port 12345 -Name "azd-app-test-12345"

# Remove rule
.\scripts\windows\add-firewall-rule.ps1 -Action Remove -Name "azd-app-test-12345"
```

Test helper in Go
- A small test helper `ListenLoopback(port int)` is available under `cli/src/internal/testing/testutil` to simplify creating loopback listeners in tests. Use it to ensure new tests bind to loopback and avoid firewall prompts.

Notes and best practices
- Prefer loopback binds in tests unless the test explicitly needs an all-interface bind.
- For CI, run all-interface tests on non-Windows runners to avoid firewall configuration.

Contact
- If you observe firewall prompts during `go test` despite these changes, file an issue and include the test file name, OS, and exact steps to reproduce.
Windows test networking notes
==============================

Why tests bind to loopback
--------------------------

Some tests create temporary TCP listeners. Binding to all interfaces (for example ":0" or "0.0.0.0:1234") on Windows will usually trigger a Windows Firewall "allow access" prompt. To avoid interactive prompts and accidental wide network exposure during automated test runs, tests in this module bind to the loopback interface only (`127.0.0.1` or `localhost`). This keeps test semantics (ephemeral ports, local connectivity) while avoiding firewall dialogs.

Running tests that intentionally bind all interfaces
--------------------------------------------------

Occasionally, an integration test may be written to require an all-interface bind (for example to simulate a service listening on all interfaces). These tests should be marked explicitly (for example with a runtime check and `t.Skip()` on Windows) or guarded by build tags so they do not run on Windows by default.

If you need to run such tests on Windows and you accept the firewall prompts, you have two options:

- Run the tests on a Unix-like environment (WSL, Linux, macOS) where the Windows Firewall isn't triggered.
- Or add persistent firewall rules so the test ports are allowed without prompting.

Examples: add firewall exception for a single TCP port (PowerShell - run as Administrator):

```powershell
# Allow inbound TCP to port 12345
New-NetFirewallRule -DisplayName "Allow azd-app test port 12345" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 12345

# Allow a range of ports (e.g. 45000-46000)
New-NetFirewallRule -DisplayName "Allow azd-app test ports 45000-46000" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 45000-46000

# Remove a rule by name
Remove-NetFirewallRule -DisplayName "Allow azd-app test port 12345"
```

Notes and best practices
------------------------

- Prefer binding to `127.0.0.1` or `localhost` in tests unless a test explicitly needs to listen on all interfaces.
- If a test must bind all interfaces, mark it to skip on Windows using a runtime check:

```go
if runtime.GOOS == "windows" {
    t.Skip("skipping all-interface bind test on Windows")
}
```

- For CI, run tests in a non-Windows runner (Linux) for any tests that require all-interface binds, or add firewall rules in the runner image if you control it.

Contact
-------

If you see unexpected firewall prompts while running tests, please open an issue describing the test file and platform so we can adjust the test to use loopback or skip on Windows.
