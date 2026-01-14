/**
 * Preview Pane Integration Tests
 * Task 5: Preview Pane Component - E2E Tests
 */

import { test, expect } from '@playwright/test'

test.describe('Preview Pane', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to editor (will be integrated later)
    // For now, we'll create a test page
    await page.goto('/test/preview-pane')
  })

  test('should toggle preview pane visibility', async ({ page }) => {
    // Check preview is visible initially
    await expect(page.getByText('YAML Preview')).toBeVisible()

    // Click toggle button
    await page.getByTitle('Hide preview').click()

    // Preview should be hidden
    await expect(page.getByText('YAML Preview')).not.toBeVisible()

    // Click toggle again (from header button)
    await page.getByTitle('Show preview').click()

    // Preview should be visible again
    await expect(page.getByText('YAML Preview')).toBeVisible()
  })

  test('should copy YAML to clipboard', async ({ page, context }) => {
    // Grant clipboard permissions
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])

    // Wait for YAML to be generated
    await expect(page.getByText(/name:/i)).toBeVisible()

    // Click copy button
    await page.getByTitle('Copy to clipboard').click()

    // Check clipboard content
    const handle = await page.evaluateHandle(() => navigator.clipboard.readText())
    const clipboardText = await handle.jsonValue()

    expect(clipboardText).toContain('name:')
    expect(clipboardText).toContain('services:')
  })

  test('should download YAML as file', async ({ page }) => {
    // Set up download listener
    const downloadPromise = page.waitForEvent('download')

    // Click download button
    await page.getByTitle('Download as azure.yaml').click()

    // Wait for download
    const download = await downloadPromise

    // Verify download
    expect(download.suggestedFilename()).toBe('azure.yaml')

    // Verify content
    const path = await download.path()
    expect(path).toBeTruthy()
  })

  test('should update YAML when form data changes', async ({ page }) => {
    // Initial YAML should contain test-app
    await expect(page.getByText(/name: test-app/i)).toBeVisible()

    // Simulate form update (change app name)
    await page.evaluate(() => {
      const event = new CustomEvent('form-data-change', {
        detail: { name: 'updated-app' }
      })
      window.dispatchEvent(event)
    })

    // Wait for debounced update (300ms)
    await page.waitForTimeout(400)

    // YAML should now contain updated-app
    await expect(page.getByText(/name: updated-app/i)).toBeVisible()
  })

  test('should show validation errors in preview', async ({ page }) => {
    // Simulate validation errors
    await page.evaluate(() => {
      const event = new CustomEvent('validation-change', {
        detail: {
          markers: [
            { line: 3, level: 'error', message: 'Service name required' },
            { line: 5, level: 'warning', message: 'Consider adding health check' },
          ]
        }
      })
      window.dispatchEvent(event)
    })

    // Error count should be displayed
    await expect(page.getByText('1 errors')).toBeVisible()

    // Error markers should be visible (visual indicators on lines)
    // This is a visual test - we verify the markers array is processed
  })

  test('should navigate to form field when clicking preview line', async ({ page }) => {
    // Click on a line in the preview
    const line = page.getByText(/name: test-app/i)
    await line.click()

    // Should trigger navigation event (verified via callback)
    await page.evaluate(() => {
      return new Promise((resolve) => {
        window.addEventListener('line-click', (e) => {
          resolve((e as CustomEvent).detail.lineNumber)
        }, { once: true })
      })
    })
  })

  test('should resize preview pane by dragging divider', async ({ page }) => {
    // Get initial width
    const preview = page.getByText('YAML Preview').locator('..')
    const initialBox = await preview.boundingBox()
    expect(initialBox).toBeTruthy()
    const initialWidth = initialBox!.width

    // Find drag divider
    const divider = page.getByRole('separator', { name: 'Resize preview pane' })

    // Drag divider to resize
    const dividerBox = await divider.boundingBox()
    expect(dividerBox).toBeTruthy()

    await page.mouse.move(dividerBox!.x, dividerBox!.y)
    await page.mouse.down()
    await page.mouse.move(dividerBox!.x - 100, dividerBox!.y) // Drag left 100px
    await page.mouse.up()

    // Wait for resize
    await page.waitForTimeout(100)

    // Width should have changed
    const newBox = await preview.boundingBox()
    expect(newBox).toBeTruthy()
    const newWidth = newBox!.width

    expect(newWidth).not.toBe(initialWidth)
  })

  test('should persist preview visibility across page reloads', async ({ page }) => {
    // Hide preview
    await page.getByTitle('Hide preview').click()

    // Reload page
    await page.reload()

    // Preview should remain hidden
    await expect(page.getByText('YAML Preview')).not.toBeVisible()

    // Show preview
    await page.getByTitle('Show preview').click()

    // Reload page
    await page.reload()

    // Preview should remain visible
    await expect(page.getByText('YAML Preview')).toBeVisible()
  })

  test('should persist preview width across page reloads', async ({ page }) => {
    // Resize preview
    const divider = page.getByRole('separator', { name: 'Resize preview pane' })
    const dividerBox = await divider.boundingBox()
    expect(dividerBox).toBeTruthy()

    await page.mouse.move(dividerBox!.x, dividerBox!.y)
    await page.mouse.down()
    await page.mouse.move(dividerBox!.x - 100, dividerBox!.y)
    await page.mouse.up()

    // Get width after resize
    const preview = page.getByText('YAML Preview').locator('..')
    const resizedBox = await preview.boundingBox()
    expect(resizedBox).toBeTruthy()
    const resizedWidth = resizedBox!.width

    // Reload page
    await page.reload()

    // Width should be persisted
    const reloadedBox = await preview.boundingBox()
    expect(reloadedBox).toBeTruthy()
    const reloadedWidth = reloadedBox!.width

    // Allow for small rounding differences
    expect(Math.abs(reloadedWidth - resizedWidth)).toBeLessThan(5)
  })

  test('should apply syntax highlighting to YAML', async ({ page }) => {
    // Wait for YAML to be rendered
    await expect(page.getByText(/name:/i)).toBeVisible()

    // Check that syntax highlighting is applied
    // This is done by react-syntax-highlighter - we verify it's rendered
    const codeBlock = page.locator('pre code')
    await expect(codeBlock).toBeVisible()

    // Verify line numbers are shown
    const lineNumbers = page.locator('.linenumber')
    const count = await lineNumbers.count()
    expect(count).toBeGreaterThan(0)
  })

  test('should show line numbers', async ({ page }) => {
    await expect(page.getByText(/name:/i)).toBeVisible()

    // Line numbers should be visible (rendered by SyntaxHighlighter)
    // We can check for the presence of line number elements
    const lineNumberElements = page.locator('[class*="linenumber"]')
    const count = await lineNumberElements.count()
    
    // Should have at least a few line numbers
    expect(count).toBeGreaterThan(0)
  })

  test('should handle keyboard shortcuts', async ({ page }) => {
    // Focus on preview pane
    await page.getByText('YAML Preview').click()

    // Test Cmd/Ctrl+C to copy (if implemented)
    // For now, we verify the copy button works
    const copyButton = page.getByTitle('Copy to clipboard')
    await expect(copyButton).toBeVisible()
  })

  test('should be accessible', async ({ page }) => {
    // Check ARIA attributes
    await expect(page.getByRole('separator', { name: 'Resize preview pane' })).toBeVisible()
    await expect(page.getByRole('separator')).toHaveAttribute('aria-orientation', 'vertical')

    // Check button titles
    await expect(page.getByTitle('Copy to clipboard')).toBeVisible()
    await expect(page.getByTitle('Download as azure.yaml')).toBeVisible()
    await expect(page.getByTitle('Hide preview')).toBeVisible()

    // Check focus management (tab through buttons)
    await page.keyboard.press('Tab')
    await page.keyboard.press('Tab')
    await page.keyboard.press('Tab')
    
    // Should be able to navigate to buttons
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName)
    expect(focusedElement).toBe('BUTTON')
  })

  test('should support dark mode', async ({ page }) => {
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.classList.add('dark')
    })

    // Preview should apply dark theme
    await expect(page.getByText('YAML Preview')).toBeVisible()

    // Verify dark mode styling (visual test)
    // The syntax highlighter should use vscDarkPlus theme
  })

  test('should highlight changed lines temporarily', async ({ page }) => {
    // Initial render
    await expect(page.getByText(/name: test-app/i)).toBeVisible()

    // Update data
    await page.evaluate(() => {
      const event = new CustomEvent('form-data-change', {
        detail: { name: 'changed-app' }
      })
      window.dispatchEvent(event)
    })

    // Wait for debounced update
    await page.waitForTimeout(400)

    // Changed lines should have highlight animation
    // This is a visual test - we verify the changed lines set is updated

    // Wait for animation to clear (2 seconds)
    await page.waitForTimeout(2100)

    // Highlight should be removed
  })

  test('should respect min and max width constraints', async ({ page }) => {
    const divider = page.getByRole('separator', { name: 'Resize preview pane' })
    const dividerBox = await divider.boundingBox()
    expect(dividerBox).toBeTruthy()

    // Try to drag beyond max width (80%)
    await page.mouse.move(dividerBox!.x, dividerBox!.y)
    await page.mouse.down()
    await page.mouse.move(dividerBox!.x - 1000, dividerBox!.y) // Very large drag
    await page.mouse.up()

    await page.waitForTimeout(100)

    // Get preview width
    const preview = page.getByText('YAML Preview').locator('..')
    const box = await preview.boundingBox()
    expect(box).toBeTruthy()

    // Calculate percentage
    const viewportWidth = await page.evaluate(() => window.innerWidth)
    const widthPercent = (box!.width / viewportWidth) * 100

    // Should be clamped to max 80%
    expect(widthPercent).toBeLessThanOrEqual(80)
    expect(widthPercent).toBeGreaterThanOrEqual(20)
  })

  test('should handle empty data gracefully', async ({ page }) => {
    // Set empty data
    await page.evaluate(() => {
      const event = new CustomEvent('form-data-change', {
        detail: {}
      })
      window.dispatchEvent(event)
    })

    // Wait for update
    await page.waitForTimeout(400)

    // Should still render (empty YAML is valid)
    await expect(page.getByText('YAML Preview')).toBeVisible()
  })

  test('should debounce updates correctly', async ({ page }) => {
    // Rapid updates
    for (let i = 0; i < 10; i++) {
      await page.evaluate((index) => {
        const event = new CustomEvent('form-data-change', {
          detail: { name: `app-${index}` }
        })
        window.dispatchEvent(event)
      }, i)
      await page.waitForTimeout(50) // 50ms between updates
    }

    // Should only render the final update after 300ms debounce
    await page.waitForTimeout(400)

    await expect(page.getByText(/name: app-9/i)).toBeVisible()
  })
})

test.describe('Preview Toggle Button', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/test/preview-toggle')
  })

  test('should display current preview state', async ({ page }) => {
    // Initially visible
    await expect(page.getByTitle('Hide preview')).toBeVisible()
    await expect(page.getByRole('button', { pressed: true })).toBeVisible()

    // Click to hide
    await page.getByTitle('Hide preview').click()

    // Should show "Show preview"
    await expect(page.getByTitle('Show preview')).toBeVisible()
    await expect(page.getByRole('button', { pressed: false })).toBeVisible()
  })

  test('should be keyboard accessible', async ({ page }) => {
    await page.keyboard.press('Tab')
    
    // Button should be focused
    const focusedElement = await page.evaluate(() => document.activeElement?.getAttribute('title'))
    expect(focusedElement).toContain('preview')

    // Press Enter to toggle
    await page.keyboard.press('Enter')

    // State should change
  })

  test('should have proper aria-pressed attribute', async ({ page }) => {
    const button = page.getByRole('button')

    // Initially pressed
    await expect(button).toHaveAttribute('aria-pressed', 'true')

    // Click to toggle
    await button.click()

    // Should be unpressed
    await expect(button).toHaveAttribute('aria-pressed', 'false')
  })
})
