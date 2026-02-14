# Container Auth Integration Test — CI/CD Runner Requirements

Research findings for running the container-auth integration test on GitHub Actions.

## 1. Runner Compatibility Matrix

| Capability | `ubuntu-latest` | `windows-latest` | `macos-latest` |
|---|---|---|---|
| **Docker pre-installed** | ✅ Yes | ❌ No | ❌ No |
| **Docker can run Linux containers** | ✅ Native | ❌ Not feasible | ❌ Not feasible |
| **Go pre-installed** | ✅ (use `actions/setup-go`) | ✅ (use `actions/setup-go`) | ✅ (use `actions/setup-go`) |
| **.NET SDK 8.0 pre-installed** | ✅ (use `actions/setup-dotnet`) | ✅ (use `actions/setup-dotnet`) | ✅ (use `actions/setup-dotnet`) |
| **PowerShell (`pwsh`)** | ✅ Yes | ✅ Yes | ✅ Yes |
| **`host.docker.internal`** | ⚠️ Needs `--add-host` | N/A | N/A |
| **Podman pre-installed** | ⚠️ Old version (3.x) | ❌ No | ❌ No |
| **WSL available** | N/A | ⚠️ Possible via `setup-wsl` action | N/A |
| **azd CLI pre-installed** | ❌ Install via script | ❌ Install via script | ❌ Install via script |

### Recommendation

**Use `ubuntu-latest` as the primary (and initially only) CI runner.** It is the only GitHub-hosted runner with Docker pre-installed and capable of running Linux containers natively. The test pulls `mcr.microsoft.com/dotnet/sdk:8.0` (a Linux image) and volume-mounts a Linux amd64 shim binary, so only Linux Docker hosts are viable.

## 2. Platform-Specific Test Adaptations

### 2.1 Container Networking (`host.docker.internal`)

**Problem:** On native Linux (which `ubuntu-latest` is), `host.docker.internal` does NOT resolve by default. It only works out-of-the-box on Docker Desktop (macOS/Windows).

**Solution:** The test script (`test.ps1`) already handles this correctly:

```powershell
$runningOnLinux = $IsLinux -or ((Test-Path "/proc/version") -and -not (Test-Path "/mnt/c/Windows"))
if ($runningOnLinux) {
    $dockerArgs += @("--add-host", "host.docker.internal:host-gateway")
}
```

This adds `--add-host host.docker.internal:host-gateway` on Linux, which maps `host.docker.internal` to the host's gateway IP (typically `172.17.0.1`). Requires Docker 20.10.0+ (satisfied on `ubuntu-latest`).

**No changes needed** — the script is already CI-ready for Ubuntu runners.

### 2.2 Architecture Detection

The test script detects Mac ARM64 and defaults to `amd64` otherwise. On `ubuntu-latest` (x86_64), this correctly produces a `linux/amd64` shim. The `AZD_SHIM_ARCH` env var override is available if needed.

**No changes needed.**

### 2.3 Temp Directory Paths

The script uses `[System.IO.Path]::GetTempPath()` which works cross-platform in PowerShell.

**No changes needed.**

## 3. Windows and macOS Runners

### 3.1 Why They Don't Work (Currently)

- **Windows runners** don't have Docker pre-installed. Even if installed, Windows Docker can only run Windows containers natively. Running Linux containers requires WSL2/Hyper-V, which is unreliable in CI.
- **macOS runners** don't have Docker pre-installed. Docker Desktop requires a license for commercial use and is not trivially installable in CI. macOS runners have also moved to Apple Silicon (arm64), adding architecture complexity.

### 3.2 Future Consideration

If we want to test the host-side auth server behavior on Windows/macOS (without Docker), we could split the test into:
1. **Unit tests** — Test cert generation, mTLS handshake, token forwarding (no Docker needed)
2. **Integration test** — Full end-to-end with Docker (Ubuntu only)

This is not needed today since the Go code is already unit-tested separately.

## 4. WSL Testing

### 4.1 Can We Test WSL on GitHub Actions?

**Yes, but it's complex and fragile.** Windows runners have WSL2 available but no distro pre-installed. The [`Vampire/setup-wsl`](https://github.com/marketplace/actions/setup-wsl) action can install Ubuntu, but:

