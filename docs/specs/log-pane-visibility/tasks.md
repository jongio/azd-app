<!-- NEXT: 0 -->
# Log Pane Visibility Defaults Tasks

## TODO

## Done

## Done: 1 - Stop filtering panes by state/health
**Assigned**: Developer
**Priority**: P1

Update ConsoleView grid rendering so pane visibility is determined only by explicit service selection, not by lifecycle state or health.

**Acceptance**:
- All services render panes by default.
- State/health UI remains visible but does not hide panes.

## Done: 2 - Add regression tests
**Assigned**: Developer
**Priority**: P1

Add/adjust dashboard tests to ensure state/health filters cannot hide panes.

**Acceptance**:
- A test sets state/health filters to exclude some statuses but still expects all panes present.
