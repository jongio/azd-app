---
title: Native Docker Compose-style container config for host: local services
status: draft
issue: https://github.com/jongio/azd-app/issues/546
scope: P1
---

# Native container config for `host: local` services

## Overview

Extend azd-app's **native container path** (services with an `image:`, launched via
individual `docker run -d`) so a project can express a realistic multi-container
local-development topology directly in `azure.yaml` — volumes, a run `command`,
multiple published ports, container-to-container name resolution, and image
pull policy — without falling back to a hand-maintained `docker-compose.yml`.

## Background

`azd app run` starts `host: local` container services by shelling out to
`docker run -d` with a `ContainerConfig` that only carries **name, image, ports,
and environment** (`cli/src/internal/docker/types.go`,
`cli/src/internal/service/container_runner.go`). That is enough for a single
stateless cache or database, but it cannot run the kind of stack most teams
actually use for local dev.

The concrete driver is a real multi-service **website** project, whose local dev
stack (a `compose.dev.yml` orchestrated by mise) has three infra containers:

| Container | What it needs |
|-----------|---------------|
| `postgres:16-alpine` | named volume, **array** `command`, healthcheck |
| `azurite` | named volume, string `command`, **3 ports** (10000/10001/10002) |
| `eventhubs-emulator` | **bind-mount** Config.json, env, `depends_on: azurite (service_healthy)`, **inter-container DNS** (`BLOB_SERVER: azurite`), `pull_policy: missing` |

None of these run under azd-app today. The goal is to make azd-app a
first-class replacement for that compose file, so the website (and any similar
project) can adopt `azd app run` for local dev.

## Goals

1. Support `volumes:` on container services — named volumes and bind mounts,
   with relative bind paths resolved against the project directory.
2. Pass a container `command:` (string **or** array) through to `docker run`.
3. Publish **all** ports listed for a container service, not just the primary.
4. Give container services a shared per-project Docker network so one container
   can reach another by service name (container→container DNS).
5. Support `pull_policy: missing | always | never`.
6. Keep the existing `uses`-based, health-gated startup ordering working so it
   expresses `depends_on: { condition: service_healthy }`.
7. Document all of the above in the v1.1 JSON schema and the CLI + web docs.

## Non-Goals

- **No `docker compose` delegation.** The native `docker run` path is retained so
  the port manager, health checks, log streaming, dashboard, and `azd app add`
  continue to work unchanged.
- **No local image build.** `azd app run` does not `docker build` a service's
  Dockerfile — that would reimplement Docker Compose. Deploy images are built by
  `azd deploy`; for local dev a service runs from source as a process (see the
  local process override in Design §6).
- **No new `depends_on` field.** The orchestrator already builds a dependency
  graph from `uses`, topologically sorts it, and waits for each level to become
  healthy before starting the next (`OrchestrateServices` /
  `waitForServiceHealthy`). `uses` already *is* `depends_on: service_healthy`.
- **No honoring compose `container_name`.** azd-app keeps its `azd-<service>`
  container naming; inter-container DNS is provided by a network alias instead.
- **No new well-known service** (Event Hubs is defined manually in `azure.yaml`).
- The website's own `azure.yaml` and docs are onboarded in a **separate**
  follow-up in that project's repo; this spec covers only azd-app.

## Design

All new fields live on the existing service object and only affect **container
services** (services with an `image:`). Native process/HTTP services are
untouched.

### 1. Volumes

Add `Volumes []string` to `Service` / `serviceRaw` / `ServiceRuntime` and
`docker.ContainerConfig`. `buildRunArgs` emits one `-v <spec>` per entry.

Classification of each `volumes:` entry:

- **Named volume** — `name:/container/path` where the left side is a bare
  volume name (`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`). Passed through unchanged;
  Docker auto-creates it.