- Adds 2-5 minutes to setup
- Docker-in-WSL requires additional configuration
- WSL networking to host is different from native Docker networking
- The test would need significant adaptation to run inside WSL

### 4.2 Recommendation

**Don't test WSL in CI for now.** The container-auth feature uses Docker (not WSL) for its primary use case. WSL testing should be done manually or in a dedicated self-hosted runner if ever needed.

## 5. Podman Support

### 5.1 Current State

- Podman is available on `ubuntu-latest` but at an outdated version (3.x from apt)
- The test script already detects Podman and adjusts the container host:

```powershell
$dockerVer = docker --version 2>&1
if ($dockerVer -match "podman") { $containerHost = "host.containers.internal" }
```

### 5.2 Podman CI Testing

To test with Podman in CI, we would need to:
1. Install a recent Podman 4.x+ (via `redhat-actions/podman-setup` or manual apt install)
2. Alias `podman` as `docker` (or modify the test to use `podman` directly)
3. Handle Podman networking differences (`host.containers.internal` vs `host.docker.internal`)

### 5.3 Recommendation

**Defer Podman CI testing.** Docker is the primary target. Podman compatibility can be validated manually until there's demand for it.

## 6. Azure Authentication in CI

### 6.1 The Challenge

The test calls `azd auth token --scope "https://management.azure.com/.default"` to verify the full auth chain. In CI, we need a non-interactive way to authenticate `azd`.

### 6.2 Option A: Federated Credentials (OIDC) — Recommended

Use GitHub Actions OIDC with Azure Workload Identity Federation. No secrets to rotate.

**Azure setup:**
1. Create an Azure AD App Registration
2. Create a Service Principal
3. Add a Federated Credential with:
   - Issuer: `https://token.actions.githubusercontent.com`
   - Subject: `repo:<org>/<repo>:ref:refs/heads/main` (or environment-scoped)
   - Audience: `api://AzureADTokenExchange`
4. Grant `Reader` role on a test subscription/resource group

**GitHub setup:**
- Repository secrets: `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`
- Workflow permissions: `id-token: write`

**Workflow:**
```yaml
- uses: azure/login@v2
  with:
    client-id: ${{ secrets.AZURE_CLIENT_ID }}
    tenant-id: ${{ secrets.AZURE_TENANT_ID }}
    subscription-id: ${{ secrets.AZURE_SUBSCRIPTION_ID }}

- name: Login azd
  run: azd auth login --client-id ${{ secrets.AZURE_CLIENT_ID }} --federated-credential-provider github --tenant-id ${{ secrets.AZURE_TENANT_ID }}
```

### 6.3 Option B: Service Principal with Client Secret

Simpler setup but requires secret rotation.

**GitHub setup:**
- Repository secrets: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`

**Workflow:**
```yaml
- name: Login azd
  run: azd auth login --client-id ${{ secrets.AZURE_CLIENT_ID }} --client-secret ${{ secrets.AZURE_CLIENT_SECRET }} --tenant-id ${{ secrets.AZURE_TENANT_ID }}
```

### 6.4 Option C: Skip Azure Tests in CI

Use the existing `-SkipAzure` flag to run only injection validation tests (no Azure subscription needed):

```yaml
- name: Run test
  run: pwsh test.ps1 -SkipAzure
```

This validates shim mounting, cert injection, and env vars without Azure credentials.

### 6.5 Recommendation

**Start with Option C (`-SkipAzure`) for initial CI.** This provides value immediately with zero Azure setup. Then add Option A (OIDC) later for full end-to-end testing.

## 7. Go and .NET SDK Availability

### 7.1 Go

Go is pre-installed on `ubuntu-latest` but the version may lag behind. The existing `ci.yml` uses `actions/setup-go@v5` with a specific version. The container-auth test needs Go 1.22+ for cross-compilation.

**Use `actions/setup-go@v5`** to ensure the correct version.

### 7.2 .NET SDK

.NET SDK 8.0 is pre-installed on `ubuntu-latest`. The test container (`mcr.microsoft.com/dotnet/sdk:8.0`) brings its own .NET SDK, so the host doesn't need it. No setup action required for .NET.

### 7.3 azd CLI

`azd` is NOT pre-installed on any runner. Install with:

```yaml
- name: Install azd
  run: curl -fsSL https://aka.ms/install-azd.sh | bash
