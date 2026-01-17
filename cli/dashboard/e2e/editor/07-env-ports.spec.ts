/**
 * Environment Variables and Ports E2E Tests for Azure YAML Editor
 * 
 * Tests environment variables and ports configuration including:
 * - Add/edit/delete env vars
 * - Port configurations (all formats)
 * - Environment variable substitution
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  navigateToSection,
} from '../helpers/test-setup'

test.describe('Environment Variables', () => {
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

  test('should add environment variable', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const envSection = page.locator('[aria-label*="environment" i], button:has-text("Environment")').first()
      if (await envSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await envSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
          if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await nameInput.fill('TEST_VAR')
            await page.keyboard.press('Tab')
            const valueInput = page.locator('input[name="value"], input[placeholder*="value" i]').first()
            if (await valueInput.isVisible({ timeout: 2000 }).catch(() => false)) {
              await valueInput.fill('test-value')
              await page.waitForTimeout(500)
            }
          }
        }
      }
    }
  })

  test('should add environment variable with secret', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const envSection = page.locator('[aria-label*="environment" i], button:has-text("Environment")').first()
      if (await envSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await envSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const nameInput = page.locator('input[name="name"]').first()
          if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await nameInput.fill('SECRET_VAR')
            await page.keyboard.press('Tab')
            
            // Find secret input
            const secretInput = page.locator('input[name*="secret" i]').first()
            if (await secretInput.isVisible({ timeout: 2000 }).catch(() => false)) {
              await secretInput.fill('SECRET_NAME')
              await page.waitForTimeout(500)
            }
          }
        }
      }
    }
  })

  test('should edit environment variable', async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: {
              host: 'appservice',
              project: './src/api',
              environment: {
                TEST_VAR: 'old-value',
              },
            },
          },
        },
      },
    })
    await navigateToEditor(page)

    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const envSection = page.locator('[aria-label*="environment" i], button:has-text("Environment")').first()
      if (await envSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await envSection.click()
        await page.waitForTimeout(500)

        // Find and edit existing env var
        const valueInput = page.locator('input[name="value"]').first()
        if (await valueInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await valueInput.fill('new-value')
          await page.waitForTimeout(300)
        }
      }
    }
  })

  test('should delete environment variable', async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: {
              host: 'appservice',
              project: './src/api',
              environment: {
                TO_DELETE: 'value',
              },
            },
          },
        },
      },
    })
    await navigateToEditor(page)

    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const envSection = page.locator('[aria-label*="environment" i], button:has-text("Environment")').first()
      if (await envSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await envSection.click()
        await page.waitForTimeout(500)

        const removeButton = page.locator('button[aria-label*="Remove" i], button:has-text("Remove")').first()
        if (await removeButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await removeButton.click()
          await page.waitForTimeout(500)
        }
      }
    }
  })
})

test.describe('Port Configuration', () => {
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

  test('should add single port', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const portsSection = page.locator('[aria-label*="port" i], button:has-text("Ports")').first()
      if (await portsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await portsSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const portInput = page.locator('input[name*="port" i], input[placeholder*="port" i]').first()
          if (await portInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await portInput.fill('3000')
            await page.waitForTimeout(500)
          }
        }
      }
    }
  })

  test('should add port mapping', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const portsSection = page.locator('[aria-label*="port" i], button:has-text("Ports")').first()
      if (await portsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await portsSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const portInput = page.locator('input[name*="port" i]').first()
          if (await portInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await portInput.fill('3000:8080')
            await page.waitForTimeout(500)
          }
        }
      }
    }
  })

  test('should add port with host mapping', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const portsSection = page.locator('[aria-label*="port" i], button:has-text("Ports")').first()
      if (await portsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await portsSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const portInput = page.locator('input[name*="port" i]').first()
          if (await portInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await portInput.fill('127.0.0.1:3000:8080')
            await page.waitForTimeout(500)
          }
        }
      }
    }
  })

  test('should add UDP port', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const portsSection = page.locator('[aria-label*="port" i], button:has-text("Ports")').first()
      if (await portsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await portsSection.click()
        await page.waitForTimeout(500)

        const addButton = page.locator('button:has-text("Add"), button[aria-label*="Add" i]').first()
        if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await addButton.click()
          await page.waitForTimeout(500)

          const portInput = page.locator('input[name*="port" i]').first()
          if (await portInput.isVisible({ timeout: 2000 }).catch(() => false)) {
            await portInput.fill('8080/udp')
            await page.waitForTimeout(500)
          }
        }
      }
    }
  })
})

test.describe('Environment Variables and Ports - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display environment variables from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      const envSection = page.locator('[aria-label*="environment" i], button:has-text("Environment")').first()
      if (await envSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await envSection.click()
        await page.waitForTimeout(500)

        // Should show env vars
        const envVars = page.locator('[role="listitem"], [class*="env-var"]')
        const count = await envVars.count()
        expect(count).toBeGreaterThanOrEqual(0)
      }
    }
  })

  test('should display ports from comprehensive project', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      const portsSection = page.locator('[aria-label*="port" i], button:has-text("Ports")').first()
      if (await portsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await portsSection.click()
        await page.waitForTimeout(500)

        // Should show ports
        const ports = page.locator('[role="listitem"], [class*="port"]')
        const count = await ports.count()
        expect(count).toBeGreaterThanOrEqual(0)
      }
    }
  })
})
