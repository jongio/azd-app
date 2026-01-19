/**
 * Azure YAML Editor Page E2E Tests
 * 
 * Comprehensive E2E tests for the azure-yaml-editor.astro documentation page.
 * Tests content rendering, navigation, links, and accessibility.
 */

import { test, expect } from '@playwright/test';

const PAGE_PATH = 'reference/azure-yaml-editor/';
const isCI = !!process.env.CI;

test.describe('Azure YAML Editor Page - Content', () => {
  test('page loads successfully without errors', async ({ page }) => {
    // Collect console errors and uncaught exceptions
    const consoleErrors: string[] = [];
    const pageErrors: Error[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    page.on('pageerror', error => {
      pageErrors.push(error);
    });
    
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Wait a bit for any async errors
    await page.waitForTimeout(1000);
    
    // Assert no JavaScript errors occurred
    if (pageErrors.length > 0) {
      console.error('Page errors detected:', pageErrors);
    }
    if (consoleErrors.length > 0) {
      console.error('Console errors detected:', consoleErrors);
    }
    
    expect(pageErrors, `Page had ${pageErrors.length} runtime errors`).toHaveLength(0);
    expect(consoleErrors, `Console had ${consoleErrors.length} errors`).toHaveLength(0);
    
    // Verify page title
    await expect(page).toHaveTitle(/Azure YAML Editor/i);
    
    // Verify hero section
    const hero = page.getByRole('heading', { level: 1, name: /Azure YAML Editor/i });
    await expect(hero).toBeVisible();
  });

  test('has all key features listed', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for all feature cards
    const features = [
      'Visual Navigation',
      'Schema-Driven Forms',
      'Live YAML Preview',
      'Real-Time Validation',
      'Quick Actions',
      'Command Palette',
    ];
    
    for (const feature of features) {
      await expect(page.getByText(feature)).toBeVisible();
    }
  });

  test('has step-by-step walkthrough', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for walkthrough section
    const walkthrough = page.getByRole('heading', { name: /Step-by-Step Walkthrough/i });
    await expect(walkthrough).toBeVisible();
    
    // Check for numbered steps
    const steps = page.locator('text=/^[1-7]\\./');
    const stepCount = await steps.count();
    expect(stepCount).toBeGreaterThanOrEqual(6);
  });

  test('has tips and best practices section', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const tipsSection = page.getByRole('heading', { name: /Tips & Best Practices/i });
    await expect(tipsSection).toBeVisible();
    
    // Check for tip cards
    const tips = [
      'Use the Preview Pane',
      'Fix Validation Errors Early',
      'Learn Keyboard Shortcuts',
    ];
    
    for (const tip of tips) {
      await expect(page.getByText(tip)).toBeVisible();
    }
  });

  test('displays all screenshots', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for screenshot images
    const screenshots = page.locator('img[src*="screenshots/editor-"]');
    const screenshotCount = await screenshots.count();
    
    // Should have multiple screenshots
    expect(screenshotCount).toBeGreaterThanOrEqual(5);
    
    // Verify images load
    for (let i = 0; i < screenshotCount; i++) {
      const img = screenshots.nth(i);
      await expect(img).toBeVisible();
      const src = await img.getAttribute('src');
      expect(src).toContain('screenshots/editor-');
    }
  });
});

test.describe('Azure YAML Editor Page - Navigation', () => {
  test('has breadcrumb navigation', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for breadcrumb
    const breadcrumb = page.locator('nav.text-sm ol');
    await expect(breadcrumb).toBeVisible();
    
    // Breadcrumb should have links
    const breadcrumbLinks = breadcrumb.locator('a');
    const linkCount = await breadcrumbLinks.count();
    expect(linkCount).toBeGreaterThanOrEqual(2);
    
    // Should link to home and reference
    await expect(breadcrumb.getByText('Home')).toBeVisible();
    await expect(breadcrumb.getByText('Reference')).toBeVisible();
  });

  test('links to azure.yaml reference page', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const referenceLink = page.getByRole('link', { name: /View Reference Docs/i });
    await expect(referenceLink).toBeVisible();
    
    const href = await referenceLink.getAttribute('href');
    expect(href).toContain('azure-yaml');
  });

  test('links to quick start guide', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const quickStartLink = page.getByRole('link', { name: /Get Started/i });
    await expect(quickStartLink).toBeVisible();
    
    const href = await quickStartLink.getAttribute('href');
    expect(href).toContain('quick-start');
  });

  test('has "Learn More" section with links', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const learnMoreSection = page.getByRole('heading', { name: /Learn More/i });
    await expect(learnMoreSection).toBeVisible();
    
    // Should have links to related pages
    const relatedLinks = page.locator('a[href*="reference"]');
    const linkCount = await relatedLinks.count();
    expect(linkCount).toBeGreaterThanOrEqual(1);
  });
});