```

This is already the pattern used in the existing `ci.yml`.

## 8. Draft GitHub Actions Workflow

```yaml
name: Container Auth Integration Test

on:
  pull_request:
    branches: [main]
    paths:
      - 'cli/src/internal/containerauth/**'
      - 'cli/tests/projects/integration/container-auth-test/**'
      - '.github/workflows/container-auth-test.yml'
  workflow_dispatch:

# For OIDC auth (Option A), uncomment:
# permissions:
#   id-token: write
#   contents: read

env:
  GO_VERSION: '1.25.7'

jobs:
  container-auth-test:
    name: Container Auth Integration
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go (bootstrap)
        uses: actions/setup-go@v5
        with:
          go-version: 'stable'
          cache: false

      - name: Install Go ${{ env.GO_VERSION }}
        run: |
          go install golang.org/dl/go${{ env.GO_VERSION }}@latest
          go${{ env.GO_VERSION }} download
          echo "$(go env GOPATH)/bin" >> $GITHUB_PATH
          ln -sf $(which go${{ env.GO_VERSION }}) $(go env GOPATH)/bin/go

      - name: Verify Go version
        run: go version

      - name: Install azd
        run: curl -fsSL https://aka.ms/install-azd.sh | bash

      # Option C: Skip Azure tests (no credentials needed)
      - name: Run container auth test (injection only)
        working-directory: cli/tests/projects/integration/container-auth-test
        run: pwsh test.ps1 -SkipAzure -Verbose

      # --- Uncomment below for full Azure tests (requires secrets) ---
      # - name: Login azd (Option A: OIDC)
      #   run: |
      #     azd auth login \
      #       --client-id ${{ secrets.AZURE_CLIENT_ID }} \
      #       --federated-credential-provider github \
      #       --tenant-id ${{ secrets.AZURE_TENANT_ID }}
      #
      # - name: Run container auth test (full)
      #   working-directory: cli/tests/projects/integration/container-auth-test
      #   run: pwsh test.ps1 -Verbose
```

## 9. Known Limitations and Workarounds

| Limitation | Impact | Workaround |
|---|---|---|
| Docker only on Ubuntu runners | Can't test on Windows/macOS in CI | Use Ubuntu as primary; test others manually |
| `host.docker.internal` not native on Linux | Container can't reach host services | `--add-host host.docker.internal:host-gateway` (already in test.ps1) |
| No WSL distro pre-installed on Windows runners | Can't test WSL scenarios easily | Use `Vampire/setup-wsl` action or test manually |
| Podman version outdated on runners | Can't reliably test Podman compat | Install newer version or defer to manual testing |
| `azd` not pre-installed | Extra setup step needed | `curl -fsSL https://aka.ms/install-azd.sh \| bash` |
| Azure auth requires setup | Full end-to-end needs credentials | Start with `-SkipAzure`; add OIDC later |
| Container image pull time | `mcr.microsoft.com/dotnet/sdk:8.0` is ~800MB | First run slower; consider caching if frequent |
| Go cross-compilation | Shim must be `linux/amd64` | Already handled (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`) |

## 10. Multi-Platform Test Strategy

### Can one `test.ps1` handle all platforms?

**Yes.** The current `test.ps1` already handles platform differences:
- Architecture detection (amd64 vs arm64)
- Linux networking (`--add-host` on native Linux)
- Podman detection (host.containers.internal)
- Cross-platform temp paths

### Recommended strategy:

1. **Phase 1 (Now):** Ubuntu-only CI with `-SkipAzure`. Validates injection mechanics.
2. **Phase 2:** Add OIDC credentials for full Azure end-to-end testing on Ubuntu.
3. **Phase 3 (If needed):** Add Podman matrix variant on Ubuntu.
4. **Phase 4 (If needed):** Add macOS arm64 testing if Docker becomes available on those runners.

No separate test project per platform is needed. The single `test.ps1` is sufficient.
