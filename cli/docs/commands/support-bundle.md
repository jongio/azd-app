# azd app support-bundle

Collect local diagnostics for support.

## Synopsis

```
azd app support-bundle [flags]
```

## Description

Collect sanitized project, service, health, and log diagnostics into a local folder so you can attach them to an issue report.

Everything the bundle writes is redacted first. `azure.yaml` is copied through a redactor, and the service, health, and log snapshots are serialized through a redacting writer, so secret-shaped values do not reach the bundle.

The bundle is written locally and never uploaded. You decide what to share.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | | Output folder path |
| `--service` | `-s` | string | | Include logs and health for specific service(s), comma-separated |
| `--tail` | | int | `200` | Recent log lines per service to include |
| `--zip` | | bool | `false` | Create a zip archive after writing the support bundle |
| `--dry-run` | | bool | `false` | Show the bundle plan without writing files |

## Bundle Contents

| File | Contents |
|------|----------|
| `manifest.json` | Timestamp, project directory, extension version, build metadata, OS and architecture, the file list, and any collection warnings |
| `azure.yaml.redacted` | The project's `azure.yaml` with secret-shaped values removed |
| `services.json` | Detected services and their configuration |
| `requirements.json` | The requirements report for the project |
| `health.json` | A health snapshot for the selected services |
| `logs.json` | The most recent log lines per service, bounded by `--tail` |

A file that cannot be collected does not fail the run. The reason is recorded in the `warnings` array of `manifest.json` and the file is left out of the bundle.

## Output Location

Without `--output`, the bundle is written to a timestamped folder under the project:

```
.azd/support-bundles/support-bundle-<YYYYMMDD-HHMMSS>/
```

`--output` accepts an absolute path, or a path relative to the project directory. Paths are validated before use.

With `--zip`, the archive is written next to the folder. If `--output` already ends in `.zip`, that path becomes the archive and the folder drops the extension.

## Examples

### Preview the bundle plan

```bash
azd app support-bundle --dry-run
```

The dry run prints the output folder, the archive path if `--zip` is set, the tail size, the service filter, and the planned file list. Nothing is written.

### Write a bundle

```bash
azd app support-bundle
```

### Write a bundle and zip it for sharing

```bash
azd app support-bundle --zip
```

### Choose the output location

```bash
azd app support-bundle --output ./diagnostics --zip
```

### Limit logs and health to selected services

```bash
azd app support-bundle --service api,web
```

### Include more log history

```bash
azd app support-bundle --tail 1000
```

## Before You Share

The bundle is redacted, but redaction is pattern-based. Skim `azure.yaml.redacted` and `logs.json` before attaching the bundle to a public issue.

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | The bundle was written, possibly with warnings recorded in `manifest.json` |
| `1` | The project could not be located, the output path was rejected, or the bundle folder, manifest, or archive could not be written |

## Related Commands

- [azd app health](health.md) - Monitor health status of services
- [azd app logs](logs.md) - View logs from running services
- [azd app reqs](reqs.md) - Check and verify required tools
- [azd app info](info.md) - Show information about running services
