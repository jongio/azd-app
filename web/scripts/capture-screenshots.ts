/**
 * Screenshot Capture Script
 *
 * Automatically captures screenshots of the azd app dashboard for the marketing website.
 * 
 * Run with: npx tsx scripts/capture-screenshots.ts
 *
 * What it does:
 * 1. Starts the demo project with `azd app run`
 * 2. Starts the dashboard dev server
 * 3. Waits for services to be ready
 * 4. Validates that all required UI elements are visible
 * 5. Captures screenshots at various viewports
 * 6. Crops and optimizes images
 * 7. Cleans up all processes
 */

import { chromium, type Browser, type Page } from 'playwright';
import { spawn, type ChildProcess, execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

// ES Module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Directories
const SCRIPTS_DIR = __dirname;
const WEB_DIR = path.dirname(SCRIPTS_DIR);
const CLI_DIR = path.join(path.dirname(WEB_DIR), 'cli');
const DEMO_DIR = path.join(CLI_DIR, 'demo');
const DASHBOARD_DIR = path.join(CLI_DIR, 'dashboard');
const SCREENSHOTS_DIR = path.join(WEB_DIR, 'public', 'screenshots');

// URLs
const API_URL = 'http://localhost:3000';

// Processes to clean up
const processes: ChildProcess[] = [];

// Required UI elements that must be visible for a valid screenshot
interface ValidationRule {
  selector: string;
  description: string;
  minCount?: number;  // Minimum number of elements expected
  mustBeVisible?: boolean;  // Element must be in viewport
  textContent?: string | RegExp;  // Expected text content
}

// Elements that should be present in the dashboard
const REQUIRED_ELEMENTS: ValidationRule[] = [
  { 
    selector: 'header[role="banner"]', 
    description: 'Header navigation',
    mustBeVisible: true 
  },
  { 
    selector: '[role="tablist"] [role="tab"]', 
    description: 'Navigation tabs',
    minCount: 3
  },
];

// Elements that indicate a healthy dashboard with services
const SERVICE_ELEMENTS: ValidationRule[] = [
  { 
    selector: 'table tbody tr, [class*="ServiceCard"], main > div, [class*="logs"]', 
    description: 'Service rows, cards, or main content',
    minCount: 1
  },
];

// Error states that should NOT be present
const ERROR_SELECTORS = [
  { selector: 'text="Error Loading Services"', description: 'Service loading error' },
  { selector: 'text="No Services Running"', description: 'No services message' },
  { selector: 'text="Failed to connect"', description: 'Connection error' },
  { selector: 'text="Reconnecting"', description: 'Reconnecting state - dashboard not connected' },
  { selector: 'text="Connection lost"', description: 'Connection lost message' },
  { selector: '[class*="error"]', description: 'Error styling', checkClass: true },
];

interface ScreenshotConfig {
  name: string;
  url: string;
  selector?: string;
  viewport: { width: number; height: number };
  waitFor?: string;
  delay?: number;
  clip?: { x: number; y: number; width: number; height: number };
  validateElements?: ValidationRule[];
  requireServices?: boolean;
  /** Actions to perform before taking screenshot (e.g., click buttons to change view) */
  actions?: ScreenshotAction[];
}

interface ScreenshotAction {
  type: 'click' | 'wait' | 'evaluate';
  selector?: string;
  script?: string;
  delay?: number;
  description: string;
}

const screenshots: ScreenshotConfig[] = [
  // Console view (default landing page)
  {
    name: 'dashboard-console',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 2000,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    // Console is the default view, no navigation needed
  },
  // Resources view - Grid (default for resources)
  {
    name: 'dashboard-resources-grid',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 1500,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 500, description: 'Wait for view to load' },
      // Ensure grid view is selected (click Grid button if visible)
      { type: 'click', selector: 'button:has-text("Grid")', description: 'Click Grid view button' },
      { type: 'wait', delay: 500, description: 'Wait for grid to render' },
    ],
  },
  // Resources view - Table
  {
    name: 'dashboard-resources-table',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 1500,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 500, description: 'Wait for view to load' },
      // Switch to table view
      { type: 'click', selector: 'button:has-text("Table")', description: 'Click Table view button' },
      { type: 'wait', delay: 500, description: 'Wait for table to render' },
    ],
  },
];

