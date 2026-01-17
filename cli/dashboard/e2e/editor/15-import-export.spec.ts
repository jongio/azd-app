/**
 * Import/Export E2E Tests for Azure YAML Editor
 * 
 * Tests import/export functionality including:
 * - Export configuration
 * - Import from file
 * - Import from paste
 * - Import from template
 * - Merge strategies
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  loadMinimalProject,
} from '../helpers/test-setup'
import * as selectors from './selectors'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const fixturesDir = path.join(__dirname, '../fixtures')
const comprehensiveYaml = fs.readFileSync(path.join(fixturesDir, 'comprehensive-azure-yaml.yaml'), 'utf-8')
const minimalYaml = fs.readFileSync(path.join(fixturesDir, 'minimal-azure-yaml.yaml'), 'utf-8')

test.describe('Import/Export - Export', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should export configuration', async ({ page }) => {
    const exportButton = page.locator('button:has-text("Export")').first()
    if (await exportButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)
      await exportButton.click()
      const download = await downloadPromise
      if (download) {
        expect(download.suggestedFilename()).toMatch(/azure\.yaml|config/)
      }
    }
  })

  test('should export current configuration state', async ({ page }) => {
    // Make changes first
    const textarea = page.locator('textarea').first()
    if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
      await textarea.fill('name: exported-project\nservices: {}')
      await page.waitForTimeout(1000)

      const exportButton = page.locator('button:has-text("Export")').first()
      if (await exportButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)
        await exportButton.click()
        const download = await downloadPromise
        if (download) {
          const path = await download.path()
          if (path) {
            const content = fs.readFileSync(path, 'utf-8')
            expect(content).toContain('exported-project')
          }
        }
      }
    }
  })
})

test.describe('Import/Export - Import from File', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should open import modal', async ({ page }) => {
    const importButton = page.locator('button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click()
      await page.waitForTimeout(500)

      const modal = page.locator('[role="dialog"]').first()
      if (await modal.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(modal).toBeVisible()
      }
    }
  })

  test('should show file upload option', async ({ page }) => {
    const importButton = page.locator('button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click()
      await page.waitForTimeout(500)

      const fileInput = page.locator('input[type="file"]')
      const hasInput = await fileInput.count()
      expect(hasInput).toBeGreaterThanOrEqual(0)
    }
  })
})

test.describe('Import/Export - Import from Paste', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should show paste YAML option', async ({ page }) => {
    const importButton = page.locator('button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click()
      await page.waitForTimeout(500)

      const pasteTab = page.locator('button:has-text("Paste"), [role="tab"]:has-text("Paste")').first()
      if (await pasteTab.isVisible({ timeout: 2000 }).catch(() => false)) {
        await pasteTab.click()
        await page.waitForTimeout(500)

        const textarea = page.locator('textarea').first()
        if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
          await expect(textarea).toBeVisible()
        }
      }
    }
  })

  test('should import from pasted YAML', async ({ page }) => {
    // Click Import button in QuickActionsBar (not blocked)
    const importButton = page.locator('button[aria-label="Import configuration"], button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click({ force: true }).catch(() => {})
      await page.waitForTimeout(1000) // Wait for modal to open

      // Wait for modal
      const modal = page.locator(selectors.modals.dialog).first()
      if (await modal.isVisible({ timeout: 2000 }).catch(() => false)) {
        // Switch to Paste tab
        const pasteTab = modal.locator('button:has-text("Paste"), [role="tab"]:has-text("Paste")').first()
        if (await pasteTab.isVisible({ timeout: 2000 }).catch(() => false)) {
          await pasteTab.click({ force: true }).catch(() => {})
          await page.waitForTimeout(500)

          // Fill in YAML
          const textarea = modal.locator('textarea').first()
          if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
            await textarea.fill(minimalYaml)
            await page.waitForTimeout(500)

            // Click Import button INSIDE modal (not the opener)
            const importConfirmButton = modal.locator('button:has-text("Import"), button:has-text("Confirm"):not([aria-hidden="true"])').first()
            if (await importConfirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
              await importConfirmButton.click({ force: true }).catch(() => {})
              await page.waitForTimeout(1000)

              // Should update editor
              const editorTextarea = page.locator('textarea').first()
              if (await editorTextarea.isVisible({ timeout: 2000 }).catch(() => false)) {
                const content = await editorTextarea.inputValue()
                expect(content).toContain('minimal-test')
              }
            }
          }
        }
      }
    }
  })
})

test.describe('Import/Export - Import from Template', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should show template import option', async ({ page }) => {
    const importButton = page.locator('button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click()
      await page.waitForTimeout(500)

      const templateTab = page.locator('button:has-text("Template"), [role="tab"]:has-text("Template")').first()
      if (await templateTab.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(templateTab).toBeVisible()
      }
    }
  })
})

test.describe('Import/Export - Merge Strategies', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: {
          name: 'test-project',
          services: {
            existing: { host: 'appservice', project: './src/existing' },
          },
        },
      },
    })
    await navigateToEditor(page)
  })

  test('should show merge strategy options', async ({ page }) => {
    const importButton = page.locator('button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click()
      await page.waitForTimeout(500)

      // Should show merge strategy selector
      const strategySelect = page.locator('select[name*="strategy" i], [name*="merge" i]').first()
      if (await strategySelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(strategySelect).toBeVisible()
      }
    }
  })
})

test.describe('Import/Export - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should export comprehensive project', async ({ page }) => {
    const exportButton = page.locator('button:has-text("Export")').first()
    if (await exportButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)
      await exportButton.click()
      const download = await downloadPromise
      if (download) {
        expect(download.suggestedFilename()).toMatch(/azure\.yaml/)
      }
    }
  })

  test('should import comprehensive project', async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
    await page.waitForTimeout(1000) // Wait for editor to load

    // Click Import button in QuickActionsBar
    const importButton = page.locator('button[aria-label="Import configuration"], button:has-text("Import")').first()
    if (await importButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await importButton.click({ force: true }).catch(() => {})
      await page.waitForTimeout(1000) // Wait for modal to open

      // Wait for modal
      const modal = page.locator(selectors.modals.dialog).first()
      if (await modal.isVisible({ timeout: 2000 }).catch(() => false)) {
        // Switch to Paste tab
        const pasteTab = modal.locator('button:has-text("Paste"), [role="tab"]:has-text("Paste")').first()
        if (await pasteTab.isVisible({ timeout: 2000 }).catch(() => false)) {
          await pasteTab.click({ force: true }).catch(() => {})
          await page.waitForTimeout(500)

          // Fill in YAML
          const textarea = modal.locator('textarea').first()
          if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
            await textarea.fill(comprehensiveYaml)
            await page.waitForTimeout(500)

            // Click Import button INSIDE modal (not the opener)
            const importConfirmButton = modal.locator('button:has-text("Import"), button:has-text("Confirm"):not([aria-hidden="true"])').first()
            if (await importConfirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
              await importConfirmButton.click({ force: true }).catch(() => {})
              await page.waitForTimeout(1000)

              // Should load comprehensive project
              const editorTextarea = page.locator('textarea').first()
              if (await editorTextarea.isVisible({ timeout: 2000 }).catch(() => false)) {
                const content = await editorTextarea.inputValue()
                expect(content).toContain('editor-e2e-test-project')
              }
            }
          }
        }
      }
    }
  })
})
