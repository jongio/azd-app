# Test Plan: Native container config for `host: local` services

Issue: https://github.com/jongio/azd-app/issues/546
Spec: ./spec.md

## Status: COVERED

Framework: Go `testing` + `stretchr/testify`, table-driven with `t.Run`.
Unit tests are Docker-free (assert on generated `docker` argv / parsed config).
Integration/e2e tests are gated behind the repo's `integration && docker` build
tags and skip when Docker is unavailable.

All planned rows are automated. Mapping to implemented tests:

| # | AC | Automated test | Status |
|---|----|----------------|--------|
| T1,T2 | AC1 | `docker.TestBuildRunArgs_Volumes` | automated |
| T3 | AC1 | `service.TestResolveVolumeSpec_RelativeBindResolves` | automated |
| T4 | AC1 | `service.TestSplitVolumeSource`, `TestResolveVolumeSpec_WindowsDriveBind` | automated |
| T5 | AC1 | `service.TestService_Volumes` | automated |
| T6 | AC1 | `docker.TestValidateVolumeSpec`, `TestContainerConfigValidate_RejectsBadFields` | automated |
| T7 | AC2 | `service.TestService_Command_StringForm` | automated |
| T8 | AC2 | `service.TestService_Command_ArrayForm`, `TestService_Command_InvalidType` | automated |
| T9 | AC2 | `docker.TestBuildRunArgs_CommandAfterImage`, `TestBuildRunArgs_NoCommandOmitsTokens` | automated |
| T10 | AC2 | `service.TestDetectContainerRuntime_WiresContainerFields` | automated |
| T11 | AC3 | `service.TestBuildContainerPortMappings_MultiPort`, `TestDetectContainerRuntime_MultiPort` | automated |
| T12 | AC3 | `service.TestBuildContainerPortMappings_HostContainerDistinct` | automated |
| T13 | AC3 | `service.TestBuildContainerPortMappings_FallbackPrimary`, `_NoPorts` | automated |
| T14 | AC4 | `docker.TestBuildRunArgs_NetworkAndAliases` | automated |
| T15 | AC4 | `service.TestDeriveNetworkName`, `TestSanitizeNetworkComponent` | automated |
| T16 | AC4 | `service.TestEnsureNetwork_IdempotentAndRemoveTolerant` (docker) | automated |
| T17 | AC4 | `docker.TestBuildRunArgs_NoNetworkWhenUnset` | automated |
| T18 | AC4 | `service.TestContainerNetwork_InterContainerDNS` (docker) | automated |
| T19 | AC4 | `service.TestContainerReuse_NetworkConnect` (docker) | automated |
| T20,T21,T22 | AC5 | `service.TestShouldPullImage` | automated |
| T23 | AC5 | `docker.TestValidatePullPolicy`; `service.TestService_PullPolicy` | automated |
| T24 | AC6 | existing `service` orchestrator/graph tests (uses ordering unchanged) | automated |
| T25 | AC7 | `service.TestV11SchemaDocumentsContainerFields` | automated |
| T26 | AC8 | `service.TestStartContainerService_WebsiteStyleTopology` (docker): full path: command + 3 ports + named volume + project network, verified via docker inspect | automated |
| T27 | AC9 | `service.TestService_RunsAsLocalProcess`: routing predicate (image=container; docker.*+command=process) | automated |
| T28 | AC9 | `service.TestDetectServiceRuntime_DockerServiceWithCommandRunsAsProcess` / `_DockerServiceWithoutCommandStaysContainer`: routing + backward compat | automated |
| T29 | AC10 | `testing.TestValidateService_ExplicitCommand_UnsupportedLanguage` / `_DefaultsFrameworkToCustom` / `TestValidateService_DockerNoExplicitCommand_Skipped`; `TestHasExplicitCommand`: explicit `test:` makes a docker/unset-language service testable; no-command service still skipped | automated |
| T30 | AC10 | `testing.TestNewRunnerForService_ExplicitConfig_FrameworkDispatch` / `_LanguageWins` / `_UnsupportedLanguage_NoExplicitCommand`: runner selected by framework for explicit-config services | automated |
| T31 | AC10 | `testing.TestExecuteServiceTests_All_ExpandsExplicitTypes`: `--type all` runs each configured explicit type and aggregates | automated |
| T32 | AC10 | `testing.TestExecuteServiceTests_UnconfiguredType_NonTestLanguage_Skipped`; `TestTypeHasExplicitCommand`; `TestIsRecognizedTestLanguage`: a non-test-language service is skipped (not run via the framework default) for a requested type it did not configure | automated |
| T33 | AC11 | `commands.TestDetectProjectsFromAzureYaml_DedupesSharedProjectDir`: three services sharing `project: .` collapse to one node project; `commands.TestGroupedNodeLabel` / `TestServiceDirsFromAzureYaml`, `installer.TestAddNodeProjectLabeled`: shared install is labeled with the covering service names | automated |
| T34 | AC12 | `service.TestInferLogLevel`: structured `level`/`severity` honored; word-boundary keywords (no `errorReporter`/`trace_worker` misfire); unclassified stderr→WARN, stdout→INFO | automated |

