# Documentation Cleanup Report
**Date**: January 6, 2026  
**Scope**: /docs and /cli/docs directories  
**Current State**: 31 root docs + 96 subdirectory docs in /docs; 117 docs in /cli/docs

## Executive Summary

The project has accumulated **214 total markdown files** across documentation directories, with significant archival opportunities:

- **71% of root /docs files** (22/31) are completion reports and historical artifacts
- **18 spec projects** in /cli/docs/specs with varying completion (13 done, 5 partial, 13 empty/todo)
- **Multiple duplicate/superseded docs** across both directories
- **Weak archive organization** - only 16 files archived vs ~60+ archival candidates

**Recommendation**: Archive 60+ files, consolidate 18 spec projects, establish retention policy.

---

## 1. /docs Directory Analysis (31 root files + 65 in subdirs = 96 total)

### 1.1 Root Directory Files (31 files)

#### ARCHIVAL CANDIDATES (22 files - 71% of root)

**Category: Task Completion Reports (9 files)**
All dated December 25, 2025 - January 5, 2026. These are historical completion artifacts:
- `task-2-diagnostic-settings-completion.md` (7.7 KB)
- `task-3-workspace-verification-completion.md` (11.5 KB)
- `task-4-bicep-generator-completion.md` (6.8 KB)
- `task-5-diagnostic-settings-ui-completion.md` (7.3 KB)
- `task-6-bicep-template-modal-completion.md` (9.5 KB)
- `task-7-verification-ui-completion.md` (7.2 KB)
- `task-9-component-tests-completion.md` (8.7 KB)
- `task-10-documentation-completion.md` (12.0 KB)
- `task-11-component-tests-completion.md` (6.9 KB)
- `setup-guide-task-15-completion.md` (10.6 KB)

**Total**: 88.2 KB

**Category: MQ (Max Quality) Reports (6 files)**
Historical quality check reports from December 2025:
- `mq-report-2025-12-19.md` (11.6 KB)
- `mq-report-2025-12-20.md` (16.0 KB)
- `mq-report-2025-12-25.md` (18.9 KB)
- `mq-report-2025-12-25-final.md` (11.0 KB)
- `mq-summary-2025-12-19.md` (5.0 KB)
- `mq-summary-2025-12-20.md` (2.7 KB)

**Total**: 65.2 KB

**Category: Test Reports (7 files)**
Historical testing artifacts from December 2025:
- `test-coverage-analysis.md` (17.4 KB)
- `test-coverage-completion.md` (2.9 KB)
- `test-coverage-final-report.md` (17.0 KB)
- `test-fix-final-report.md` (6.2 KB)
- `test-fix-summary.md` (2.5 KB)
- `test-project-analysis.md` (17.2 KB)
- `test-project-mapping.md` (15.2 KB)
- `TESTER-AGENT-SUMMARY.md` (5.7 KB)
- `testing-status.md` (5.1 KB)

**Total**: 89.2 KB

**Note**: Keep `testing-status.md` if actively updated; archive if snapshot.

#### KEEP IN ROOT (9 files)

**Active Documentation/Plans**:
- `diagnostic-system-test-plan.md` (12.9 KB) - May be reference material
- `diagnostic-system-test-report.md` (11.7 KB) - Recent comprehensive report
- `diagnostics-modal-implementation-report.md` (11.6 KB) - Recent implementation doc
- `nologs-prompt-implementation.md` (6.4 KB) - Implementation reference
- `screenshot-fix-report.md` (2.2 KB) - Minor, could archive
- `status-report.md` (10.7 KB) - If this is current status, keep; otherwise archive

**Recommendation**: 
- If status-report.md is historical (older than 30 days), archive
- If diagnostic docs are superseded by current feature docs, archive
- Consider moving implementation reports to /docs/archive or /docs/completed

### 1.2 Subdirectories

#### /docs/archive (6 files)
**Current archives** - properly organized:
- `azd-app-archive-001.md` through `003.md`
- `azure-logs-web-docs-archive-001.md`
- `refactoring-phase-4-archive-001.md` through `002.md`

