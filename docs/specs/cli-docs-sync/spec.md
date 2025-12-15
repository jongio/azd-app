# CLI Docs Sync

## Goal
Keep CLI command documentation consistent across:
- The actual CLI command set implemented in Go
- The CLI reference markdown
- Per-command markdown files
- The website CLI reference pages

## In Scope
- Ensure every top-level `azd app <command>` implemented in the CLI is documented in `cli/docs/cli-reference.md`.
- Ensure every top-level command has a corresponding file in `cli/docs/commands/<command>.md`.
- Ensure command docs reflect current flags, arguments, and subcommands.
- Ensure the website contains a reference page for each documented command.

## Out of Scope
- Changing command behavior.
- Removing existing documentation unless explicitly requested.
- Documenting internal-only commands beyond minimal “internal/hidden” notes.

## Source of Truth
- CLI command set and flags are derived from the Cobra command definitions in `cli/src/cmd/app/commands/`.

## Acceptance Criteria
- `cli/docs/cli-reference.md` includes all implemented top-level commands.
- Each implemented top-level command has a matching `cli/docs/commands/<command>.md`.
- Website CLI reference pages exist for each documented command and align with the markdown docs.
