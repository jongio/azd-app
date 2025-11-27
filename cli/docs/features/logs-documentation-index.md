# Multi-Pane Logs Dashboard - Complete Documentation Index

## Quick Navigation

### 📋 For Product/PM
1. **Start here**: `logs-multi-pane-summary.md` - High-level overview of all features
2. **Detailed requirements**: `logs-multi-pane-view.md` - Complete functional spec
3. **Inspiration**: `logs-ux-enhancements.md` - 30+ ideas for future improvements
4. **Research**: `configuration-research.md` - Background on how azd handles config

### 👨‍💻 For Developers
1. **Start here**: `logs-multi-pane-quickstart.md` - Build phases and high-level approach
2. **Main spec**: `logs-multi-pane-view.md` - All requirements and technical details
3. **Task list**: `logs-multi-pane-tasks.md` - 35+ actionable dev tasks
4. **Configuration**: `configuration-management.md` - Implementation guide for config files
5. **Research**: `configuration-research.md` - Storage architecture and best practices

### 🧪 For QA/Testers
1. **Acceptance criteria**: See "Acceptance Criteria" section in `logs-multi-pane-view.md`
2. **Task breakdown**: `logs-multi-pane-tasks.md` - Completion criteria for each phase
3. **Configuration testing**: `configuration-management.md` - How to test config loading/saving

---

## Documentation Files Overview

### Features Documentation (`cli/docs/features/`)

#### 📄 logs-multi-pane-summary.md
**What**: Executive summary of complete feature
**Who**: Product managers, stakeholders
**Length**: ~3 pages
**Contains**:
- What's new (vs original request)
- Architecture overview
- Configuration file formats
- Acceptance criteria checklist
- Next steps for each role

#### 📄 logs-multi-pane-view.md
**What**: Complete functional specification
**Who**: Developers, product team
**Length**: ~8 pages
**Contains**:
- User story
- Functional requirements (all 8 areas)
- Acceptance criteria (40+ items)
- Component structure
- State management architecture
- Styling approach
- Performance considerations
- Accessibility requirements

#### 📄 logs-ux-enhancements.md
**What**: Future UX ideas and improvements
**Who**: Product team, designers
**Length**: ~12 pages
**Contains**:
- 10 major improvement areas
- 30+ specific feature ideas
- Implementation priority phases
- User research recommendations
- Not commitments, but inspiration for v2+

### Development Documentation (`cli/docs/dev/`)

#### 📄 logs-multi-pane-quickstart.md
**What**: Developer reference guide
**Who**: Developers (primary)
**Length**: ~4 pages
**Contains**:
- High-level overview
- Phase breakdown (7 phases)
- Build order and dependencies
- Files to create/modify
- Common gotchas
- Running tests
- Acceptance criteria

#### 📄 logs-multi-pane-tasks.md
**What**: Detailed task breakdown
**Who**: Project manager, developers
**Length**: ~8 pages
**Contains**:
- 35+ specific tasks
- Organized in 7 phases
- Dependencies between tasks
- Completion criteria per task
- Success metrics

#### 📄 configuration-management.md
**What**: Implementation guide for config storage
**Who**: Backend developers
**Length**: ~10 pages
**Contains**:
- File structure and schema definitions
- TypeScript code examples
- React hook patterns
- API endpoint specifications
- Best practices (permissions, atomic writes)
- Testing patterns
- Troubleshooting guide

#### 📄 configuration-research.md
**What**: Research on azd configuration patterns
**Who**: Developers, architects
**Length**: ~9 pages
**Contains**:
- Two-level storage architecture
- Directory structure details
- File format standards
- Permission conventions
- Cache invalidation strategies
- How azd actually uses `.azure/`
- Cross-platform compatibility
- Error handling patterns

---

## Feature Breakdown

### Core Features (Phase 1)
From spec: Multi-pane grid with configurable columns, status indicators, independent scroll

**Documentation**:
- Main spec: Section "1. Multi-Pane Layout"
- Tasks: Tasks T1.1-T1.5 (Phase 1)
- Quickstart: "Phase 1: Core Layout"

### Status Indicators (Phase 2)
Red+blink for errors, yellow for warnings, white for info

**Documentation**:
- Main spec: Section "2. Status Indicators"
- Tasks: Tasks T2.1-T2.4 (Phase 2)
- Quickstart: "Phase 2: Status Indicators"

### False Positive/Negative Markers (Phase 3)
Per-line marking + global pattern configuration

