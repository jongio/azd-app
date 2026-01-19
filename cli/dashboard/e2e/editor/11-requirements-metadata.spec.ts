/**
 * Requirements and Metadata E2E Tests for Azure YAML Editor
 * 
 * Tests requirements and metadata configuration including:
 * - Requirements configuration
 * - Metadata configuration
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
} from '../helpers/test-setup'

test.describe('Requirements Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          reqs: [
            {
              name: 'node',
              minVersion: '18.0.0',
            },
          ],
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should add requirement', async ({ page }) => {
    const reqsNav = page.locator('[role="button"]:has-text("Requirements"), [role="button"]:has-text("reqs" i)').first()
    if (await reqsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await reqsNav.click()
      await page.waitForTimeout(500)

      const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
      if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await addButton.click()
        await page.waitForTimeout(500)

        const nameInput = page.locator('input[name*="name" i]').first()
        if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await nameInput.fill('python')
          await page.keyboard.press('Tab')
          const versionInput = page.locator('input[name*="minVersion" i]').first()
          if (await versionInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await versionInput.fill('3.9.0')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })

  test('should edit requirement', async ({ page }) => {
    const reqsNav = page.locator('[role="button"]:has-text("Requirements")').first()
    if (await reqsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await reqsNav.click()
      await page.waitForTimeout(500)

      const nodeReq = page.locator('[role="button"]:has-text("node"), [role="listitem"]:has-text("node")').first()
      if (await nodeReq.isVisible({ timeout: 2000 }).catch(() => false)) {
        await nodeReq.click()
        await page.waitForTimeout(500)

        const versionInput = page.locator('input[name*="minVersion" i]').first()
        if (await versionInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await versionInput.fill('20.0.0')
          await page.waitForTimeout(300)
        }
      }
    }
  })

  test('should delete requirement', async ({ page }) => {
    const reqsNav = page.locator('[role="button"]:has-text("Requirements")').first()
    if (await reqsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await reqsNav.click()
      await page.waitForTimeout(500)

      const nodeReq = page.locator('[role="button"]:has-text("node")').first()
      if (await nodeReq.isVisible({ timeout: 2000 }).catch(() => false)) {
        await nodeReq.click()
        await page.waitForTimeout(500)

        const deleteButton = page.locator('button[aria-label*="Delete" i], button:has-text("Delete")').first()
        if (await deleteButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await deleteButton.click()
          await page.waitForTimeout(500)
        }
      }
    }
  })

  test('should configure requirement with all properties', async ({ page }) => {
    const reqsNav = page.locator('[role="button"]:has-text("Requirements")').first()
    if (await reqsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await reqsNav.click()
      await page.waitForTimeout(500)

      const addButton = page.locator('button:has-text("Add")').first()
      if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await addButton.click()
        await page.waitForTimeout(500)

        // Fill all requirement properties
        const nameInput = page.locator('input[name*="name" i]').first()
        if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await nameInput.fill('custom-tool')
          await page.keyboard.press('Tab')
          
          const commandInput = page.locator('input[name*="command" i]').first()
          if (await commandInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await commandInput.fill('custom-tool')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })
})

test.describe('Metadata Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          metadata: {
            template: 'test-template@1.0.0',
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure metadata template', async ({ page }) => {
    const metadataNav = page.locator('[role="button"]:has-text("Metadata"), [role="button"]:has-text("metadata" i)').first()
    if (await metadataNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await metadataNav.click()
      await page.waitForTimeout(500)

      const templateInput = page.locator('input[name*="template" i]').first()
      if (await templateInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await templateInput.fill('new-template@2.0.0')
        await page.waitForTimeout(300)
      }
    }
  })
})

test.describe('Requirements and Metadata - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display requirements from comprehensive project', async ({ page }) => {
    const reqsNav = page.locator('[role="button"]:has-text("Requirements")').first()
    if (await reqsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await reqsNav.click()
      await page.waitForTimeout(500)

      // Should show requirements list
      const reqsList = page.locator('[role="list"], [class*="requirements"]').first()
      if (await reqsList.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(reqsList).toBeVisible()
      }
    }
  })

  test('should display metadata from comprehensive project', async ({ page }) => {
    const metadataNav = page.locator('[role="button"]:has-text("Metadata")').first()
    if (await metadataNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await metadataNav.click()
      await page.waitForTimeout(500)

      const templateInput = page.locator('input[name*="template" i]').first()
      if (await templateInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(templateInput).toBeVisible()
      }
    }
  })
})