async function ensureDir(dir: string): Promise<void> {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForUrl(url: string, timeout = 30000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return true;
      }
    } catch {
      // Service not ready yet
    }
    await sleep(500);
  }
  return false;
}

function startProcess(
  command: string,
  args: string[],
  cwd: string,
  name: string,
  onOutput?: (line: string) => void
): ChildProcess {
  console.log(`🚀 Starting ${name}...`);
  console.log(`   Command: ${command} ${args.join(' ')}`);
  console.log(`   Dir: ${cwd}`);

  const isWindows = process.platform === 'win32';
  const proc = spawn(command, args, {
    cwd,
    stdio: ['ignore', 'pipe', 'pipe'],
    shell: isWindows,
    detached: !isWindows,
  });

  proc.stdout?.on('data', (data) => {
    const lines = data.toString().trim().split('\n');
    lines.forEach((line: string) => {
      if (line.trim()) {
        console.log(`   [${name}] ${line}`);
        onOutput?.(line);
      }
    });
  });

  proc.stderr?.on('data', (data) => {
    const lines = data.toString().trim().split('\n');
    lines.forEach((line: string) => {
      if (line.trim()) {
        console.log(`   [${name}] ${line}`);
        onOutput?.(line);
      }
    });
  });

  processes.push(proc);
  return proc;
}

function killProcess(proc: ChildProcess): void {
  if (!proc.killed) {
    const isWindows = process.platform === 'win32';
    if (isWindows) {
      // On Windows, use taskkill to kill process tree
      try {
        execSync(`taskkill /pid ${proc.pid} /T /F`, { stdio: 'ignore' });
      } catch {
        proc.kill('SIGTERM');
      }
    } else {
      // On Unix, kill the process group
      try {
        process.kill(-proc.pid!, 'SIGTERM');
      } catch {
        proc.kill('SIGTERM');
      }
    }
  }
}

function cleanup(): void {
  console.log('\n🧹 Cleaning up processes...');
  processes.forEach((proc) => killProcess(proc));
}

/**
 * Validates that required UI elements are present and visible
 */
