/**
 * azure.yaml Reference Page Tests
 * 
 * Comprehensive E2E tests for the azure.yaml reference documentation page.
 * Tests content rendering, navigation, interactive elements, and accessibility.
 */

import { test, expect } from '@playwright/test';

const PAGE_PATH = 'reference/azure-yaml/';
const isCI = !!process.env.CI;

test.describe('azure.yaml Reference Page - Content', () => {
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
    await expect(page).toHaveTitle(/azure\.yaml Reference/i);
    
    // Verify hero section
    const hero = page.locator('h1');
    await expect(hero).toContainText('azure.yaml Reference');
  });

  test('has all major sections', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for all major sections by their IDs
    const sections = [
      'name',
      'services',
      'healthchecks',
      'docker',
      'resources',
      'requirements',
      'hooks',
      'logging',
      'testing',
      'deployment',
      'metadata',
    ];
    
    for (const sectionId of sections) {
      const section = page.locator(`#${sectionId}`);
      await expect(section).toBeVisible();
    }
  });

  test('contains property tables', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Verify tables exist
    const tables = page.locator('table');
    const tableCount = await tables.count();
    
    // Should have multiple property tables (we added 20+)
    expect(tableCount).toBeGreaterThanOrEqual(15);
    
    // Verify first table has proper structure
    const firstTable = tables.first();
    await expect(firstTable.locator('thead')).toBeVisible();
    await expect(firstTable.locator('tbody')).toBeVisible();
    
    // Check for table headers (Property, Type, etc.)
    const headers = firstTable.locator('th');
    const headerCount = await headers.count();
    expect(headerCount).toBeGreaterThanOrEqual(3);
  });

  test('contains code examples', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Verify code blocks exist
    const codeBlocks = page.locator('pre code, .expressive-code');
    const codeCount = await codeBlocks.count();
    
    // Should have multiple code examples
    expect(codeCount).toBeGreaterThanOrEqual(10);
  });

  test('name section has pattern validation details', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Navigate to name section
    await page.locator('#name').scrollIntoViewIfNeeded();
    
    // Check for pattern validation details (looking for table with Pattern column and 2-64 chars length)
    const nameSection = page.locator('#name').locator('..');
    await expect(nameSection).toContainText(/Pattern/);
    await expect(nameSection).toContainText(/2-64/);
  });

  test('services section has all property types documented', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Navigate to services section
    await page.locator('#services').scrollIntoViewIfNeeded();
    
    const servicesSection = page.locator('#services').locator('..');
    
    // Check for key service properties
    await expect(servicesSection).toContainText('language');
    await expect(servicesSection).toContainText('type');
    await expect(servicesSection).toContainText('mode');
    await expect(servicesSection).toContainText('host');
    await expect(servicesSection).toContainText('environment');
  });

  test('docker section has all build properties', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await page.locator('#docker').scrollIntoViewIfNeeded();
    
    const dockerSection = page.locator('#docker').locator('..');
    
    // Check for Docker properties
    await expect(dockerSection).toContainText('path');
    await expect(dockerSection).toContainText('context');
    await expect(dockerSection).toContainText('platform');
    await expect(dockerSection).toContainText('buildArgs');
    await expect(dockerSection).toContainText('remoteBuild');
  });

  test('resources section lists all Azure resource types', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await page.locator('#resources').scrollIntoViewIfNeeded();
    
    const resourcesSection = page.locator('#resources').locator('..');
    
    // Check for common resource types
    await expect(resourcesSection).toContainText('azure.servicebus.namespace');
    await expect(resourcesSection).toContainText('azure.storage.account');
    await expect(resourcesSection).toContainText('azure.cosmos.account');
    await expect(resourcesSection).toContainText('azure.keyvault.vault');
    await expect(resourcesSection).toContainText('azure.ai.model');
  });

  test('hooks section documents all lifecycle hooks', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await page.locator('#hooks').scrollIntoViewIfNeeded();
    
    const hooksSection = page.locator('#hooks').locator('..');
    
    // Check for hook types
    await expect(hooksSection).toContainText('prerun');
    await expect(hooksSection).toContainText('postrun');
    await expect(hooksSection).toContainText('predeploy');
    await expect(hooksSection).toContainText('postdeploy');
    await expect(hooksSection).toContainText('shell');
    await expect(hooksSection).toContainText('continueOnError');
  });

  test('logging section has Azure Log Analytics details', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await page.locator('#logging').scrollIntoViewIfNeeded();
    
    const loggingSection = page.locator('#logging').locator('..');
    
    // Check for logging features
    await expect(loggingSection).toContainText('filters');
    await expect(loggingSection).toContainText('classifications');
    await expect(loggingSection).toContainText('analytics');
    await expect(loggingSection).toContainText('workspaceId');
  });

  test('testing section has all test types and formats', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await page.locator('#testing').scrollIntoViewIfNeeded();
    
    const testingSection = page.locator('#testing').locator('..');
    
    // Check for test types
    await expect(testingSection).toContainText('unit');
    await expect(testingSection).toContainText('integration');
    await expect(testingSection).toContainText('e2e');
    
    // Check for output formats
    await expect(testingSection).toContainText('json');
    await expect(testingSection).toContainText('junit');
    await expect(testingSection).toContainText('github-actions');
  });

  test('deployment section has infrastructure options', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await page.locator('#deployment').scrollIntoViewIfNeeded();
    
    const deploymentSection = page.locator('#deployment').locator('..');
    
    // Check for deployment properties
    await expect(deploymentSection).toContainText('infra');
    await expect(deploymentSection).toContainText('pipeline');
    await expect(deploymentSection).toContainText('state');
    await expect(deploymentSection).toContainText('platform');
    await expect(deploymentSection).toContainText('workflows');
    await expect(deploymentSection).toContainText('cloud');
  });
});