**Documentation**:
- Main spec: Section "3. False Positive / False Negative Detection"
- Config guide: Pattern file format and merging logic
- Tasks: Tasks T3.1-T3.5 (Phase 3)
- Research: Pattern management strategies

### Grid Configuration (Phase 4)
Column slider (1-6), row height slider

**Documentation**:
- Main spec: Section "4. Grid & Pane Layout Configuration"
- Tasks: Task T4.4 (Row height UI)
- Quickstart: "Feature: Grid configuration"

### View Switcher (Phase 4)
Toggle between grid and unified view

**Documentation**:
- Main spec: Section "5. View Switcher: Grid vs. Unified"
- Tasks: Tasks T4.1-T4.3 (per-pane controls)
- Quickstart: "Feature: View modes"

### Copy Functionality (Phase 4)
Copy per-pane, per-line, multi-pane

**Documentation**:
- Main spec: Section "6. Copy Functionality"
- Tasks: Tasks T5.3 (export formats)
- Config guide: Copy format preferences

---

## Configuration Architecture

### File Locations

| File | Location (Windows) | Location (macOS/Linux) | Commits to Repo? |
|------|-------------------|----------------------|-----------------|
| User Preferences | `%APPDATA%\.azure\logs-dashboard\preferences.json` | `~/.azure/logs-dashboard/preferences.json` | NO |
| User Patterns | `%APPDATA%\.azure\logs-dashboard\patterns.json` | `~/.azure/logs-dashboard/patterns.json` | NO |
| App Config | `.azure\logs-dashboard\config.json` | `.azure/logs-dashboard/config.json` | YES |
| App Patterns | `.azure\logs-dashboard\patterns.json` | `.azure/logs-dashboard/patterns.json` | YES |

### Configuration Precedence
1. Environment variables (if implemented)
2. App-level patterns (`.azure/logs-dashboard/`)
3. User-level preferences (`~/.azure/logs-dashboard/`)
4. Hardcoded defaults

**Documentation**: See `configuration-management.md` for full implementation guide

---

## Acceptance Criteria Organization

### By Phase

**Phase 1 (Core Layout)**: T1.1-T1.5 acceptance criteria
- Multi-pane renders ✅
- Logs independent ✅
- Per-pane auto-scroll ✅

**Phase 2 (Status Indicators)**: T2.1-T2.4 acceptance criteria
- Error = red + blink ✅
- Warning = yellow ✅
- Info = white ✅

**Phase 3 (Patterns)**: T3.1-T3.5 acceptance criteria
- Per-line marking works ✅
- Global patterns load ✅
- User + app level works ✅

**Phase 4 (UI)**: T4.1-T5.4 acceptance criteria
- Sliders work (columns, height) ✅
- View switcher works ✅
- Copy functionality works ✅
- Settings modal works ✅

**Phase 6 (Testing)**: T6.1-T6.6 acceptance criteria
- 80% test coverage ✅
- All e2e tests pass ✅
- Accessibility audit passes ✅
- Responsive design verified ✅

**See**: `logs-multi-pane-view.md` "Acceptance Criteria" section for complete checklist

---

## Implementation Path

### Week 1: Foundation
- Phase 1 (Core Layout)
- Phase 2 (Status Indicators)
- Documentation: Review config architecture

### Week 2: Configuration & Features
- Phase 3 (Patterns & Configuration)
- Phase 4 (Layout Controls)
- Implementation: Config file loading/saving

### Week 3: Polish & Test
- Phase 5 (Global Controls)
- Phase 6 (Testing & Accessibility)
- Fix bugs, optimize performance

### Week 4: Ship
- Phase 7 (Optional enhancements)
- Final QA pass
- Release notes

---

## File Dependencies

```
logs-multi-pane-view.md (main spec)
├── Referenced by: logs-multi-pane-tasks.md
├── Referenced by: logs-multi-pane-quickstart.md
├── Referenced by: configuration-management.md
└── Referenced by: logs-multi-pane-summary.md (overview)

configuration-management.md (implementation guide)
├── References: configuration-research.md
├── Used by: Backend developers
└── Needed by: Phase 3 (Pattern configuration)

configuration-research.md (research)
├── Referenced by: configuration-management.md
├── Referenced by: logs-multi-pane-summary.md
└── Optional reading for architects

logs-ux-enhancements.md (future ideas)
├── Independent (reference only)
├── Not required for v1
└── Reference for Phase 2+ planning
```

---

