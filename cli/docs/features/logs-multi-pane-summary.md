# Multi-Pane Logs Dashboard - Complete Specification Summary

## What's New in This Enhancement

Based on user research and azd configuration patterns, the logs dashboard now includes:

### 1. **Global Pattern Configuration** (User + App Levels)
Store false positive patterns in two places:
- **User Level** (`~/.azure/logs-dashboard/patterns.json`): Applies to all projects
- **App Level** (`.azure/logs-dashboard/patterns.json`): Project-specific, can commit to repo

This allows teams to share common patterns while preserving individual customizations.

### 2. **Grid & Layout Configuration**
- **Column Slider**: Dynamically adjust 1-6 columns with real-time preview
- **Row Height Slider**: Adjust pane heights (300-800px)
- Settings persist to `~/.azure/logs-dashboard/preferences.json`
- Mobile auto-detection: collapses to 1 column on small screens

### 3. **View Mode Switcher**
Two powerful log display modes:
- **Grid View**: Each service in isolated pane (monitoring mode)
- **Unified View**: All services in single scrollable stream (debugging/correlation mode)
- Toggle with button or `Ctrl+Shift+L`
- Preference persists

### 4. **Copy Functionality**
- Per-pane copy button (top-right of header)
- Per-line copy on right-click/hover
- Multi-pane export (all services)
- Format options: plaintext, JSON, Markdown, CSV
- Visual feedback with line count notification

### 5. **Enhanced UX Ideas Document**
See `logs-ux-enhancements.md` for 30+ potential improvements organized by priority, including:
- Log correlation & timeline view
- Advanced query syntax
- Faceted search
- Incident sharing
- Keyboard shortcuts
- And more...

---

## Documentation Files

| File | Purpose |
|------|---------|
| `logs-multi-pane-view.md` | **Main specification** - Complete functional requirements, acceptance criteria, technical details |
| `logs-ux-enhancements.md` | **Inspiration document** - 30+ UX ideas organized by phase, not commitments but reference for future work |
| `logs-multi-pane-tasks.md` | **Task breakdown** - 35+ actionable dev tasks across 7 phases with completion criteria |
| `logs-multi-pane-quickstart.md` | **Developer reference** - High-level overview, build phases, common gotchas |

---

## Key Differences from Original Request

✅ **Added**:
- Global pattern storage at user + app levels (not just session/localStorage)
- Grid/column configuration via sliders
- View switcher (grid vs unified mode)
- Copy buttons per pane and per line
- Configuration file structure following azd conventions
- Extensive UX ideas for future enhancements

✅ **Preserved from Original**:
- Multi-pane isolated view
- Status indicators (red+blink for errors, yellow for warnings, white for info)
- False positive/negative marking (per-line + global patterns)
- Per-pane search, filter, scroll
- Auto-scroll pause on user scroll

---

## Configuration Architecture

### User Settings
**File**: `~/.azure/logs-dashboard/preferences.json` (created automatically)
```json
{
  "version": "1.0",
  "ui": {
    "gridColumns": 2,
    "paneHeight": 500,
    "viewMode": "grid",
    "selectedServices": ["web", "api", "db"]
  },
  "behavior": {
    "autoScroll": true,
    "pauseOnScroll": true
  }
}
```

### Patterns
**User Level**: `~/.azure/logs-dashboard/patterns.json`
**App Level**: `.azure/logs-dashboard/patterns.json` (team-shared)

Both files use same format:
```json
{
  "version": "1.0",
  "patterns": [
    {
      "id": "pattern-001",
      "name": "Zero Errors",
      "regex": "^.*\\b0\\s+errors?\\b.*$",
      "enabled": true,
      "source": "user|app"
    }
  ]
}
```

**Precedence**: App patterns override user patterns (merge at runtime)

---

## File Storage Locations

| OS | User Settings | User Patterns | App Config | App Patterns |
|----|---------------|---------------|-----------|--------------|
| Windows | `%APPDATA%\.azure\logs-dashboard\preferences.json` | `%APPDATA%\.azure\logs-dashboard\patterns.json` | `.azure\logs-dashboard\config.json` | `.azure\logs-dashboard\patterns.json` |
| macOS/Linux | `~/.azure/logs-dashboard/preferences.json` | `~/.azure/logs-dashboard/patterns.json` | `.azure/logs-dashboard/config.json` | `.azure/logs-dashboard/patterns.json` |

---

## Acceptance Criteria Checklist

### Core Features (Must Have)
- ✅ Multi-pane grid with configurable columns (1-6)
- ✅ Status indicators: red+blink (error), yellow (warning), white (info)
- ✅ False positive/negative per-line marking
- ✅ Global pattern configuration (user + app levels)
- ✅ View mode switcher (grid ↔ unified)
- ✅ Copy per-pane and per-line
- ✅ Grid/column configuration via sliders
- ✅ Preferences persist to `~/.azure/logs-dashboard/`

### Quality Metrics
- Tests: ≥80% coverage
- Accessibility: WCAG AA minimum
- TypeScript: No `any` types
- Performance: <100ms search/filter, 60fps scroll

---

## Next Steps

### For Developer
1. Review main spec: `logs-multi-pane-view.md`
2. Skim task breakdown: `logs-multi-pane-tasks.md`
3. Reference quickstart: `logs-multi-pane-quickstart.md`
4. Start with Phase 1 tasks (layout + multi-pane rendering)

### For Product
1. Optional: Review UX ideas in `logs-ux-enhancements.md`
2. Prioritize Phase 2+ features with team
3. Consider user research before advanced features

### For QA
1. Acceptance criteria in main spec
2. Test matrices: mobile (1 col), tablet (2 col), desktop (3-6 col)
3. Pattern configuration UI testing
4. Cross-browser testing for copy functionality

---

## Research References

This spec is based on:
- **Azure Developer CLI Configuration Patterns**: How `.azure/` directory and `~/.azure/` home directory storage work
- **User Feedback**: Requirements for copy functionality, layout configuration, and pattern management
- **Industry Best Practices**: Multi-pane UX (IDE editors, devtools), pattern matching (linters, formatters)

For detailed research on azd configuration storage, see agent research file referenced in development notes.
