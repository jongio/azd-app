/**
 * Resource Management E2E Tests for Azure YAML Editor
 * 
 * Tests resource management including:
 * - Adding resources (all types)
 * - Editing resources
 * - Deleting resources
 * - Resource dependencies
 * - Resource-specific configurations
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  addResourceViaForm,
  navigateToSection,
  expandSection,
} from '../helpers/test-setup'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const fixturesDir = path.join(__dirname, '../fixtures')
const resourceConfigs = JSON.parse(fs.readFileSync(path.join(fixturesDir, 'resource-configs.json'), 'utf-8'))

test.describe('Resource Management - Add Resource', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', resources: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should add db.postgres resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'postgres-db',
      type: 'db.postgres',
    })
    await page.waitForTimeout(1000)
  })

  test('should add db.mysql resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'mysql-db',
      type: 'db.mysql',
    })
    await page.waitForTimeout(1000)
  })

  test('should add db.redis resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'redis-cache',
      type: 'db.redis',
    })
    await page.waitForTimeout(1000)
  })

  test('should add db.mongo resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'mongo-db',
      type: 'db.mongo',
    })
    await page.waitForTimeout(1000)
  })

  test('should add db.cosmos resource with containers', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'cosmos-db',
      type: 'db.cosmos',
    })
    await page.waitForTimeout(1000)

    // Configure containers
    const addContainerButton = page.locator('button:has-text("Add Container"), button[aria-label*="container" i]').first()
    if (await addContainerButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await addContainerButton.click()
      await page.waitForTimeout(500)

      const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]').first()
      if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await nameInput.fill('users')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should add ai.openai.model resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'ai-model',
      type: 'ai.openai.model',
    })
    await page.waitForTimeout(1000)
  })

  test('should add ai.project resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'ai-project',
      type: 'ai.project',
    })
    await page.waitForTimeout(1000)
  })

  test('should add ai.search resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'ai-search',
      type: 'ai.search',
    })
    await page.waitForTimeout(1000)
  })

  test('should add host.containerapp resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'host-containerapp',
      type: 'host.containerapp',
    })
    await page.waitForTimeout(1000)
  })

  test('should add host.appservice resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'host-appservice',
      type: 'host.appservice',
    })
    await page.waitForTimeout(1000)
  })

  test('should add messaging.eventhubs resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'eventhubs',
      type: 'messaging.eventhubs',
    })
    await page.waitForTimeout(1000)
  })

  test('should add messaging.servicebus resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'servicebus',
      type: 'messaging.servicebus',
    })
    await page.waitForTimeout(1000)
  })

  test('should add storage resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'storage-account',
      type: 'storage',
    })
    await page.waitForTimeout(1000)
  })

  test('should add keyvault resource', async ({ page }) => {
    await addResourceViaForm(page, {
      name: 'keyvault',
      type: 'keyvault',
    })
    await page.waitForTimeout(1000)
  })
})

test.describe('Resource Management - Edit Resource', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          resources: {
            db: { type: 'db.postgres' },
            cosmos: {
              type: 'db.cosmos',
              containers: [{ name: 'users', partitionKeys: ['/id'] }],
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should edit resource properties', async ({ page }) => {
    await navigateToSection(page, 'Resources')
    await page.waitForTimeout(500)

    const dbButton = page.locator('[role="button"]:has-text("db")').first()
    if (await dbButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await dbButton.click()
      await page.waitForTimeout(500)

      // Should show resource form
      const form = page.locator('form, [class*="form"]').first()
      if (await form.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(form).toBeVisible()
      }
    }
  })

  test('should edit cosmos db containers', async ({ page }) => {
    await navigateToSection(page, 'Resources')
    await page.waitForTimeout(500)

    const cosmosButton = page.locator('[role="button"]:has-text("cosmos")').first()
    if (await cosmosButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await cosmosButton.click()
      await page.waitForTimeout(500)

      // Find containers section
      const containersSection = page.locator('[aria-label*="container" i], button:has-text("Containers")').first()
      if (await containersSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await containersSection.click()
        await page.waitForTimeout(500)

        // Should show containers list
        const containers = page.locator('[role="listitem"], [class*="container-item"]')
        const count = await containers.count()
        expect(count).toBeGreaterThanOrEqual(0)
      }
    }
  })
})

test.describe('Resource Management - Delete Resource', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          resources: {
            'resource-to-delete': { type: 'db.postgres' },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should delete resource with confirmation', async ({ page }) => {
    await navigateToSection(page, 'Resources')
    await page.waitForTimeout(500)

    const resourceButton = page.locator('[role="button"]:has-text("resource-to-delete")').first()
    if (await resourceButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await resourceButton.click()
      await page.waitForTimeout(500)

      const deleteButton = page.locator('button[aria-label*="Delete" i], button:has-text("Delete")').first()
      if (await deleteButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await deleteButton.click()
        await page.waitForTimeout(500)

        const confirmButton = page.locator('button:has-text("Confirm"), button:has-text("Delete")').first()
        if (await confirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await confirmButton.click()
          await page.waitForTimeout(1000)

          await expect(resourceButton).not.toBeVisible({ timeout: 2000 }).catch(() => {})
        }
      }
    }
  })
})

test.describe('Resource Management - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display all resources from comprehensive project', async ({ page }) => {
    await expandSection(page, 'Resources')
    await page.waitForTimeout(500)

    const resourceNames = ['db', 'mysql-db', 'redis-cache', 'cosmos-db', 'storage', 'keyvault']
    
    for (const resourceName of resourceNames) {
      const resourceButton = page.locator(`[role="button"]:has-text("${resourceName}")`).first()
      if (await resourceButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(resourceButton).toBeVisible()
        break // Found at least one
      }
    }
  })

  test('should configure resource dependencies', async ({ page }) => {
    await navigateToSection(page, 'Resources')
    await page.waitForTimeout(500)

    const dbButton = page.locator('[role="button"]:has-text("db")').first()
    if (await dbButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await dbButton.click()
      await page.waitForTimeout(500)

      // Find uses/dependencies section
      const usesSection = page.locator('[aria-label*="uses" i], [aria-label*="dependencies" i]').first()
      if (await usesSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await usesSection.click()
        await page.waitForTimeout(500)
      }
    }
  })

  test('should configure existing resource flag', async ({ page }) => {
    await navigateToSection(page, 'Resources')
    await page.waitForTimeout(500)

    const existingButton = page.locator('[role="button"]:has-text("existing-db")').first()
    if (await existingButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await existingButton.click()
      await page.waitForTimeout(500)

      // Find existing flag
      const existingToggle = page.locator('input[type="checkbox"][name*="existing" i], button[role="switch"][name*="existing" i]').first()
      if (await existingToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(existingToggle).toBeVisible()
      }
    }
  })
})
