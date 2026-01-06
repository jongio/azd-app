/**
 * Logs UX E2E Tests
 * Coverage for: services dropdown removal, timeframe presets, refresh bounds,
 * diagnostics entry points, and local-only service override behavior.
 */
import { test, expect } from '@playwright/test'
import {
  setupTest,
  scenarios,
  waitForDashboardReady,
  createServiceFixture,
  createHealthCheckFixture,
} from './helpers/test-setup'

test.describe('Console - Logs UX', () => {
  test('does not show services dropdown in console view', async ({ page }) => {
    await setupTest(page, { scenario: scenarios.standard() })
    await page.goto('/')
    await waitForDashboardReady(page)

    // Old logs view had an "All Services" <select> option; it should not appear on Console.
    await expect(page.locator('option', { hasText: 'All Services' })).toHaveCount(0)

    // Console should still show service panes.
    await expect(page.getByText('api').first()).toBeVisible()
    await expect(page.getByText('web').first()).toBeVisible()
  })

  test('timeframe presets exclude 1 hour and include 30 min (Azure mode)', async ({ page }) => {
    await setupTest(page, {
      scenario: scenarios.standard(),
      azure: { enabled: true, status: 'connected', mode: 'azure' },
    })
    await page.goto('/')
    await waitForDashboardReady(page)

    // Options exist in the DOM even when the select is not opened.
    await expect(page.locator('option', { hasText: '1 hour' })).toHaveCount(0)
    await expect(page.locator('option', { hasText: '30 min' })).toHaveCount(1)
  })

  test.skip('refresh interval clamps low values from localStorage (min 5s)', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('logs-sync-interval', '1000')
    })

    await setupTest(page, {
      scenario: scenarios.standard(),
      clearStorage: false,
      azure: { enabled: true, status: 'connected', mode: 'azure' },
    })

    await page.goto('/')
    await waitForDashboardReady(page)

    const refreshSelect = page.getByText('Refresh:').locator('..').locator('select')
    await expect(refreshSelect).toHaveValue('5000')
  })

  test.skip('refresh interval clamps high values from localStorage (max 5m)', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('logs-sync-interval', String(999999999))
    })

    await setupTest(page, {
      scenario: scenarios.standard(),
      clearStorage: false,
      azure: { enabled: true, status: 'connected', mode: 'azure' },
    })

    await page.goto('/')
    await waitForDashboardReady(page)

    const refreshSelect = page.getByText('Refresh:').locator('..').locator('select')
    await expect(refreshSelect).toHaveValue('300000')
  })

  test('diagnostics button is visible only in Azure mode', async ({ page }) => {
    await setupTest(page, {
      scenario: scenarios.standard(),
      azure: { enabled: true, status: 'connected', mode: 'azure' },
    })
    await page.goto('/')
    await waitForDashboardReady(page)

    // Check that diagnostics buttons are visible (one per service + header)
    await expect(page.getByRole('button', { name: 'Diagnostics' }).first()).toBeVisible()

    // Switch to local mode via the ModeToggle button.
    await page.getByRole('button', { name: 'View local logs' }).click()
    await expect(page.getByRole('button', { name: 'Diagnostics' })).toHaveCount(0)
  })

  test('local-only services do not use Azure logs even when global mode is Azure', async ({ page }) => {
    let azureRequestedForWeb = false
    let azureRequestedForApi = false

    page.on('request', req => {
      const url = req.url()
      if (!url.includes('/api/azure/logs')) return
      try {
        const parsed = new URL(url)
        const service = parsed.searchParams.get('service')
        if (service === 'web') azureRequestedForWeb = true
        if (service === 'api') azureRequestedForApi = true
      } catch {
        // Ignore URL parsing issues
      }
    })

    const scenario = {
      services: [
        createServiceFixture({ name: 'api', status: 'running', health: 'healthy', port: 3001 }),
        createServiceFixture({ name: 'web', status: 'running', health: 'healthy', port: 3000, host: 'local' }),
      ],
      healthChecks: [
        createHealthCheckFixture('api', 'healthy', { port: 3001 }),
        createHealthCheckFixture('web', 'healthy', { port: 3000 }),
      ],
      healthSummary: { total: 2, healthy: 2, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'healthy' },
    } as const

    await setupTest(page, {
      scenario,
      azure: { enabled: true, status: 'connected', mode: 'azure' },
    })

    await page.goto('/')
    await waitForDashboardReady(page)

    // The page starts in local mode and then asynchronously reads /api/mode.
    // Wait until the UI reflects Azure mode and the Azure-pane request has occurred.
    await expect(page.getByText('Viewing Azure Logs')).toBeVisible()
    await expect.poll(() => azureRequestedForApi, { timeout: 5000 }).toBe(true)

    // The local-only service should never request Azure logs.
    await page.waitForTimeout(300)
    expect(azureRequestedForWeb).toBeFalsy()

    // UI should reflect mixed log sources.
    await expect(page.getByText('Viewing Local Logs')).toBeVisible()
  })
})