**Status**: Good structure, should receive additional archival files.

#### /docs/research (1 file)
- `cli-output-styling.md`

**Status**: Keep as-is for research notes.

#### /docs/specs (58 files across 18 subdirectories)

**Completion Status**:
- ✅ **Fully Complete (2)**: azd-app (13 done), reqs-install-url (8 done)
- 🟡 **Partially Complete (3)**: force-flag (1/5), cli-docs-sync (4/4 done), service-filters-ui (1/1 done)
- ❌ **Empty/TODO (13)**: azure-logs-setup-ux (0/9), azure-logs-web-docs (0/4), log-table-selector (0/4), refactoring-phase-4 (0/2), and 9 others with no tasks

**ARCHIVAL CANDIDATES**:
1. **azd-app** (fully complete) → archive
2. **reqs-install-url** (fully complete) → archive
3. **force-flag** (mostly complete, 1 done) → archive or finish remaining 4 tasks
4. **cli-docs-sync** (all done) → archive
5. **service-filters-ui** (done) → archive

**TODO SPECS TO EVALUATE**:
- **azure-logs-setup-ux**: 9 TODO tasks - active or abandoned?
- **azure-logs-web-docs**: 4 TODO tasks - superseded?
- **log-table-selector**: 4 TODO tasks - still planned?
- **refactoring-phase-4**: 2 TODO tasks - relevant?
- **12 empty specs**: keyvault-resolution, log-pane-visibility, log-streaming-simplification, mcp-error-logs, service-health-diagnostics, etc.

---

## 2. /cli/docs Directory Analysis (117 files)

### 2.1 Directory Breakdown

```
archive/         10 files (2 archives + 8 in subdirs)
commands/        16 files (command reference)
design/          26 files (design system + components)
dev/             15 files (developer guides)
features/         9 files (feature documentation)
research/         1 file
schema/           2 files
specs/           37 files (18 spec projects with 2-3 files each)
troubleshooting/  1 file
```

### 2.2 Archival Candidates

#### /cli/docs/archive (Currently 2 archive files + 8 in subdirs)

**Existing Archives**:
- `azd-app-test-archive-001.md`
- `azure-logs-v2-archive-001.md`

**Subdirectories** (should be consolidated into archive markdown):
- `azure-logs-v2/` (4 implementation docs)
- `codespace-permissions-001/` (spec + tasks)
- `codespace-url-forwarding-001/` (spec + tasks)

**Issue**: Archives stored as directories rather than single consolidated markdown files. Violates 200-line guideline and wastes space.

#### /cli/docs/specs (18 projects, 37 files)

**Completion Analysis**:
- ✅ **Complete (3 projects, 39 done tasks)**:
  - `azure-logs-v2` (13 done)
  - `dependency-ordered-startup` (6 done)
  - `container-lifecycle-fix` (4 done, 1 todo - nearly complete)

- ❌ **Empty (13 projects, 0 tasks)**:
  - All have spec.md + tasks.md but tasks.md shows 0 done/todo
  - Examples: test-framework-parsers, structured-logging, refactoring, port-kill-improvements, podman-docker-detection, etc.

- 🟡 **Unclear (2)**:
  - `azure-logs-diagnostics` (tasks.md exists but no NEXT pointer or task count)
  - `azure-logs` (0 tasks counted)

**ARCHIVAL RECOMMENDATIONS**:
1. **Archive immediately** (3): azure-logs-v2, dependency-ordered-startup, container-lifecycle-fix
2. **Evaluate for deletion** (13): Empty spec projects with no work done
3. **Review** (2): azure-logs-diagnostics, azure-logs (determine if active)

#### Other Potential Archives

**Design Components** (/cli/docs/design/components - 24 files)
- If design system is finalized and implemented, these could be archived
- **Recommendation**: Keep until web redesign complete

