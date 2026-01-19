/**
 * Screenshot Capture Module - Playwright screenshot logic
 */
import { type Page } from 'playwright';
import * as fs from 'fs';
import * as path from 'path';
import type { ValidationRule, ScreenshotConfig } from './screenshot-config.js';
import { SERVICE_ELEMENTS, ERROR_SELECTORS } from './screenshot-config.js';

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Calculate clip region from element bounding box with padding
 */
async function getElementClip(
  page: Page, 
  selector: string, 
  padding: number = 20
): Promise<{ x: number; y: number; width: number; height: number } | null> {
  try {
    const element = await page.$(selector);
    if (!element) return null;
    
    const box = await element.boundingBox();
    if (!box) return null;
    
    const viewport = page.viewportSize();
    if (!viewport) return null;
    
    // Add padding for context, but ensure we don't go outside viewport
    return {
      x: Math.max(0, Math.min(box.x - padding, viewport.width - (box.width + padding * 2))),
      y: Math.max(0, Math.min(box.y - padding, viewport.height - (box.height + padding * 2))),
      width: Math.min(box.width + (padding * 2), viewport.width),
      height: Math.min(box.height + (padding * 2), viewport.height)
    };
  } catch {
    return null;
  }
}

async function checkElement(page: Page, rule: ValidationRule): Promise<string[]> {
  const errors: string[] = [];
  try {
    const elements = await page.$$(rule.selector);
    if (elements.length === 0) return [`Missing: ${rule.description} (selector: ${rule.selector})`];
    if (rule.minCount && elements.length < rule.minCount) {
      errors.push(`Not enough: ${rule.description} - found ${elements.length}, expected ${rule.minCount}+`);
    }
    if (rule.textContent) {
      const foundMatch = await Promise.all(elements.map(async el => {
        const text = await el.textContent();
        return text && (typeof rule.textContent === 'string' 
          ? text.includes(rule.textContent) 
          : rule.textContent.test(text));
      })).then(results => results.some(Boolean));
      if (!foundMatch) errors.push(`Wrong content: ${rule.description}`);
    }
    if (rule.mustBeVisible && !(await elements[0].isVisible())) {
      errors.push(`Not visible: ${rule.description}`);
    }
  } catch (e) {
    errors.push(`Error checking ${rule.description}: ${e}`);
  }
  return errors;
}

async function checkServices(page: Page): Promise<boolean> {
  for (const rule of SERVICE_ELEMENTS) {
    if ((await page.$$(rule.selector)).length >= (rule.minCount || 1)) return true;
  }
  return false;
}

async function checkErrorStates(page: Page): Promise<string[]> {
  const errors: string[] = [];
  for (const ec of ERROR_SELECTORS.filter(e => !e.checkClass)) {
    try {
      const el = await page.$(ec.selector);
      if (el && await el.isVisible()) errors.push(`Error state: ${ec.description}`);
    } catch {}
  }
  return errors;
}

export async function validateDashboard(
  page: Page, 
  rules: ValidationRule[],
  requireServices: boolean
): Promise<{ valid: boolean; errors: string[] }> {
  const errors: string[] = [];
  for (const rule of rules) errors.push(...await checkElement(page, rule));
  if (requireServices && !(await checkServices(page))) {
    errors.push('No services displayed');
  }
  errors.push(...await checkErrorStates(page));
  return { valid: errors.length === 0, errors };
}

async function isReconnecting(page: Page): Promise<boolean> {
  for (const msg of ['text="Reconnecting"', 'text="Connection lost"']) {
    const el = await page.$(msg);
    if (el && await el.isVisible().catch(() => false)) return true;
  }
  return false;
}

async function hasContent(page: Page): Promise<boolean> {
  // Check if we're on the editor route
  const isEditor = page.url().includes('/editor');
  if (isEditor) {
    // For editor, check for editor-specific elements
    const editorElements = await page.$$('[role="navigation"][aria-label*="Azure YAML Editor" i], [class*="editor"], [role="tree"], nav, main');
    return editorElements.length > 0;
  }
  // For dashboard, check for dashboard-specific elements
  if (!(await page.$('h1, h2')) || !(await page.$('header[role="banner"]'))) return false;
  const counts = (await page.$$('table tbody tr')).length +
                 (await page.$$('[class*="ServiceCard"]')).length +
                 (await page.$$('[class*="LogsPane"], [class*="log-"], main[class] > div')).length;
  return counts > 0 || !!(await page.$('text="No Services Running"'));
}

export async function waitForDashboardReady(page: Page, timeout = 30000): Promise<boolean> {
  const start = Date.now();
  console.log('  ⏳ Waiting for dashboard to load...');
  while (Date.now() - start < timeout) {
    await page.waitForLoadState('networkidle').catch(() => {});
    if (await isReconnecting(page)) {
      console.log('  ⏳ Dashboard reconnecting...');
      await sleep(1000);
      continue;
    }
    if (await hasContent(page) && !(await isReconnecting(page))) {
      console.log('  ✓ Dashboard loaded');
      return true;
    }
    await sleep(500);
  }
  console.log('  ⚠️ Timeout - proceeding anyway');
  return false;
}

