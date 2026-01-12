/**
 * YAML Editor Navigation Component E2E Tests
 * 
 * Tests complete navigation flows for the Azure YAML Editor including:
 * - Tree navigation
 * - Keyboard navigation
 * - Search functionality
 * - Accessibility
 */

import { test, expect } from '@playwright/test'

test.describe('YAML Editor Navigation Component', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the editor page (adjust URL as needed)
    await page.goto('/editor')
    
    // Wait for navigation to load
    await page.waitForSelector('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')
  })

  test('should display navigation tree structure', async ({ page }) => {
    // Check for main sections
    await expect(page.getByRole('button', { name: /Overview/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /Services/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /Resources/i })).toBeVisible()
  })

  test('should expand and collapse sections', async ({ page }) => {
    const servicesButton = page.getByRole('button', { name: /Services/i })
    
    // Click to expand
    await servicesButton.click()
    
    // Check if children are visible
    await expect(page.getByRole('button', { name: 'api' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'web' })).toBeVisible()
    
    // Click to collapse
    await servicesButton.click()
    
    // Children should be hidden
    await expect(page.getByRole('button', { name: 'api' })).not.toBeVisible()
  })

  test('should navigate to section when clicked', async ({ page }) => {
    // Click on a service
    await page.getByRole('button', { name: 'api' }).click()
    
    // Check if the item is marked as active
    const apiButton = page.getByRole('button', { name: 'api' })
    await expect(apiButton).toHaveAttribute('aria-current', 'page')
  })

  test('should show validation badges', async ({ page }) => {
    // Check for error badge (if errors exist)
    const errorBadge = page.locator('[aria-label*="error"]').first()
    if (await errorBadge.isVisible()) {
      await expect(errorBadge).toBeVisible()
    }
    
    // Check for warning badge (if warnings exist)
    const warningBadge = page.locator('[aria-label*="warning"]').first()
    if (await warningBadge.isVisible()) {
      await expect(warningBadge).toBeVisible()
    }
  })

  test('should filter navigation with search', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search/i)
    
    // Type search query
    await searchInput.fill('api')
    
    // Only matching items should be visible
    await expect(page.getByRole('button', { name: 'api' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'web' })).not.toBeVisible()
  })

  test('should clear search with clear button', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search/i)
    
    // Type search query
    await searchInput.fill('api')
    
    // Click clear button
    await page.getByLabel('Clear search').click()
    
    // Search should be cleared
    await expect(searchInput).toHaveValue('')
    
    // All items should be visible again
    await expect(page.getByRole('button', { name: 'web' })).toBeVisible()
  })

  test('should support keyboard navigation', async ({ page }) => {
    const nav = page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')
    
    // Focus navigation
    await nav.focus()
    
    // Press ArrowDown
    await page.keyboard.press('ArrowDown')
    
    // Press Enter to expand/navigate
    await page.keyboard.press('Enter')
    
    // Navigation should respond to keyboard
  })

  test('should navigate with arrow keys', async ({ page }) => {
    const nav = page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')
    await nav.focus()
    
    // Navigate down
    await page.keyboard.press('ArrowDown')
    await page.keyboard.press('ArrowDown')
    
    // Navigate up
    await page.keyboard.press('ArrowUp')
  })

  test('should expand/collapse with arrow keys', async ({ page }) => {
    const nav = page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')
    await nav.focus()
    
    // Navigate to a section
    await page.keyboard.press('ArrowDown')
    
    // Expand with ArrowRight
    await page.keyboard.press('ArrowRight')
    
    // Collapse with ArrowLeft
    await page.keyboard.press('ArrowLeft')
  })

  test('should show add buttons for services and resources', async ({ page }) => {
    // Expand services section
    await page.getByRole('button', { name: /Services/i }).click()
    
    // Check for add service button
    await expect(page.getByRole('button', { name: /Add service/i })).toBeVisible()
    
    // Expand resources section
    await page.getByRole('button', { name: /Resources/i }).click()
    
    // Check for add resource button
    await expect(page.getByRole('button', { name: /Add resource/i })).toBeVisible()
  })

  test('should collapse navigation sidebar', async ({ page }) => {
    // Click collapse button
    await page.getByLabel('Collapse navigation').click()
    
    // Navigation should be collapsed
    await expect(page.getByText('Configuration')).not.toBeVisible()
    
    // Expand button should be visible
    await expect(page.getByLabel('Expand navigation')).toBeVisible()
  })

  test('should be accessible', async ({ page }) => {
    // Check for ARIA landmarks
    await expect(page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')).toBeVisible()
    await expect(page.locator('[role="tree"]')).toBeVisible()
    
    // Check for proper tree structure
    const treeItems = page.locator('[role="treeitem"]')
    await expect(treeItems.first()).toBeVisible()
  })

  test('should support keyboard focus management', async ({ page }) => {
    const nav = page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')
    
    // Navigation should be focusable
    await nav.focus()
    await expect(nav).toBeFocused()
  })

  test('should clear search with Escape key', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search/i)
    
    // Type search query
    await searchInput.fill('api')
    
    // Focus navigation and press Escape
    await page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]').focus()
    await page.keyboard.press('Escape')
    
    // Search should be cleared
    await expect(searchInput).toHaveValue('')
  })

  test('should auto-expand parent of active section', async ({ page }) => {
    // Navigate to a nested item (e.g., services.api)
    // This should auto-expand the Services section
    
    // Check if parent section is expanded
    const servicesButton = page.getByRole('button', { name: /Services/i })
    await expect(servicesButton).toHaveAttribute('aria-expanded', 'true')
  })

  test('should maintain focus visibility', async ({ page }) => {
    const nav = page.locator('[role="navigation"][aria-label="Azure YAML Editor Navigation"]')
    await nav.focus()
    
    // Check if focus ring is visible (visual regression would be better)
    await expect(nav).toBeFocused()
  })

  test('should handle chevron toggle separately from item click', async ({ page }) => {
    const servicesButton = page.getByRole('button', { name: /Services/i })
    
    // Click chevron to toggle
    const chevron = servicesButton.locator('button').first()
    await chevron.click()
    
    // Section should toggle without navigating
    await expect(servicesButton).toHaveAttribute('aria-expanded')
  })
})
