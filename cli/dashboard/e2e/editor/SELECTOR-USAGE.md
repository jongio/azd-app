# Selector Registry Usage Guide

## Overview

The selector registry (`selectors.ts`) provides centralized, reliable selectors for all editor UI elements. All selectors use multiple fallback patterns for maximum reliability.

## Importing Selectors

```typescript
import * as selectors from './selectors'
// or
import { navigation, header, modals, forms } from './selectors'
```

## Usage Examples

### Navigation

```typescript
// Navigate to Services tab
const servicesTab = page.locator(selectors.tabs.services).first()
await servicesTab.click()

// Find a service in navigation
const webService = page.locator(selectors.navigation.item('web')).first()
await webService.click()

// Click Add Service button in navigation
const addButton = page.locator(selectors.navigation.addService).first()
await addButton.click()
```

### Header Buttons

```typescript
// Click Add Service button in header
const addServiceBtn = page.locator(selectors.header.addService).first()
await addServiceBtn.click()

// Click Save button
const saveBtn = page.locator(selectors.header.save).first()
await saveBtn.click()
```

### Modals

```typescript
// Wait for Add Service modal
const modal = page.locator(selectors.modals.dialog).first()
await modal.waitFor({ state: 'visible' })

// Fill service name in modal
const nameInput = modal.locator(selectors.modals.addService.nameInput).first()
await nameInput.fill('my-service')

// Select host type
const hostSelect = modal.locator(selectors.modals.addService.hostSelect).first()
await hostSelect.selectOption('appservice')

// Click save button (inside modal, not the opener)
const saveButton = modal.locator(selectors.modals.addService.saveButton).first()
await saveButton.click({ force: true })
```

### Forms

```typescript
// Fill text input
const nameField = page.locator(selectors.forms.textInput('name')).first()
await nameField.fill('value')

// Select dropdown
const hostSelect = page.locator(selectors.forms.select('host')).first()
await hostSelect.selectOption('containerapp')

// Toggle switch
const toggle = page.locator(selectors.forms.switch('enabled')).first()
await toggle.click()
```

### Combining Selectors

```typescript
// Use combineSelectors for multiple fallbacks
import { combineSelectors } from './selectors'

const button = page.locator(
  combineSelectors(
    'button:has-text("Save")',
    '[data-testid="save-button"]',
    'button[type="submit"]'
  )
).first()
```

## Best Practices

1. **Always use `.first()`** - Selectors may match multiple elements
2. **Use defensive checks** - `isVisible().catch(() => false)`
3. **Wait for modals** - Always wait for `[role="dialog"]` before interacting
4. **Use modal-scoped selectors** - Find buttons inside modal, not page-wide
5. **Combine with helper functions** - Use helpers that already use selectors

## Selector Priority

When multiple selectors are available, they're ordered by reliability:

1. **data-testid** - Most reliable (if available)
2. **aria-label/aria-labelledby** - Semantic and accessible
3. **role attributes** - Semantic HTML
4. **Text content** - `:has-text()` matcher
5. **Class names** - Last resort (fragile)

## Updating Selectors

If UI changes:
1. Update selector in `selectors.ts`
2. Add fallback patterns
3. Test with a few key tests
4. Update all tests if needed

## Common Patterns

### Modal Interaction Pattern

```typescript
// 1. Wait for modal
const modal = page.locator(selectors.modals.dialog).first()
await modal.waitFor({ state: 'visible', timeout: 5000 })

// 2. Interact with fields inside modal
const field = modal.locator(selectors.modals.addService.nameInput).first()
await field.fill('value')

// 3. Click button inside modal (not opener)
const saveBtn = modal.locator(selectors.modals.addService.saveButton).first()
await saveBtn.click({ force: true }) // Force to bypass backdrop
```

### Navigation Pattern

```typescript
// 1. Try tab first (for main sections)
const tab = page.locator(selectors.tabs.services).first()
if (await tab.isVisible({ timeout: 2000 }).catch(() => false)) {
  await tab.click()
}

// 2. Fallback to navigation tree
const treeItem = page.locator(selectors.navigation.item('services')).first()
if (await treeItem.isVisible({ timeout: 2000 }).catch(() => false)) {
  await treeItem.click()
}
```

### Form Field Pattern

```typescript
// Try multiple selectors
const field = page.locator(
  combineSelectors(
    selectors.forms.textInput('name'),
    `input[id*="name" i]`,
    `input[placeholder*="name" i]`
  )
).first()

if (await field.isVisible({ timeout: 2000 }).catch(() => false)) {
  await field.fill('value')
}
```