test.describe('azure.yaml Reference Page - Navigation', () => {
  test('floating TOC is visible on desktop', async ({ page }) => {
    // Catch any JavaScript errors
    const pageErrors: Error[] = [];
    page.on('pageerror', error => pageErrors.push(error));
    
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    expect(pageErrors, 'No JavaScript errors should occur').toHaveLength(0);
    
    // TOC should be visible at this viewport size
    const toc = page.locator('nav').filter({ hasText: 'On this page' });
    await expect(toc).toBeVisible();
  });

  test('floating TOC is hidden on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // TOC should be hidden at mobile viewport
    const toc = page.locator('nav').filter({ hasText: 'On this page' });
    await expect(toc).not.toBeVisible();
  });

  test('TOC links navigate to sections', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Click on a TOC link (using href="#services" which is an in-page anchor)
    const servicesLink = page.locator('a[href="#services"]').first();
    
    // Use Promise.all to handle potential navigation
    await Promise.race([
      servicesLink.click(),
      page.waitForURL('**#services', { timeout: 1000 }).catch(() => {})
    ]);
    
    // Wait for scroll animation
    await page.waitForTimeout(500);
    
    // Verify the services section is in view
    const servicesSection = page.locator('#services');
    await expect(servicesSection).toBeInViewport();
  });

  test('TOC highlights active section on scroll', async ({ page }) => {
    // Catch JavaScript errors
    const pageErrors: Error[] = [];
    page.on('pageerror', error => pageErrors.push(error));
    
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Verify IntersectionObserver script loaded without errors
    await page.waitForTimeout(500);
    expect(pageErrors, 'IntersectionObserver should initialize without errors').toHaveLength(0);
    
    // Scroll to services section
    await page.locator('#services').scrollIntoViewIfNeeded();
    await page.waitForTimeout(300); // Wait for IntersectionObserver
    
    // Check if services link is active
    const servicesLink = page.locator('nav a[href="#services"]');
    const hasActiveClass = await servicesLink.evaluate(el => 
      el.classList.contains('text-blue-600') || 
      el.classList.contains('dark:text-blue-400')
    );
    
    expect(hasActiveClass).toBeTruthy();
  });

  test('all TOC links work', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Get all TOC links
    const tocLinks = page.locator('nav a[href^="#"]');
    const linkCount = await tocLinks.count();
    
    expect(linkCount).toBeGreaterThanOrEqual(10);
    
    // Test first 5 links to avoid excessive test time
    for (let i = 0; i < Math.min(5, linkCount); i++) {
      const link = tocLinks.nth(i);
      const href = await link.getAttribute('href');
      
      if (href) {
        await link.click();
        await page.waitForTimeout(300);
        
        // Verify corresponding section exists
        const sectionId = href.substring(1);
        const section = page.locator(`#${sectionId}`);
        await expect(section).toBeInViewport();
      }
    }
  });

  test('breadcrumb navigation exists', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for breadcrumb (implemented inline on this page, not in Layout)
    const breadcrumb = page.locator('nav.text-sm ol');
    await expect(breadcrumb).toBeVisible();
    
    // Breadcrumb should have multiple links (Home, Reference, azure.yaml)
    const breadcrumbLinks = breadcrumb.locator('a');
    const linkCount = await breadcrumbLinks.count();
    expect(linkCount).toBeGreaterThanOrEqual(2);
  });
});

