import { test, expect } from '@playwright/test';

test.describe('Dashboard - Resources View', () => {
  test.beforeEach(async ({ page }) => {
    // Mock the API responses
    await page.route('/api/project', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ name: 'test-project' }),
      });
    });

    await page.route('/api/services', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            name: 'api',
            language: 'python',
            framework: 'flask',
            local: {
              status: 'ready',
              health: 'healthy',
              url: 'http://localhost:5000',
              port: 5000,
              pid: 12345,
              startTime: new Date(Date.now() - 60000).toISOString(),
              lastChecked: new Date().toISOString(),
            },
          },
          {
            name: 'web',
            language: 'node',
            framework: 'express',
            local: {
              status: 'ready',
              health: 'healthy',
              url: 'http://localhost:5001',
              port: 5001,
              pid: 12346,
              startTime: new Date(Date.now() - 120000).toISOString(),
              lastChecked: new Date().toISOString(),
            },
          },
        ]),
      });
    });

    await page.route('/api/logs*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/');
  });

  test('should display project name in header', async ({ page }) => {
    await expect(page.getByText('test-project')).toBeVisible();
  });

  test('should display services in table view by default', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Resources' })).toBeVisible();
    await expect(page.getByText('api')).toBeVisible();
    await expect(page.getByText('web')).toBeVisible();
  });

  test('should switch between table and grid view', async ({ page }) => {
    // Default is table view
    await expect(page.getByRole('button', { name: /table/i })).toBeVisible();
    
    // Switch to grid view
    await page.getByRole('button', { name: /grid/i }).click();
    
    // Grid view should be active (check for grid container)
    const gridContainer = page.locator('.grid.grid-cols-1');
    await expect(gridContainer).toBeVisible();
    
    // Switch back to table view
    await page.getByRole('button', { name: /table/i }).click();
  });

  test('should display service details in cards', async ({ page }) => {
    // Switch to grid view
    await page.getByRole('button', { name: /grid/i }).click();
    
    // Check that service cards are visible
    await expect(page.getByText('api')).toBeVisible();
    await expect(page.getByText('flask')).toBeVisible();
    await expect(page.getByText('python')).toBeVisible();
  });

  test('should display service status', async ({ page }) => {
    await expect(page.getByText('Running').first()).toBeVisible();
  });

  test('should show search filter input', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Filter...');
    await expect(searchInput).toBeVisible();
    
    // Type in search
    await searchInput.fill('api');
  });

  test('should navigate between views', async ({ page }) => {
    // Click console view
    await page.getByRole('button', { name: /console/i }).click();
    await expect(page.getByRole('heading', { name: 'Console' })).toBeVisible();
    
    // Click back to resources
    await page.getByRole('button', { name: /resources/i }).click();
    await expect(page.getByRole('heading', { name: 'Resources' })).toBeVisible();
  });

  test('should show coming soon for unimplemented views', async ({ page }) => {
    await page.getByRole('button', { name: /traces/i }).click();
    await expect(page.getByText('Coming Soon')).toBeVisible();
    await expect(page.getByText('This view is not yet implemented')).toBeVisible();
  });

  test('should display header buttons', async ({ page }) => {
    // Check that header action buttons are present
    const header = page.locator('header');
    await expect(header).toBeVisible();
  });

  test('should preserve view preference in localStorage', async ({ page }) => {
    // Switch to grid view
    await page.getByRole('button', { name: /grid/i }).click();
    
    // Reload page
    await page.reload();
    
    // Grid view should still be active
    const gridContainer = page.locator('.grid.grid-cols-1');
    await expect(gridContainer).toBeVisible();
  });
});

test.describe('Dashboard - Console View', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('/api/project', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ name: 'test-project' }),
      });
    });

    await page.route('/api/services', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.route('/api/logs*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            service: 'api',
            message: 'Application started',
            level: 0,
            timestamp: new Date().toISOString(),
            isStderr: false,
          },
          {
            service: 'web',
            message: 'Server listening on port 5001',
            level: 0,
            timestamp: new Date().toISOString(),
            isStderr: false,
          },
        ]),
      });
    });

    await page.goto('/');
  });

  test('should navigate to console view', async ({ page }) => {
    await page.getByRole('button', { name: /console/i }).click();
    await expect(page.getByRole('heading', { name: 'Console' })).toBeVisible();
  });

  test('should display log controls', async ({ page }) => {
    await page.getByRole('button', { name: /console/i }).click();
    
    // Check for service filter heading
    await expect(page.getByRole('heading', { name: 'Services' })).toBeVisible();
  });
});

test.describe('Dashboard - Error States', () => {
  test('should display loading state', async ({ page }) => {
    // Delay the API response
    await page.route('/api/services', async route => {
      await new Promise(resolve => setTimeout(resolve, 500));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.route('/api/project', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ name: 'test-project' }),
      });
    });

    await page.goto('/');
    
    // Should show loading spinner (use first() since there may be multiple spinners)
    const spinner = page.locator('.animate-spin').first();
    await expect(spinner).toBeVisible();
  });

  test('should display empty state when no services', async ({ page }) => {
    await page.route('/api/project', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ name: 'test-project' }),
      });
    });

    await page.route('/api/services', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/');
    
    await expect(page.getByText('No Services Running')).toBeVisible();
    await expect(page.getByText('azd app run')).toBeVisible();
  });
});

test.describe('Dashboard - Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('/api/project', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ name: 'test-project' }),
      });
    });

    await page.route('/api/services', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            name: 'api',
            language: 'python',
            framework: 'flask',
            local: {
              status: 'ready',
              health: 'healthy',
              url: 'http://localhost:5000',
              port: 5000,
            },
          },
        ]),
      });
    });

    await page.route('/api/logs*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/');
  });

  test('should have proper heading structure', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Resources' })).toBeVisible();
  });

  test('should have accessible buttons', async ({ page }) => {
    const tableButton = page.getByRole('button', { name: /table/i });
    await expect(tableButton).toBeVisible();
    await expect(tableButton).toBeEnabled();
  });

  test('should have keyboard navigation', async ({ page }) => {
    // Test that buttons can be focused and activated with keyboard
    await page.getByRole('button', { name: /grid/i }).focus();
    await page.keyboard.press('Enter');
    
    // Grid view should be active
    const gridContainer = page.locator('.grid.grid-cols-1');
    await expect(gridContainer).toBeVisible();
  });
});
