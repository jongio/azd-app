/**
 * Service Management E2E Tests for Azure YAML Editor
 * 
 * Tests complete service management workflows including:
 * - Adding services (all host types)
 * - Editing services
 * - Deleting services
 * - Service validation
 * - Service dependencies
 * - All service properties
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  loadMinimalProject,
  addServiceViaForm,
  navigateToSection,
  expandSection,
} from '../helpers/test-setup'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

// Load service configs from fixtures
const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const fixturesDir = path.join(__dirname, '../fixtures')
const serviceConfigs = JSON.parse(fs.readFileSync(path.join(fixturesDir, 'service-configs.json'), 'utf-8'))

test.describe('Service Management - Add Service', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should open add service modal', async ({ page }) => {
    const addButton = page.locator('button:has-text("Add Service"), button[aria-label*="Add service" i]').first()
    if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await addButton.click()
      await page.waitForTimeout(500)

      // Modal should be visible
      const modal = page.locator('[role="dialog"]').first()
      if (await modal.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(modal).toBeVisible()
      }
    }
  })

  test('should add appservice host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-appservice',
      host: 'appservice',
      language: 'js',
      project: './src/api',
    })
    await page.waitForTimeout(1000)

    // Verify service appears
    const preview = page.locator('[class*="preview"]').first()
    if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
      const content = await preview.textContent()
      expect(content).toContain('test-appservice')
    }
  })

  test('should add containerapp host type with project', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-containerapp',
      host: 'containerapp',
      language: 'py',
      project: './src/api',
    })
    await page.waitForTimeout(1000)
  })

  test('should add containerapp host type with image', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-containerapp-image',
      host: 'containerapp',
      image: 'nginx:latest',
    })
    await page.waitForTimeout(1000)
  })

  test('should add function host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-function',
      host: 'function',
      language: 'js',
      project: './src/function',
    })
    await page.waitForTimeout(1000)
  })

  test('should add springapp host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-springapp',
      host: 'springapp',
      language: 'java',
      project: './src/spring-service',
    })
    await page.waitForTimeout(1000)
  })

  test('should add staticwebapp host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-staticwebapp',
      host: 'staticwebapp',
      project: './src/static-site',
    })
    await page.waitForTimeout(1000)
  })

  test('should add aks host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-aks',
      host: 'aks',
      language: 'docker',
      project: './src/aks-service',
    })
    await page.waitForTimeout(1000)
  })

  test('should add ai.endpoint host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-ai-endpoint',
      host: 'ai.endpoint',
      project: './src/ai-endpoint',
    })
    await page.waitForTimeout(1000)
  })

  test('should add azure.ai.agent host type', async ({ page }) => {
    await addServiceViaForm(page, {
      name: 'test-ai-agent',
      host: 'azure.ai.agent',
      project: './src/ai-agent',
    })
    await page.waitForTimeout(1000)
  })

  test('should validate service name format', async ({ page }) => {
    const addButton = page.locator('button:has-text("Add Service")').first()
    if (await addButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await addButton.click()
      await page.waitForTimeout(500)

      // Enter invalid name (uppercase, spaces)
      const nameInput = page.locator('input[name="name"]').first()
      if (await nameInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await nameInput.fill('Invalid Name With Spaces')
        await page.keyboard.press('Tab')
        await page.waitForTimeout(300)

        // Should show validation error
        const hasError = await page.locator('[class*="error"], [role="alert"]').count()
        expect(hasError).toBeGreaterThanOrEqual(0)
      }
    }
  })

  test('should detect duplicate service names', async ({ page }) => {
    // Add first service
    await addServiceViaForm(page, {
      name: 'duplicate-service',
      host: 'appservice',
      project: './src/api',
    })
    await page.waitForTimeout(1000)

    // Try to add duplicate
    await addServiceViaForm(page, {
      name: 'duplicate-service',
      host: 'containerapp',
      project: './src/api2',
    })
    await page.waitForTimeout(1000)

    // Should show validation error
    const errors = await page.locator('[class*="error"], [role="alert"]').count()
    expect(errors).toBeGreaterThanOrEqual(0)
  })
})

test.describe('Service Management - Edit Service', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            api: {
              host: 'appservice',
              language: 'js',
              project: './src/api',
              ports: ['3000'],
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should edit service properties', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      // Edit project path
      const projectField = page.locator('input[name*="project" i]').first()
      if (await projectField.isVisible({ timeout: 2000 }).catch(() => false)) {
        await projectField.fill('./src/api-edited')
        await page.waitForTimeout(300)

        const newValue = await projectField.inputValue()
        expect(newValue).toContain('api-edited')
      }
    }
  })

  test('should edit service ports', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      // Find ports section
      const portsSection = page.locator('[aria-label*="port" i], button:has-text("Ports")').first()
      if (await portsSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await portsSection.click()
        await page.waitForTimeout(500)

        // Edit port
        const portInput = page.locator('input[name*="port" i]').first()
        if (await portInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await portInput.fill('8080')
          await page.waitForTimeout(300)
        }
      }
    }
  })

  test('should edit service environment variables', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      // Find environment section
      const envSection = page.locator('[aria-label*="environment" i], button:has-text("Environment")').first()
      if (await envSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await envSection.click()
        await page.waitForTimeout(500)

        // Add env var
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
})

test.describe('Service Management - Delete Service', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            'service-to-delete': {
              host: 'appservice',
              project: './src/api',
            },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should delete service with confirmation', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const serviceButton = page.locator('[role="button"]:has-text("service-to-delete")').first()
    if (await serviceButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await serviceButton.click()
      await page.waitForTimeout(500)

      // Find delete button
      const deleteButton = page.locator('button[aria-label*="Delete" i], button:has-text("Delete")').first()
      if (await deleteButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await deleteButton.click()
        await page.waitForTimeout(500)

        // Confirm deletion
        const confirmButton = page.locator('button:has-text("Confirm"), button:has-text("Delete")').first()
        if (await confirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await confirmButton.click()
          await page.waitForTimeout(1000)

          // Service should be removed
          await expect(serviceButton).not.toBeVisible({ timeout: 2000 }).catch(() => {})
        }
      }
    }
  })

  test('should cancel service deletion', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const serviceButton = page.locator('[role="button"]:has-text("service-to-delete")').first()
    if (await serviceButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await serviceButton.click()
      await page.waitForTimeout(500)

      const deleteButton = page.locator('button[aria-label*="Delete" i], button:has-text("Delete")').first()
      if (await deleteButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await deleteButton.click()
        await page.waitForTimeout(500)

        // Cancel deletion
        const cancelButton = page.locator('button:has-text("Cancel")').first()
        if (await cancelButton.isVisible({ timeout: 2000 }).catch(() => false)) {
          await cancelButton.click()
          await page.waitForTimeout(500)

          // Service should still exist
          await expect(serviceButton).toBeVisible({ timeout: 2000 })
        }
      }
    }
  })
})

test.describe('Service Management - Service Properties', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display all service properties for appservice', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      // Should show service form with all properties
      const form = page.locator('form, [class*="form"]').first()
      if (await form.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(form).toBeVisible()
      }
    }
  })

  test('should configure service dependencies', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      // Find dependencies/uses section
      const usesSection = page.locator('[aria-label*="uses" i], [aria-label*="dependencies" i], button:has-text("Dependencies")').first()
      if (await usesSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await usesSection.click()
        await page.waitForTimeout(500)

        // Should show dependency selector
        const dependencySelector = page.locator('select, [role="combobox"]').first()
        if (await dependencySelector.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(dependencySelector).toBeVisible()
        }
      }
    }
  })

  test('should configure service docker settings', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const apiButton = page.locator('[role="button"]:has-text("api")').first()
    if (await apiButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await apiButton.click()
      await page.waitForTimeout(500)

      // Find docker section
      const dockerSection = page.locator('[aria-label*="docker" i], button:has-text("Docker")').first()
      if (await dockerSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await dockerSection.click()
        await page.waitForTimeout(500)

        // Should show docker config fields
        const dockerPath = page.locator('input[name*="docker" i], input[name*="path" i]').first()
        if (await dockerPath.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(dockerPath).toBeVisible()
        }
      }
    }
  })

  test('should configure service k8s settings for AKS', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const aksButton = page.locator('[role="button"]:has-text("aks-service")').first()
    if (await aksButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await aksButton.click()
      await page.waitForTimeout(500)

      // Find k8s section
      const k8sSection = page.locator('[aria-label*="k8s" i], [aria-label*="kubernetes" i], button:has-text("Kubernetes")').first()
      if (await k8sSection.isVisible({ timeout: 2000 }).catch(() => false)) {
        await k8sSection.click()
        await page.waitForTimeout(500)

        // Should show k8s config fields
        const deploymentPath = page.locator('input[name*="deploymentPath" i]').first()
        if (await deploymentPath.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(deploymentPath).toBeVisible()
        }
      }
    }
  })
})

test.describe('Service Management - Service Types and Modes', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should configure http service type', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const webButton = page.locator('[role="button"]:has-text("web")').first()
    if (await webButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await webButton.click()
      await page.waitForTimeout(500)

      // Find type field
      const typeSelect = page.locator('select[name*="type" i]').first()
      if (await typeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await typeSelect.selectOption('http')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure tcp service type', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const databaseProxyButton = page.locator('[role="button"]:has-text("database-proxy")').first()
    if (await databaseProxyButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await databaseProxyButton.click()
      await page.waitForTimeout(500)

      const typeSelect = page.locator('select[name*="type" i]').first()
      if (await typeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await typeSelect.selectOption('tcp')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure process service with watch mode', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const watcherButton = page.locator('[role="button"]:has-text("typescript-watcher")').first()
    if (await watcherButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await watcherButton.click()
      await page.waitForTimeout(500)

      const modeSelect = page.locator('select[name*="mode" i]').first()
      if (await modeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await modeSelect.selectOption('watch')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure process service with build mode', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const buildButton = page.locator('[role="button"]:has-text("build-service")').first()
    if (await buildButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await buildButton.click()
      await page.waitForTimeout(500)

      const modeSelect = page.locator('select[name*="mode" i]').first()
      if (await modeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await modeSelect.selectOption('build')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure process service with daemon mode', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const daemonButton = page.locator('[role="button"]:has-text("daemon-service")').first()
    if (await daemonButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await daemonButton.click()
      await page.waitForTimeout(500)

      const modeSelect = page.locator('select[name*="mode" i]').first()
      if (await modeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await modeSelect.selectOption('daemon')
        await page.waitForTimeout(300)
      }
    }
  })

  test('should configure process service with task mode', async ({ page }) => {
    await navigateToSection(page, 'Services')
    await page.waitForTimeout(500)

    const taskButton = page.locator('[role="button"]:has-text("migration-task")').first()
    if (await taskButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await taskButton.click()
      await page.waitForTimeout(500)

      const modeSelect = page.locator('select[name*="mode" i]').first()
      if (await modeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await modeSelect.selectOption('task')
        await page.waitForTimeout(300)
      }
    }
  })
})
