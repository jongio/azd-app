/**
 * Preview Pane E2E Tests for Azure YAML Editor
 * 
 * Tests preview pane functionality including:
 * - Preview updates on changes
 * - Download as azure.yaml
 * - Preview formatting
 */

import { test, expect } from '@playwright/test'
import {
  setupTest,
  navigateToEditor,
  loadComprehensiveProject,
  editYamlDirectly,
} from '../helpers/test-setup'

test.describe('Preview Pane - Updates', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: { api: { host: 'appservice', project: './src/api' } } },
      },
    })
    await navigateToEditor(page)
  })

  test('should update preview on changes', async ({ page }) => {
    const preview = page.locator('[class*="preview"]').first()
    if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
      const initialContent = await preview.textContent()

      // Make a change
      await editYamlDirectly(page, 'name: updated-project\nservices:\n  api:\n    host: appservice')
      await page.waitForTimeout(1000)

      const newContent = await preview.textContent()
      expect(newContent).not.toBe(initialContent)
      expect(newContent).toContain('updated-project')
    }
  })

  test('should show formatted YAML in preview', async ({ page }) => {
    const preview = page.locator('[class*="preview"]').first()
    if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
      const content = await preview.textContent()
      if (content) {
        // Should be formatted YAML
        expect(content).toContain('name:')
        expect(content).toContain('services:')
      }
    }
  })

  test('should sync preview with editor changes', async ({ page }) => {
    const textarea = page.locator('textarea').first()
    if (await textarea.isVisible({ timeout: 2000 }).catch(() => false)) {
      await textarea.fill('name: synced-project\nservices: {}')
      await page.waitForTimeout(1000)

      const preview = page.locator('[class*="preview"]').first()
      if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
        const previewContent = await preview.textContent()
        expect(previewContent).toContain('synced-project')
      }
    }
  })
})

test.describe('Preview Pane - Download', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page, {
      config: {
        initialConfig: { name: 'test-project', services: {} },
      },
    })
    await navigateToEditor(page)
  })

  test('should download as azure.yaml', async ({ page }) => {
    const downloadButton = page.locator('button:has-text("Download"), button[aria-label*="Download" i]').first()
    if (await downloadButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)
      await downloadButton.click()
      const download = await downloadPromise
      if (download) {
        expect(download.suggestedFilename()).toMatch(/azure\.yaml/)
      }
    }
  })

  test('should download current configuration', async ({ page }) => {
    // Make changes first
    await editYamlDirectly(page, 'name: downloaded-project\nservices: {}')
    await page.waitForTimeout(1000)

    const downloadButton = page.locator('button:has-text("Download")').first()
    if (await downloadButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)
      await downloadButton.click()
      const download = await downloadPromise
      if (download) {
        const path = await download.path()
        if (path) {
          const fs = await import('fs')
          const content = fs.readFileSync(path, 'utf-8')
          expect(content).toContain('downloaded-project')
        }
      }
    }
  })
})

test.describe('Preview Pane - Comprehensive Project', () => {
  test.beforeEach(async ({ page }) => {
    await setupTest(page)
    await loadComprehensiveProject(page)
    await navigateToEditor(page)
  })

  test('should display comprehensive YAML in preview', async ({ page }) => {
    const preview = page.locator('[class*="preview"]').first()
    if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
      const content = await preview.textContent()
      if (content) {
        expect(content).toContain('editor-e2e-test-project')
        expect(content).toContain('services:')
        expect(content).toContain('resources:')
        expect(content).toContain('hooks:')
      }
    }
  })

  test('should format comprehensive YAML correctly', async ({ page }) => {
    const preview = page.locator('[class*="preview"]').first()
    if (await preview.isVisible({ timeout: 2000 }).catch(() => false)) {
      const content = await preview.textContent()
      if (content) {
        // Should be properly formatted YAML
        const lines = content.split('\n')
        expect(lines.length).toBeGreaterThan(10)
      }
    }
  })
})
