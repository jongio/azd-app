/**
 * Debug test to understand the log viewer DOM structure
 */
import { test } from '@playwright/test'
import {
  setupTest,
  scenarios,
  waitForDashboardReady,
} from './helpers/test-setup'

test.describe('Debug Logs DOM', () => {
  test('examine log structure', async ({ page }) => {
    const mockLogs = [
      { service: 'api', message: 'Server started on port 3001', level: 1, timestamp: new Date().toISOString(), isStderr: false },
      { service: 'api', message: 'Connected to database', level: 1, timestamp: new Date().toISOString(), isStderr: false },
    ]

    await setupTest(page, { 
      scenario: scenarios.standard(),
      azure: { enabled: false, status: 'disabled', mode: 'local' }
    })
    
    await page.route('/api/logs*', async route => {
      const url = new URL(route.request().url())
      const service = url.searchParams.get('service')
      const filteredLogs = service ? mockLogs.filter(l => l.service === service) : mockLogs
      console.log(`LOG REQUEST: ${route.request().url()} - returning ${filteredLogs.length} logs`)
      console.log('Filtered logs:', JSON.stringify(filteredLogs))
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(filteredLogs),
      })
    })
    
    await page.goto('/')
    
    await page.goto('/')
    await waitForDashboardReady(page)
    
    // Wait for logs to load
    await page.waitForTimeout(2000)
    
    // Take a screenshot
    await page.screenshot({ path: 'test-results/debug-logs-structure.png', fullPage: true })
    
    // Look for log containers
    const logContainers = page.locator('[role="log"]')
    const count = await logContainers.count()
    console.log('Found log containers:', count)
    
    for (let i = 0; i < count; i++) {
      const container = logContainers.nth(i)
      const containerHTML = await container.innerHTML()
      console.log(`=== LOG CONTAINER ${i} ===`)
      console.log(containerHTML.substring(0, 1000))
    }
    
    // Look for select-text divs
    const selectTextDivs = page.locator('div.select-text')
    const selectTextCount = await selectTextDivs.count()
    console.log('Found select-text divs:', selectTextCount)
    
    if (selectTextCount > 0) {
      const firstSelectText = selectTextDivs.first()
      const text = await firstSelectText.textContent()
      console.log('First select-text content:', text)
    }
    
    // Look for group divs
    const groupDivs = page.locator('div.group')
    const groupCount = await groupDivs.count()
    console.log('Found group divs:', groupCount)
  })
})
