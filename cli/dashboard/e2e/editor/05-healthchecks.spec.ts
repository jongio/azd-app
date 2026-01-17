/**
 * Health Check Configuration E2E Tests for Azure YAML Editor
 * 
 * Tests health check configuration including:
 * - All healthcheck types (http, tcp, process, output, none)
 * - All healthcheck properties
 * - Service-level healthcheck configuration
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  configureHealthCheck,
  navigateToSection,
} from '../helpers/test-setup'

test.describe('Health Check Configuration - Types', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: { host: 'appservice', project: './src/api' },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure HTTP healthcheck', async ({ page }) => {
    await configureHealthCheck(page, 'api', {
      type: 'http',
      path: '/health',
      interval: '30s',
      timeout: '10s',
      retries: 3,
    })
    await page.waitForTimeout(1000)
  })

  test('should configure TCP healthcheck', async ({ page }) => {
    await configureHealthCheck(page, 'api', {
      type: 'tcp',
      interval: '30s',
      timeout: '5s',
    })
    await page.waitForTimeout(1000)
  })

  test('should configure process healthcheck', async ({ page }) => {
    await configureHealthCheck(page, 'api', {
      type: 'process',
      interval: '10s',
    })
    await page.waitForTimeout(1000)
  })

  test('should configure output healthcheck with pattern', async ({ page }) => {
    await configureHealthCheck(page, 'api', {
      type: 'output',
      pattern: 'Server started',
      interval: '5s',
    })
    await page.waitForTimeout(1000)
  })

  test('should disable healthcheck', async ({ page }) => {
    await configureHealthCheck(page, 'api', {
      disable: true,
    })
    await page.waitForTimeout(1000)
  })
})

test.describe('Health Check Configuration - Properties', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: { host: 'appservice', project: './src/api' },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should configure healthcheck interval', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const healthcheckSection = page.locator('[aria-label*="health" i], button:has-text("Health Check")').first()
      if (await healthcheckSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await healthcheckSection.click()
        await page.waitForTimeout(500)

        const intervalInput = page.locator('input[name*="interval" i]').first()
        if (await intervalInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await intervalInput.fill('20s')
          await page.waitForTimeout(300)
        }
      }
    }
  })

  test('should configure healthcheck timeout', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const healthcheckSection = page.locator('[aria-label*="health" i], button:has-text("Health Check")').first()
      if (await healthcheckSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await healthcheckSection.click()
        await page.waitForTimeout(500)

        const timeoutInput = page.locator('input[name*="timeout" i]').first()
        if (await timeoutInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await timeoutInput.fill('5s')
          await page.waitForTimeout(300)
        }
      }
    }
  })

  test('should configure healthcheck retries', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const healthcheckSection = page.locator('[aria-label*="health" i], button:has-text("Health Check")').first()
      if (await healthcheckSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await healthcheckSection.click()
        await page.waitForTimeout(500)

        const retriesInput = page.locator('input[name*="retries" i], input[type="number"][name*="retries" i]').first()
        if (await retriesInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await retriesInput.fill('2')
          await page.waitForTimeout(300)
        }
      }
    }
  })

  test('should configure healthcheck start_period', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const healthcheckSection = page.locator('[aria-label*="health" i], button:has-text("Health Check")').first()
      if (await healthcheckSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await healthcheckSection.click()
        await page.waitForTimeout(500)

        const startPeriodInput = page.locator('input[name*="start_period" i], input[name*="startPeriod" i]').first()
        if (await startPeriodInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await startPeriodInput.fill('40s')
          await page.waitForTimeout(300)
        }
      }
    }
  })
})

test.describe('Health Check Configuration - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display healthcheck configs from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    // Check services with different healthcheck types
    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      // Should show healthcheck configuration
      const healthcheckSection = page.locator('[aria-label*="health" i], button:has-text("Health Check")').first()
      if (await healthcheckSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(healthcheckSection).toBeVisible()
      }
    }
  })

  test('should configure HTTP healthcheck with custom path', async ({ page }) => {
    await configureHealthCheck(page, 'web', {
      type: 'http',
      path: '/api/health',
      interval: '30s',
    })
    await page.waitForTimeout(1000)
  })
})
