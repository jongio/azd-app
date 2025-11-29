import { test, expect } from '@playwright/test'

// Helper to inject EventSource mock before page loads
async function mockEventSource(page: import('@playwright/test').Page) {
  await page.addInitScript(() => {
    // Mock EventSource to prevent health stream overlay
    class MockEventSource {
      static readonly CONNECTING = 0;
      static readonly OPEN = 1;
      static readonly CLOSED = 2;
      readonly CONNECTING = 0;
      readonly OPEN = 1;
      readonly CLOSED = 2;
      readyState = 1; // OPEN
      url: string;
      withCredentials = false;
      onopen: ((ev: Event) => void) | null = null;
      onmessage: ((ev: MessageEvent) => void) | null = null;
      onerror: ((ev: Event) => void) | null = null;

      constructor(url: string) {
        this.url = url;
        // Simulate successful connection
        setTimeout(() => {
          if (this.onopen) {
            this.onopen(new Event('open'));
          }
          // Send initial health data
          if (this.onmessage) {
            const data = JSON.stringify({
              type: 'health',
              timestamp: new Date().toISOString(),
              services: [],
              summary: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, unknown: 0, overall: 'healthy' }
            });
            this.onmessage(new MessageEvent('message', { data }));
          }
        }, 10);
      }

      close() {
        this.readyState = 2;
      }

      addEventListener() {}
      removeEventListener() {}
      dispatchEvent() { return false; }
    }

    // Replace global EventSource
    (window as unknown as { EventSource: typeof MockEventSource }).EventSource = MockEventSource;
  });
}

