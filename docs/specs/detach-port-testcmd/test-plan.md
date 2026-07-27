# Test Plan: Reliable unattended runs

Issues: [#555](https://github.com/jongio/azd-app/issues/555),
[#556](https://github.com/jongio/azd-app/issues/556),
[#557](https://github.com/jongio/azd-app/issues/557)
Spec: ./spec.md

## Status: AUTOMATED

Framework: Go `testing` plus `stretchr/testify`, table-driven with `t.Run`.

Platform-specific spawn behaviour is tested through the platform hooks
(`detachSpawnAttempts`, `isBreakawayRejected`) so the shared retry logic is
asserted on every OS, with `//go:build` files pinning the per-platform flag
values. The retry loop is exercised through `spawnWithAttempts`, which takes the
starter as a parameter, so rejection and fallback are tested without depending on
a real Job Object.

Live kill-on-close job behaviour cannot be simulated in a unit test. It was
verified with a purpose-built end-to-end harness that creates a real job, spawns
the built binary under it, closes the handle, and inspects survival plus
`run.log`. Results are recorded in the spec.

## Coverage

| # | AC | Test | Location |
|---|----|------|----------|
| T1 | AC1 | `TestDetachSpawnAttempts` | `run_detach_windows_test.go`: attempt 1 sets `CREATE_BREAKAWAY_FROM_JOB` with detached plus new group |
| T2 | AC2 | `TestDetachSpawnAttempts` | `run_detach_windows_test.go`: attempt 2 equals the previous flags exactly |
| T3 | AC1 | `TestDetachSpawnAttempts` | `run_detach_unix_test.go`: exactly one attempt, `Setsid` set |
| T4 | AC2 | `TestIsBreakawayRejected` | both platform test files, including `fs.PathError` unwrapping and nil |
| T5 | AC2 | `TestSpawnWithAttemptsFallsBackWhenBreakawayRejected` | `run_detach_windows_test.go` |
| T6 | AC2 | `TestSpawnWithAttempts` | `run_detach_test.go`: non-breakaway failure returns after one attempt |
| T7 | AC2 | `TestSpawnWithAttempts` | `run_detach_test.go`: each attempt receives a distinct `exec.Cmd` |
| T8 | AC2 | `TestSpawnWithAttemptsReportsLastErrorWhenAllRejected` | `run_detach_windows_test.go` |
| T9 | AC3 | `TestMaybeStartDetachedRunSpawns` | `run_detach_test.go`: run state readable before the parent returns |
| T10 | AC3 | `TestRecordDetachedRunState` | `run_detach_test.go`: PID and start time persisted |
| T10b | AC3 | `TestRecordDetachedRunStateDoesNotClobberChildState` | `run_detach_test.go`: a late seed leaves the child's richer state alone |
| T10c | AC3 | `TestRecordDetachedRunStateReplacesStaleStateFromAnotherPID` | `run_detach_test.go`: a previous run's state is replaced |
| T11 | AC3 | `TestRecordDetachedRunStateSurvivesWriteFailure` | `run_detach_test.go`: unwritable state dir does not fail the detach |
| T12 | AC4 | `TestIsDetachedChild` | `run_detach_test.go`: only the exact marker value counts as the child |
| T13 | AC4 | end-to-end measurement | `run.log` first-byte latency with `-e`, recorded in the spec |
| T14 | AC5 | compile-time | the duplicate directory parameter is removed, so the two cannot diverge |
| T15 | AC6 | `TestClearRunStateIfManagerDeadKeepsStateForLiveProcess` | `stop_test.go`: live PID keeps its state |
| T15b | AC6 | `TestClearRunStateIfManagerDeadRemovesStateForDeadProcess` | `stop_test.go`: crashed manager's stale state is cleared |
| T15c | AC6 | `TestClearRunStateIfManagerDeadIgnoresMissingState` | `stop_test.go` |
| T16 | AC7 | `TestReadPromptLine` | `prompts_test.go`: empty and whitespace-only streams report `errNoInput` |
| T17 | AC8 | `TestReadPromptLine` | `prompts_test.go`: answer without a trailing newline is honoured |
| T18 | AC9 | `TestReadPromptLineDiscardsPartialDataOnNonEOFError` | `prompts_test.go`: partial data discarded, `errNoInput` not reported |
| T19 | AC7 | `TestParsePortConflictChoice` | `prompts_test.go`: every menu choice maps correctly, unknown input cancels |
| T20 | AC7 | `TestHandlePortConflict_ForceMode` and siblings | `prompts_test.go`: pre-existing force and always-kill paths unchanged |
| T21 | AC10 | `TestValidateServiceForType_ReportsExplicitCommandForRequestedType` | `explicit_config_test.go` |
| T22 | AC10 | `TestValidateServiceForType_AllListsEveryConfiguredCommand` | `explicit_config_test.go` |
| T23 | AC10 | `TestValidateServiceForType_AllWithOneCommandOmitsPrefix` | `explicit_config_test.go` |
| T24 | AC10 | `TestValidateServiceDelegatesToAllTypes` | `explicit_config_test.go` |
| T25 | AC10 | `TestValidateServiceForType_FrameworkDrivenReportsNoCommand` | `explicit_config_test.go` |
| T26 | AC11 | `TestValidateServiceForType_UnconfiguredTypeOnUnrecognizedLanguage_Skipped` | `explicit_config_test.go` |
| T27 | AC11 | `TestValidateServiceForType_ConfiguredTypeOnUnrecognizedLanguage_Testable` | `explicit_config_test.go` |
| T28 | AC11 | `TestValidateServiceForType_UnconfiguredTypeOnRecognizedLanguage_StillTestable` | `explicit_config_test.go` |
| T29 | AC11 | `TestValidateServiceForTypeMatchesExecutionGuard` | `explicit_config_test.go`: validation and execution agree for every test type |

| T30 | AC12 | `TestIsDetachedChildConsumesMarker` | `run_detach_test.go`: marker removed from the environment, answer still cached |
| T31 | AC12 | `TestIsDetachedChildLeavesForeignValueIntact` | `run_detach_test.go`: a non-matching value is not touched |
| T32 | AC13 | `TestStripDetachFlag/keeps_positional_args_after_the_terminator_verbatim` | `run_detach_test.go` |
| T33 | AC13 | `TestStripDetachFlag/keeps_a_lone_terminator` | `run_detach_test.go` |
| T34 | AC6 | `TestRunStopAppClearsStateForDeadManager` | `stop_extra_test.go`: end-to-end, pins the up-front cleanup call site |
| T35 | AC7 | `TestHandlePortConflict_InteractiveStdinYieldingEOF_DoesNotError` | `prompts_test.go`: end-to-end reproduction of #556 through the real entry point |
| T36 | AC7 | `TestHandlePortConflict_NonInteractiveStdin_AutoKills` | `prompts_test.go`: pre-existing guard still short-circuits |
| T37 | AC10 | `TestDisplayValidationSummary_ExplicitCommandReportedOverFramework` | `test_test.go`: the user-visible #557 output |
| T38 | AC10 | `TestDisplayValidationSummary_FrameworkReportedWhenNoCommand` | `test_test.go` |
| T39 | AC10 | `TestDisplayValidationSummary_ReportsSkippedServices` | `test_test.go` |
| T40 | AC10 | `TestDisplayValidationSummary_SilentInJSONMode` | `test_test.go`: JSON contract stays machine-only |

## Notes

T14 is structural rather than behavioural: the duplicate-parameter removal is
enforced by the compiler and is covered indirectly by the existing `commands`
suite, which continues to pass.

## Results

Full suite: 39 packages, 0 failures. `mage lint`: 0 issues. `gofmt`/`go vet`: clean.
Cross-compiles clean for linux/amd64, darwin/arm64 and windows/amd64.

Every test above that guards a fix was mutation-tested: the fix was temporarily
reverted and the test confirmed to fail, then restored. This was done for the
#556 EOF path, the #557 reporting path, the `stop` cleanup call site, and the
run-state seed guard, so none of them are tautological.

## Determinism

The suite was re-run with `-count=2` to prove the new tests are order-independent
and leave no global state behind. That surfaced two pre-existing isolation
defects, neither introduced here and neither visible to CI (CI runs without
`-count`, so every test executes once):

1. `cmd/app/commands`: `NewStatusCommand` binds `--watch`/`--interval`/`--service`
   to package-level vars, so `TestRunStatusWatchIntervalTooSmall` left
   `statusWatch=true, statusInterval=100ms` behind and the second pass of
   `TestRunStatusNotRunning` failed with `--interval must be at least 1s`.
   Fixed here with a `restoreStatusFlags` helper, because the new tests added by
   this change share that test binary and would otherwise inherit the flakiness.
2. `internal/service`: the log-manager singleton is never reset between runs, so
   `TestLogManagerGetAllBuffers`, `TestLogManagerGetAllLogs` and
   `TestLogManagerGetServiceNames` fail on the second pass. That package is not
   touched by this change and the defect is left as-is to keep the diff scoped.

After fix 1, `cmd/app/commands` passes at `-count=2`.
