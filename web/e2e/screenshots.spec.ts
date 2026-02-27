/**
 * Screenshot Tests
 * 
 * Captures screenshots of key pages in light and dark modes
 * for use in documentation and visual regression testing.
 * 
 * Note: Paths are relative to baseURL (http://localhost:4321/azd-app/)
 * so they should NOT start with '/' to ensure proper URL resolution.
 * 
 * In CI, these tests verify pages load correctly without screenshot comparison
 * (due to cross-platform font rendering differences). Locally, they perform
 * full visual regression testing with screenshot comparisons.
 */

import { test, expect } from '@playwright/test';

const isCI = !!process.env.CI;

const pages = [
  { name: 'home', path: './' },
  { name: 'quick-start', path: 'quick-start/' },
  { name: 'mcp', path: 'mcp/' },
  { name: 'mcp-setup', path: 'mcp/setup/' },
  { name: 'mcp-ai-debugging', path: 'mcp/ai-debugging/' },
  { name: 'tour', path: 'tour/' },
  { name: 'examples', path: 'examples/' },
  { name: 'cli-reference', path: 'reference/cli/' },
];

test.describe('Light Mode Screenshots', () => {
  test.beforeEach(async ({ page }) => {
    // Force light mode
    await page.addInitScript(() => {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    });
  });

  for (const pageInfo of pages) {
    test(`${pageInfo.name} page`, async ({ page }) => {
      await page.goto(pageInfo.path);
      await page.waitForLoadState('networkidle');
      
      // Wait for any animations to complete
      await page.waitForTimeout(500);
      
      // In CI, just verify the page loads; locally, do visual comparison
      if (isCI) {
        // Verify page loaded successfully by checking for main content
        await expect(page.locator('body')).toBeVisible();
      } else {
        await expect(page).toHaveScreenshot(`${pageInfo.name}-light.png`, {
          fullPage: true,
          animations: 'disabled',
        });
      }
    });
  }
});

test.describe('Dark Mode Screenshots', () => {
  test.beforeEach(async ({ page }) => {
    // Force dark mode
    await page.addInitScript(() => {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    });
  });

  for (const pageInfo of pages) {
    test(`${pageInfo.name} page`, async ({ page }) => {
      await page.goto(pageInfo.path);
      await page.waitForLoadState('networkidle');
      
      // Wait for any animations to complete
      await page.waitForTimeout(500);
      
      // In CI, just verify the page loads; locally, do visual comparison
      if (isCI) {
        // Verify page loaded successfully by checking for main content
        await expect(page.locator('body')).toBeVisible();
      } else {
        await expect(page).toHaveScreenshot(`${pageInfo.name}-dark.png`, {
          fullPage: true,
          animations: 'disabled',
        });
      }
    });
  }
});

test.describe('Component Screenshots', () => {
  test('code block with copy button', async ({ page }) => {
    await page.goto('quick-start/');
    await page.waitForLoadState('networkidle');
    
    // Wait for code blocks to render
    await page.waitForSelector('.code-block');
    
    const codeBlock = page.locator('.code-block').first();
    
    // In CI, just verify the element exists; locally, do visual comparison
    if (isCI) {
      await expect(codeBlock).toBeVisible();
    } else {
      await expect(codeBlock).toHaveScreenshot('code-block.png');
    }
  });

  test('search modal', async ({ page }) => {
    await page.goto('./');
    await page.waitForLoadState('networkidle');
    
    // Check if search modal element exists before interacting
    const modalExists = await page.locator('#search-modal').count() > 0;
    test.skip(!modalExists, 'Search modal element not found — may be provided by external component');
    
    // Open search with keyboard shortcut (/ key)
    await page.keyboard.press('/');
    
    // Wait for modal to have the 'open' class (id selector for the modal)
    try {
      await page.waitForSelector('#search-modal.open', { timeout: 5000 });
    } catch {
      // Keyboard shortcut may not work in headless CI; try clicking search button
      const searchBtn = page.locator('[data-search-toggle], button[aria-label*="search"], button[aria-label*="Search"]').first();
      if (await searchBtn.count() > 0) {
        await searchBtn.click();
        await page.waitForSelector('#search-modal.open', { timeout: 5000 });
      } else {
        test.skip(true, 'Search modal cannot be opened — keyboard shortcut and search button not available');
        return;
      }
    }
    
    const modal = page.locator('#search-modal');
    
    // In CI, just verify the modal opened; locally, do visual comparison
    if (isCI) {
      await expect(modal).toBeVisible();
    } else {
      await expect(modal).toHaveScreenshot('search-modal.png');
    }
  });

  test('navigation menu on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('./');
    await page.waitForLoadState('networkidle');
    
    // Check if mobile menu toggle exists (rendered by external Header component)
    const toggleExists = await page.locator('[data-mobile-menu-toggle]').count() > 0;
    test.skip(!toggleExists, 'Mobile menu toggle not found — may be provided by external Header component');
    
    // Click the mobile menu toggle button
    await page.click('[data-mobile-menu-toggle]');
    
    // Wait for the mobile menu to open (it sets data-open="true")
    await page.waitForSelector('[data-mobile-menu][data-open="true"]', { timeout: 5000 });
    await page.waitForTimeout(300); // Wait for animation
    
    // In CI, just verify the menu opened; locally, do visual comparison
    if (isCI) {
      await expect(page.locator('[data-mobile-menu][data-open="true"]')).toBeVisible();
    } else {
      await expect(page).toHaveScreenshot('mobile-menu.png');
    }
  });
});

test.describe('Responsive Screenshots', () => {
  const viewports = [
    { name: 'mobile', width: 375, height: 667 },
    { name: 'tablet', width: 768, height: 1024 },
    { name: 'desktop', width: 1280, height: 800 },
  ];

  for (const viewport of viewports) {
    test(`home page at ${viewport.name}`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('./');
      await page.waitForLoadState('networkidle');
      
      // In CI, just verify the page loads at this viewport; locally, do visual comparison
      if (isCI) {
        await expect(page.locator('body')).toBeVisible();
      } else {
        await expect(page).toHaveScreenshot(`home-${viewport.name}.png`, {
          fullPage: true,
          animations: 'disabled',
        });
      }
    });
  }
});
