# Phase 2: UI Discovery and Selector Updates - Progress

## Completed ✅

### 1. UI Structure Discovery
- ✅ Identified main editor component: `YamlEditor.tsx`
- ✅ Found navigation structure: `NavigationSidebar.tsx` with `role="navigation"` and `role="treeitem"`
- ✅ Found tab structure: `role="tablist"` with `aria-label="Azure YAML Editor views"`
- ✅ Found modal structure: `role="dialog"` for all modals
- ✅ Found header buttons: Add Service, Save, Cancel, Preview, Help, etc.
- ✅ Found form structure: Standard HTML inputs with React Hook Form

### 2. Selector Registry Created ✅
- ✅ Created `selectors.ts` with comprehensive selector registry
- ✅ Organized by component (navigation, header, modals, forms, etc.)
- ✅ Multiple fallback patterns for each selector
- ✅ Priority: data-testid → aria-label → role → text → class

### 3. Key Findings

**Navigation:**
- Uses `role="navigation"` with `aria-label="Azure YAML Editor Navigation"`
- Items use `role="treeitem"` with `aria-label` containing label text
- Tabs use `role="tablist"` with `role="tab"` for individual tabs
- Add buttons: `[role="treeitem"][aria-label="New service"]` or `button:has-text("New Service")`

**Header:**
- "Add Service" button: `button:has-text("Add Service")`
- "Save" button: `button:has-text("Save")`
- "Cancel" button: `button:has-text("Cancel")`
- Preview toggle: `button:has-text("Preview")`

**Modals:**
- All modals use `role="dialog"`
- Add Service Modal has tabs: Well-Known, Application, Container
- Form fields use standard `input[name="..."]` or `select[name="..."]`
- Save button: `button:has-text("Add Service")` or `button[type="submit"]` inside modal

**Forms:**
- Service name: `input[id="app-service-name"]` or `input[name="name"]`
- Host select: `select[name="host"]`
- Project input: `input[name="project"]`
- Language select: `select[name="language"]`

## Next Steps

### 1. Update Helper Functions
Update `test-setup.ts` helpers to use selector registry:
- `navigateToSection()` - Use `tabs.tab()` or `navigation.item()`
- `expandSection()` - Use `navigation.item()` with expand logic
- `findInNavigation()` - Use `navigation.item()`
- `addServiceViaForm()` - Use `modals.addService.*` selectors
- `addResourceViaForm()` - Use `modals.addResource.*` selectors
- `fillFormField()` - Use `forms.textInput()`, `forms.select()`, etc.

### 2. Update Test Files
Update all 22 test files to:
- Import selector registry
- Use selectors from registry instead of hardcoded strings
- Add defensive checks where needed

### 3. Test Selector Accuracy
Run a few tests to verify selectors work:
- Navigation tests (already passing)
- One service test
- One resource test

## Selector Patterns Identified

### Navigation Items
```typescript
// Main sections (tabs)
'[role="tab"]:has-text("Services")'

// Navigation tree items
'[role="treeitem"][aria-label*="Services" i]'

// Specific service/resource
'[role="treeitem"][aria-label*="web" i]'
```

### Modal Interactions
```typescript
// Wait for modal
'[role="dialog"]'

// Find button inside modal
modal.locator('button:has-text("Add Service"):not([aria-hidden="true"])')
```

### Form Fields
```typescript
// Service name in Add Service modal
'input[id="app-service-name"], input[name="name"]'

// Host select
'select[name="host"]'

// Project path
'input[name="project"]'
```

## Files Modified

1. ✅ `e2e/editor/selectors.ts` - Created selector registry
2. ⏳ `e2e/helpers/test-setup.ts` - Update helpers (next step)
3. ⏳ Test files - Update to use selectors (after helpers)

## Estimated Time Remaining

- Update helpers: 2-3 hours
- Update test files: 4-6 hours
- Testing and fixes: 2-3 hours
- **Total: 8-12 hours**
