# Service run output

**Overview**
- CLI summary for `azd app run` service startup; one block per service, no columns, no URL truncation.
- Show run context (profile, service count, optional elapsed), phase header, services list, ready/footer line.

**Components**
- Header line: `azd app run — profile: {name} — services: {count} — elapsed: {time}` (elapsed optional if available).
- Phase line: `Starting services…` before listing services.
- Service block: status symbol and service name on first line; subsequent lines for endpoints (indented two spaces) labeled `local:`, `custom:`, `azure:`, `domain:`; blank line between services.
- Footer: `Ready — all services healthy — logs: azd app logs --follow` (or error variant when failures exist).

**States**
- Service status: ok (healthy), warn (degraded/unknown), err (failed). Use ✓/⚠/✗ with color when available; ASCII fallback [OK]/[WARN]/[ERR] when not.
- Phase: starting, ready, error (if any service failed). Ready only when all services ok or warn resolved.

**Interactions**
- Non-interactive output; no columns. Natural wrapping handled by terminal.
- On warn/err, append a short reason on the status line (e.g., `✗ api — port 8080 in use`). Still list any known endpoints below.
- Verbose mode may add extra lines per service (e.g., health details) beneath existing labels without changing the base layout.

**A11y**
- Do not rely on color alone: always pair symbols with ASCII tokens in no-color environments.
- Keep indentation minimal (two spaces) and avoid column padding so screen readers read cleanly.
- Ensure output works with NO_COLOR/TERM=dumb; no ANSI boxes; only basic color codes when allowed.

**Responsive**
- Supports narrow terminals via natural wrapping; each label on its own line prevents overflow issues.
- Blank line between services to maintain readability even when wrapped.

**Tokens**
- Reuse existing CLI palette: green for success, yellow for warn, red for error, dim for labels. No new tokens required; fallback to ASCII-only when colors disabled.
