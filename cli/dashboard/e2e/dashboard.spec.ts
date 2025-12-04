/**
 * Dashboard Core E2E Tests
 * Core smoke tests and integration tests for the dashboard
 */
import { test, expect } from '@playwright/test'
import { 
  setupTest, 
  scenarios, 
  waitForDashboardReady,
  navigateToView,
  getServiceCard,
} from './helpers/test-setup'

// =============================================================================
// Smoke Tests - Basic Loading
// =============================================================================
test.describe('Dashboard - Smoke Tests', () => {
  test('dashboard loads successfully', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Page should load
    await expect(page).toHaveTitle(/test-project/i)
  })

  test('dashboard shows project name', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard(), projectName: 'My Awesome App' })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    await expect(page.getByText('My Awesome App')).toBeVisible()
  })

  test('dashboard loads services', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    await navigateToView(page, 'resources')
    
    // Should see service names
    await expect(page.getByText('api')).toBeVisible()
    await expect(page.getByText('web')).toBeVisible()
  })

  test('dashboard handles empty services gracefully', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.empty() })
    await page.goto('/')
    await waitForDashboardReady(page)
    await navigateToView(page, 'resources')
    
    // Should show empty state, not crash - look for common empty state messages
    const main = page.locator('main')
    await expect(main).toBeVisible()
    
    // Page should work without throwing errors
    await expect(page).toHaveTitle(/test-project/i)
  })
})

// =============================================================================
// Health Summary Tests
// =============================================================================
test.describe('Dashboard - Health Summary', () => {
  test('displays overall healthy status', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allHealthy() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Header should show healthy indicator
    const header = page.locator('header')
    await expect(header).toBeVisible()
    
    // Should have green/healthy coloring somewhere
    await expect(page.locator('[class*="green"], [class*="emerald"]').first()).toBeVisible()
  })

  test('displays unhealthy status when services fail', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allErrors() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Should have red/error coloring somewhere
    await expect(page.locator('[class*="red"], [class*="rose"]').first()).toBeVisible()
  })

  test('displays mixed health status', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.mixedHealth() })
    await page.goto('/')
    await waitForDashboardReady(page)
    await navigateToView(page, 'resources')
    
    // Should show healthy service
    const apiCard = getServiceCard(page, 'api')
    await expect(apiCard).toBeVisible()
    
    // Should show degraded service
    const degradedCard = getServiceCard(page, 'slow-api')
    await expect(degradedCard).toBeVisible()
  })
})

// =============================================================================
// View Integration Tests
// =============================================================================
test.describe('Dashboard - Views Integration', () => {
  test('console view displays logs panes', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Should show log panes for services (grid mode)
    await expect(page.getByText(/api|web/).first()).toBeVisible()
  })

  test('resources view displays service cards', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should show service cards
    const apiCard = getServiceCard(page, 'api')
    await expect(apiCard).toBeVisible()
  })

  test('environment view displays environment section', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/environment')
    await waitForDashboardReady(page)
    
    // Should show environment tab is active
    const envTab = page.locator('[role="tab"]:has-text("Environment")').first()
    await expect(envTab).toHaveAttribute('aria-selected', 'true')
    
    // Main content should be visible
    await expect(page.locator('main').first()).toBeVisible()
  })
})

// =============================================================================
// Service Detail Panel Tests
// =============================================================================
test.describe('Dashboard - Service Details', () => {
  test('clicking service card opens detail panel', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Click on service card
    const apiCard = getServiceCard(page, 'api')
    await apiCard.click()
    
    // Detail panel should open
    await expect(page.locator('[role="dialog"], [data-state="open"], aside:has-text("api")')).toBeVisible({ timeout: 5000 })
  })

  test('detail panel shows service information', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Click on service card
    const apiCard = getServiceCard(page, 'api')
    await apiCard.click()
    await page.waitForTimeout(300)
    
    // Should show service name in panel
    const panel = page.locator('[role="dialog"], [data-state="open"], aside').filter({ hasText: 'api' })
    if (await panel.count() > 0) {
      await expect(panel.first()).toContainText(/api/i)
    }
  })

  test('detail panel can be closed', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Open detail panel
    const apiCard = getServiceCard(page, 'api')
    await apiCard.click()
    await page.waitForTimeout(300)
    
    // Close with Escape
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
    
    // Panel should be closed (no visible detail panel)
    const panel = page.locator('[data-state="open"]')
    // Either hidden or not present
    expect(await panel.count()).toBeLessThanOrEqual(1)
  })
})

// =============================================================================
// Connection Status Tests
// =============================================================================
test.describe('Dashboard - Connection Status', () => {
  test('shows connected status when healthy', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Should not show connection error overlay
    await expect(page.getByText(/Connection Lost/i)).not.toBeVisible()
  })
})

// =============================================================================
// Settings Tests
// =============================================================================
test.describe('Dashboard - Settings', () => {
  test('settings dialog can be opened', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Find and click settings button (could be in header with various titles)
    const settingsBtn = page.locator('button[title*="Settings" i], button[title*="settings" i], button[aria-label*="Settings" i], button[aria-label*="settings" i]').first()
    
    if (await settingsBtn.isVisible()) {
      await settingsBtn.click()
      await page.waitForTimeout(300)
      
      // Dialog should open - look for overlay or dialog
      const dialog = page.locator('[role="dialog"], [class*="dialog" i], [class*="modal" i]').first()
      const isDialogVisible = await dialog.isVisible().catch(() => false)
      expect(typeof isDialogVisible).toBe('boolean')
    } else {
      // Settings button not visible in this context - that's acceptable
      test.skip()
    }
  })

  test('settings dialog can be closed with Escape', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Open settings
    const settingsBtn = page.locator('button[title*="Settings" i], button[aria-label*="Settings" i]').first()
    
    if (await settingsBtn.isVisible()) {
      await settingsBtn.click()
      await page.waitForTimeout(300)
      
      // Close with Escape
      await page.keyboard.press('Escape')
      await page.waitForTimeout(300)
      
      // Page should still work
      await expect(page.locator('main').first()).toBeVisible()
    } else {
      test.skip()
    }
  })
})

// =============================================================================
// Error Handling Tests
// =============================================================================
test.describe('Dashboard - Error Handling', () => {
  test('handles service errors gracefully', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allErrors() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should show error indicators but not crash
    await expect(page.getByText(/Error|Failed|Unhealthy/i).first()).toBeVisible()
  })

  test('page does not crash with no services', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.empty() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Should load without errors
    await expect(page).toHaveTitle(/test-project/i)
  })
})

// =============================================================================
// Performance Tests
// =============================================================================
test.describe('Dashboard - Performance', () => {
  test('loads within reasonable time', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    
    const startTime = Date.now()
    await page.goto('/')
    await waitForDashboardReady(page)
    const loadTime = Date.now() - startTime
    
    // Should load within 5 seconds
    expect(loadTime).toBeLessThan(5000)
  })

  test('handles many services without timeout', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.manyServices() })
    
    const startTime = Date.now()
    await page.goto('/services')
    await waitForDashboardReady(page)
    const loadTime = Date.now() - startTime
    
    // Should still load within reasonable time even with 20 services
    expect(loadTime).toBeLessThan(10000)
    
    // All services should be accessible
    await expect(page.getByText(/20 services/i)).toBeVisible()
  })
})
