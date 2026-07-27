---
issue: https://github.com/jongio/azd-app/issues/555
author: "@jongio"
status: shipped
---

# Reliable unattended runs: detach, non-interactive prompts, explicit test commands

## Problem

Three separate reports describe the same underlying theme: `azd app` works when a
human is watching an interactive terminal, and misbehaves the moment nobody is.

**1. `azd app run --detach` dies immediately on Windows ([#555](https://github.com/jongio/azd-app/issues/555)).**
The command prints a PID and a log path, then the child is gone within a second.
`run.log` is created but empty. `azd app status` reports "App is not running" and
`azd app stop` has nothing to stop, yet service processes started before the death
keep running and self-restart, holding ports. The user is left with orphans and no
handle to kill them.

Investigation found this is not one bug but five, each independently capable of
producing part of the report. The orphaned services are downstream of the manager
dying, so the fixes below target why it dies and why it leaves no handle behind.

**2. Port-conflict prompt crashes under non-interactive stdin ([#556](https://github.com/jongio/azd-app/issues/556)).**
When a port is busy, `azd app run` prints the "what would you like to do" menu and
then fails with `failed to read user input: EOF`. The code already has a documented
non-interactive path (print a message, auto-kill the conflicting process), but the
guard that selects it does not fire, so the run aborts instead of degrading.

**3. `azd app test` appears to ignore explicit commands ([#557](https://github.com/jongio/azd-app/issues/557)).**
A service configured with both `framework: vitest` and an explicit
`test.e2e.command` is reported as `web: vitest detected` and
`web (vitest) - Running...`, followed by `0 passed, 0 total` and
`All tests passed!`. Every signal the CLI emits says the framework runner ran and
found nothing, so the explicit command looks ignored.

### Root causes

**#555a: the detached child is still inside the parent's Job Object.**
`detachSysProcAttr` sets `DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP`. Neither flag
removes the child from the parent's job. When the launching process (an azd host, a
terminal wrapper, VS Code) owns a job with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` and
that job handle closes, Windows terminates the "detached" child. Verified with a
purpose-built probe:

| Spawn conditions | Child started | Child survived |
|---|---|---|
| No job, `DETACHED_PROCESS` | yes | yes |
| Kill-on-close job, `DETACHED_PROCESS` only (current code) | yes | **no** |
| Kill-on-close job plus `CREATE_BREAKAWAY_FROM_JOB` | yes | **yes** |

The child dies after `CreateProcess` succeeds but before it writes anything, which
is exactly why `run.log` exists and is empty.

`CREATE_BREAKAWAY_FROM_JOB` cannot be applied unconditionally: if the job lacks
`JOB_OBJECT_LIMIT_BREAKAWAY_OK`, `CreateProcess` fails outright with
`ERROR_ACCESS_DENIED`. The probe confirmed this too. So the spawn needs an ordered
set of attempts, not a single flag change.

**#555b: run state is written far too late.**
`startDetachedRun` returns right after `cmd.Start()` without persisting anything.
The child writes `run.json` only from `monitorServicesUntilShutdown`, after every
service has started and after `waitForDashboardURL` waits up to 10 seconds. During
that window (or forever, if the child dies) `status` and `stop` have no PID to work
with.

**#555c: the detached child redundantly reloads the azd environment.**
`PersistentPreRunE` in `main.go` calls `env.LoadAzdEnvironment` whenever `-e` is
set. That helper shells out to `azd env get-values`, and retries without
`--output json` on failure, so it costs up to two subprocess round-trips. It runs
before any command output exists, so the process is completely silent while it
blocks. The detached child inherits `-e` (only `--detach` is stripped), so it
repeats this work against an azd host that is already exiting. Measured on a
trivial single-service project:

| Invocation | First bytes in `run.log` |
|---|---|
| `run --detach` (no `-e`) | ~320 ms |
| `run --detach -e <env>` (before) | ~1590 ms |
| `run --detach -e <env>` (after) | ~328 ms |

The reload is pure waste: cobra runs `PersistentPreRunE` before `RunE`, and
`maybeStartDetachedRun` is called from `RunE`, so the parent has already resolved
every value and exported it into its own environment. `cmd.Env` is built from
`os.Environ()`, so the child inherits all of them. This is the root cause of the
0-byte `run.log` in the report: a job kill alone always left 11-15 KB of output in
testing, never an empty file.

**#555d: parent and child key run state by different directories.**
The parent identifies the project by `filepath.Dir(azureYamlPath)`, while the
child's `writeRunState` was reached with `os.Getwd()`. Running from a
subdirectory produced two different state files, so the child never overwrote the
parent's seed record and `stop`, which uses the azure.yaml directory, looked in a
third place. This also broke the dashboard port and token files the same way.

**#555e: `stop` deleted run state even when it failed.**
`runStopApp` removed the state file from a `defer` that ran unconditionally. A
`stop` that could not reach the dashboard discarded the only PID record, leaving a
live manager with no handle at all. This directly defeats the point of writing the
PID early.

**#556: EOF is treated as a hard error.**
`handlePortConflict` guards the prompt with `isStdinInteractive()`, which only tests
`os.ModeCharDevice`. The azd host hands down a console-like stdin with no reader, so
the guard passes, the menu prints, and `reader.ReadString('\n')` returns `io.EOF`,
which is converted into a fatal error. The existing non-interactive fallback is
never reached.

**#557: the message describes detection, not execution.**
The execution path is already correct on `main`: `ValidateService` short-circuits on
`HasExplicitCommand()` and `runExplicitTypes` runs the configured command. Verified
empirically with marker files for both recognized (`js`) and unrecognized (`docker`)
languages. What is wrong is the reporting: `ValidateService` has no idea which test
type was requested, so it labels the service with the configured `framework`, and
`displayValidationSummary` prints `"%s: %s detected"`. A user running
`--type e2e` against an explicit e2e command is told `vitest detected`. The
`0 passed, 0 total` line follows because a custom command emits no framework-shaped
counts.

**#557b: validation claimed types that execution silently skips.**
`HasExplicitCommand()` is type-agnostic. A service whose language has no framework
runner (for example `docker`) and which configures only `test.unit.command` was
reported as testable for `--type e2e`. `executeServiceTests` already refuses that
combination and returns `Success: true` without running anything, so the user was
shown a passing e2e run that never happened.

## Goals

- `azd app run --detach` on Windows survives the launching process exiting, including
  when that process owns a kill-on-close Job Object.
- A failed breakaway degrades to the previous behaviour instead of failing the run.
- `azd app status` and `azd app stop` work the instant `--detach` returns.
- The detached child starts producing output immediately instead of blocking on work
  its parent already did.
- One project identity is used for run state, dashboard port, and token files, no
  matter which directory the command is run from.
- Port-conflict handling degrades to the documented non-interactive behaviour when
  stdin yields EOF, instead of aborting the run.
- `azd app test` reports the command it will actually run for the requested type, and
  never reports a type it will silently skip.

## Non-Goals

- Reworking `--detach` into a service manager, supervisor, or daemon.
- Parsing arbitrary custom test command output into pass/fail counts. Exit code
  remains the contract for custom commands.
- Changing which process wins a port conflict, or the auto-kill policy itself.
- Unix orphan reaping via process groups. Unix already gets `setsid` isolation and
  the reported harm is Windows-specific.
- Replacing `isStdinInteractive` with a full TTY-detection dependency.
- **OS-enforced service teardown via a Job Object.** Deferred to a follow-up issue.
  An architecture review found that attaching a kill-on-close job from the shared
  `service.StartService` would kill services started by the short-lived
  `azd app start` and tie MCP-started services to the MCP host's lifetime, that
  post-start attachment races with wrapper processes such as npm and mise which
  spawn grandchildren before the assignment lands, and that it would not cover
  Aspire, containers, or Unix. Doing it correctly requires an explicitly owned
  group threaded through orchestration plus suspended process creation, which is a
  new subsystem in the service start path used by every command. The reported
  orphan symptom is a consequence of the manager dying, which the fixes above
  address directly.

## Solution

### 1. Ordered detach spawn attempts (#555a)

Replace the single `detachSysProcAttr()` platform hook with an ordered list of
attempts plus a predicate that identifies a breakaway rejection.

- Windows: `[breakaway | detached | new group, detached | new group]`, and
  `isBreakawayRejected` returns true for `ERROR_ACCESS_DENIED`.
- Unix: a single `{Setsid: true}` attempt, and `isBreakawayRejected` always false.

`startDetachedRun` iterates the attempts. Because a failed `exec.Cmd` cannot be
restarted, each attempt constructs a fresh `exec.Cmd`. Retry happens only when the
platform predicate says the failure was a breakaway rejection, so genuine spawn
errors still surface immediately.

### 2. Persist run state at spawn time (#555b)

Immediately after a successful `cmd.Start()`, write `run.json` with the child's PID
and start time. The child later overwrites the same file with the dashboard URL and
service list. `cmd.Process.Pid` is the child's own `os.Getpid()`, so the two writes
agree on identity. A failure to persist is logged and does not fail the detach: the
run is already alive, and returning an error would strand it.

### 3. Skip the redundant environment reload in the detached child (#555c)

Export `commands.IsDetachedChild()`, which reports whether `AZD_APP_DETACHED_RUN`
marks this process as the spawned background run, and consult it in
`PersistentPreRunE`:

```go
if extCtx.Environment != "" && !commands.IsDetachedChild() {
    ...LoadAzdEnvironment...
}
```

Skipping the reload is preferred over stripping `-e` from the child's arguments,
because `extCtx.Environment` has other consumers and clearing it would change more
than the reload.

### 3b. One project identity (#555d)

`runAzdMode` passed `os.Getwd()` down the orchestration chain as the project key
while `stop`, `status`, and the new seed write all used the azure.yaml directory.
The two parameters that had to be equal (`cwd` and `azureYamlDir`) are collapsed
into a single `projectDir`, which is the azure.yaml directory. Removing the second
parameter removes the ability for them to diverge again.

### 3c. `stop` clears state based on manager liveness (#555e)

The unconditional `defer runstate.Remove(projectDir)` is replaced by two paths.
On success the state is cleared as before. On failure the state is kept when the
recorded PID is still alive, so `status` and a retry can still reach the manager,
and cleared when that PID is gone, so a crashed manager does not leave a phantom
run that nothing would ever clean up. Keeping state unconditionally on failure
would trade one bug for the other.

### 3d. The detached marker is consumed, not just read

`IsDetachedChild()` reads `AZD_APP_DETACHED` once, then removes it from the
process environment and caches the answer.

Services and lifecycle hooks build their environment from `os.Environ()`, so a
marker left in place would propagate to every child. Any nested `azd app run`
would then classify itself as a detached child and skip `LoadAzdEnvironment`,
silently running against the default azd environment instead of the one the user
selected with `-e`. Caching matters because callers run both before the unset
(`main.go`, in `PersistentPreRunE`) and after it (`maybeStartDetachedRun`, in
`RunE`); without it the child would fail to recognise itself and would spawn a
second detached run.

The marker is only removed when it actually matched, so an unrelated value the
user set is left untouched.

This does not defend against a user who exports `AZD_APP_DETACHED=1` in their
shell profile. Anyone able to set that variable can already set the azd values
directly, so this is a footgun guard rather than a security boundary.

### 4. EOF-tolerant port-conflict prompt (#556)

Extract `readPromptLine`, which decides what a read result means:

1. No error: the trimmed line is the answer.
2. A non-EOF error: return the error and **discard** any partial data. A truncated
   read from a broken stream is not consent to pick a destructive menu entry.
3. `io.EOF` with non-empty partial data: honour it, because
   `bufio.Reader.ReadString` returns both the buffered data and `io.EOF` when the
   user's final answer had no trailing newline.
4. `io.EOF` with nothing to return: report `errNoInput`, and let
   `handlePortConflict` fall through to the existing
   `printNonInteractiveKillMessage` plus `ActionKill` path that already serves
   detectably non-interactive stdin.

`promptUpdateAzureYaml` uses the same helper, since it had the identical bug.

### 5. Report the effective test command (#557)

Thread the requested test type into validation via a new
`ValidateServiceForType(service, testType)`; `ValidateService` delegates with
`TestTypeAll` so existing callers are unaffected. When the requested type has an
explicit command, record it on a new `ServiceValidation.Command` field.

`displayValidationSummary` prints the command when present:

```
web: custom command (mise exec -- npx tsx scripts/smoke-pat.ts --fresh)
```

instead of `web: vitest detected`. `Framework` keeps its current meaning so
`SaveTestConfigToAzureYaml` and other consumers are untouched.

### 5b. Validation agrees with execution (#557b)

When the requested type has no explicit command and the service's language has no
framework runner, `ValidateServiceForType` now reports the service as skipped with
a reason naming the missing type, mirroring the guard in `executeServiceTests`. A
test asserts the two agree for every test type so they cannot drift.

## Acceptance criteria

| # | Criterion |
|---|---|
| AC1 | On Windows the detached spawn requests `CREATE_BREAKAWAY_FROM_JOB` first, so the child survives a parent whose job is kill-on-close |
| AC2 | A breakaway rejection (`ERROR_ACCESS_DENIED`) retries with the previous flags on a fresh `exec.Cmd`; other spawn errors do not retry |
| AC3 | `run.json` carries the detached child's PID before `run --detach` returns, and a persist failure does not fail the run |
| AC4 | The detached child does not reload the azd environment its parent already resolved, so `run.log` starts filling at the same speed with and without `-e` |
| AC5 | Run state, dashboard port, and token files use the azure.yaml directory as the single project identity, regardless of the working directory |
| AC6 | A failed `azd app stop` leaves the run state intact while the manager is alive, and clears it once the manager is gone |
| AC7 | A port-conflict prompt that reads EOF with no input falls back to the non-interactive auto-kill path instead of erroring |
| AC8 | A final answer with no trailing newline is honoured rather than discarded |
| AC9 | A non-EOF read error still fails and its partial data is discarded, rather than silently selecting a menu entry |
| AC10 | `azd app test --type <t>` reports the explicit `test.<t>.command` when configured, and the framework otherwise |
| AC11 | A service that cannot run the requested type is reported as skipped rather than passing |
| AC12 | The detached marker is consumed at startup so services, hooks, and any nested `azd app run` do not inherit it |
| AC13 | Positional arguments after a `--` terminator reach the detached child verbatim |

## Trade-offs and risks

- **Two-attempt spawn adds a failure mode to reason about.** Mitigated by retrying
  only on the specific breakaway rejection, and by keeping the fallback identical to
  today's behaviour, so the worst case is the status quo.
- **Early run-state write can be overwritten by the child.** The seed is skipped
  when the file already holds richer state for the same PID, so a lost race
  cannot reduce a published dashboard URL back to a bare PID. Stale state left by
  a previous run under a different PID is still replaced.
- **The state file is written in place rather than atomically replaced.** A
  temp-file-plus-rename was tried and rejected: Windows refuses to rename over a
  file a reader currently has open, which converts a rare short read window into
  a writer that fails outright. The read window is pre-existing, unchanged in
  kind by this work, and costs at most one retryable `status` invocation.
- **Skipping the environment reload assumes the child inherits the parent's values.**
  Guaranteed by ordering: `PersistentPreRunE` runs before `RunE`, the reload calls
  `os.Setenv`, and the child's environment is built from `os.Environ()`.
- **Switching the project key from the working directory to the azure.yaml
  directory changes where state lives** for anyone who ran from a subdirectory.
  Those runs were already broken, since `stop` looked in the azure.yaml directory
  all along; this makes every command agree.
- **`ServiceValidation` gains a field.** Additive, so no consumer breaks.
- **Marking unrunnable type/service pairs as skipped changes summary output.** It
  replaces a false "passed" with an accurate skip, which is the point.