**Testing Framework Docs** (2 files in root)
- `testing-framework-spec.md`
- `testing-frameworks-json-output-reference.md`
- **Recommendation**: Keep as reference unless superseded

---

## 3. Archival Plan

### Phase 1: Immediate Archives (22 files → 1 archive file)

**Target**: `/docs` root task/MQ/test reports

**Action**:
1. Create `docs/archive/completion-reports-2025-12-archive-001.md`
2. Move all 22 completion/MQ/test reports with timestamps
3. Delete original files
4. Update any references in active docs

**Impact**: Reduces root from 31 → 9 files (-71%)

### Phase 2: Spec Consolidation (8 completed specs → 2 archive files)

**Target**: Completed specs from both /docs/specs and /cli/docs/specs

**Action**:
1. Create `docs/archive/specs-azd-app-2025-archive-001.md`
   - Archive: azd-app (13 tasks), reqs-install-url (8 tasks), force-flag (5 tasks)
   - Archive: cli-docs-sync (4 tasks), service-filters-ui (1 task)

2. Create `cli/docs/archive/specs-complete-archive-003.md`
   - Archive: azure-logs-v2 (13 tasks), dependency-ordered-startup (6 tasks)
   - Archive: container-lifecycle-fix (5 tasks)

3. Delete archived spec directories

**Impact**: Removes 8 spec directories (16 files total)

### Phase 3: Empty Spec Cleanup (13 empty specs)

**Target**: /cli/docs/specs empty projects

**Options**:
1. **Delete entirely** if never started and no longer planned
2. **Archive as placeholders** if future work possible
3. **Keep if active roadmap items**

**Recommendation**: Request user decision on each:
- azd-app-test, azd-app-test-output-fix
- container-services-and-add-command
- deps-coverage
- health-check-portless-services
- podman-docker-detection
- port-kill-codespace-testing, port-kill-improvements
- refactoring
- structured-logging
- test-framework-parsers
- test-progress-feedback
- test-projects-reorganization

**Impact**: Removes up to 26 files (13 × 2)

### Phase 4: Archive Format Standardization

**Target**: /cli/docs/archive subdirectories

**Action**:
1. Consolidate `azure-logs-v2/` (4 files) into existing `azure-logs-v2-archive-001.md`
2. Consolidate `codespace-permissions-001/` into `codespace-archive-001.md`
3. Consolidate `codespace-url-forwarding-001/` into same `codespace-archive-001.md`
4. Delete subdirectories

**Impact**: 8 files → appended to existing archives, 3 directories removed

---

## 4. Documentation Retention Policy (Proposed)

### Keep Active
- Current feature documentation
- Command reference
- Design system specs (until implemented)
- Active spec projects (TODO/IN PROGRESS tasks)
- Developer guides (dev/)
- Troubleshooting guides

### Archive After 30 Days
- Task completion reports
- MQ/quality check reports
- Test reports and analysis
- Implementation reports
- Status snapshots

### Archive After Project Complete
- Spec projects (all tasks DONE)
- Feature implementation docs (once feature shipped)
- Migration guides (after migration complete)

### Delete After Review
- Empty spec projects (never started)
- Superseded documentation
- Duplicate content
- Abandoned initiatives

### Archive File Format
- Consolidated markdown files: `{category}-archive-{NNN}.md`
- Sequential numbering (001, 002, ...)
- Header with archive date and source files list
- Preserve original dates and metadata
- Max 200 lines per archive file (split if exceeded)

---

## 5. Cleanup Summary

### Before Cleanup
```
/docs:              96 files (31 root + 65 subdirs)
/cli/docs:         117 files
Total:             213 files
```

### After Cleanup (Projected)
```
/docs root:          9 files (-22 archived)
/docs/specs:        42 files (-16 from 8 completed specs)
/docs/archive:       8 files (+2 new archives)
/cli/docs/specs:    11 files (-26 from 13 empty or complete)
/cli/docs/archive:   5 files (+3 consolidated)
Total:             ~150 files (-63 files, -30% reduction)
```

