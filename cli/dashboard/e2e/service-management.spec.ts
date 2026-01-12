/**
 * Service Management E2E Tests
 * 
 * Tests the complete flow of adding, editing, and deleting services
 * using the AddServiceModal and DeleteServiceDialog components.
 */

import { test, expect } from '@playwright/test'

test.describe('Service Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the editor page
    await page.goto('/editor')
  })

  test('should open add service modal when clicking add service button', async ({ page }) => {
    // Click the "+ Add Service" button
    await page.getByRole('button', { name: /add service/i }).click()

    // Modal should be visible
    const modal = page.getByRole('dialog', { name: /add service/i })
    await expect(modal).toBeVisible()

    // All three tabs should be present
    await expect(page.getByRole('button', { name: 'Well-Known Services' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Application Service' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Container Service' })).toBeVisible()
  })

  test('should add a well-known service (Azurite)', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Wait for well-known services to load
    await page.waitForSelector('text=Azurite')

    // Click on Azurite service card
    await page.getByText('Azurite').click()

    // Service should be selected
    await expect(page.getByText(/Selected: Azurite/i)).toBeVisible()

    // Click Add Service button
    await page.getByRole('button', { name: 'Add Service', exact: true }).click()

    // Modal should close
    await expect(page.getByRole('dialog', { name: /add service/i })).not.toBeVisible()

    // Service should appear in navigation
    await expect(page.getByText('azurite')).toBeVisible()
  })

  test('should add an application service', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Switch to Application Service tab
    await page.getByRole('button', { name: 'Application Service' }).click()

    // Fill in form
    await page.getByLabel(/Service Name/).fill('my-api')
    await page.getByLabel(/Project Path/).fill('./src/api')
    await page.getByLabel(/Language/).selectOption('node')

    // Submit form
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Modal should close
    await expect(page.getByRole('dialog', { name: /add service/i })).not.toBeVisible()

    // Service should appear in navigation
    await expect(page.getByText('my-api')).toBeVisible()
  })

  test('should add a container service', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Switch to Container Service tab
    await page.getByRole('button', { name: 'Container Service' }).click()

    // Fill in form
    await page.getByLabel(/Service Name/).fill('my-nginx')
    await page.getByLabel(/Docker Image/).fill('nginx:alpine')
    await page.getByLabel(/Port Mappings/).fill('80:80')

    // Submit form
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Modal should close
    await expect(page.getByRole('dialog', { name: /add service/i })).not.toBeVisible()

    // Service should appear in navigation
    await expect(page.getByText('my-nginx')).toBeVisible()
  })

  test('should prevent duplicate service names', async ({ page }) => {
    // Add first service
    await page.getByRole('button', { name: /add service/i }).click()
    await page.getByRole('button', { name: 'Container Service' }).click()
    await page.getByLabel(/Service Name/).fill('duplicate-test')
    await page.getByLabel(/Docker Image/).fill('nginx:alpine')
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Wait for modal to close
    await page.waitForTimeout(500)

    // Try to add service with same name
    await page.getByRole('button', { name: /add service/i }).click()
    await page.getByRole('button', { name: 'Container Service' }).click()
    await page.getByLabel(/Service Name/).fill('duplicate-test')
    await page.getByLabel(/Docker Image/).fill('redis:alpine')
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Should show alert
    page.on('dialog', async (dialog) => {
      expect(dialog.message()).toContain('already exists')
      await dialog.accept()
    })

    // Modal should remain open
    await expect(page.getByRole('dialog', { name: /add service/i })).toBeVisible()
  })

  test('should validate service name format', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Switch to Application Service tab
    await page.getByRole('button', { name: 'Application Service' }).click()

    // Enter invalid name (uppercase)
    await page.getByLabel(/Service Name/).fill('MyAPI')
    await page.getByLabel(/Project Path/).fill('./src/api')

    // Submit form
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Should show validation error
    await expect(page.getByText(/must contain only lowercase/i)).toBeVisible()

    // Modal should remain open
    await expect(page.getByRole('dialog', { name: /add service/i })).toBeVisible()
  })

  test('should delete a service', async ({ page }) => {
    // First add a service
    await page.getByRole('button', { name: /add service/i }).click()
    await page.getByRole('button', { name: 'Container Service' }).click()
    await page.getByLabel(/Service Name/).fill('to-delete')
    await page.getByLabel(/Docker Image/).fill('nginx:alpine')
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Wait for service to appear
    await expect(page.getByText('to-delete')).toBeVisible()

    // Click service in navigation
    await page.getByText('to-delete').click()

    // Click delete button
    await page.getByRole('button', { name: /delete/i }).click()

    // Confirmation dialog should appear
    const deleteDialog = page.getByRole('dialog', { name: /delete service/i })
    await expect(deleteDialog).toBeVisible()
    await expect(page.getByText(/Delete "to-delete"\?/)).toBeVisible()

    // Confirm deletion
    await page.getByRole('button', { name: 'Delete Service', exact: true }).click()

    // Dialog should close
    await expect(deleteDialog).not.toBeVisible()

    // Service should be removed from navigation
    await expect(page.getByText('to-delete')).not.toBeVisible()
  })

  test('should cancel service deletion', async ({ page }) => {
    // First add a service
    await page.getByRole('button', { name: /add service/i }).click()
    await page.getByRole('button', { name: 'Container Service' }).click()
    await page.getByLabel(/Service Name/).fill('keep-me')
    await page.getByLabel(/Docker Image/).fill('nginx:alpine')
    await page.getByRole('button', { name: 'Add Service' }).click()

    // Wait for service to appear
    await expect(page.getByText('keep-me')).toBeVisible()

    // Click service and delete
    await page.getByText('keep-me').click()
    await page.getByRole('button', { name: /delete/i }).click()

    // Cancel deletion
    await page.getByRole('button', { name: 'Cancel' }).click()

    // Dialog should close
    await expect(page.getByRole('dialog', { name: /delete service/i })).not.toBeVisible()

    // Service should still exist
    await expect(page.getByText('keep-me')).toBeVisible()
  })

  test('should filter well-known services by category', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Wait for services to load
    await page.waitForSelector('text=Azurite')

    // Click database category
    await page.getByRole('button', { name: 'database' }).click()

    // Should show only database services
    await expect(page.getByText('PostgreSQL')).toBeVisible()
    await expect(page.getByText('MongoDB')).toBeVisible()

    // Storage services should not be visible
    await expect(page.getByText('Azurite')).not.toBeVisible()
  })

  test('should show service preview when well-known service selected', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Wait for services to load
    await page.waitForSelector('text=Redis')

    // Click Redis service card
    await page.getByText('Redis Cache').click()

    // Preview should show configuration
    await expect(page.getByText(/Configuration Preview/i)).toBeVisible()
    await expect(page.getByText(/redis:7-alpine/i)).toBeVisible()
    await expect(page.getByText(/6379:6379/i)).toBeVisible()
  })

  test('should pre-fill common container images', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Switch to Container Service tab
    await page.getByRole('button', { name: 'Container Service' }).click()

    // Click Nginx example
    await page.getByText('Nginx').click()

    // Should pre-fill image and ports
    const imageInput = page.getByLabel(/Docker Image/)
    await expect(imageInput).toHaveValue('nginx:alpine')

    const portsInput = page.getByLabel(/Port Mappings/)
    await expect(portsInput).toHaveValue('80:80')
  })

  test('should close modal on escape key', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Modal should be visible
    await expect(page.getByRole('dialog', { name: /add service/i })).toBeVisible()

    // Press Escape
    await page.keyboard.press('Escape')

    // Modal should close
    await expect(page.getByRole('dialog', { name: /add service/i })).not.toBeVisible()
  })

  test('should close modal on backdrop click', async ({ page }) => {
    // Open add service modal
    await page.getByRole('button', { name: /add service/i }).click()

    // Modal should be visible
    const modal = page.getByRole('dialog', { name: /add service/i })
    await expect(modal).toBeVisible()

    // Click backdrop
    await page.click('.fixed.inset-0', { force: true })

    // Modal should close
    await expect(modal).not.toBeVisible()
  })
})