async function executeActions(page: Page, actions: any[]): Promise<void> {
  console.log('  🎬 Executing actions...');
  for (const action of actions) {
    console.log(`     - ${action.description}`);
    try {
      if (action.type === 'click' && action.selector) {
        // Wait for element to be visible first, then click
        try {
          // Try to wait for the element with a reasonable timeout
          await page.waitForSelector(action.selector, { state: 'visible', timeout: 5000 }).catch(() => {});
          
          // Try multiple selector strategies for better reliability
          const element = await page.$(action.selector).catch(() => null);
          if (element) {
            const isVisible = await element.isVisible().catch(() => false);
            if (isVisible) {
              await element.scrollIntoViewIfNeeded().catch(() => {});
              await element.click({ timeout: 5000 }).catch(() => {
                // Fallback to page.click
                page.click(action.selector, { timeout: 5000 }).catch(() => console.log(`       ⚠️ Could not click`));
              });
            } else {
              // Element exists but not visible, try page.click
              await page.click(action.selector, { timeout: 5000 }).catch(() => console.log(`       ⚠️ Could not click - element not visible`));
            }
          } else {
            // Element not found, try page.click directly
            await page.click(action.selector, { timeout: 5000 }).catch(() => console.log(`       ⚠️ Could not click - selector not found: ${action.selector.substring(0, 50)}`));
          }
        } catch (e) {
          console.log(`       ⚠️ Click failed: ${String(e).substring(0, 100)}`);
        }
      } else if (action.type === 'type' && action.selector && action.text) {
        await page.type(action.selector, action.text, { timeout: 5000 }).catch(() => console.log(`       ⚠️ Could not type`));
      } else if (action.type === 'wait') {
        if (action.selector) {
          // Wait for selector to appear
          await page.waitForSelector(action.selector, { state: 'visible', timeout: action.delay || 5000 }).catch(() => {});
        } else if (action.delay) {
          await page.waitForTimeout(action.delay);
        }
      } else if (action.type === 'evaluate' && action.script) {
        await page.evaluate(action.script);
      } else if (action.type === 'keyboard' && action.key) {
        // Use Playwright's keyboard API for better reliability
        if (action.modifier) {
          // Use specified modifier (e.g., 'Control', 'Meta')
          await page.keyboard.press(`${action.modifier}+${action.key}`);
        } else {
          // Default: use Control on Windows/Linux, Meta on Mac
          const isMac = await page.evaluate(() => navigator.platform.includes('Mac'));
          const modifier = isMac ? 'Meta' : 'Control';
          await page.keyboard.press(`${modifier}+${action.key}`);
        }
      }
      await page.waitForTimeout(100);
    } catch (e) {
      // Log error but continue
      if (action.type !== 'wait') {
        console.log(`       ⚠️ Action failed: ${String(e).substring(0, 100)}`);
      }
    }
  }
}

export async function captureScreenshot(
  page: Page,
  config: ScreenshotConfig & { skipGoto?: boolean },
  screenshotsDir: string
): Promise<{ success: boolean; errors: string[] }> {
  console.log(`📸 Capturing: ${config.name} (${config.viewport.width}x${config.viewport.height})`);
  await page.setViewportSize(config.viewport);
  
  if (!config.skipGoto) {
    await page.goto(config.url, { waitUntil: 'domcontentloaded', timeout: 30000 });
  }
  
  await waitForDashboardReady(page);

  if (config.actions?.length) {
    await executeActions(page, config.actions);
    await waitForDashboardReady(page);
  }
  if (config.waitFor) await page.waitForSelector(config.waitFor);
  if (config.delay) await page.waitForTimeout(config.delay);

  if (config.validateElements || config.requireServices) {
    const v = await validateDashboard(page, config.validateElements || [], config.requireServices || false);
    if (!v.valid) {
      console.log('  ⚠️ Validation warnings:');
      v.errors.forEach(err => console.log(`     - ${err}`));
    } else {
      console.log('  ✓ Dashboard validation passed');
    }
  }

  const screenshotPath = path.join(screenshotsDir, `${config.name}.png`);
  
  // Determine screenshot method: selector > clipSelector > clip > full page
  if (config.selector) {
    const element = await page.$(config.selector);
    if (!element) return { success: false, errors: [`Selector not found: ${config.selector}`] };
    await element.screenshot({ path: screenshotPath });
  } else if (config.clipSelector) {
    // Calculate clip from element bounding box
    const clip = await getElementClip(page, config.clipSelector);
    if (clip) {
      await page.screenshot({ path: screenshotPath, clip });
    } else {
      // Fallback to fixed clip if provided, otherwise full page
      if (config.clip) {
        console.log(`  ⚠️ clipSelector "${config.clipSelector}" not found, using fallback clip`);
        await page.screenshot({ path: screenshotPath, clip: config.clip });
      } else {
        console.log(`  ⚠️ clipSelector "${config.clipSelector}" not found, falling back to full page`);
        await page.screenshot({ path: screenshotPath });
      }
    }
  } else if (config.clip) {
    await page.screenshot({ path: screenshotPath, clip: config.clip });
  } else {
    await page.screenshot({ path: screenshotPath });
  }

  const sizeKB = (fs.statSync(screenshotPath).size / 1024).toFixed(1);
  console.log(`  ✓ Saved: ${config.name}.png (${sizeKB} KB)`);
  return { success: true, errors: [] };
}
