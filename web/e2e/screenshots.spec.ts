/**
 * Screenshot Tests
 * 
 * Captures screenshots of key pages in light and dark modes
 * for use in documentation and visual regression testing.
 */

import { test, expect } from '@playwright/test';

const pages = [
  { name: 'home', path: '/' },
  { name: 'quick-start', path: '/quick-start/' },
  { name: 'mcp', path: '/mcp/' },
  { name: 'mcp-setup', path: '/mcp/setup/' },
  { name: 'mcp-ai-debugging', path: '/mcp/ai-debugging/' },
  { name: 'tour', path: '/tour/' },
  { name: 'examples', path: '/examples/' },
  { name: 'cli-reference', path: '/reference/cli/' },
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
      
      await expect(page).toHaveScreenshot(`${pageInfo.name}-light.png`, {
        fullPage: true,
        animations: 'disabled',
      });
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
      
      await expect(page).toHaveScreenshot(`${pageInfo.name}-dark.png`, {
        fullPage: true,
        animations: 'disabled',
      });
    });
  }
});

test.describe('Component Screenshots', () => {
  test('code block with copy button', async ({ page }) => {
    await page.goto('/quick-start/');
    await page.waitForLoadState('networkidle');
    
    const codeBlock = page.locator('.code-block').first();
    await expect(codeBlock).toHaveScreenshot('code-block.png');
  });

  test('search modal', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Open search with keyboard shortcut
    await page.keyboard.press('/');
    await page.waitForSelector('.search-modal.open');
    
    const modal = page.locator('.search-modal');
    await expect(modal).toHaveScreenshot('search-modal.png');
  });

  test('navigation menu on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Open mobile menu
    await page.click('[data-mobile-menu-toggle]');
    await page.waitForTimeout(300);
    
    await expect(page).toHaveScreenshot('mobile-menu.png');
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
      await page.goto('/');
      await page.waitForLoadState('networkidle');
      
      await expect(page).toHaveScreenshot(`home-${viewport.name}.png`, {
        fullPage: true,
        animations: 'disabled',
      });
    });
  }
});