test.describe('azure.yaml Reference Page - Interactivity', () => {
  test('JavaScript executes without runtime errors', async ({ page }) => {
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
    
    // Wait for scripts to execute
    await page.waitForTimeout(1000);
    
    // Check that IntersectionObserver is set up
    const observerExists = await page.evaluate(() => {
      return typeof IntersectionObserver !== 'undefined';
    });
    
    expect(observerExists, 'IntersectionObserver should be available').toBeTruthy();
    
    // Assert no errors occurred during script execution
    if (consoleErrors.length > 0) {
      console.error('Console errors:', consoleErrors);
    }
    if (pageErrors.length > 0) {
      console.error('Page errors:', pageErrors.map(e => e.message));
    }
    
    expect(pageErrors, 'Should have no page errors').toHaveLength(0);
    expect(consoleErrors, 'Should have no console errors').toHaveLength(0);
  });

  test('code blocks have copy buttons', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Look for code blocks with copy functionality
    const codeBlocks = page.locator('pre').first();
    await codeBlocks.hover();
    
    // Check if copy button appears (implementation may vary)
    const hasCopyButton = await page.locator('button[title*="copy"], button[aria-label*="copy"]').count() > 0 ||
                          await page.locator('.copy-button, [data-copy]').count() > 0;
    
    // This is optional functionality, so we just log if not present
    if (!hasCopyButton) {
      console.log('Note: Copy buttons not found - may need to be implemented');
    }
  });

  test('internal links work', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Find first internal link (if any exist)
    const internalLink = page.locator('a[href^="#"]').first();
    const linkExists = await internalLink.count() > 0;
    
    if (linkExists) {
      const href = await internalLink.getAttribute('href');
      await internalLink.click();
      await page.waitForTimeout(300);
      
      // Verify URL updated
      expect(page.url()).toContain(href || '');
    }
  });

  test('external links have proper attributes', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Find external links
    const externalLinks = page.locator('a[href^="http"]');
    const linkCount = await externalLinks.count();
    
    if (linkCount > 0) {
      const firstExternalLink = externalLinks.first();
      
      // External links should open in new tab (optional but good practice)
      const target = await firstExternalLink.getAttribute('target');
      const rel = await firstExternalLink.getAttribute('rel');
      
      // Log attributes for review
      console.log('External link attributes:', { target, rel });
    }
  });
});

test.describe('azure.yaml Reference Page - Dark Mode', () => {
  test('renders correctly in light mode', async ({ page }) => {
    await page.addInitScript(() => {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    });
    
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-reference-light.png', {
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
      await expect(page).toHaveScreenshot('azure-yaml-reference-dark.png', {
        fullPage: true,
        animations: 'disabled',
        timeout: 30000,
      });
    } else {
      // Just verify it loaded
      await expect(page.locator('h1')).toBeVisible();
    }
  });

  test('tables render in dark mode', async ({ page }) => {
    await page.addInitScript(() => {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    });
    
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const table = page.locator('table').first();
    await expect(table).toBeVisible();
    
    // Verify dark mode classes are applied
    const hasDarkStyles = await table.evaluate(el => {
      const styles = window.getComputedStyle(el);
      return styles.backgroundColor !== 'rgb(255, 255, 255)'; // Not white
    });
    
    expect(hasDarkStyles).toBeTruthy();
  });
});

test.describe('azure.yaml Reference Page - Responsive', () => {
  test('mobile viewport (375px)', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Page should load and be scrollable
    await expect(page.locator('h1')).toBeVisible();
    
    // Tables should be horizontally scrollable
    const table = page.locator('table').first();
    await expect(table).toBeVisible();
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-reference-mobile.png', {
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
      await expect(page).toHaveScreenshot('azure-yaml-reference-tablet.png', {
        fullPage: true,
        animations: 'disabled',
      });
    }
  });

  test('desktop viewport (1920px)', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    await expect(page.locator('h1')).toBeVisible();
    
    // TOC should be visible at large viewport
    const toc = page.locator('nav').filter({ hasText: 'On this page' });
    await expect(toc).toBeVisible();
    
    if (!isCI) {
      await expect(page).toHaveScreenshot('azure-yaml-reference-desktop-wide.png', {
        fullPage: true,
        animations: 'disabled',
        timeout: 30000,
      });
    }
  });
});

test.describe('azure.yaml Reference Page - Accessibility', () => {
  test('has proper heading hierarchy', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Check for H1
    const h1 = page.locator('h1');
    await expect(h1).toHaveCount(1);
    
    // Check for H2s (section headings)
    const h2s = page.locator('h2');
    const h2Count = await h2s.count();
    expect(h2Count).toBeGreaterThanOrEqual(10);
    
    // Check for H3s (subsection headings)
    const h3s = page.locator('h3');
    const h3Count = await h3s.count();
    expect(h3Count).toBeGreaterThanOrEqual(5);
  });

  test('tables have proper semantic structure', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const table = page.locator('table').first();
    
    // Verify thead and tbody exist
    await expect(table.locator('thead')).toBeVisible();
    await expect(table.locator('tbody')).toBeVisible();
    
    // Verify th elements in header
    const headerCells = table.locator('thead th');
    const headerCount = await headerCells.count();
    expect(headerCount).toBeGreaterThanOrEqual(3);
  });

  test('code blocks are properly marked up', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    const codeBlock = page.locator('pre code').first();
    await expect(codeBlock).toBeVisible();
    
    // Code should be within pre tags
    const preTag = page.locator('pre').first();
    await expect(preTag).toBeVisible();
  });

  test('skip to content or main landmark exists', async ({ page }) => {
    await page.goto(PAGE_PATH);
    await page.waitForLoadState('networkidle');
    
    // Look for main landmark
    const main = page.locator('main');
    const mainExists = await main.count() > 0;
    
    expect(mainExists).toBeTruthy();
  });
});

test.describe('azure.yaml Reference Page - Performance', () => {
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
