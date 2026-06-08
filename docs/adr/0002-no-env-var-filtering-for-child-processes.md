# ADR-0002: No Environment Variable Filtering for Child Processes

## Status

Accepted

## Date

2026-06-07

## Context

PR #319 (security remediation) introduced a sensitive environment variable filter that
stripped variables matching certain patterns (`*_TOKEN`, `*_SECRET`, `*_KEY`, `*_PASSWORD`,
`AWS_*`, `AZURE_*`) from child service processes spawned by `azd app run`.

The intent was to prevent credential leakage (CWE-200, CWE-526) from the parent process
to child services. A narrow allowlist preserved a few known-safe Azure config variables.

However, this caused a regression: the parent `azd` process injects all environment values
from `azd env` (bicep outputs, service URLs, resource names, connection strings) into the
extension's process environment. The `AZURE_*` prefix denylist was too broad and blocked
legitimate, non-sensitive values that services need to function:

- `AZURE_ENV_NAME`
- `AZURE_RESOURCE_GROUP_NAME`
- `AZURE_CONTAINER_APPS_ENVIRONMENT_NAME`
- `AZURE_CONTAINER_REGISTRY_NAME`
- `AZURE_KEY_VAULT_NAME`
- Any custom bicep output with `AZURE_` prefix

## Decision

**Remove all environment variable filtering from child process spawning.**

Child processes inherit the full parent environment without modification. The rationale:

1. **azd's contract**: The parent `azd` CLI deliberately injects environment values into
   the extension process. These values are the primary mechanism for passing deployment
   configuration to services during local development. Filtering them breaks the contract.

2. **Local dev context**: `azd app run` is a local development tool. The developer's own
   machine already has access to these credentials. Filtering them from child processes
   provides no meaningful security boundary - the developer can `echo $GITHUB_TOKEN` at
   any time.

3. **False positive cost is too high**: Any prefix/suffix heuristic will either block
   legitimate values (breaking functionality) or require an ever-growing allowlist that
   must be updated for every new bicep output pattern.

4. **Credential isolation belongs elsewhere**: In production, credential isolation is
   handled by managed identity, Key Vault references, and network policies. In local dev,
   the responsibility is on the developer's environment configuration, not on the
   orchestrator.

## Consequences

- Child service processes see ALL environment variables from the parent, including any
  credentials (e.g., `GITHUB_TOKEN`, `AWS_SECRET_ACCESS_KEY`) present in the developer's
  shell.
- Input validation for environment variable names is retained (protection against injection
  via malformed variable names with control characters).
- If credential isolation for child processes is needed in the future, it should be opt-in
  via explicit configuration in `azure.yaml` rather than a heuristic filter.

## Security Notes

The following mitigations remain in place:

- Environment variable name validation (`isValidEnvVarName`) prevents injection attacks
  via malformed names containing control characters, shell metacharacters, or null bytes.
- Key Vault reference resolution allows secrets to be stored securely and only resolved
  at runtime.
- The `environment:` section in `azure.yaml` allows explicit control over which variables
  a service receives (opt-in approach for sensitive values from Key Vault).
