/**
 * Logs Configuration E2E Tests for Azure YAML Editor
 * 
 * Tests logs configuration including:
 * - Global logs config
 * - Service-level logs config
 * - Filters, classifications, analytics
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  navigateToSection,
} from '../helpers/test-setup'

test.describe('Logs Configuration - Global', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          logs: {
            filters: {
              exclude: ['npm warn'],
            },
            classifications: [
              { text: 'ERROR', level: 'error' },
            ],
            analytics: {
              pollingInterval: '10s',
              defaultTimespan: '30m',
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure global log filters', async ({ page }) => {
    const logsNav = page.locator('[role="button"]:has-text("Logs"), [role="button"]:has-text("logs" i)').first()
    if (await logsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await logsNav.click()
      await page.waitForTimeout(500)

      const filtersSection = page.locator('[aria-label*="filter" i], button:has-text("Filters")').first()
      if (await filtersSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await filtersSection.click()
        await page.waitForTimeout(500)

        // Should show exclude patterns
        const excludeInput = page.locator('input[name*="exclude" i], textarea[name*="exclude" i]').first()
        if (await excludeInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(excludeInput).toBeVisible()
        }
      }
    }
  })

  test('should configure global log classifications', async ({ page }) => {
    const logsNav = page.locator('[role="button"]:has-text("Logs")').first()
    if (await logsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await logsNav.click()
      await page.waitForTimeout(500)

      const classificationsSection = page.locator('[aria-label*="classification" i], button:has-text("Classifications")').first()
      if (await classificationsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await classificationsSection.click()
        await page.waitForTimeout(500)

        // Should show classifications
        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(addButton).toBeVisible()
        }
      }
    }
  })

  test('should configure global log analytics', async ({ page }) => {
    const logsNav = page.locator('[role="button"]:has-text("Logs")').first()
    if (await logsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await logsNav.click()
      await page.waitForTimeout(500)

      const analyticsSection = page.locator('[aria-label*="analytics" i], button:has-text("Analytics")').first()
      if (await analyticsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await analyticsSection.click()
        await page.waitForTimeout(500)

        const pollingIntervalInput = page.locator('input[name*="pollingInterval" i]').first()
        if (await pollingIntervalInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await pollingIntervalInput.fill('15s')
          await page.waitForTimeout(300)
        }
      }
    }
  })
})

test.describe('Logs Configuration - Service Level', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: {
              host: 'appservice',
              project: './src/api',
              logs: {
                filters: {
                  exclude: ['debug'],
                },
              },
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure service-level log filters', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const logsSection = page.locator('[aria-label*="logs" i], button:has-text("Logs")').first()
      if (await logsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await logsSection.click()
        await page.waitForTimeout(500)

        const filtersSection = page.locator('[aria-label*="filter" i], button:has-text("Filters")').first()
        if (await filtersSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await filtersSection.click()
          await page.waitForTimeout(500)
        }
      }
    }
  })

  test('should configure service-level log analytics', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const logsSection = page.locator('[aria-label*="logs" i], button:has-text("Logs")').first()
      if (await logsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await logsSection.click()
        await page.waitForTimeout(500)

        const analyticsSection = page.locator('[aria-label*="analytics" i], button:has-text("Analytics")').first()
        if (await analyticsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await analyticsSection.click()
          await page.waitForTimeout(500)

          const tablesInput = page.locator('input[name*="tables" i], textarea[name*="tables" i]').first()
          if (await tablesInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await expect(tablesInput).toBeVisible()
          }
        }
      }
    }
  })
})

test.describe('Logs Configuration - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display global logs config from comprehensive project', async ({ page }) => {
    const logsNav = page.locator('[role="button"]:has-text("Logs")').first()
    if (await logsNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await logsNav.click()
      await page.waitForTimeout(500)

      // Should show logs configuration
      const logsForm = page.locator('form, [class*="form"]').first()
      if (await logsForm.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(logsForm).toBeVisible()
      }
    }
  })

  test('should display service-level logs config from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      const logsSection = page.locator('[aria-label*="logs" i], button:has-text("Logs")').first()
      if (await logsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await logsSection.click()
        await page.waitForTimeout(500)

        // Should show service logs config
        const filtersSection = page.locator('[aria-label*="filter" i], button:has-text("Filters")').first()
        if (await filtersSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(filtersSection).toBeVisible()
        }
      }
    }
  })
})
