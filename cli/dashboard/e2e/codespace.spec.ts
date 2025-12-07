/**
 * Codespace URL Forwarding E2E Tests
 * Tests that localhost URLs are correctly transformed to Codespace URLs
 * when running in a GitHub Codespace environment.
 */
import { test, expect, type Page, type Route } from '@playwright/test'
import { setupTest, scenarios, waitForDashboardReady, getServiceCard } from './helpers/test-setup'

// =============================================================================
// Codespace Environment Mock
// =============================================================================

const CODESPACE_CONFIG = {
  codespace: {
    enabled: true,
    name: 'silver-space-xyzzy',
    domain: 'app.github.dev',
  },
}

const NON_CODESPACE_CONFIG = {
  codespace: {
    enabled: false,
    name: '',
    domain: '',
  },
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Mock the /api/environment endpoint to return Codespace configuration
 */
async function mockCodespaceEnvironment(page: Page, enabled: boolean = true) {
  await page.route('**/api/environment', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(enabled ? CODESPACE_CONFIG : NON_CODESPACE_CONFIG),
    })
  })
}

// =============================================================================
// Tests
// =============================================================================

test.describe('Codespace URL Forwarding', () => {
  test.describe('Service Card URLs', () => {
    test('transforms localhost URL to Codespace URL in service card', async ({ page }) => {
      // Clear sessionStorage before any page loads
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      // Setup test with standard scenario first
      await setupTest(page, { scenario: scenarios.standard() })
      
      // Then set up Codespace mock (after setupTest so it takes precedence)
      await mockCodespaceEnvironment(page, true)
      
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      // Find the API service card (port 3001)
      const apiCard = getServiceCard(page, 'api')
      await expect(apiCard).toBeVisible()
      
      // Find the URL link in the card
      const urlLink = apiCard.locator('a[href*="app.github.dev"]')
      await expect(urlLink).toBeVisible()
      
      // Verify the href is transformed to Codespace URL
      const href = await urlLink.getAttribute('href')
      expect(href).toBe('https://silver-space-xyzzy-3001.app.github.dev/')
    })

    test('keeps localhost URL when not in Codespace', async ({ page }) => {
      // Clear sessionStorage before any page loads
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      // Setup test first
      await setupTest(page, { scenario: scenarios.standard() })
      
      // Then set up non-Codespace mock
      await mockCodespaceEnvironment(page, false)
      
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      const apiCard = getServiceCard(page, 'api')
      await expect(apiCard).toBeVisible()
      
      // Find the URL link - should be localhost
      const urlLink = apiCard.locator('a[href*="localhost"]')
      await expect(urlLink).toBeVisible()
      
      const href = await urlLink.getAttribute('href')
      expect(href).toBe('http://localhost:3001')
    })

    test('transforms multiple service URLs correctly', async ({ page }) => {
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      await setupTest(page, { scenario: scenarios.standard() })
      await mockCodespaceEnvironment(page, true)
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      // Check API service (port 3001)
      const apiLink = page.locator('a[href="https://silver-space-xyzzy-3001.app.github.dev/"]')
      await expect(apiLink).toBeVisible()
      
      // Check Web service (port 3000)
      const webLink = page.locator('a[href="https://silver-space-xyzzy-3000.app.github.dev/"]')
      await expect(webLink).toBeVisible()
    })
  })

  test.describe('Service URL Click', () => {
    test('opens Codespace URL in new tab when clicked', async ({ page, context }) => {
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      await setupTest(page, { scenario: scenarios.standard() })
      await mockCodespaceEnvironment(page, true)
      
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      // Find and click the service URL link
      const urlLink = page.locator('a[href*="silver-space-xyzzy-3001"]').first()
      await expect(urlLink).toBeVisible()
      
      // Intercept the new tab opening
      const [newPage] = await Promise.all([
        context.waitForEvent('page'),
        urlLink.click(),
      ])
      
      // Verify the new tab URL is the Codespace URL
      expect(newPage.url()).toContain('silver-space-xyzzy-3001.app.github.dev')
    })
  })

  test.describe('Process Services', () => {
    test('process services without ports are not affected', async ({ page }) => {
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      await setupTest(page, { scenario: scenarios.processServices() })
      await mockCodespaceEnvironment(page, true)
      
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      // Process services should not have URL links
      const watcherCard = getServiceCard(page, 'typescript-watcher')
      await expect(watcherCard).toBeVisible()
      
      // Should not have any external link to Codespace URL
      const urlLink = watcherCard.locator('a[href*="app.github.dev"]')
      await expect(urlLink).toHaveCount(0)
    })
  })

  test.describe('Environment API', () => {
    test('fetches environment info on load', async ({ page }) => {
      let apiCalled = false
      
      // Clear sessionStorage before page load
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      // Setup test first (sets up default mocks)
      await setupTest(page, { scenario: scenarios.standard() })
      
      // Then set up our tracking route (after setupTest so it takes precedence)
      await page.route('**/api/environment', async (route: Route) => {
        apiCalled = true
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(CODESPACE_CONFIG),
        })
      })
      
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      // Wait a bit for the API call
      await page.waitForTimeout(500)
      
      expect(apiCalled).toBe(true)
    })

    test('handles environment API errors gracefully', async ({ page }) => {
      // Clear sessionStorage before page load
      await page.addInitScript(() => {
        sessionStorage.clear()
      })
      
      // Setup test first
      await setupTest(page, { scenario: scenarios.standard() })
      
      // Then mock API error (after setupTest so it takes precedence)
      await page.route('**/api/environment', async (route: Route) => {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal server error' }),
        })
      })
      
      await page.goto('/services')
      await waitForDashboardReady(page)
      
      // Dashboard should still load and show localhost URLs (fallback)
      const apiCard = getServiceCard(page, 'api')
      await expect(apiCard).toBeVisible()
      
      // URL should fall back to localhost
      const urlLink = apiCard.locator('a[href*="localhost"]')
      await expect(urlLink).toBeVisible()
    })
  })
})
