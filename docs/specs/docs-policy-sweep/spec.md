# Docs policy compliance sweep

## Context
Docs in this repo include user documentation, engineering notes, and planning artifacts. Over time, some docs drift from the repo’s conventions (especially task trackers), which makes it harder to navigate and maintain.

## Goals
- Ensure task tracker docs are structurally consistent and easy to maintain.
- Remove obvious structural corruption in docs (duplicate headers/markers).
- Keep changes mechanical and low-risk: no semantic rewrites unless needed.

## Non-Goals
- Rewriting large specs for tone or completeness.
- Converting all historical planning docs to a new template.
- Changing product behavior or code as part of the docs sweep.

## Policies enforced by this sweep
- `tasks.md` files must have exactly one `<!-- NEXT: ... -->` marker and it must be the first non-empty content in the file.
- Task tracker docs must not contain duplicated top-level headers caused by accidental concatenation.
- Small mechanical fixes only (ordering, headers, markers, whitespace).

## Acceptance criteria
- No `tasks.md` contains multiple `<!-- NEXT: ... -->` markers.
- Task trackers missing a `<!-- NEXT: ... -->` marker get one added at the top.
- Accidental duplicate headers/sections are removed while preserving the intended content.
