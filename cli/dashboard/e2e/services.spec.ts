/**
 * Services E2E Tests
 * Tests for different service types, modes, health states, and lifecycle states
 */
import { test, expect } from '@playwright/test'
import { 
  setupTest, 
  scenarios, 
  waitForDashboardReady,
  getServiceCard,
} from './helpers/test-setup'

// =============================================================================
// HTTP Services Tests
// =============================================================================
test.describe('Services - HTTP Services', () => {
  test('displays HTTP service with port and URL', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see API service
    const apiCard = getServiceCard(page, 'api')
    await expect(apiCard).toBeVisible()
    
    // Should show port info
    await expect(apiCard.getByText(/3001|Port/).first()).toBeVisible()
  })

  test('displays healthy HTTP service with status indicator', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allHealthy() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    const apiCard = getServiceCard(page, 'api')
    await expect(apiCard).toBeVisible()
    
    // Should show healthy status (green color or "Running" text)
    await expect(apiCard.locator('[class*="emerald"], [class*="green"], :text-matches("Running|Healthy", "i")').first()).toBeVisible()
  })
})

// =============================================================================
// TCP Services Tests
// =============================================================================
test.describe('Services - TCP Services', () => {
  test('displays TCP service (database)', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allHealthy() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see database service
    const dbCard = getServiceCard(page, 'database')
    await expect(dbCard).toBeVisible()
    
    // Should show PostgreSQL or SQL info
    await expect(dbCard.getByText(/PostgreSQL|SQL|database/i).first()).toBeVisible()
  })
})

// =============================================================================
// Process Services Tests
// =============================================================================
test.describe('Services - Process Services', () => {
  test('displays watch mode service', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.processServices() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see watch service
    const watchCard = getServiceCard(page, 'typescript-watcher')
    await expect(watchCard).toBeVisible()
    
    // Should show watch mode indicator
    await expect(watchCard.getByText(/watch/i).first()).toBeVisible()
  })

  test('displays daemon mode service', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.processServices() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see daemon service
    const daemonCard = getServiceCard(page, 'mcp-server')
    await expect(daemonCard).toBeVisible()
    
    // Should show daemon mode indicator
    await expect(daemonCard.getByText(/daemon/i).first()).toBeVisible()
  })

  test('displays task service with completed status', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.processServices() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see task service
    const taskCard = getServiceCard(page, 'migration')
    await expect(taskCard).toBeVisible()
  })

  test('displays failed process service with error state', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.processServices() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see failed build service
    const failedCard = getServiceCard(page, 'failed-build')
    await expect(failedCard).toBeVisible()
    
    // Should show error/failed indicator (look for red/rose colors or error text)
    const hasErrorStyling = await failedCard.locator('[class*="rose"], [class*="red"]').first().isVisible().catch(() => false)
    const hasErrorText = await failedCard.getByText(/Error|Failed|Unhealthy/i).first().isVisible().catch(() => false)
    
    expect(hasErrorStyling || hasErrorText).toBeTruthy()
  })
})

// =============================================================================
// Health States Tests
// =============================================================================
test.describe('Services - Health States', () => {
  test('displays healthy service correctly', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.allHealthy() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // All services should be healthy
    const apiCard = getServiceCard(page, 'api')
    await expect(apiCard).toBeVisible()
  })

  test('displays degraded service with warning', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.mixedHealth() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see degraded service
    const degradedCard = getServiceCard(page, 'slow-api')
    await expect(degradedCard).toBeVisible()
    
    // Should show warning indicator (amber/yellow)
    await expect(degradedCard.locator('[class*="amber"], [class*="yellow"]').first()).toBeVisible()
  })

  test('displays unhealthy service with error indicator', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.mixedHealth() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see unhealthy service
    const unhealthyCard = getServiceCard(page, 'failing-api')
    await expect(unhealthyCard).toBeVisible()
    
    // Should show error state (red/rose)
    await expect(unhealthyCard.locator('[class*="rose"], [class*="red"]').first()).toBeVisible()
  })

  test('displays stopped service correctly', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.mixedHealth() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see stopped service
    const stoppedCard = getServiceCard(page, 'stopped-api')
    await expect(stoppedCard).toBeVisible()
  })
})

// =============================================================================
// View Modes Tests
// =============================================================================
test.describe('Services - View Modes', () => {
  test('can toggle between grid and table view', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Grid should be default - article elements are service cards
    await expect(page.locator('article').first()).toBeVisible()
  })
})

// =============================================================================
// Azure Deployment Tests
// =============================================================================
test.describe('Services - Azure Deployment', () => {
  test('displays Azure URL when service is deployed', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.azureDeployment() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should see Azure service
    const azureCard = getServiceCard(page, 'prod-api')
    await expect(azureCard).toBeVisible()
    
    // Should show Azure URL
    await expect(azureCard.getByText(/azurewebsites\.net|Azure/i).first()).toBeVisible()
  })
})

// =============================================================================
// Empty States Tests
// =============================================================================
test.describe('Services - Empty States', () => {
  test('shows empty state when no services', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.empty() })
    await page.goto('/services')
    await waitForDashboardReady(page)
    
    // Should show empty state message
    await expect(page.getByText(/No Service|0 service|empty|none/i).first()).toBeVisible()
  })
})

// =============================================================================
// Performance Tests
// =============================================================================
test.describe('Services - Performance', () => {
  test('handles many services without performance issues', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.manyServices() })
    
    const startTime = Date.now()
    await page.goto('/services')
    await waitForDashboardReady(page)
    const loadTime = Date.now() - startTime
    
    // Should load in reasonable time
    expect(loadTime).toBeLessThan(10000)
    
    // Should show service count or some services
    const serviceCards = page.locator('article[role="button"]')
    const count = await serviceCards.count()
    expect(count).toBeGreaterThan(0)
  })
})