## Key Decisions Documented

### 1. Two-Level Configuration
**Why**: Teams need shared defaults, individuals need personal overrides
**Where**: `configuration-research.md` section 1 + `logs-multi-pane-view.md` section 9
**Impact**: Phase 3+ tasks require backend support

### 2. File-Based Not Browser-Only
**Why**: Configuration must be team-shareable via git
**Where**: `logs-multi-pane-view.md` section 9 + `configuration-research.md`
**Impact**: Backend must handle file I/O with proper permissions

### 3. Grid Layout Default (not List)
**Why**: Matches user request "see diff panes all at same time"
**Where**: `logs-multi-pane-view.md` section 5 + `logs-multi-pane-summary.md`
**Impact**: Phase 1 requires CSS Grid implementation

### 4. Three-Tier Status System
**Why**: Error (critical), Warning (caution), Info (normal) is standard UX
**Where**: `logs-multi-pane-view.md` section 2 + `logs-ux-enhancements.md` section 1.2
**Impact**: Phase 2 requires pattern matching + animation

### 5. Pattern Suggestions (Not Auto-Inference)
**Why**: Too many false positives if ML-based without training data
**Where**: `logs-multi-pane-view.md` section 3 (Pattern Suggestions)
**Impact**: Phase 3 includes UI for quick pattern creation

---

## Cross-References

### By User Story

**"As a developer, I want to see all services at once"**
- Feature: Multi-pane grid layout
- Docs: `logs-multi-pane-view.md` section 1
- Tasks: T1.1-T1.5
- Config: Grid columns setting

**"I need to quickly identify errors"**
- Feature: Status indicators
- Docs: `logs-multi-pane-view.md` section 2
- Tasks: T2.1-T2.4
- UX ideas: Section 1.2 (Severity distribution)

**"False positives make it hard to find real errors"**
- Feature: Pattern-based filtering
- Docs: `logs-multi-pane-view.md` section 3
- Tasks: T3.1-T3.5
- Config: `configuration-management.md` patterns section

**"I need to share logs with my team"**
- Feature: Copy + export
- Docs: `logs-multi-pane-view.md` section 6
- Tasks: T5.3 (multi-pane export)
- UX ideas: Section 4.2 (Report generation)

---

## Testing Strategy

### Unit Tests
**Location**: `cli/dashboard/src/components/`
**Coverage Target**: ≥80%
**Focus**: Pattern matching, state management, config loading

### E2E Tests
**Location**: `cli/dashboard/e2e/`
**Scenarios**: All workflows in "Acceptance Criteria"
**Tool**: Playwright (existing in project)

### Accessibility Tests
**Standard**: WCAG AA
**Check**: Color contrast, keyboard nav, screen reader
**Tools**: axe-core, manual review

**Documentation**: `logs-multi-pane-tasks.md` Phase 6 tasks

---

## Rollout Plan

### User Impact
- **v1**: Grid view only (both modes available in Phase 4)
- **v1.1**: View switcher (grid + unified)
- **v2**: Advanced features from `logs-ux-enhancements.md`

### Developer Training
- `logs-multi-pane-quickstart.md` for overview
- `logs-multi-pane-view.md` for detailed requirements
- `configuration-management.md` for config architecture

### Release Notes Template
```markdown
## Logs Dashboard Multi-Pane View (v1.0)

### Features
- View multiple services in isolated panes
- Status indicators (error/warning/info)
- False positive pattern filtering
- Configurable grid layout
- Copy logs to clipboard

### Configuration
- Settings stored in ~/.azure/logs-dashboard/
- Team-shared patterns in .azure/logs-dashboard/
- [See documentation](link-to-config-guide)

### Migration
- Existing single-pane view preserved
- Choose between modes with toggle
- No action needed for users
```

---

## Questions & Decisions Needed

### Before Phase 1 Starts
- [ ] Confirm: Should multi-pane replace single-pane view or coexist?
- [ ] Confirm: API endpoints for pattern CRUD provided by backend?

### Before Phase 3 Starts
- [ ] Decide: Import built-in pattern library or manual entry only?
- [ ] Decide: localStorage fallback if file I/O fails?

### Before Phase 4+ Starts
- [ ] Review: UX enhancements priorities (from `logs-ux-enhancements.md`)
- [ ] Decide: Virtual scrolling if perf issues detected?

---

**Last Updated**: November 23, 2025  
**Version**: 1.0 (Complete Specification)  
**Status**: Ready for Developer Handoff