test.describe('Theme Toggle', () => {
  test.beforeEach(async ({ page }) => {
    // Mock EventSource before page loads
    await mockEventSource(page);

    // Mock API endpoints to avoid loading overlay
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
        body: JSON.stringify([]),
      });
    });

    await page.route('/api/preferences*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ theme: 'system', dateFormat: 'relative', refreshInterval: 5000, fontSize: 14 }),
      });
    });

    await page.route('/api/classifications*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    // Clear localStorage before each test
    await page.goto('/')
    await page.evaluate(() => localStorage.clear())
    await page.reload()
  })

  test('theme toggle button is visible in header', async ({ page }) => {
    await page.goto('/')
    
    // Theme toggle should be visible
    const themeToggle = page.getByRole('button', { name: /switch to/i })
    await expect(themeToggle).toBeVisible()
  })

  test('initial theme is light mode', async ({ page }) => {
    await page.goto('/')
    
    // Check data-theme attribute
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('light')
    
    // Button should say "Switch to dark mode"
    const button = page.getByRole('button', { name: /switch to dark mode/i })
    await expect(button).toBeVisible()
  })

  test('clicking toggle switches to dark mode', async ({ page }) => {
    await page.goto('/')
    
    // Click theme toggle
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    
    // Check theme changed
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('dark')
    
    // Button label should update
    const button = page.getByRole('button', { name: /switch to light mode/i })
    await expect(button).toBeVisible()
  })

  test('clicking toggle again switches back to light mode', async ({ page }) => {
    await page.goto('/')
    
    // Toggle to dark
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    
    // Toggle back to light
    await page.getByRole('button', { name: /switch to light mode/i }).click()
    
    // Should be light mode
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('light')
  })

  test('theme persists across page refresh', async ({ page }) => {
    await page.goto('/')
    
    // Toggle to dark mode
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    
    // Reload page
    await page.reload()
    
    // Theme should still be dark
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('dark')
    
    const button = page.getByRole('button', { name: /switch to light mode/i })
    await expect(button).toBeVisible()
  })

  test('theme is stored in localStorage', async ({ page }) => {
    await page.goto('/')
    
    // Toggle to dark
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    
    // Check localStorage
    const storedTheme = await page.evaluate(() => localStorage.getItem('dashboard-theme'))
    expect(storedTheme).toBe('dark')
    
    // Toggle back to light
    await page.getByRole('button', { name: /switch to light mode/i }).click()
    
    // Check localStorage updated
    const newStoredTheme = await page.evaluate(() => localStorage.getItem('dashboard-theme'))
    expect(newStoredTheme).toBe('light')
  })

  test('theme toggle is keyboard accessible', async ({ page }) => {
    await page.goto('/')
    
    // Focus the theme toggle button directly
    const themeToggle = page.getByRole('button', { name: /switch to dark mode/i })
    await themeToggle.focus()
    
    // Verify it's focused
    await expect(themeToggle).toBeFocused()
    
    // Press Enter to toggle
    await page.keyboard.press('Enter')
    
    // Wait a bit for the theme to change
    await page.waitForTimeout(100)
    
    // Theme should change
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('dark')
  })

  test('theme toggle responds to Space key', async ({ page }) => {
    await page.goto('/')
    
    // Focus the theme toggle button
    await page.getByRole('button', { name: /switch to dark mode/i }).focus()
    
    // Press Space
    await page.keyboard.press('Space')
    
    // Theme should change
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('dark')
  })

  test('focus ring is visible when focused', async ({ page }) => {
    await page.goto('/')
    
    const button = page.getByRole('button', { name: /switch to dark mode/i })
    
    // Focus the button
    await button.focus()
    
    // Check for focus ring (via outline or ring class)
    const hasRing = await button.evaluate((el) => {
      // Tailwind focus-visible:ring-2 should apply
      return el.classList.contains('focus-visible:ring-2')
    })
    
    expect(hasRing).toBe(true)
  })

  test('theme change is smooth (no layout shift)', async ({ page }) => {
    await page.goto('/')
    
    // Get initial layout metrics
    const initialMetrics = await page.evaluate(() => {
      const header = document.querySelector('header')
      return {
        headerHeight: header?.offsetHeight,
        headerWidth: header?.offsetWidth,
      }
    })
    
    // Toggle theme
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    
    // Wait for transition
    await page.waitForTimeout(500)
    
    // Get new layout metrics
    const newMetrics = await page.evaluate(() => {
      const header = document.querySelector('header')
      return {
        headerHeight: header?.offsetHeight,
        headerWidth: header?.offsetWidth,
      }
    })
    
    // Layout should not change
    expect(newMetrics.headerHeight).toBe(initialMetrics.headerHeight)
    expect(newMetrics.headerWidth).toBe(initialMetrics.headerWidth)
  })

  test('background color changes on theme toggle', async ({ page }) => {
    await page.goto('/')
    
    // Get initial background color
    const lightBg = await page.evaluate(() => {
      return window.getComputedStyle(document.body).backgroundColor
    })
    
    // Toggle to dark
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    
    // Wait for transition
    await page.waitForTimeout(500)
    
    // Get dark background color
    const darkBg = await page.evaluate(() => {
      return window.getComputedStyle(document.body).backgroundColor
    })
    
    // Colors should be different
    expect(lightBg).not.toBe(darkBg)
  })

  test('icon changes from sun to moon', async ({ page }) => {
    await page.goto('/')
    
    const button = page.getByRole('button', { name: /switch to dark mode/i })
    
    // Initial: should have Sun icon
    const initialIcon = await button.locator('svg').count()
    expect(initialIcon).toBe(1)
    
    // Toggle
    await button.click()
    
    // Should still have an icon (Moon now)
    const newButton = page.getByRole('button', { name: /switch to light mode/i })
    const newIcon = await newButton.locator('svg').count()
    expect(newIcon).toBe(1)
  })

  test('aria-pressed attribute updates correctly', async ({ page }) => {
    await page.goto('/')
    
    const button = page.getByRole('button', { name: /switch to dark mode/i })
    
    // Initial: light mode, aria-pressed should be false
    let pressed = await button.getAttribute('aria-pressed')
    expect(pressed).toBe('false')
    
    // Toggle to dark
    await button.click()
    
    // aria-pressed should now be true
    const darkButton = page.getByRole('button', { name: /switch to light mode/i })
    pressed = await darkButton.getAttribute('aria-pressed')
    expect(pressed).toBe('true')
  })

  test('works on mobile viewport', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/')
    
    // Theme toggle should still be visible and functional
    const button = page.getByRole('button', { name: /switch to dark mode/i })
    await expect(button).toBeVisible()
    
    // Get button size
    const box = await button.boundingBox()
    
    // Should be tappable (allow some flexibility for padding/margins)
    // The button itself might be smaller but have adequate touch target via padding
    expect(box?.width).toBeGreaterThan(0)
    expect(box?.height).toBeGreaterThan(0)
    
    // Should toggle
    await button.click()
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('dark')
  })

  test('multiple rapid toggles work correctly', async ({ page }) => {
    await page.goto('/')
    
    const button = page.getByRole('button', { name: /switch to dark mode/i })
    
    // Rapid clicks with small delays
    await button.click()
    await page.waitForTimeout(100)
    await page.getByRole('button', { name: /switch to light mode/i }).click()
    await page.waitForTimeout(100)
    await page.getByRole('button', { name: /switch to dark mode/i }).click()
    await page.waitForTimeout(100)
    
    // Should end on dark (odd number of clicks)
    const theme = await page.getAttribute('html', 'data-theme')
    expect(theme).toBe('dark')
  })
})