Original planned matrix retained below for traceability.


## Planned Tests

| # | AC | Type | Location | Description | Status |
|---|----|------|----------|-------------|--------|
| T1 | AC1 | unit | docker/exec_test.go | `buildRunArgs` emits `-v name:/path` for a named volume | planned |
| T2 | AC1 | unit | docker/exec_test.go | `buildRunArgs` emits `-v <abs>:/path` for a bind mount (host path already absolute) | planned |
| T3 | AC1 | unit | service/container_runner_test.go | relative bind path (`./cfg.json:/c`) resolves to project-dir-absolute before docker args | planned |
| T4 | AC1 | unit | service/container_runner_test.go | Windows drive path (`C:\data:/c`) classified as bind mount, not `name:path` | planned |
| T5 | AC1 | unit | service/types_test.go | `volumes:` YAML sequence unmarshals into `Service.Volumes` | planned |
| T6 | AC1 | unit | docker/types_test.go | volume spec with shell metachars / injection is rejected by validation | planned |
| T7 | AC2 | unit | service/types_test.go | `command:` string form unmarshals (existing behavior preserved) | planned |
| T8 | AC2 | unit | service/types_test.go | `command:` array form (`["postgres","-c","x=y"]`) unmarshals to tokens | planned |
| T9 | AC2 | unit | docker/exec_test.go | `buildRunArgs` appends command tokens AFTER the image, as discrete argv | planned |
| T10 | AC2 | unit | service/container_runner_test.go | container ContainerConfig carries the command (wired from runtime) | planned |
| T11 | AC3 | unit | service/container_runner_test.go | `buildContainerPortMappings` maps ALL 3 azurite ports (10000/1/2) | planned |
| T12 | AC3 | unit | service/container_runner_test.go | host:container port form (`3000:8080`) maps distinct host/container ports | planned |
| T13 | AC3 | unit | service/container_runner_test.go | non-container service still uses single primary port (no regression) | planned |
| T14 | AC4 | unit | docker/exec_test.go | `buildRunArgs` includes `--network <net>` and `--network-alias <svc>` when set | planned |
| T15 | AC4 | unit | service/*_test.go | project network name derivation is stable/sanitized for a given project dir | planned |
| T16 | AC4 | unit | docker/*_test.go | network create is idempotent (already-exists is not an error) | planned |
| T17 | AC4 | unit | docker/exec_test.go | no `--network` emitted when networking not configured (backward compat) | planned |
| T18 | AC4 | integration | service/container_integration_test.go | azurite + eventhubs on shared network: eventhubs resolves `azurite` by alias | planned |
| T19 | AC4 | unit | service/container_runner_test.go | reused (already-running) container is reconnected to the network with its alias | planned |
| T20 | AC5 | unit | service/container_runner_test.go | `pull_policy: missing` → pull only when image absent | planned |
| T21 | AC5 | unit | service/container_runner_test.go | `pull_policy: never` → pull never called | planned |
| T22 | AC5 | unit | service/container_runner_test.go | `pull_policy: always` / unset → pull attempted (current behavior) | planned |
| T23 | AC5 | unit | service/types_test.go | `pull_policy` invalid value rejected with clear error | planned |
| T24 | AC6 | unit | service/orchestrator_test.go | `uses` builds level ordering; dependent starts after dependency healthy (regression) | planned |
| T25 | AC7 | unit | schema structure | v1.1 schema file declares `volumes` (array), `pull_policy` (enum missing/always/never), and `command` (oneOf string\|array) | planned |
| T26 | AC8 | e2e | manual + scripted | website 3-container topology comes up under `azd app run`; documented in Phase 4 | planned |

## Functionality Inventory (Phase 3 reconciliation)

Enumerated against `git diff origin/main`: every unit of new functionality maps
to a covering test. **Zero gaps.**

| Functionality | Covering test(s) |
|---------------|------------------|
| Volume specs → `-v` args | `docker.TestBuildRunArgs_Volumes` |
| Command tokens after image | `docker.TestBuildRunArgs_CommandAfterImage`, `_NoCommandOmitsTokens` |
| Env emission | `docker.TestBuildRunArgs_Environment` |
| Network/alias args | `docker.TestBuildRunArgs_NetworkAndAliases`, `_NoNetworkWhenUnset` |
| `--pull never` arg | `docker.TestBuildRunArgs_PullNeverEmitsFlag` |
| Multi-port args | `docker.TestBuildRunArgs_MultiPort` |
| Field validators | `docker.TestValidateVolumeSpec`, `TestValidatePullPolicy`, `TestValidateNetworkName`, `TestContainerConfigValidate_RejectsBadFields` |
| Docker stderr classifiers (idempotency) | `docker.TestStderrClassifiers` |
| Command string/array unmarshal | `service.TestService_Command_{StringForm,ArrayForm,InvalidType}` |
| Volumes/pull_policy unmarshal | `service.TestService_Volumes`, `TestService_PullPolicy` |
| `GetCommandArgs` | `service.TestGetCommandArgs` |
| Command tokenizer (+ unclosed quote) | `service.TestParseCommandLine` |
| Network name derivation/sanitize | `service.TestDeriveNetworkName`, `TestSanitizeNetworkComponent` |
| Project-root normalization (subdir bug fix) | `service.TestProjectNetworkDir_*` |
| Volume resolution/classification | `service.TestResolveVolumeSpec_*`, `TestSplitVolumeSource`, `TestIsNamedVolume` |
| Container runtime wiring | `service.TestDetectContainerRuntime_*` |
| **Process array-command whitespace (cr fix)** | `service.TestDetectServiceRuntime_ProcessArrayCommandPreservesWhitespace` |
| Multi-port mapping / fallback | `service.TestBuildContainerPortMappings_*` |
| Pull policy decision | `service.TestShouldPullImage` |
| Schema documents new fields | `service.TestV11SchemaDocumentsContainerFields` |
| Network create/remove/connect/DNS (real Docker) | `service.TestEnsureNetwork_IdempotentAndRemoveTolerant`, `TestContainerNetwork_InterContainerDNS`, `TestContainerReuse_NetworkConnect` |

No `GAP` rows remain.

## Notes

- T18/T26 require Docker; they are the true acceptance signals for the
  networking + DNS feature and the overall goal. They must be runnable locally
  even if CI skips them without a daemon.
- Every new exported function gets at least one direct unit test (repo 80%
  coverage gate).
