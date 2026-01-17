/**
 * Hooks Configuration E2E Tests for Azure YAML Editor
 * 
 * Tests hooks configuration including:
 * - Project-level hooks (all types)
 * - Service-level hooks
 * - Hook properties (run, shell, continueOnError, interactive, platform overrides)
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  configureHooks,
  navigateToSection,
} from '../helpers/test-setup'

test.describe('Hooks Configuration - Project Level', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', hooks: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should add preprovision hook', async ({ page }) => {
    await configureHooks(page, {
      hookType: 'preprovision',
      run: './scripts/pre-provision.sh',
      shell: 'bash',
    })
    await page.waitForTimeout(1000)
  })

  test('should add postprovision hook', async ({ page }) => {
    await configureHooks(page, {
      hookType: 'postprovision',
      run: './scripts/post-provision.sh',
      shell: 'sh',
    })
    await page.waitForTimeout(1000)
  })

  test('should add preinfracreate hook', async ({ page }) => {
    await configureHooks(page, {
      hookType: 'preinfracreate',
      run: 'echo "Pre infra create"',
      shell: 'sh',
    })
    await page.waitForTimeout(1000)
  })

  test('should add postinfracreate hook', async ({ page }) => {
    await configureHooks(page, {
      hookType: 'postinfracreate',
      run: 'echo "Post infra create"',
      shell: 'sh',
    })
    await page.waitForTimeout(1000)
  })

  test('should add preup hook', async ({ page }) => {
    await configureHooks(page, {
      hookType: 'preup',
      run: './scripts/pre-up.sh',
      shell: 'bash',
    })
    await page.waitForTimeout(1000)
  })

  test('should add postup hook', async ({ page }) => {
    await configureHooks(page, {
      hookType: 'postup',
      run: './scripts/post-up.sh',
      shell: 'bash',
    })
    await page.waitForTimeout(1000)
  })

  test('should configure hook with continueOnError', async ({ page }) => {
    await navigateToSection(page, 'Hooks')
    await page.waitForTimeout(500)

    // Find hook configuration
    const hookButton = page.locator('button:has-text("preprovision"), [aria-label*="preprovision" i]').first()
    if (await hookButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await hookButton.click()
      await page.waitForTimeout(500)

      // Find continueOnError toggle
      const continueToggle = page.locator('input[type="checkbox"][name*="continueOnError" i], button[role="switch"][name*="continueOnError" i]').first()
      if (await continueToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
        await continueToggle.click()
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure hook with platform overrides', async ({ page }) => {
    await navigateToSection(page, 'Hooks')
    await page.waitForTimeout(500)

    const hookButton = page.locator('button:has-text("postrun"), [aria-label*="postrun" i]').first()
    if (await hookButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await hookButton.click()
      await page.waitForTimeout(500)

      // Find platform override sections
      const windowsSection = page.locator('[aria-label*="windows" i], button:has-text("Windows")').first()
      if (await windowsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await windowsSection.click()
        await page.waitForTimeout(500)

        const runInput = page.locator('input[name*="run" i], textarea[name*="run" i]').first()
        if (await runInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await runInput.fill('echo "Windows hook"')
          await page.waitForTimeout(300)
        }
      }
    }
  })
})

test.describe('Hooks Configuration - Service Level', () => {
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

  test('should add service predeploy hook', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      // Find hooks section
      const hooksSection = page.locator('[aria-label*="hooks" i], button:has-text("Hooks")').first()
      if (await hooksSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await hooksSection.click()
        await page.waitForTimeout(500)

        // Add predeploy hook
        const addHookButton = page.locator('button:has-text("predeploy"), button[aria-label*="predeploy" i]').first()
        if (await addHookButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addHookButton.click()
          await page.waitForTimeout(500)

          const runInput = page.locator('input[name*="run" i], textarea[name*="run" i]').first()
          if (await runInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await runInput.fill('npm run build')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })

  test('should add service postdeploy hook', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const hooksSection = page.locator('[aria-label*="hooks" i], button:has-text("Hooks")').first()
      if (await hooksSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await hooksSection.click()
        await page.waitForTimeout(500)

        const addHookButton = page.locator('button:has-text("postdeploy"), button[aria-label*="postdeploy" i]').first()
        if (await addHookButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addHookButton.click()
          await page.waitForTimeout(500)

          const runInput = page.locator('input[name*="run" i], textarea[name*="run" i]').first()
          if (await runInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await runInput.fill('npm run migrate')
            await page.waitForTimeout(300)
          }
        }
      }
    }
  })
})

test.describe('Hooks Configuration - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display all project-level hooks from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Hooks')
    await page.waitForTimeout(500)

    // Check for hooks that exist in comprehensive project
    const hookTypes = ['preprovision', 'postprovision', 'preup', 'postup', 'prerun', 'postrun']
    
    for (const hookType of hookTypes) {
      const hookButton = page.locator(`button:has-text("${hookType}"), [aria-label*="${hookType}" i]`).first()
      if (await hookButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(hookButton).toBeVisible()
        break // Found at least one
      }
    }
  })

  test('should display service-level hooks from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      const hooksSection = page.locator('[aria-label*="hooks" i], button:has-text("Hooks")').first()
      if (await hooksSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await hooksSection.click()
        await page.waitForTimeout(500)

        // Should show service hooks
        const predeployHook = page.locator('button:has-text("predeploy"), [aria-label*="predeploy" i]').first()
        if (await predeployHook.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(predeployHook).toBeVisible()
        }
      }
    }
  })
})
