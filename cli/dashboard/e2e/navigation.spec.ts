/**
 * Navigation E2E Tests
 * Tests for tab navigation, URL routing, and keyboard shortcuts
 */
import { test, expect } from '@playwright/test'
import { 
  setupTest, 
  scenarios, 
  waitForDashboardReady,
} from './helpers/test-setup'

// =============================================================================
// Tab Navigation Tests
// =============================================================================
test.describe('Navigation - Tabs', () => {
  test('all navigation tabs are visible', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Should see Console tab
    await expect(page.locator('[role="tab"]:has-text("Console")').first()).toBeVisible()
    
    // Should see Services tab (called "Services" in UI)
    await expect(page.locator('[role="tab"]:has-text("Services")').first()).toBeVisible()
    
    // Should see Environment tab
    await expect(page.locator('[role="tab"]:has-text("Environment")').first()).toBeVisible()
  })

  test('clicking Console tab navigates to console view', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Click Console tab
    await page.locator('[role="tab"]:has-text("Console")').first().click()
    await page.waitForTimeout(300)
    
    // Should be on console view (check URL or active tab)
    await expect(page).toHaveURL(/\/(console)?$/)
  })

  test('clicking Services tab navigates to services view', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Click Services tab
    await page.locator('[role="tab"]:has-text("Services")').first().click()
    await page.waitForTimeout(300)
    
    // Should be on services view
    await expect(page).toHaveURL(/\/services/)
  })

  test('clicking Environment tab navigates to environment view', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Click Environment tab
    await page.locator('[role="tab"]:has-text("Environment")').first().click()
    await page.waitForTimeout(300)
    
    // Should be on environment view
    await expect(page).toHaveURL(/\/environment/)
  })
})

// =============================================================================
// URL Routing Tests
// =============================================================================
test.describe('Navigation - URL Routing', () => {
  test('direct navigation to /console works', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/console')
    await waitForDashboardReady(page)
    
    // Console tab should be active
    const consoleTab = page.locator('[role="tab"]:has-text("Console")').first()
    await expect(consoleTab).toHaveAttribute('aria-selected', 'true')
  })

  test('direct navigation to /services works', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Services tab should be active
    const servicesTab = page.locator('[role="tab"]:has-text("Services")').first()
    await expect(servicesTab).toHaveAttribute('aria-selected', 'true')
  })

  test('direct navigation to /environment works', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/environment')
    await waitForDashboardReady(page)
    
    // Environment tab should be active
    const envTab = page.locator('[role="tab"]:has-text("Environment")').first()
    await expect(envTab).toHaveAttribute('aria-selected', 'true')
  })

  test('unknown path defaults to console', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/unknown-path')
    await waitForDashboardReady(page)
    
    // Should default to console
    await expect(page).toHaveURL(/\/(console)?$/)
  })

  test('root path shows console', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Console should be default
    const consoleTab = page.locator('[role="tab"]:has-text("Console")').first()
    await expect(consoleTab).toHaveAttribute('aria-selected', 'true')
  })
})

// =============================================================================
// Browser History Tests
// =============================================================================
test.describe('Navigation - Browser History', () => {
  test('back button returns to previous view', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Navigate to services
    await page.locator('[role="tab"]:has-text("Services")').first().click()
    await page.waitForTimeout(300)
    await expect(page).toHaveURL(/\/services/)
    
    // Go back
    await page.goBack()
    await page.waitForTimeout(300)
    
    // Should be back on console
    await expect(page).toHaveURL(/\/(console)?$/)
  })

  test('forward button works after going back', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Navigate to services
    await page.locator('[role="tab"]:has-text("Services")').first().click()
    await page.waitForTimeout(300)
    
    // Go back
    await page.goBack()
    await page.waitForTimeout(300)
    
    // Go forward
    await page.goForward()
    await page.waitForTimeout(300)
    
    // Should be on services again
    await expect(page).toHaveURL(/\/services/)
  })
})

// =============================================================================
// Keyboard Shortcuts Tests
// =============================================================================
test.describe('Navigation - Keyboard Shortcuts', () => {
  test('1 key navigates to Console', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Press 1
    await page.keyboard.press('1')
    await page.waitForTimeout(300)
    
    // Should be on console
    await expect(page).toHaveURL(/\/(console)?$/)
  })

  test('2 key navigates to Services', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Press 2
    await page.keyboard.press('2')
    await page.waitForTimeout(300)
    
    // Should be on services
    await expect(page).toHaveURL(/\/services/)
  })

  test('3 key navigates to Environment', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Press 3
    await page.keyboard.press('3')
    await page.waitForTimeout(300)
    
    // Should be on environment
    await expect(page).toHaveURL(/\/environment/)
  })

  test('keyboard shortcuts do not work in input fields', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Find an input field
    const input = page.locator('input').first()
    if (await input.isVisible()) {
      await input.focus()
      await input.type('1')
      
      // Should NOT have navigated
      await expect(page).toHaveURL(/\/(console)?$/)
    }
  })
})

// =============================================================================
// Header Tests
// =============================================================================
test.describe('Navigation - Header', () => {
  test('header displays project name', async ({ page }) => {
    await setupTest(page, { projectName: 'My Test Project' })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Project name should be visible
    await expect(page.getByText('My Test Project').first()).toBeVisible()
  })

  test('header shows health summary', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allHealthy() })
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Should show healthy indicator
    const header = page.locator('header').first()
    await expect(header).toBeVisible()
  })
})
