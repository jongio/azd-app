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
    // Wait until the UI reflects Azure mode.
    await expect(page.getByText('Viewing Azure Logs')).toBeVisible()

    // The in-page fetch interceptor (addInitScript) tracks which services
    // triggered GetAzureLogs calls via window.__azureLogsCalledFor. This
    // avoids the abort race between React state updates and the Connect
    // transport's async setup that makes page.on('request') unreliable.
    await expect.poll(async () => {
      const calls = await page.evaluate(() =>
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (window as any).__azureLogsCalledFor as string[],
      )
      return calls.includes('api')
    }, { timeout: 15000 }).toBe(true)

    // The local-only service should never request Azure logs.
    await page.waitForTimeout(500)
    const azureCalls = await page.evaluate(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (window as any).__azureLogsCalledFor as string[],
    )
    expect(azureCalls).not.toContain('web')

    // UI should reflect mixed log sources.
    await expect(page.getByText('Viewing Local Logs')).toBeVisible()
  })
})