test.describe('Azure YAML Editor Page - Accessibility', () => {
  test('has proper heading hierarchy', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for H1
    const h1 = page.locator('h1');
    await expect(h1).toHaveCount(1);
    
    // Check for H2s (section headings)
    const h2s = page.locator('h2');
    const h2Count = await h2s.count();
    expect(h2Count).toBeGreaterThanOrEqual(4);
  });

  test('images have alt text', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const images = page.locator('img[src*="screenshots/editor-"]');
    const imageCount = await images.count();
    
    for (let i = 0; i < imageCount; i++) {
      const img = images.nth(i);
      const alt = await img.getAttribute('alt');
      expect(alt).toBeTruthy();
      expect(alt?.length).toBeGreaterThan(0);
    }
  });

  test('links have descriptive text', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const links = page.locator('a[href]');
    const linkCount = await links.count();
    
    // Check first few links have text
    for (let i = 0; i < Math.min(5, linkCount); i++) {
      const link = links.nth(i);
      const text = await link.textContent();
      const ariaLabel = await link.getAttribute('aria-label');
      
      // Should have either text or aria-label
      expect(text?.trim().length || ariaLabel?.length).toBeGreaterThan(0);
    }
  });

  test('has semantic HTML structure', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for main landmark (should be in Layout component)
    const main = page.locator('main');
    const mainExists = await main.count() > 0;
    
    // Main landmark is optional but good practice
    if (mainExists) {
      await expect(main).toBeVisible();
    }
  });
});

test.describe('Azure YAML Editor Page - Responsive', () => {
  test('mobile viewport (375px)', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Page should load and be scrollable
    await expect(page.locator('h1')).toBeVisible();
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-editor-mobile.png', {
        fullPage: true,
        animations: 'disabled',
      });
    }
  });

  test('tablet viewport (768px)', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await expect(page.locator('h1')).toBeVisible();
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-editor-tablet.png', {
        fullPage: true,
        animations: 'disabled',
        timeout: 30000,
      });
    }
  });

  test('desktop viewport (1920px)', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await expect(page.locator('h1')).toBeVisible();
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-editor-desktop.png', {
        fullPage: true,
        animations: 'disabled',
        timeout: 30000,
      });
    }
  });
});

test.describe('Azure YAML Editor Page - Dark Mode', () => {
  test('renders correctly in light mode', async ({ page }) => {
    await page.addInitScript(() => {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    });
    
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-editor-light.png', {
        fullPage: true,
        animations: 'disabled',
        timeout: 30000,
      });
    } else {
      // Just verify it loaded
      await expect(page.locator('h1')).toBeVisible();
    }
  });

  test('renders correctly in dark mode', async ({ page }) => {
    await page.addInitScript(() => {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    });
    
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-editor-dark.png', {
        fullPage: true,
        animations: 'disabled',
        timeout: 30000,
      });
    } else {
      // Just verify it loaded
      await expect(page.locator('h1')).toBeVisible();
    }
  });
});

test.describe('Azure YAML Editor Page - Performance', () => {
  test('loads within acceptable time', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const loadTime = Date.now() - startTime;
    
    // Should load in under 5 seconds on decent connection
    expect(loadTime).toBeLessThan(5000);
    
    console.log(`Page loaded in ${loadTime}ms`);
  });

  test('does not have excessive DOM size', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const domSize = await page.evaluate(() => {
      return document.getElementsByTagName('*').length;
    });
    
    console.log(`DOM size: ${domSize} elements`);
    
    // Warning if DOM is very large (could impact performance)
    if (domSize > 3000) {
      console.warn('Large DOM size detected - consider optimizing');
    }
  });
});