- **Bind mount** — left side is a path (`.`, `..`, `/`, `~`, or a Windows drive
  like `C:\`). The host side is resolved to an **absolute** path relative to the
  project directory before being passed to Docker.
- **Anonymous volume** — a single `/container/path` with no `:` host side.

Each entry is passed to `docker run -v` as a discrete argv element (never through
a shell) and validated to reject empty, oversized, or control-character specs, so
it stays injection-safe under gosec G204.

### 2. Command

`Service.Command` already exists as a string. Extend YAML unmarshalling so
`command:` also accepts a **sequence** (`["postgres", "-c", "..."]`). For
container services the tokens are appended after the image in `docker run`.
Array form is stored verbatim; string form is tokenized with shell-style
splitting consistent with how the process path already handles `command`.

### 3. Multi-port containers

`buildContainerPortMappings` currently maps only `runtime.Port`. For container
services it will map **every** entry in the service `ports` list, reusing the
existing `ParsePortMappings` logic (host:container, bind IP, `/udp`). Non-
container services keep their single-primary-port behavior.

### 4. Shared network + DNS

Container services are attached to a per-project user-defined bridge network so
they can resolve each other by service name (compose-equivalent).

- **Name**: `azd-app-<sanitized-basename>-<short-hash-of-abs-project-dir>`,
  derived purely from the project directory so it is identical across the
  `run`, restart, and dashboard code paths without threading a name parameter.
- **Creation**: idempotent `EnsureNetwork` (`docker network create`, tolerating
  "already exists") performed by each container as it starts. Safe under
  parallel level startup because the "already exists" error is treated as
  success — no serialization needed.
- **Attachment**: each container runs with `--network <net>` and
  `--network-alias <serviceName>`, so `BLOB_SERVER: azurite` resolves to the
  azurite container regardless of its `azd-<name>` container name. A **reused**
  (already-running) container is (idempotently) connected to the network with
  the same alias so DNS works after a fast restart.
- **Lifecycle**: azd-app container services are **persistent** — `azd app stop`
  and Ctrl+C run a graceful shutdown that leaves running containers in place
  (`shutdownAllServices` stops only OS processes; containers are reused on the
  next `azd app run`). The network therefore **persists with its containers** and
  is reused across runs; it is *not* torn down while containers remain attached
  (removing an in-use network would fail). Containers are removed on a failed
  start or a forced restart, and a `RemoveNetwork` client method is provided for
  a future `azd app clean`.
- **Backward compatibility**: single-container projects still publish their
  ports to the host and are health-checked from the host via those ports; the
  network is additive and requires no config.

### 5. Pull policy

Add `pull_policy` (`missing` | `always` | `never`) to gate the existing
`client.Pull()` call:

- `missing` (recommended for pinned emulator images) — pull only if the image
  is not present locally.
- `always` — always pull; the container fails to start if the pull fails.
- `never` — never pull; fail only if the image is absent at `docker run` time.
- **Default (unset)** preserves today's behavior (attempt pull, tolerate
  failure, continue with cached image).

### Health-gated ordering (existing, unchanged)

`eventhubs uses: [azurite]` → the orchestrator starts azurite, waits for it to
become healthy, then starts eventhubs. This is the existing behavior and is
covered by a regression test, not new code.

### 6. Local process override for deploy services

A single `azure.yaml` often describes services for **deployment** as container
images (top-level `image:` for a prebuilt image, or `docker.*` to build one for
`azd deploy`). For **local dev**, those same services usually run from source as
a process (`npm run dev`, `uvicorn`, …). To let one file serve both, `azd app
run` treats an explicit local **`command`** (or **`type: process`**) as an opt-in
to run the service as a **process**, using `docker.*`/`image` only for
`azd deploy`:

- Top-level **`image:`** (a prebuilt emulator, e.g. `postgres`, `azurite`) is
  always a container; its `command` is a **container command override** (F2).
- A service whose container-ness comes from **`docker.*`** (a build-and-deploy
  service) runs as a **process** when it has a local `command`/`type: process`.
- A `docker.*` service **without** a local command keeps today's container
  (pull) behavior — this is backward compatible.

This is a routing rule only (`Service.RunsAsLocalProcess()`), not a local image
build — building the Dockerfile locally would reimplement Docker Compose and is
explicitly out of scope.

### 7. `azd app test` for explicit-command services

`azd app test` discovers testable services by **language** (js/ts, python, go,
.NET) and skips the rest, so a service that runs as a container/emulator
(`language: docker`, or no language) can't be tested even when the project has a
suite for it. To let such a service opt in, an explicit **`test:`** block with a
per-type `command` now makes the service testable **regardless of its
language**:

- A service with an explicit `test.<type>.command` is **testable** even when its
  language isn't a recognized test language (validation bypasses the language
  gate). The runner is chosen from the configured **`framework`** (e.g. `vitest`
  → Node runner), defaulting to the Node runner, which runs the command and
  reports pass/fail from the process **exit code**.
- `azd app test` with no `--type` (i.e. `all`) **runs each explicitly-configured
  type** (unit, then integration, then e2e) and aggregates, instead of falling
  back to the framework's default command — so the declared commands are always
  the ones executed. `--type unit|integration|e2e` runs just that command.
- Services **without** an explicit `test:` block are unchanged: language
  auto-detection still applies, and container/emulator services without a suite
  are still skipped.

### 8. Deduped installs for shared `project` directories

A monorepo commonly points several services at one directory (e.g. `project: .`
on each, backed by a single root `package.json`). The deps step collected one
install task **per service**, so the same directory was installed — and rendered
as its own progress bar — once per service (N identical `website (npm)` bars).
Project collection (`detectProjectsFromAzureYaml`) now **dedupes by resolved
project directory**, so a shared directory is collected, installed, and shown
**once**. `azd app deps --dry-run` and `azd app run`'s install phase are
consistent.

To keep it clear that the single install covers **all** the services (not just
the directory), the install line is **labeled with the service names** that
share the directory — e.g. `web, ingest, +6 more (npm)` (sorted, truncated for
readability). A directory used by a single service keeps its default
directory-name label (no change).

### 9. Accurate service-log levels

`azd app run`/`azd app logs` classified each captured output line with
`inferLogLevel`, which did a loose **substring** match on the raw text and
ignored the stream. That mislabeled lines two ways: a JSON log like
`{"level":"info","role":"trace-worker"}` matched the substring `trace` and was
dropped to `DEBUG`, while an unstructured stderr diagnostic (`​.env not found`)
with no keyword defaulted to `INFO`. The classifier now:

- **honors structured logs** — a JSON line with a `level`/`severity` field uses
  that level (the emitter's own classification wins);
- uses **word-boundary** keyword matching, so identifiers (`errorReporter`,
  `trace_worker`) no longer misfire;
- is **stream-aware** — an unclassified **stderr** line surfaces as `WARN`
  (where programs write diagnostics) instead of `INFO`.

## Risks / trade-offs

- **Network lifecycle**: the network must be created idempotently (parallel
  startup) and cleaned up best-effort (a leftover empty network is harmless and
  reused next run). Container **reuse** (an already-running container) must not
  be broken by network changes — reused containers are assumed already attached.
- **Arg injection**: volumes and command introduce user-controlled `docker run`
  arguments. Each is validated and passed as discrete `exec` argv elements
  (never a shell string), consistent with the existing G204-scoped exec pattern.
- **Windows paths**: bind-mount host paths may be `C:\...`; resolution and the
  named-vs-bind heuristic must handle drive letters without misclassifying them
  as `name:path`.

## Acceptance criteria

- **AC1** — `volumes:` supports named volumes and bind mounts; relative bind
  paths resolve against the project dir; entries are injection-safe.
- **AC2** — container `command:` accepts string and array forms; tokens reach
  `docker run`.
- **AC3** — every `ports:` entry is published for a container service.
- **AC4** — container services share a per-project network (created
  idempotently); a container can reach another by service name; a reused
  container is reconnected with its alias; single-container projects still work;
  the network is safely reused across runs (persists with its containers).
- **AC5** — `pull_policy: missing|always|never` gates image pulls; unset
  preserves current behavior.
- **AC6** — `uses` still health-gates startup ordering (regression).
- **AC7** — v1.1 JSON schema + CLI/web docs document volumes, array command,
  multi-port, pull_policy, and container networking.
- **AC8** — the website's 3-container topology (postgres + azurite +
  eventhubs) starts under `azd app run` (end-to-end validation).
- **AC9** — a service with `docker.*`/`image` **and** an explicit local
  `command`/`type: process` runs as a **process** under `azd app run` (its
  `docker.*` stays deploy-only); a `docker.*` service **without** a local command
  is unchanged (still a container). No local image build is performed.
- **AC10** — a service whose `language` isn't a recognized test language (e.g.
  `docker`) but which declares an explicit `test.<type>.command` is testable
  under `azd app test`; the runner is selected from `framework`; `--type all`
  runs each configured type and aggregates; services without a `test:` block are
  unaffected (auto-detection unchanged).
- **AC11** — when several services share one resolved `project` directory, deps
  collection yields a single install task for it (one progress bar, one install),
  and that line is **labeled with the service names** it covers (sorted,
  truncated, e.g. `web, ingest, +6 more (npm)`); a single-service directory keeps
  its directory-name label; `azd app deps --dry-run` and the `azd app run` install
  phase agree; distinct directories and package managers remain separate.
- **AC12** — service-log level classification honors an explicit `level`/
  `severity` on a structured (JSON) line; keyword detection is whole-word (no
  identifier misfires); an unclassified stderr line is `WARN`, an unclassified
  stdout line is `INFO`.

<!-- Pipeline tracking (auto-managed, not part of product spec) -->
## Pipeline Status
Phase: SHIPPING

### Live end-to-end validation (a real multi-service website)
Built locally via `mage build` (v0.19.5+dev) and ran a real multi-service
website's dev stack under `azd app run` (dev config mirroring `compose.dev.yml`):
- postgres (array command + named volume), azurite (**3 published ports** +
  string command + named volume), eventhubs (bind-mount `Config.json` +
  `pull_policy: missing` + `uses: [azurite]`) all came up; the **Event Hubs
  emulator resolved `azurite` by name over the project network** and reported
  "Successfully Up!" (its metadata backend).
- A `uses: [postgres]` **task** applied the full DB schema; **web (Vite)** served
  **HTTP 200** on :3000.
- Containers + network **persisted** across stop, as designed.
- No azd-app defects surfaced in the e2e (the two fixes shipped here were found
  in review: subdir network-name derivation, and array-command on process
  services).
- Note (pre-existing, out of scope): `azd app run --detach` exits silently on
  Windows (empty run.log) — unrelated to this change (`run_detach.go`).

### Deferred (follow-up)
- **Container-exec health checks** (`healthcheck.test: ["CMD-SHELL", ...]`) that run
  *inside* a container via `docker exec` are not yet honored — container health uses
  host-side TCP/HTTP checks against the published port. This is adequate for the
  emulators in scope (they open their ports when ready), and `uses` health-gating
  works on that signal. A dedicated follow-up can add a `container-exec` health type.
- **Command tokenizer consolidation** — `parseCommandLine` is a third copy of a
  quote-aware splitter (the `testing` package has two). Consolidating into a shared
  `internal` util is a low-risk cleanup deferred to avoid touching unrelated
  test-infra code in this PR.

### Security note (accepted trust model)
Volume/command/network/pull_policy values flow to `docker run` as **discrete argv**
(never a shell), and the image positional is validated to not start with `-`, so
shell- and flag-injection are neutralized (security review: no findings). Bind mounts
to **absolute** host paths are intentionally allowed — the same trust model as
`docker compose` for a developer-authored `azure.yaml`. Relative binds that escape the
project directory are rejected.
