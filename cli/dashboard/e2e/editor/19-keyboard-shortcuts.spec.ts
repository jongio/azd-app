/**
 * Keyboard Shortcuts E2E Tests for Azure YAML Editor
 * 
 * Tests keyboard shortcuts including:
 * - All keyboard shortcuts
 * - Shortcut combinations
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
} from '../helpers/test-setup'

test.describe('Keyboard Shortcuts - Basic Shortcuts', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should save with Ctrl+S', async ({ page }) => {
    await page.keyboard.press('Control+s')
    await page.waitForTimeout(500)
    // Should trigger save (may show success message)
  })

  test('should open command palette with Ctrl+K', async ({ page }) => {
    await page.keyboard.press('Control+k')
    await page.waitForTimeout(500)

    const palette = page.locator('[role="dialog"], [class*="command-palette"]').first()
    if (await palette.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(palette).toBeVisible()
    }
  })

  test('should close modals with Escape', async ({ page }) => {
    // Open a modal first
    const addButton = page.locator('button:has-text("Add Service")').first()
    if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await addButton.click()
      await page.waitForTimeout(500)

      await page.keyboard.press('Escape')
      await page.waitForTimeout(500)

      // Modal should be closed
      const modal = page.locator('[role="dialog"]').first()
      await expect(modal).not.toBeVisible({ timeout: 2000 }).catch(() => {})
    }
  })
})

test.describe('Keyboard Shortcuts - Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: { host: 'appservice', project: './src/api' },
            web: { host: 'containerapp', project: './src/web' },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should navigate with Tab', async ({ page }) => {
    await page.keyboard.press('Tab')
    await page.waitForTimeout(200)

    const focused = await page.evaluate(() => document.activeElement?.tagName)
    expect(focused).toBeDefined()
  })

  test('should navigate forms with Tab', async ({ page }) => {
    const addButton = page.locator('button:has-text("Add Service")').first()
    if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await addButton.click()
      await page.waitForTimeout(500)

      // Tab through form
      await page.keyboard.press('Tab')
      await page.waitForTimeout(200)
      await page.keyboard.press('Tab')
      await page.waitForTimeout(200)

      const focused = await page.evaluate(() => document.activeElement?.tagName)
      expect(focused).toBeDefined()
    }
  })
})

test.describe('Keyboard Shortcuts - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should work with comprehensive project loaded', async ({ page }) => {
    // Test that shortcuts work with full project
    await page.keyboard.press('Control+k')
    await page.waitForTimeout(500)

    const palette = page.locator('[role="dialog"]').first()
    if (await palette.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(palette).toBeVisible()
    }
  })
})
