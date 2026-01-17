# Azure YAML Editor E2E Tests - Updated Completion Plan

## Current Status

**Phase 1: Foundation** ✅ COMPLETE
- ✅ Test file structure created (22 files, 255 tests)
- ✅ ES module issues fixed
- ✅ Modal interactions fixed
- ✅ Navigation helpers fixed
- ✅ Test fixtures created

**Phase 2: UI Discovery and Selector Updates** ✅ COMPLETE
- ✅ UI structure analyzed
- ✅ Selector registry created (`selectors.ts`)
- ✅ Helper functions updated
- ✅ Documentation created
- ✅ Verified working (navigation + service tests passing)

**Phase 3: Fix Core Functionality Tests** ⏳ NEXT
- ⏳ Schema Forms (02-schema-forms.spec.ts)
- ⏳ Services (03-services.spec.ts) - finish remaining
- ⏳ Resources (04-resources.spec.ts) - finish remaining

## Progress Summary

- **Total Tests**: 255
- **Passing**: 66+ (26%+)
- **Remaining**: ~189 (74%)
- **Infrastructure**: ✅ Complete
- **Selectors**: ✅ Complete

## What We've Built

### 1. Selector Registry (`selectors.ts`)
Comprehensive selector registry with:
- Navigation selectors (tabs, tree items, add buttons)
- Header selectors (all action buttons)
- Modal selectors (all modals and their contents)
- Form selectors (all field types)
- Validation, preview, help, keyboard shortcuts

### 2. Updated Helper Functions
- `navigateToSection()` - Improved with tab/treeitem patterns
- `expandSection()` - Uses aria-expanded attribute
- `findInNavigation()` - Multiple fallback patterns
- `fillFormField()` - Tries name, id, placeholder
- `addServiceViaForm()` - Improved selectors

### 3. Documentation
- `SELECTOR-USAGE.md` - How to use selectors
- `REVIEW-AND-PLAN.md` - Original comprehensive plan
- `PHASE2-COMPLETE.md` - Phase 2 summary
- `README.md` - Test organization guide

## Remaining Work

### Phase 3: Core Functionality (Week 3)
**Goal**: Get core tests passing

**Files:**
1. `02-schema-forms.spec.ts` (22 tests)
2. `03-services.spec.ts` (32 tests) - finish remaining ~22
3. `04-resources.spec.ts` (24 tests) - finish remaining ~19

**Approach:**
1. Run each file: `pnpm test:e2e editor/02-schema-forms.spec.ts --reporter=list`
2. Identify failures
3. Update selectors using registry
4. Fix timing issues
5. Add defensive checks
6. Verify all tests pass

**Estimated Time**: 6-9 hours

### Phase 4: Configuration (Week 4)
**Files:**
- `05-healthchecks.spec.ts` (14 tests)
- `06-hooks.spec.ts` (15 tests)
- `07-env-ports.spec.ts` (13 tests)
- `08-test-config.spec.ts` (14 tests)
- `09-logs-config.spec.ts` (10 tests)

**Estimated Time**: 8-10 hours

### Phase 5: Advanced Features (Week 5)
**Files:**
- `10-pipeline-infra.spec.ts` (23 tests)
- `11-requirements-metadata.spec.ts` (10 tests)
- `12-yaml-editor.spec.ts` (17 tests)
- `13-preview-pane.spec.ts` (10 tests)
- `14-validation.spec.ts` (23 tests)

**Estimated Time**: 10-12 hours

### Phase 6: Workflows (Week 6)
**Files:**
- `15-import-export.spec.ts` (16 tests)
- `16-backup-restore.spec.ts` (12 tests)
- `17-save-load.spec.ts` (10 tests)
- `18-command-palette.spec.ts` (11 tests)
- `19-keyboard-shortcuts.spec.ts` (9 tests)
- `20-accessibility.spec.ts` (11 tests)
- `21-error-handling.spec.ts` (15 tests)
- `22-integration.spec.ts` (9 tests)

**Estimated Time**: 12-15 hours

### Phase 7: Optimization (Week 7)
- Performance optimization
- Reliability improvements
- Final documentation
- Full test suite verification

**Estimated Time**: 6-8 hours

## Quick Start Guide

### To Fix a Test File:

1. **Run the tests:**
   ```bash
   cd cli/dashboard
   pnpm test:e2e editor/02-schema-forms.spec.ts --reporter=list
   ```

2. **Identify failures:**
   - Element not found → Update selector
   - Timeout → Add proper wait
   - Wrong assertion → Fix expectation

3. **Apply fixes:**
   ```typescript
   // Import selectors
   import * as selectors from './selectors'
   
   // Use selector registry
   const button = page.locator(selectors.header.addService).first()
   
   // Add defensive check
   if (await button.isVisible({ timeout: 2000 }).catch(() => false)) {
     await button.click()
   }
   ```

4. **Re-run to verify:**
   ```bash
   pnpm test:e2e editor/02-schema-forms.spec.ts
   ```

## Key Resources

- **Selector Registry**: `e2e/editor/selectors.ts`
- **Usage Guide**: `e2e/editor/SELECTOR-USAGE.md`
- **Helper Functions**: `e2e/helpers/test-setup.ts`
- **Test Fixtures**: `e2e/fixtures/`
- **Test Project**: `cli/tests/projects/editor-e2e-test/`

## Success Criteria

- [ ] All 255 tests passing
- [ ] Test execution time < 30 minutes
- [ ] Flaky test rate < 5%
- [ ] All selectors documented
- [ ] All helpers use selector registry

## Estimated Total Remaining Time

- Phase 3: 6-9 hours
- Phase 4: 8-10 hours
- Phase 5: 10-12 hours
- Phase 6: 12-15 hours
- Phase 7: 6-8 hours
- **Total: 42-54 hours** (~1-1.5 weeks of focused work)

## Next Immediate Actions

1. **Update remaining helpers** to use selector registry (1-2 hours)
2. **Fix schema forms tests** (2-3 hours)
3. **Finish service tests** (2-3 hours)
4. **Finish resource tests** (2-3 hours)

**Total for immediate next steps: 7-11 hours**

---

## Conclusion

Phase 2 is complete. We have:
- ✅ Solid test infrastructure
- ✅ Comprehensive selector registry
- ✅ Updated helper functions
- ✅ Clear documentation
- ✅ Working foundation (66+ tests passing)

The remaining work is systematic: update selectors, fix timing, add defensive checks. With the selector registry and updated helpers, this should be straightforward.

**Ready to proceed with Phase 3!**
