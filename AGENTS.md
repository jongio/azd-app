# AGENTS.md — jongio/azd-app

## Overview

**azd-app** is an Azure Developer CLI (azd) extension that orchestrates multi-service application development. It provides service discovery, lifecycle management, health checks, log streaming, and a real-time dashboard — all driven from a single `azd app run` command.

## Architecture

Monorepo with three major components:

- **cli/** — Go CLI extension (the core product)
- **web/** — Astro 6 documentation site
- **proto/** — Protobuf service definitions (Connect-RPC v2)

### CLI (Go)

- **Module**: `github.com/jongio/azd-app/cli`
- **Go version**: Declared in `cli/go.mod`; CI reads it directly
- **Framework**: Cobra (CLI), Connect-RPC (dashboard ↔ CLI transport)
- **Core dependency**: `github.com/jongio/azd-core` (shared extension SDK)
- **Build tool**: Mage (`magefile.go` at `cli/magefile.go`)
- **Package structure**:
  - `cli/src/cmd/app/` — Entry point
  - `cli/src/cmd/app/commands/` — Command implementations (run, logs, health, test, etc.)
  - `cli/src/internal/` — Domain packages (service, detector, executor, orchestrator, portmanager, etc.)
  - `cli/src/gen/proto/` — Generated protobuf Go code
- **Dashboard**: `cli/dashboard/` — Vite + React 19 SPA, communicates via Connect-RPC

### Web (Astro)

- **Framework**: Astro 6 with MDX, Tailwind CSS v4, Expressive Code
- **Package manager**: pnpm (version 9+)
- **TypeScript**: strict mode, v5.9.3
- **Build**: `astro build && pagefind --site dist`
- **Testing**: Playwright e2e with snapshot updates

### Proto (Buf)

- **9 proto files**: azure, bicep, common, health, lifecycle, logs, mode, project, services
- **Code gen**: Go via protoc-gen-connect-go, TypeScript via protoc-gen-es v2
- **Lint rule**: STANDARD (minus PACKAGE_NO_IMPORT_CYCLE)

## Conventions

### Commits

Conventional Commits strictly enforced:
- `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`, `ci:`, `deps:`
- Scoped variants: `feat(logging):`, `fix(detector):`

### Go Code Style

- **Error handling**: `fmt.Errorf` with `%w` wrapping — always add context
- **Logging**: slog-based via logutil, component-scoped: `NewLogger("component-name")`
- **Naming**: PascalCase exports, camelCase unexported, descriptive domain package names
- **Interfaces**: Suffix with role (e.g., `*Credential`, `*Logger`, `*Provider`)
- **Comments**: Package-level docs required; exported symbols documented; minimal inline comments

### Testing

- **Framework**: `testing` stdlib + `github.com/stretchr/testify` (assert/require)
- **Pattern**: Table-driven tests with subtests (`t.Run`)
- **Coverage target**: 80% minimum (codecov enforced)
- **File naming**: `*_test.go` (unit), `*_integration_test.go`, `*_e2e_test.go`, `benchmark_test.go`
- **Helpers**: `t.TempDir()` for filesystem tests, mocks for external deps

### Linting

- **Config**: `.golangci.yml` — 24 linters enabled, 5-minute timeout
- **Key linters**: errcheck, govet, staticcheck, gosec, revive, dupl, exhaustive
- **Security**: gosec enabled (G204/G304 excluded for CLI exec patterns)
- **Test exclusions**: Broad — most linters disabled for `_test.go` files
- **Run**: `mage preflight` (runs format, imports, security, lint)

## CI/CD

- **Main CI**: `.github/workflows/ci.yml` — preflight, lint, test on ubuntu/windows/macos matrix
- **Go version**: `cli/go.mod`, Node: 22, pnpm: 9
- **Race detector**: Enabled on Linux/Windows, disabled on macOS
- **Coverage**: codecov integration with threshold enforcement
- **Security**: CodeQL + govulncheck (push/PR/weekly schedule)
- **Release**: Manual trigger via `release.yml` (patch/minor/major)
- **Website**: Auto-deploy on push to main (web/ changes)

## Dependency Management

- **Dependabot**: Weekly scans for gomod + github-actions
  - Commit prefix: `deps:` for Go, `ci:` for Actions
  - No npm/pnpm ecosystem configured
- **Go modules**: Standard (no vendoring)
- **Node**: pnpm with lockfile

## Dev Environment

- **Devcontainer**: Latest Go 1.26 patch, Node LTS, Python 3.12, .NET 8, Docker-in-Docker
- **IDE**: VS Code with golang.go, Copilot, Azure Dev extensions
- **Gopls settings**: nilness, shadow, ST1003, unusedparams, unusedwrite, useany, staticcheck

## Build Commands

| Command | Purpose |
|---------|---------|
| `mage build` | Build CLI binary |
| `mage test` | Run tests |
| `mage lint` | Run linters |
| `mage preflight` | Full quality gate (format + imports + security + lint) |
| `mage coverage` | Generate coverage report |
| `cd web && pnpm install && pnpm build` | Build docs site |
| `cd cli/dashboard && pnpm install && pnpm build` | Build dashboard |
| `buf generate` | Regenerate proto stubs |

## Registry

- **Extension ID**: `jongio.azd.app`
- **Namespace**: `app`
- **registry.json**: Production registry (stable releases)
- **pr-registry.json**: PR testing registry (pre-release features incl. MCP server)
