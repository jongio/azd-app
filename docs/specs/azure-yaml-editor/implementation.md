# Azure YAML Editor - Implementation Overview

## Architecture

### Component Hierarchy

```
YamlEditor (Main Integration)
├── YamlEditorHeader (Actions, Command Palette)
├── YamlEditorLayout
│   ├── NavigationSidebar
│   │   ├── NavigationSearch (Fuzzy search)
│   │   └── NavigationItem (Tree nodes)
│   ├── Editor Pane (Dynamic)
│   │   ├── SchemaForm (Schema-driven forms)
│   │   │   ├── FieldRenderer (Dispatches to field types)
│   │   │   ├── StringField / NumberField / BooleanField
│   │   │   ├── EnumField (Dropdown selection)
│   │   │   ├── ArrayField (Dynamic lists)
│   │   │   └── ObjectField (Nested forms)
│   │   └── Modals (Add/Edit/Import/Export)
│   └── PreviewPane (Live YAML output)
├── ValidationSummaryPanel (Error/warning display)
├── QuickActionsBar (Context-sensitive actions)
└── ErrorBoundary (Graceful error handling)
```

### Data Flow

```
1. Schema Loading
   Remote URL (GitHub raw)
      ↓ (5s timeout)
   Bundled Fallback ──→ JSON Schema
      ↓
   parseSchema() ──→ Internal Model (TypeScript)
      ↓
   SchemaContext (React Context + Cache)
      ↓
   Components consume via useSchema()

2. Form State
   User edits form field
      ↓
   react-hook-form updates state
      ↓
   Zustand store (useEditorState)
      ↓
   YAML serializer (state → YAML string)
      ↓
   Preview pane + Validation

3. Save Flow
   User clicks Save
      ↓
   Validate against schema (Ajv)
      ↓
   POST /api/editor/config
      ↓
   Go backend creates backup (.backup.{timestamp})
      ↓
   Atomic write to azure.yaml
      ↓
   Dashboard auto-reloads
```

### State Management

**Zustand Store** (`useEditorState`)
- Current configuration (parsed YAML)
- Dirty state (unsaved changes)
- Selected navigation path
- Validation errors
- Backup list

**react-hook-form**
- Individual form field values
- Field-level validation
- Form submission handling

**Why Zustand over Context?**
- Better performance (no unnecessary re-renders)
- DevTools support for debugging
- Simpler API for complex state updates
- Type-safe selectors

## Backend Integration

### Go API Endpoints

**Configuration**
- `GET /api/editor/config` - Load current azure.yaml
- `POST /api/editor/config` - Save configuration (auto-creates backup)

**Backups**
- `GET /api/editor/backups` - List all backups (sorted newest first)
- `GET /api/editor/backups/{timestamp}` - Get backup content
- `POST /api/editor/backups/{timestamp}/restore` - Restore backup
- `DELETE /api/editor/backups/{timestamp}` - Delete backup
- `POST /api/editor/backups` - Create manual backup

**Schema & Validation**
- `GET /api/editor/schema` - Get azure.yaml JSON Schema
- `POST /api/editor/validate` - Validate YAML content
- `GET /api/editor/well-known-services` - Get service templates

**Implementation**: `cli/src/internal/dashboard/editor_handlers.go`

### API Client

**Location**: `cli/dashboard/src/lib/editor/config-api.ts`

All API functions are strongly typed with TypeScript interfaces:
```typescript
loadConfig(): Promise<ConfigResponse>
saveConfig(content: string): Promise<SaveConfigResponse>
validateConfig(content: string): Promise<ValidationResponse>
listBackups(): Promise<BackupsListResponse>
// ... etc
```

## Schema System

### Architecture

**Components**:
1. `schema-loader.ts` - Loads JSON Schema (remote + bundled fallback)
2. `schema-parser.ts` - Parses JSON Schema → TypeScript model
3. `SchemaContext.tsx` - React context provider for schema state
4. `bundled-schema.json` - Local copy for offline use

