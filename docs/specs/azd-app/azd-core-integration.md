---
description: Specification for integrating azd-core Key Vault resolution into azd-app
project: azd-app
date: 2026-01-08
status: complete
---

# Integrate azd-core into azd-app

## Scope
Initial effort focuses solely on integrating `azd-core` for Key Vault environment variable reference resolution within `azd-app`. No additions to `azd-core` and no migration of other core utilities in this phase.

## Goals
- Use `azd-core` for Key Vault reference resolution of environment variables.
- Minimize API churn within `azd-app` by adapting thin integration layers.
- Maintain build, tests, and packaging across Windows/Linux.

## Non-Goals
- Large-scale redesign of CLI UX.
- Migrating other core utilities (logging/config/helpers) to `azd-core` in this phase.
- Changing feature behavior beyond necessary refactors for the integration.

## Integration Plan
1. Module dependency
   - Use the existing Go workspace (`go.work`) under `c:\code` to include `azd-app` and `azd-core` for local development, so no `go.mod replace` is needed.
   - Ensure `azd-app` can import `azd-core` via the workspace setup during local dev.
   - For CI: pin to a tagged version of `azd-core` in `go.mod` (CI will not rely on the workspace), and avoid local replaces.
2. Migration map
   - Inventory usages of Key Vault env resolution and other core helpers in `azd-app`.
   - Map each to `azd-core` equivalents and note any adapter needed.
3. Refactor Key Vault env resolver
   - Replace internal implementations with calls to `azd-core`.
   - Keep existing public interfaces stable where possible; add adapters if needed.
4. Migrate other core methods (Deferred)
   - Out of scope for the initial Key Vault-only integration; plan as a future phase.
5. Tests
   - Update unit and integration tests to exercise `azd-core` paths.
   - Add coverage for env var KV resolution including error paths.
6. Build & packaging
   - Validate `go build` and release packaging (mage/build scripts) on Windows/Linux.
7. CI/CD
   - Ensure workflows use a tagged `azd-core` version; run preflight checks.
8. Docs
   - Add contributor guidance for local `replace` usage and module pinning.
   - Update CHANGELOG and CLI docs where behavior or references change.

## Acceptance Criteria
- All tests pass on Windows/Linux; coverage ≥ 80% for KV-related paths.
- No duplicated KV resolver implementation remains in `azd-app`.
- CI builds use a tagged `azd-core` version without local replace.
- Documentation updated to reflect `azd-core` KV integration and contributor workflow.

## Risks & Mitigations
- API mismatches: Use adapters/shims to avoid churn.
- CI dependency drift: Pin `azd-core` versions and monitor with preflight.
- Local vs CI differences: Document workspace-based dev flow vs tagged CI dependencies; validate both paths.

## Rollout
- Implement behind a short-lived feature branch, open PR.
- Incrementally migrate components starting with Key Vault, then other core methods.
- Merge when tests are green and docs updated.