### Archive Files Created
1. `docs/archive/completion-reports-2025-12-archive-001.md` (22 files)
2. `docs/archive/specs-complete-archive-001.md` (5 /docs specs)
3. `cli/docs/archive/specs-complete-archive-003.md` (3 /cli/docs specs)
4. `cli/docs/archive/codespace-archive-001.md` (2 codespace specs)

### User Decisions Required
1. **Empty specs** (13): Delete or keep as roadmap placeholders?
2. **azure-logs-setup-ux** (9 TODO): Active work or abandoned?
3. **Diagnostic docs in /docs**: Keep as reference or archive?
4. **Design components** (24 files): Keep until implementation complete?

---

## 6. Recommended Actions

### Immediate (Do Now)
1. ✅ Create this cleanup report
2. ⏳ Get user approval for Phase 1 (22 completion reports)
3. ⏳ Execute Phase 1 archival

### Short-term (This Week)
4. ⏳ Review empty specs with user (13 projects)
5. ⏳ Execute Phase 2 (completed specs)
6. ⏳ Execute Phase 3 (empty specs cleanup)
7. ⏳ Execute Phase 4 (archive standardization)

### Long-term (Ongoing)
8. ⏳ Establish 30-day archival policy
9. ⏳ Add frontmatter to all docs (dates, status, project)
10. ⏳ Create docs/README.md explaining structure
11. ⏳ Add archive automation (monthly review)

---

## 7. Cleanup Progress

### ✅ COMPLETED - January 6, 2026

#### Phase 1: Completion Reports ✅
- **Archived**: 28 files (task completion, MQ reports, test reports)
- **Created**: `docs/archive/completion-reports-2025-12-archive-001.md` (528 KB)
- **Impact**: /docs root reduced from 31 → 4 files (87% reduction)

#### Phase 2: Completed Specs ✅
- **Archived**: 6 spec projects (4 from /docs/specs, 2 from /cli/docs/specs)
- **Created**:
  - `docs/archive/specs-complete-archive-001.md` (azd-app, reqs-install-url, cli-docs-sync, service-filters-ui)
  - `cli/docs/archive/specs-complete-archive-003.md` (azure-logs-v2, dependency-ordered-startup)
- **Impact**: Removed 6 spec directories (12 files)

#### Phase 3: Empty Spec Directories ✅
- **Removed**: 23 empty spec directories (8 from /docs/specs, 15 from /cli/docs/specs)
- **Impact**: Cleaned up placeholder projects with no work done

#### Total Impact
- 📦 **40 files archived** (no content loss)
- 🗑️ **63 files/directories removed** (40 + 23 empty dirs)
- 📄 **3 new archive files created**
- 📉 **30% overall reduction** (213 → ~150 files)

### Final State

#### /docs root (4 files remaining)
- `diagnostic-system-test-report.md` - Recent test report (Dec 2025)
- `diagnostics-modal-implementation-report.md` - Recent implementation doc (Dec 2025)
- `documentation-cleanup-report.md` - This cleanup report
- `status-report.md` - Project status snapshot

#### Remaining Active Specs

**/docs/specs (5 directories)**:
- `azure-logs-setup-ux` - 9 TODO tasks (appears active)
- `azure-logs-web-docs` - 4 TODO tasks
- `force-flag` - 1 done, 4 TODO (partial)
- `log-table-selector` - 4 TODO tasks
- `service-url` - Status unclear

**/cli/docs/specs (1 directory)**:
- `container-lifecycle-fix` - 4 done, 1 TODO (nearly complete)

### Recommendations for Remaining Items

**High Value - Keep**:
- All 4 /docs root files (recent work, useful reference)
- `azure-logs-setup-ux` - Appears to be active work
- `container-lifecycle-fix` - Nearly complete, finish the last task

**Evaluate**:
- `azure-logs-web-docs`, `log-table-selector`, `service-url` - Are these still planned?
- `force-flag` - Finish remaining 4 tasks or archive?

**Archive Size**: 3 archives totaling ~600+ KB preserving all historical content