**Parsed Schema Structure**:
```typescript
{
  name: string
  properties: Record<string, SchemaProperty>
  required: string[]
  definitions: Record<string, SchemaProperty>
}

SchemaProperty: {
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array' | 'enum'
  title?: string
  description?: string  // Used for tooltips
  required: boolean
  defaultValue?: unknown
  validation: ValidationRule[]
  enumValues?: string[]  // For dropdowns
  properties?: Record<string, SchemaProperty>  // For objects
  items?: SchemaProperty  // For arrays
}
```

**Usage**:
```typescript
import { useSchema } from '@/contexts/SchemaContext'

const { schema, isLoading, error } = useSchema()
const hostProperty = getPropertyByPath(schema, 'services.api.host')
// hostProperty.enumValues = ['local', 'containerapp', 'appservice', ...]
```

### Form Generation

**Schema → UI Mapping**:
- `string` → `<StringField>` (text input)
- `number` → `<NumberField>` (number input)
- `boolean` → `<BooleanField>` (toggle switch)
- `enum` → `<EnumField>` (dropdown)
- `array` → `<ArrayField>` (add/remove items)
- `object` → `<ObjectField>` (nested form)

**FieldRenderer** dispatches based on schema property type.

## Testing Strategy

### Unit Tests (Vitest + Testing Library)

**Coverage**: ~85-90% for core components

**Location**: Co-located with components (`*.test.tsx`)

**Key test utilities**:
- `TestWrapper` - Provides react-hook-form context
- `mockConfigApi()` - Mocks API responses
- Mock fixtures in `src/mocks/`

**Run**: `cd cli/dashboard && pnpm test`

### Integration Tests

**Purpose**: Test frontend/backend communication with real Go server

**Location**: `cli/dashboard/src/lib/editor/config-api-integration.test.ts`

**Setup**:
1. Build server: `cd cli && ./build.ps1`
2. Tests spawn server on port 3333
3. Make real HTTP requests
4. Verify responses match contracts

**Skipped by default**: Require `TEST_INTEGRATION=true` environment variable

**Run**: `pnpm test:integration`

**Test categories**:
- Configuration load/save (round-trip, line endings)
- Validation (schema violations, missing fields)
- Backup operations (create, list, restore, delete)
- Error handling (network failures, corruption)

### E2E Tests (Playwright)

**Coverage**: 22 comprehensive test files covering all features

**Location**: `cli/dashboard/e2e/editor/`

**Organization**:
- `01-navigation.spec.ts` - Navigation tree
- `02-schema-forms.spec.ts` - Form generation
- `03-services.spec.ts` - Service management
- `04-resources.spec.ts` - Resource management
- `05-healthchecks.spec.ts` - Health checks
- `06-hooks.spec.ts` - Lifecycle hooks
- `07-env-ports.spec.ts` - Environment variables
- `08-test-config.spec.ts` - Test configuration
- ... (22 total)

**Test helpers** (`e2e/helpers/test-setup.ts`):
- `setupTest()` - Mock API, load fixtures
- `navigateToEditor()` - Navigate to editor page
- `addServiceViaForm()` - Add service through UI
- `editYamlDirectly()` - Edit YAML textarea
- `waitForValidation()` - Wait for validation pass
- `expectValidationError()` - Assert specific error

**Fixtures** (`e2e/fixtures/`):
- `comprehensive-azure-yaml.yaml` - Full feature test
- `minimal-azure-yaml.yaml` - Minimal valid config
- `service-configs.json` - Service templates
- `resource-configs.json` - Resource templates

**Test project**: `cli/tests/projects/editor-e2e-test/`
- Complete azure.yaml exercising all schema features
- Minimal source files for services
- Hook scripts and infrastructure templates

**Run**: `pnpm test:e2e editor/`

## Development Workflows

### Local Development Setup

**Start Backend**:
```bash
cd cli
./build.ps1  # or ./build.sh on Unix
./bin/azd.exe monitor --port 3333 --cwd .
```

**Start Frontend**:
```bash
cd cli/dashboard
pnpm install
pnpm dev  # Starts on http://localhost:5173
```

**Access Editor**: Navigate to dashboard, click "Edit Configuration"

### Adding a New Form Field Type

