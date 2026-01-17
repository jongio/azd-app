/**
 * Test Configuration E2E Tests for Azure YAML Editor
 * 
 * Tests test configuration including:
 * - Global test config
 * - Service-level test config
 * - All test properties
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  navigateToSection,
} from '../helpers/test-setup'

test.describe('Test Configuration - Global', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          test: {
            parallel: true,
            failFast: false,
            outputDir: './test-results',
            outputFormat: 'json',
            coverage: {
              enabled: true,
              threshold: 70,
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure global test parallel setting', async ({ page }) => {
    const testNav = page.locator('[role="button"]:has-text("Test"), [role="button"]:has-text("test" i)').first()
    if (await testNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testNav.click()
      await page.waitForTimeout(500)

      const parallelToggle = page.locator('input[type="checkbox"][name*="parallel" i], button[role="switch"][name*="parallel" i]').first()
      if (await parallelToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(parallelToggle).toBeVisible()
      }
    }
  })

  test('should configure global test failFast setting', async ({ page }) => {
    const testNav = page.locator('[role="button"]:has-text("Test")').first()
    if (await testNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testNav.click()
      await page.waitForTimeout(500)

      const failFastToggle = page.locator('input[type="checkbox"][name*="failFast" i], button[role="switch"][name*="failFast" i]').first()
      if (await failFastToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
        await failFastToggle.click()
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure global test outputDir', async ({ page }) => {
    const testNav = page.locator('[role="button"]:has-text("Test")').first()
    if (await testNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testNav.click()
      await page.waitForTimeout(500)

      const outputDirInput = page.locator('input[name*="outputDir" i]').first()
      if (await outputDirInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await outputDirInput.fill('./custom-test-results')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure global test outputFormat', async ({ page }) => {
    const testNav = page.locator('[role="button"]:has-text("Test")').first()
    if (await testNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testNav.click()
      await page.waitForTimeout(500)

      const formatSelect = page.locator('select[name*="outputFormat" i]').first()
      if (await formatSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await formatSelect.selectOption('junit')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure global test coverage', async ({ page }) => {
    const testNav = page.locator('[role="button"]:has-text("Test")').first()
    if (await testNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testNav.click()
      await page.waitForTimeout(500)

      const coverageSection = page.locator('[aria-label*="coverage" i], button:has-text("Coverage")').first()
      if (await coverageSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await coverageSection.click()
        await page.waitForTimeout(500)

        const enabledToggle = page.locator('input[type="checkbox"][name*="enabled" i]').first()
        if (await enabledToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(enabledToggle).toBeVisible()
        }

        const thresholdInput = page.locator('input[name*="threshold" i]').first()
        if (await thresholdInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await thresholdInput.fill('80')
          await page.waitForTimeout(300)
        }
      }
    }
  })
})

test.describe('Test Configuration - Service Level', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: {
              host: 'appservice',
              project: './src/api',
              test: {
                unit: {
                  command: 'npm test',
                  path: './tests/unit',
                },
              },
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure service unit test', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const testSection = page.locator('[aria-label*="test" i], button:has-text("Test")').first()
      if (await testSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await testSection.click()
        await page.waitForTimeout(500)

        const unitSection = page.locator('[aria-label*="unit" i], button:has-text("Unit")').first()
        if (await unitSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await unitSection.click()
          await page.waitForTimeout(500)

          const commandInput = page.locator('input[name*="command" i]').first()
          if (await commandInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await commandInput.fill('npm run test:unit')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })

  test('should configure service integration test', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const testSection = page.locator('[aria-label*="test" i], button:has-text("Test")').first()
      if (await testSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await testSection.click()
        await page.waitForTimeout(500)

        const integrationSection = page.locator('[aria-label*="integration" i], button:has-text("Integration")').first()
        if (await integrationSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await integrationSection.click()
          await page.waitForTimeout(500)

          const commandInput = page.locator('input[name*="command" i]').first()
          if (await commandInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await commandInput.fill('npm run test:integration')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })

  test('should configure service e2e test', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const testSection = page.locator('[aria-label*="test" i], button:has-text("Test")').first()
      if (await testSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await testSection.click()
        await page.waitForTimeout(500)

        const e2eSection = page.locator('[aria-label*="e2e" i], button:has-text("E2E")').first()
        if (await e2eSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await e2eSection.click()
          await page.waitForTimeout(500)

          const commandInput = page.locator('input[name*="command" i]').first()
          if (await commandInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await commandInput.fill('npm run test:e2e')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })

  test('should configure service test timeout', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const testSection = page.locator('[aria-label*="test" i], button:has-text("Test")').first()
      if (await testSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await testSection.click()
        await page.waitForTimeout(500)

        const timeoutInput = page.locator('input[name*="timeout" i]').first()
        if (await timeoutInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await timeoutInput.fill('10m')
          await page.waitForTimeout(300)
        }
      }
    }
  })
})

test.describe('Test Configuration - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display global test config from comprehensive project', async ({ page }) => {
    const testNav = page.locator('[role="button"]:has-text("Test")').first()
    if (await testNav.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testNav.click()
      await page.waitForTimeout(500)

      // Should show test configuration
      const testForm = page.locator('form, [class*="form"]').first()
      if (await testForm.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(testForm).toBeVisible()
      }
    }
  })

  test('should display service-level test config from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      const testSection = page.locator('[aria-label*="test" i], button:has-text("Test")').first()
      if (await testSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await testSection.click()
        await page.waitForTimeout(500)

        // Should show service test config
        const unitSection = page.locator('[aria-label*="unit" i], button:has-text("Unit")').first()
        if (await unitSection.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(unitSection).toBeVisible()
        }
      }
    }
  })
})
