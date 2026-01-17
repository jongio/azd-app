# Azure YAML Editor E2E Tests

This directory contains organized E2E tests for the Azure YAML Editor, covering all features of the azure.yaml 1.1 schema.

## Test Organization

Tests are organized into 22 feature-based test suites:

1. **01-navigation.spec.ts** - Navigation tree functionality
2. **02-schema-forms.spec.ts** - Schema form generation
3. **03-services.spec.ts** - Service management (all host types)
4. **04-resources.spec.ts** - Resource management (all resource types)
5. **05-healthchecks.spec.ts** - Health check configuration
6. **06-hooks.spec.ts** - Hooks configuration (project + service level)
7. **07-env-ports.spec.ts** - Environment variables and ports
8. **08-test-config.spec.ts** - Test configuration
9. **09-logs-config.spec.ts** - Logs configuration
10. **10-pipeline-infra.spec.ts** - Pipeline, infrastructure, state, platform, workflows, cloud, requiredVersions
11. **11-requirements-metadata.spec.ts** - Requirements and metadata
12. **12-yaml-editor.spec.ts** - Direct YAML editing
13. **13-preview-pane.spec.ts** - Preview pane
14. **14-validation.spec.ts** - Validation (schema + business rules)
15. **15-import-export.spec.ts** - Import/export workflows
16. **16-backup-restore.spec.ts** - Backup/restore operations
17. **17-save-load.spec.ts** - Save/load operations
18. **18-command-palette.spec.ts** - Command palette
19. **19-keyboard-shortcuts.spec.ts** - Keyboard shortcuts
20. **20-accessibility.spec.ts** - Accessibility features
21. **21-error-handling.spec.ts** - Error handling
22. **22-integration.spec.ts** - End-to-end integration workflows

## Running Tests

### Run All Editor Tests
```bash
cd cli/dashboard
pnpm test:e2e editor/
```

### Run Specific Test File
```bash
pnpm test:e2e editor/01-navigation.spec.ts
```

### Run Tests in UI Mode
```bash
pnpm test:e2e:ui editor/
```

### Run Tests with Specific Browser
```bash
pnpm test:e2e editor/ --project=chromium
```

### List All Tests
```bash
pnpm test:e2e -- --list
```

## Test Helpers

All tests use helper functions from `../helpers/test-setup.ts`:

- `setupTest()` - Set up test environment with mocked APIs
- `navigateToEditor()` - Navigate to editor page
- `loadComprehensiveProject()` - Load comprehensive test project
- `loadMinimalProject()` - Load minimal test project
- `loadInvalidProject()` - Load invalid project for error testing
- `navigateToSection()` - Navigate to specific section
- `expandSection()` - Expand navigation section
- `addServiceViaForm()` - Add service via form modal
- `addResourceViaForm()` - Add resource via form modal
- `configureHealthCheck()` - Configure health check
- `configureHooks()` - Configure hooks
- `editYamlDirectly()` - Edit YAML in textarea
- `getYamlContent()` - Get current YAML content
- `waitForValidation()` - Wait for validation to complete
- `getValidationErrors()` - Get validation errors
- `expectValidationError()` - Expect specific validation error
- `expectNoValidationErrors()` - Expect no validation errors

## Test Fixtures

Test fixtures are located in `../fixtures/`:

- `comprehensive-azure-yaml.yaml` - Full azure.yaml with all features
- `minimal-azure-yaml.yaml` - Minimal valid azure.yaml
- `invalid-azure-yaml.yaml` - Invalid YAML syntax for error testing
- `schema-violations.yaml` - YAML with schema violations
- `service-configs.json` - Service configuration examples
- `resource-configs.json` - Resource configuration examples

## Test Project

The comprehensive test project is located at:
`cli/tests/projects/editor-e2e-test/`

This project contains a complete `azure.yaml` that exercises every feature of the 1.1 schema.

## Common Issues and Fixes

### Issue: `__dirname is not defined in ES module scope`

**Fix:** Use `import.meta.url` with `fileURLToPath`:
```typescript
import { fileURLToPath } from 'url'
const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
```

### Issue: Tests fail because UI elements not found

**Fix:** Tests use defensive checks with `isVisible({ timeout: 2000 }).catch(() => false)`. If elements are consistently not found:
1. Check that the editor page is loading correctly
2. Verify the selectors match the actual UI
3. Increase timeout values if needed

### Issue: Tests fail due to async timing

**Fix:** Tests include `await page.waitForTimeout()` calls. If timing issues persist:
1. Use `waitForSelector()` instead of `waitForTimeout()` when possible
2. Wait for specific conditions rather than fixed delays
3. Use `waitForValidation()` helper for validation-related waits

### Issue: Mock API not working

**Fix:** Ensure `setupTest()` is called in `beforeEach` and `mockConfigApi()` is properly configured. Check that:
1. Route handlers are set up before navigation
2. API endpoints match what the editor expects
3. Response format matches expected structure

## Debugging Tests

### Run Single Test
```bash
pnpm test:e2e editor/01-navigation.spec.ts -g "should display all main sections"
```

### Run with Debug Mode
```bash
PWDEBUG=1 pnpm test:e2e editor/01-navigation.spec.ts
```

### Run with Headed Browser
Modify `playwright.config.ts` to set `headless: false` or use:
```bash
pnpm test:e2e editor/ --headed
```

### View Test Traces
```bash
pnpm test:e2e editor/ --trace on
```

Then open traces with:
```bash
npx playwright show-trace trace.zip
```

## Test Coverage

These tests cover:
- ✅ All top-level azure.yaml 1.1 properties
- ✅ All service host types (8 types)
- ✅ All resource types (14 types)
- ✅ All service types and modes
- ✅ All healthcheck types
- ✅ All hooks (project + service level)
- ✅ Test and logs configurations
- ✅ Pipeline, infrastructure, state, platform, workflows, cloud, requiredVersions
- ✅ Requirements and metadata
- ✅ YAML editing, validation, preview
- ✅ Import/export, backup/restore, save/load
- ✅ Command palette, keyboard shortcuts
- ✅ Accessibility, error handling
- ✅ Integration workflows

## Migration Status

The old test files are still present but will be removed after verification:
- `editor-integration.spec.ts` → Distributed across multiple suites
- `yaml-editor-navigation.spec.ts` → `editor/01-navigation.spec.ts`
- `service-management.spec.ts` → `editor/03-services.spec.ts`
- `preview-pane.spec.ts` → `editor/13-preview-pane.spec.ts`
- `schema-form.spec.ts` → `editor/02-schema-forms.spec.ts`
- `editor-accessibility.spec.ts` → `editor/20-accessibility.spec.ts`
- `azure-yaml-editor-comprehensive.spec.ts` → Distributed across all suites
