<!-- NEXT: 0 -->
# CLI Docs Sync Tasks

## Done

## DONE: Inventory Implemented CLI Commands (1)
- Confirmed registered commands from `cli/src/cmd/app/main.go` and Cobra command files.
- Identified internal/hidden commands (`listen`, `mcp`).

## DONE: Sync CLI Reference (2)
- Added `add` and `listen` to `cli/docs/cli-reference.md` (including a new `add` section).
- Documented the missing global `--cwd/-C` flag.

## DONE: Sync Per-Command Docs (3)
- Added `cli/docs/commands/listen.md`.
- Fixed `cli/docs/commands/notifications.md` to use valid Go duration examples.

## DONE: Sync Website Pages (4)
- Updated `web/scripts/generate-cli-reference.ts` to generate a `listen` page but hide it from the index.
- Regenerated website pages in `web/src/pages/reference/cli/`.
