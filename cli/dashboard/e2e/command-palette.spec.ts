/**
 * Command Palette E2E Tests
 */

import { test, expect } from '@playwright/test'

test.describe('Command Palette', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Wait for app to load
    await page.waitForSelector('[data-testid="app-loaded"]', { timeout: 5000 })
  })
  
  test('should open with Cmd+K shortcut', async ({ page }) => {
    // Press Cmd+K (use Meta key for Mac, Control for others)
    const modifier = process.platform === 'darwin' ? 'Meta' : 'Control'
    await page.keyboard.press(`${modifier}+K`)
    
    // Palette should be visible
    await expect(page.getByRole('dialog', { name: /command palette/i })).toBeVisible()
  })
  
  test('should focus search input when opened', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    const searchInput = page.getByRole('textbox', { name: /search commands/i })
    await expect(searchInput).toBeFocused()
  })
  
  test('should search and filter commands', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Type search query
    await page.getByRole('textbox').fill('service')
    
    // Should show service-related commands
    await expect(page.getByText('Go to Services')).toBeVisible()
    await expect(page.getByText('Add Service')).toBeVisible()
    
    // Should hide non-matching commands
    await expect(page.getByText('Go to Overview')).not.toBeVisible()
  })
  
  test('should navigate with keyboard arrows', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // First result should be selected
    const firstResult = page.locator('[data-selected="true"]').first()
    await expect(firstResult).toBeVisible()
    
    // Press down arrow
    await page.keyboard.press('ArrowDown')
    
    // Selection should move
    await page.waitForTimeout(100)
    const selectedResults = page.locator('[data-selected="true"]')
    await expect(selectedResults).toHaveCount(1)
  })
  
  test('should execute command on Enter', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Search for specific command
    await page.getByRole('textbox').fill('overview')
    
    // Press Enter
    await page.keyboard.press('Enter')
    
    // Palette should close
    await expect(page.getByRole('dialog', { name: /command palette/i })).not.toBeVisible()
    
    // Navigation should occur (check URL or page state)
    // This depends on your app's navigation implementation
  })
  
  test('should execute command on click', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Click a command
    await page.getByText('Go to Services').click()
    
    // Palette should close
    await expect(page.getByRole('dialog', { name: /command palette/i })).not.toBeVisible()
  })
  
  test('should close on Escape', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Palette should be open
    await expect(page.getByRole('dialog', { name: /command palette/i })).toBeVisible()
    
    // Press Escape
    await page.keyboard.press('Escape')
    
    // Palette should close
    await expect(page.getByRole('dialog', { name: /command palette/i })).not.toBeVisible()
  })
  
  test('should close on backdrop click', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Click backdrop
    await page.locator('.fixed.inset-0').first().click({ position: { x: 10, y: 10 } })
    
    // Palette should close
    await expect(page.getByRole('dialog', { name: /command palette/i })).not.toBeVisible()
  })
  
  test('should close on close button click', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Click close button
    await page.getByRole('button', { name: /close command palette/i }).click()
    
    // Palette should close
    await expect(page.getByRole('dialog', { name: /command palette/i })).not.toBeVisible()
  })
  
  test('should show grouped results by category', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Should show category headers
    await expect(page.getByText(/navigation/i)).toBeVisible()
    await expect(page.getByText(/actions/i)).toBeVisible()
  })
  
  test('should show recent commands when no query', async ({ page }) => {
    // Execute a command first to add to history
    await page.keyboard.press('Control+K')
    await page.getByText('Go to Overview').click()
    
    // Open palette again
    await page.keyboard.press('Control+K')
    
    // Should show recent section
    await expect(page.getByText(/recent/i)).toBeVisible()
    await expect(page.getByText('Go to Overview')).toBeVisible()
  })
  
  test('should clear recent history', async ({ page }) => {
    // Add command to history
    await page.keyboard.press('Control+K')
    await page.getByText('Go to Overview').click()
    
    // Open palette again
    await page.keyboard.press('Control+K')
    
    // Click clear button
    await page.getByText(/clear/i).click()
    
    // Recent section should disappear
    await expect(page.getByText(/recent/i)).not.toBeVisible()
  })
  
  test('should show empty state when no matches', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Search for non-existent command
    await page.getByRole('textbox').fill('zzz-nonexistent-xyz')
    
    // Should show empty state
    await expect(page.getByText(/no commands found/i)).toBeVisible()
  })
  
  test('should display keyboard shortcuts for commands', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Search for command with shortcut
    await page.getByRole('textbox').fill('add service')
    
    // Should show shortcut hint
    await expect(page.getByText('Cmd+N')).toBeVisible()
  })
  
  test('should update results count', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Check initial count (depends on number of commands)
    const footer = page.locator('text=/\\d+ results?/')
    await expect(footer).toBeVisible()
    
    // Search to reduce results
    await page.getByRole('textbox').fill('overview')
    
    // Count should update
    await expect(page.locator('text=/1 result/')).toBeVisible()
  })
  
  test('should persist recent history across sessions', async ({ page, context }) => {
    // Execute a command
    await page.keyboard.press('Control+K')
    await page.getByText('Go to Services').click()
    
    // Close and reopen page
    await page.close()
    const newPage = await context.newPage()
    await newPage.goto('/')
    await newPage.waitForSelector('[data-testid="app-loaded"]', { timeout: 5000 })
    
    // Open palette
    await newPage.keyboard.press('Control+K')
    
    // Recent command should still be there
    await expect(newPage.getByText(/recent/i)).toBeVisible()
    await expect(newPage.getByText('Go to Services')).toBeVisible()
  })
  
  test('should highlight matching characters', async ({ page }) => {
    await page.keyboard.press('Control+K')
    
    // Search with specific query
    await page.getByRole('textbox').fill('serv')
    
    // Should highlight matched characters in results
    const highlighted = page.locator('mark')
    await expect(highlighted.first()).toBeVisible()
  })
})
