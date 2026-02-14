# Container Auth Test

Tests the `local.credentials: azd` feature in azure.yaml, which automatically forwards Azure credentials into Docker containers via an mTLS auth server and azd CLI shim.

## What It Tests

1. **Shim injection** — Verifies the azd CLI shim binary is mounted at `/usr/local/bin/azd`
2. **Certificate injection** — Verifies mTLS certificates are mounted at `/run/secrets/azd-auth/`
3. **Environment variables** — Verifies `AZD_AUTH_HOST`, `AZD_AUTH_PORT`, `AZD_AUTH_CERTS_DIR` are set
4. **Token acquisition** — Acquires an Azure token via `DefaultAzureCredential` → `AzureDeveloperCliCredential` → shim → mTLS → host
5. **Azure API call** — Lists resource groups from inside the container using the acquired token

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Container (mcr.microsoft.com/dotnet/sdk:8.0)   │
│                                                 │
│  DefaultAzureCredential                         │
│    → AzureDeveloperCliCredential                │
│      → /usr/local/bin/azd auth token --scope X  │
│        (shim binary, volume-mounted)            │
│                                                 │
│  Reads mTLS certs from /run/secrets/azd-auth/   │
│  POST https://host.docker.internal:<port>/token │
└────────────────────┬────────────────────────────┘
                     │ mTLS (TLS 1.3, mutual cert auth)
┌────────────────────▼────────────────────────────┐
│  Host: azd auth server (auto-started)           │
│  - Validates client certificate                 │
│  - Validates scope format + allowlist           │
│  - Rate limits (30 req/min per IP)              │
│  - Calls real: azd auth token --scope X         │
│  - Returns token to container                   │
└─────────────────────────────────────────────────┘
```

## Prerequisites

- Go 1.22+ (for cross-compiling the shim)
- Docker Desktop (or compatible container runtime)
- `azd auth login` completed on the host (for Azure tests; optional with `-SkipAzure`)

## Running

### Full test (requires Azure credentials):
```bash
pwsh test.ps1
```

### Injection-only test (no Azure credentials needed):
```bash
pwsh test.ps1 -SkipAzure
```

### Manual test:
```bash
cd cli/tests/projects/integration/container-auth-test
azd app run
# Then in another terminal:
curl http://localhost:8080/check   # Verify injection
curl http://localhost:8080/token   # Acquire token
curl http://localhost:8080/azure   # List resource groups
```

### CI

See [CI-TESTING.md](CI-TESTING.md) for GitHub Actions setup, runner compatibility, and Azure authentication options.

## Configuration

The `azure.yaml` uses `local.credentials: azd` under the service's `local` block:

```yaml
services:
  testapp:
    host: containerapp
    local:
      credentials: azd
```

## Endpoints

| Endpoint | Description | Requires Azure Login |
|----------|-------------|---------------------|
| `GET /health` | Basic health check | No |
| `GET /check` | Validates shim + certs + env vars are injected | No |
| `GET /token` | Acquires Azure token via DefaultAzureCredential | Yes |
| `GET /azure` | Lists resource groups using Azure Resource Manager | Yes |