1. **Create field component**: `src/components/editor/forms/fields/MyField.tsx`
2. **Add to FieldRenderer**: Update switch statement in `FieldRenderer.tsx`
3. **Update schema parser**: Handle new type in `schema-parser.ts` if needed
4. **Add unit tests**: `MyField.test.tsx`
5. **Add to Storybook**: `MyField.stories.tsx` (if using)

### Adding a New Well-Known Service

1. **Update backend**: Add to `internal/wellknown/services.go`
2. **Add mock data**: Update `src/mocks/editorApiMocks.ts`
3. **Test integration**: Verify in UI via Quick Actions bar
4. **Add E2E test**: Update `e2e/editor/03-services.spec.ts`

### Adding a New Validation Rule

**Schema validation** (automatic):
- Update `schemas/v1.1/azure.yaml.json`
- Editor picks up changes automatically

**Custom business rules**:
1. Add to `src/lib/editor/validation-rules.ts`
2. Call in `ValidationSummaryPanel.tsx`
3. Add unit test in `validation-rules.test.ts`

## File Locations Reference

### Core Components
```
cli/dashboard/src/
├── components/editor/
│   ├── YamlEditor.tsx              # Main integration
│   ├── YamlEditorHeader.tsx        # Top bar with actions
│   ├── YamlEditorLayout.tsx        # 3-pane layout
│   ├── NavigationSidebar.tsx       # Tree navigation
│   ├── NavigationItem.tsx          # Tree nodes
│   ├── NavigationSearch.tsx        # Fuzzy search
│   ├── PreviewPane.tsx             # Live YAML preview
│   ├── ValidationSummaryPanel.tsx  # Error display
│   ├── QuickActionsBar.tsx         # Bottom actions
│   ├── CommandPalette.tsx          # Cmd+K palette
│   ├── forms/
│   │   ├── SchemaForm.tsx          # Main form renderer
│   │   ├── FieldRenderer.tsx       # Field type dispatcher
│   │   ├── FieldLabel.tsx          # Label with tooltip
│   │   ├── FieldError.tsx          # Error display
│   │   └── fields/
│   │       ├── StringField.tsx
│   │       ├── NumberField.tsx
│   │       ├── BooleanField.tsx
│   │       ├── EnumField.tsx
│   │       ├── ArrayField.tsx
│   │       └── ObjectField.tsx
│   └── modals/
│       ├── AddServiceModal.tsx
│       ├── ImportModal.tsx
│       └── BackupManagerModal.tsx
```

### Schema System
```
├── lib/schema/
│   ├── schema-loader.ts            # Remote + bundled loading
│   ├── schema-parser.ts            # JSON Schema → TypeScript
│   └── bundled-schema.json         # Offline fallback
├── contexts/
│   └── SchemaContext.tsx           # React context provider
```

### API Integration
```
├── lib/editor/
│   ├── config-api.ts               # API client functions
│   ├── yaml-serializer.ts          # State → YAML
│   └── validation-rules.ts         # Custom validation
```

### State Management
```
├── hooks/
│   └── useEditorState.ts           # Zustand store
```

### Testing
```
├── components/editor/**/*.test.tsx  # Unit tests (co-located)
├── lib/editor/config-api-integration.test.ts  # Integration tests
└── e2e/editor/*.spec.ts            # E2E tests (22 files)
```

### Backend
```
cli/src/internal/dashboard/
├── editor_handlers.go              # API endpoints
└── wellknown/
    └── services.go                 # Service templates
```

## Key Technical Decisions

### Why Schema-Driven Forms?

**Alternative**: Hardcode forms for each azure.yaml property

**Decision**: Generate forms from JSON Schema

**Rationale**:
- Single source of truth (schema drives UI + validation)
- Automatic support for schema updates
- Consistent validation logic
- Reduced code duplication
- Better type safety

### Why Backup-Before-Write?

**Alternative**: Git-style version control

**Decision**: Timestamped backups on every save

**Rationale**:
- Simple, predictable behavior
- No Git dependency
- Easy rollback for users
- Automatic cleanup (limit to 10 backups)
- Works even without version control

### Why React Hook Form + Zustand?

**Alternative**: Context API only, or Redux

**Decision**: react-hook-form for field state, Zustand for global state

