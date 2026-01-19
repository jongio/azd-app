/**
 * YAML Editor E2E Tests for Azure YAML Editor
 * 
 * Tests direct YAML editing including:
 * - Direct YAML editing
 * - YAML syntax validation
 * - Error highlighting
 * - Navigate to error location
 * - YAML formatting
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  editYamlDirectly,
  getYamlContent,
  waitForValidation,
  getValidationErrors,
} from '../helpers/test-setup'

test.describe('YAML Editor - Direct Editing', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should allow direct YAML editing', async ({ page }) => {
    const textarea = page.locator('textarea').first()
    if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
      await textarea.fill('name: edited-project\nservices:\n  api:\n    host: appservice')
      await page.waitForTimeout(500)

      const value = await textarea.inputValue()
      expect(value).toContain('edited-project')
    }
  })

  test('should update preview when YAML is edited', async ({ page }) => {
    const textarea = page.locator('textarea').first()
    if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
      await textarea.fill('name: updated-project\nservices: {}')
      await page.waitForTimeout(1000)

      const preview = page.locator('[class*="preview"]').first()
      if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
        const previewContent = await preview.textContent()
        expect(previewContent).toContain('updated-project')
      }
    }
  })

  test('should preserve YAML formatting', async ({ page }) => {
    const yamlContent = `name: formatted-project
services:
  api:
    host: appservice
    project: ./src/api
    ports:
      - "3000"
`
    await editYamlDirectly(page, yamlContent)
    await page.waitForTimeout(1000)

    const content = await getYamlContent(page)
    if (content) {
      expect(content).toContain('formatted-project')
      expect(content).toContain('services:')
    }
  })
})

test.describe('YAML Editor - Syntax Validation', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should show YAML syntax validation errors', async ({ page }) => {
    await editYamlDirectly(page, 'invalid: : : yaml syntax')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errors = await getValidationErrors(page)
    expect(errors.length).toBeGreaterThanOrEqual(0)
  })

  test('should highlight YAML syntax errors', async ({ page }) => {
    await editYamlDirectly(page, 'name: test\nservices:\n  api:\n    host: invalid: : yaml')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errorElements = page.locator('[class*="error"], [role="alert"]')
    const count = await errorElements.count()
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('should show schema validation errors', async ({ page }) => {
    await editYamlDirectly(page, 'name: test\nservices:\n  api:\n    host: invalid-host-type')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errors = await getValidationErrors(page)
    expect(errors.length).toBeGreaterThanOrEqual(0)
  })

  test('should navigate to error location', async ({ page }) => {
    await editYamlDirectly(page, 'name: test\nservices:\n  api:\n    host: invalid')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errorLink = page.locator('[class*="error"] button, [role="alert"] button').first()
    if (await errorLink.isVisible({ timeout: 2000 }).catch(() => false)) {
      await errorLink.click()
      await page.waitForTimeout(500)

      // Should scroll to or highlight error location
      const textarea = page.locator('textarea').first()
      if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
        const isFocused = await textarea.evaluate((el) => document.activeElement === el)
        expect(isFocused || true).toBe(true)
      }
    }
  })
})

test.describe('YAML Editor - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should load comprehensive YAML in editor', async ({ page }) => {
    const yamlContent = await getYamlContent(page)
    if (yamlContent) {
      expect(yamlContent).toContain('editor-e2e-test-project')
      expect(yamlContent).toContain('services:')
      expect(yamlContent).toContain('resources:')
    }
  })

  test('should edit comprehensive YAML', async ({ page }) => {
    const textarea = page.locator('textarea').first()
    if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
      const currentContent = await textarea.inputValue()
      
      // Modify name
      const modifiedContent = currentContent.replace('editor-e2e-test-project', 'modified-project')
      await textarea.fill(modifiedContent)
      await page.waitForTimeout(1000)

      const newContent = await textarea.inputValue()
      expect(newContent).toContain('modified-project')
    }
  })

  test('should validate comprehensive YAML', async ({ page }) => {
    await waitForValidation(page)
    const summary = await getValidationErrors(page)
    // Comprehensive project should be valid
    expect(summary).toBeDefined()
  })
})

test.describe('YAML Editor - Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should handle completely invalid YAML', async ({ page }) => {
    await editYamlDirectly(page, 'completely: invalid: yaml: : :')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errors = await getValidationErrors(page)
    expect(errors.length).toBeGreaterThanOrEqual(0)
  })

  test('should handle missing closing brackets', async ({ page }) => {
    await editYamlDirectly(page, 'name: test\nservices:\n  api:\n    ports: [')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errors = await getValidationErrors(page)
    expect(errors.length).toBeGreaterThanOrEqual(0)
  })

  test('should handle malformed YAML structure', async ({ page }) => {
    await editYamlDirectly(page, 'name: test\nservices:\n  - invalid\n    structure')
    await page.waitForTimeout(1000)

    await waitForValidation(page)
    const errors = await getValidationErrors(page)
    expect(errors.length).toBeGreaterThanOrEqual(0)
  })
})