async function validateDashboard(
  page: Page, 
  rules: ValidationRule[],
  requireServices: boolean
): Promise<{ valid: boolean; errors: string[] }> {
  const errors: string[] = [];

  // Check required elements
  for (const rule of rules) {
    try {
      const elements = await page.$$(rule.selector);
      
      if (elements.length === 0) {
        errors.push(`Missing: ${rule.description} (selector: ${rule.selector})`);
        continue;
      }

      if (rule.minCount && elements.length < rule.minCount) {
        errors.push(`Not enough: ${rule.description} - found ${elements.length}, expected at least ${rule.minCount}`);
        continue;
      }

      if (rule.textContent) {
        let foundMatch = false;
        for (const el of elements) {
          const text = await el.textContent();
          if (text) {
            if (typeof rule.textContent === 'string') {
              if (text.includes(rule.textContent)) foundMatch = true;
            } else {
              if (rule.textContent.test(text)) foundMatch = true;
            }
          }
        }
        if (!foundMatch) {
          errors.push(`Wrong content: ${rule.description} - expected "${rule.textContent}"`);
        }
      }

      if (rule.mustBeVisible) {
        const firstElement = elements[0];
        const isVisible = await firstElement.isVisible();
        if (!isVisible) {
          errors.push(`Not visible: ${rule.description}`);
        }
      }
    } catch (e) {
      errors.push(`Error checking ${rule.description}: ${e}`);
    }
  }

  // Check for services if required
  if (requireServices) {
    let hasServices = false;
    for (const rule of SERVICE_ELEMENTS) {
      const elements = await page.$$(rule.selector);
      if (elements.length >= (rule.minCount || 1)) {
        hasServices = true;
        break;
      }
    }
    if (!hasServices) {
      errors.push('No services displayed - dashboard may be empty or services failed to load');
    }
  }

  // Check for error states
  for (const errorCheck of ERROR_SELECTORS) {
    try {
      // Skip class-based checks as they might have false positives
      if (errorCheck.checkClass) continue;
      
      const errorElement = await page.$(errorCheck.selector);
      if (errorElement) {
        const isVisible = await errorElement.isVisible();
        if (isVisible) {
          errors.push(`Error state detected: ${errorCheck.description}`);
        }
      }
    } catch {
      // Selector not found is fine
    }
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Waits for the dashboard to be fully loaded with services
 */
async function waitForDashboardReady(page: Page, timeout = 30000): Promise<boolean> {
  const start = Date.now();
  
  console.log('  ⏳ Waiting for dashboard to load...');
  
  while (Date.now() - start < timeout) {
    // Wait for the page to be stable
    await page.waitForLoadState('networkidle').catch(() => {});
    
    // Check for reconnecting state - if present, wait more
    const reconnecting = await page.$('text="Reconnecting"');
    if (reconnecting) {
      const isVisible = await reconnecting.isVisible().catch(() => false);
      if (isVisible) {
        console.log('  ⏳ Dashboard is reconnecting, waiting...');
        await sleep(1000);
        continue;
      }
    }
    
    // Check for connection lost state
    const connectionLost = await page.$('text="Connection lost"');
    if (connectionLost) {
      const isVisible = await connectionLost.isVisible().catch(() => false);
      if (isVisible) {
        console.log('  ⏳ Dashboard connection lost, waiting for reconnection...');
        await sleep(1000);
        continue;
      }
    }
    
    // Check if main content is loaded
      const hasTitle = await page.$('h1, h2');
      const hasHeader = await page.$('header[role="banner"]');
    if (hasTitle && hasHeader) {
      // Check for services (table rows, cards, or log entries)
      const tableRows = await page.$$('table tbody tr');
      const serviceCards = await page.$$('[class*="ServiceCard"]');
      const logPanes = await page.$$('[class*="LogsPane"], [class*="log-"], main[class] > div');
      
      if (tableRows.length > 0 || serviceCards.length > 0 || logPanes.length > 0) {
        // Double check no reconnecting message
        const stillReconnecting = await page.$('text="Reconnecting"');
        if (stillReconnecting) {
          const isVisible = await stillReconnecting.isVisible().catch(() => false);
          if (isVisible) {
            console.log('  ⏳ Still reconnecting...');
            await sleep(1000);
            continue;
          }
        }
        
        console.log(`  ✓ Dashboard loaded with ${tableRows.length + serviceCards.length + logPanes.length} content element(s)`);
        return true;
      }
      
      // Check for "No Services" message - this is also a valid state
      const noServices = await page.$('text="No Services Running"');
      if (noServices) {
        console.log('  ⚠️ Dashboard loaded but no services are running');
        return true;
      }
    }
    
    await sleep(500);
  }
  
  console.log('  ⚠️ Dashboard load timeout - proceeding anyway');
  return false;
}

async function captureScreenshot(
  page: Page,
  config: ScreenshotConfig
): Promise<{ success: boolean; errors: string[] }> {
  console.log(`📸 Capturing: ${config.name} (${config.viewport.width}x${config.viewport.height})`);

  await page.setViewportSize(config.viewport);
  
  // Use 'domcontentloaded' instead of 'networkidle' since dashboard has persistent connections
  await page.goto(config.url, { waitUntil: 'domcontentloaded', timeout: 30000 });

  // Wait for dashboard to be fully ready
  await waitForDashboardReady(page);

  // Execute any pre-screenshot actions
  if (config.actions && config.actions.length > 0) {
    console.log('  🎬 Executing actions...');
    for (const action of config.actions) {
      try {
        console.log(`     - ${action.description}`);
        switch (action.type) {
          case 'click':
            if (action.selector) {
              await page.click(action.selector, { timeout: 5000 }).catch(() => {
                console.log(`       ⚠️ Could not click: ${action.selector}`);
              });
            }
            break;
          case 'wait':
            if (action.delay) {
              await page.waitForTimeout(action.delay);
            }
            break;
          case 'evaluate':
            if (action.script) {
              await page.evaluate(action.script);
            }
            break;
        }
        // Small delay between actions
        await page.waitForTimeout(100);
      } catch (e) {
        console.log(`       ⚠️ Action failed: ${e}`);
      }
    }
    // Wait for any view changes to settle
    await waitForDashboardReady(page);
  }

  if (config.waitFor) {
    await page.waitForSelector(config.waitFor);
  }

  if (config.delay) {
    await page.waitForTimeout(config.delay);
  }

  // Validate the dashboard state before capturing
  if (config.validateElements || config.requireServices) {
    const validation = await validateDashboard(
      page, 
      config.validateElements || [], 
      config.requireServices || false
    );
    
    if (!validation.valid) {
      console.log('  ⚠️ Validation warnings:');
      validation.errors.forEach(err => console.log(`     - ${err}`));
      // Continue anyway but report the issues
    } else {
      console.log('  ✓ Dashboard validation passed');
    }
  }

  const screenshotPath = path.join(SCREENSHOTS_DIR, `${config.name}.png`);

  if (config.selector) {
    const element = await page.$(config.selector);
    if (!element) {
      console.error(`  ❌ Selector not found: ${config.selector}`);
      return { success: false, errors: [`Selector not found: ${config.selector}`] };
    }
    await element.screenshot({ path: screenshotPath });
  } else if (config.clip) {
    await page.screenshot({ path: screenshotPath, clip: config.clip });
  } else {
    await page.screenshot({ path: screenshotPath });
  }

  // Get file size
  const stats = fs.statSync(screenshotPath);
  const sizeKB = (stats.size / 1024).toFixed(1);
  console.log(`  ✓ Saved: ${config.name}.png (${sizeKB} KB)`);
  
  return { success: true, errors: [] };
}

async function optimizeImages(): Promise<void> {
  console.log('\n🔧 Optimizing images...');

  // Check if sharp is available for optimization
  try {
    // Dynamic require to handle missing module gracefully
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const sharp = require('sharp');

    const files = fs.readdirSync(SCREENSHOTS_DIR).filter((f) => f.endsWith('.png'));

    for (const file of files) {
      const filePath = path.join(SCREENSHOTS_DIR, file);
      const originalSize = fs.statSync(filePath).size;

      // Optimize with sharp
      const optimized = await sharp(filePath)
        .png({ quality: 80, compressionLevel: 9 })
        .toBuffer();

      fs.writeFileSync(filePath, optimized);

      const newSize = fs.statSync(filePath).size;
      const savings = (((originalSize - newSize) / originalSize) * 100).toFixed(1);
      console.log(`  ✓ ${file}: ${(newSize / 1024).toFixed(1)} KB (${savings}% smaller)`);
    }
  } catch {
    console.log('  ⚠️ sharp not available, skipping optimization');
    console.log('  Install with: pnpm add -D sharp');
  }
}

async function findAzdAppBinary(): Promise<string> {
  // Look for the built binary - prefer the NEWEST one to avoid stale binary issues
  const binDir = path.join(CLI_DIR, 'bin');
  const isWindows = process.platform === 'win32';
  
  if (fs.existsSync(binDir)) {
    const files = fs.readdirSync(binDir);
    const ext = isWindows ? '.exe' : '';
    const platformArch = isWindows ? 'windows-amd64' : `${process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}`;
    
    // Find all matching binaries for this platform (excluding .old files)
    const candidates = files
      .filter(f => 
        (f === `azd-app${ext}` || f.includes(platformArch)) &&
        (isWindows ? f.endsWith('.exe') : !f.includes('.')) &&
        !f.endsWith('.old')
      )
      .map(f => ({
        name: f,
        path: path.join(binDir, f),
        mtime: fs.statSync(path.join(binDir, f)).mtime.getTime()
      }))
      .sort((a, b) => b.mtime - a.mtime); // Sort by newest first
    
    if (candidates.length > 0) {
      const newest = candidates[0];
      console.log(`  Found ${candidates.length} candidate(s), using newest: ${newest.name}`);
      if (candidates.length > 1) {
        const oldest = candidates[candidates.length - 1];
        const ageMinutes = Math.round((newest.mtime - oldest.mtime) / 60000);
        if (ageMinutes > 5) {
          console.log(`  ⚠️  Warning: ${oldest.name} is ${ageMinutes} minutes older - consider cleaning up stale binaries`);
        }
      }
      return newest.path;
    }
  }
  
  // Fall back to azd-app in PATH (if installed)
  return 'azd-app';
}

async function main(): Promise<void> {
  console.log('═══════════════════════════════════════════════════════════');
  console.log('  📸 azd app Dashboard Screenshot Capture');
  console.log('═══════════════════════════════════════════════════════════\n');

  // Register cleanup handlers
  process.on('SIGINT', () => {
    cleanup();
    process.exit(1);
  });
  process.on('SIGTERM', () => {
    cleanup();
    process.exit(1);
  });

  await ensureDir(SCREENSHOTS_DIR);

  let browser: Browser | null = null;
  let dashboardUrl = '';

  try {
    // Step 1: Find azd-app binary
    const azdAppBin = await findAzdAppBinary();
    console.log(`📦 Using azd-app: ${azdAppBin}\n`);

    // Step 2: Start the demo project and capture the dashboard URL from output
    // azd app run outputs: "Dashboard  http://localhost:XXXXX"
    let dashboardDetected = false;
    
    startProcess(azdAppBin, ['run'], DEMO_DIR, 'azd-app', (line) => {
      // Look for dashboard URL in output
      const dashboardMatch = line.match(/Dashboard\s+(https?:\/\/[^\s]+)/i);
      if (dashboardMatch && !dashboardDetected) {
        dashboardUrl = dashboardMatch[1];
        dashboardDetected = true;
        console.log(`\n  🎯 Detected dashboard URL: ${dashboardUrl}`);
      }
    });

    // Step 3: Wait for services and dashboard to be ready
    console.log('\n⏳ Waiting for services to start...');

    const apiReady = await waitForUrl(`${API_URL}/items`, 15000);
    if (!apiReady) {
      console.log('  ⚠️ API not responding, continuing anyway...');
    } else {
      console.log('  ✓ Demo API ready');
    }

    // Wait for dashboard URL to be detected from azd app run output
    const dashboardTimeout = 30000;
    const dashboardStart = Date.now();
    while (!dashboardDetected && Date.now() - dashboardStart < dashboardTimeout) {
      await sleep(500);
    }

    if (!dashboardDetected || !dashboardUrl) {
      throw new Error('Dashboard URL not detected from azd app run output');
    }

    // Wait for the dashboard to be accessible
    console.log(`  ⏳ Waiting for dashboard at ${dashboardUrl}...`);
    const dashboardReady = await waitForUrl(dashboardUrl, 30000);
    if (!dashboardReady) {
      throw new Error(`Dashboard not responding at ${dashboardUrl}`);
    }
    console.log(`  ✓ Dashboard ready at ${dashboardUrl}\n`);

    // Extra wait for full initialization
    await sleep(3000);

    // Step 4: Launch browser and capture screenshots
    browser = await chromium.launch({
      headless: true,
    });

    const context = await browser.newContext({
      deviceScaleFactor: 2, // Retina quality
      colorScheme: 'dark', // Use dark mode for better contrast
    });

    const page = await context.newPage();

    const results: { name: string; success: boolean; errors: string[] }[] = [];

    for (const config of screenshots) {
      try {
        // Use detected dashboard URL instead of default
        const configWithUrl = { ...config, url: dashboardUrl };
        const result = await captureScreenshot(page, configWithUrl);
        results.push({ name: config.name, ...result });
      } catch (e) {
        console.error(`  ❌ Failed: ${config.name}`, e);
        results.push({ name: config.name, success: false, errors: [String(e)] });
      }
    }

    // Step 5: Optimize images
    await optimizeImages();

    // Step 6: Report summary
    console.log('\n═══════════════════════════════════════════════════════════');
    console.log('  📊 Screenshot Capture Summary');
    console.log('═══════════════════════════════════════════════════════════\n');
    
    const successful = results.filter(r => r.success);
    const failed = results.filter(r => !r.success);
    
    console.log(`  ✅ Successful: ${successful.length}/${results.length}`);
    successful.forEach(r => console.log(`     - ${r.name}.png`));
    
    if (failed.length > 0) {
      console.log(`\n  ❌ Failed: ${failed.length}/${results.length}`);
      failed.forEach(r => {
        console.log(`     - ${r.name}`);
        r.errors.forEach(err => console.log(`       Error: ${err}`));
      });
    }

    console.log(`\n  📁 Screenshots saved to: ${SCREENSHOTS_DIR}`);
    console.log('═══════════════════════════════════════════════════════════\n');

    // Exit with error if any failed
    if (failed.length > 0) {
      process.exit(1);
    }

  } catch (error) {
    console.error('\n❌ Error:', error);
    process.exit(1);
  } finally {
    if (browser) {
      await browser.close();
    }
    cleanup();
  }
}

main().catch((error) => {
  console.error('Fatal error:', error);
  cleanup();
  process.exit(1);
});