**Rationale**:
- react-hook-form: Excellent performance, minimal re-renders, built-in validation
- Zustand: Simple API, no boilerplate, TypeScript-first
- Separation of concerns: Fields (RHF) vs. app state (Zustand)

## Common Issues & Solutions

### Schema Not Loading

**Symptom**: Editor shows loading spinner indefinitely

**Causes**:
1. Network timeout fetching remote schema
2. Bundled schema missing/corrupt
3. Schema parse error

**Debug**:
```typescript
// Check schema loading
const { schema, error, source } = useSchema()
console.log('Schema source:', source)  // 'remote' | 'bundled'
console.log('Schema error:', error)
```

**Fix**: Bundled schema should always work as fallback. If not, check:
- `src/lib/schema/bundled-schema.json` exists
- JSON is valid
- Schema matches expected structure

### Form Validation Not Working

**Symptom**: Invalid values accepted or valid values rejected

**Causes**:
1. Schema validation rules incorrect
2. Custom validation rules conflicting
3. Field type mismatch

**Debug**:
```typescript
// Check parsed schema property
const property = getPropertyByPath(schema, 'services.api.host')
console.log('Validation rules:', property.validation)
```

**Fix**:
- Verify schema has correct validation (pattern, min, max, etc.)
- Check custom validation in `validation-rules.ts`
- Ensure field type matches schema type

### Backup Restore Failed

**Symptom**: Error when restoring backup

**Causes**:
1. Backup file missing/corrupted
2. Backup contains invalid YAML
3. Permissions issue

**Debug**: Check backup list API response and file existence

**Fix**:
- Backups stored in `.azd/backups/` with timestamp naming
- Verify file exists and is readable
- Check YAML validity before restore

## Performance Considerations

### Large Configurations (100+ services)

**Optimizations applied**:
- Virtual scrolling for navigation tree (react-window)
- Lazy loading for editor sections (React.lazy)
- Debounced validation (500ms)
- Memoized schema parsing (useMemo)
- Code-split components (dynamic imports)

**Targets**:
- Initial load: <500ms
- Field update: <50ms
- Save operation: <1s

### Validation Performance

**Strategy**:
- Schema validation: Cached Ajv validator
- Debounced: 500ms delay after last change
- Async: Non-blocking UI updates
- Progressive: Field-level → form-level → global

## Accessibility (WCAG AA)

**Keyboard Navigation**:
- Tab through all interactive elements
- Cmd/Ctrl+K for command palette
- Escape to close modals
- Enter to submit forms

**Screen Reader Support**:
- ARIA labels on all buttons/inputs
- Live regions for validation errors
- Announcements for save/restore actions

**Visual Accessibility**:
- 4.5:1 minimum contrast ratio
- Focus indicators (2px outline)
- No color-only error indicators (icon + text)

**Testing**: All components pass axe-core WCAG AA tests

## Related Documentation

- **Requirements**: [spec.md](spec.md) - Feature requirements and user stories
- **Tasks**: [tasks.md](tasks.md) - Development task tracking
- **Testing**: [testing-plan.md](testing-plan.md) - Test strategy and coverage
- **Accessibility**: [accessibility.md](accessibility.md) - WCAG AA implementation
- **Schema Comparison**: [schema-comparison.md](schema-comparison.md) - v1.0 vs v1.1 analysis

## Quick Reference

### Start Development
```bash
# Terminal 1: Backend
cd cli && ./build.ps1 && ./bin/azd.exe monitor

# Terminal 2: Frontend
cd cli/dashboard && pnpm dev
```

### Run Tests
```bash
# Unit tests
pnpm test

# Integration tests (requires server build)
pnpm test:integration

# E2E tests
pnpm test:e2e editor/

# All tests with coverage
pnpm test:coverage
```

### Build for Production
```bash
cd cli/dashboard
pnpm build
# Output: cli/dashboard/dist/
```

### Debug Tips
```bash
# Enable verbose logging
DEBUG=azd:editor pnpm dev

# Check schema loading
# Open DevTools → Console → Look for schema source

# Inspect Zustand state
# Install Redux DevTools extension
# State visible in Redux DevTools
```
