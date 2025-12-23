/**
 * Screenshot Capture Script
 *
 * Automatically captures screenshots of the azd app dashboard for the marketing website.
 * 
 * Prerequisites:
 * - Azure CLI authenticated (az login)
 * - Azure resources deployed in azure-logs-test project
 * 
 * Run with: npx tsx scripts/capture-screenshots.ts
 *
 * What it does:
 * 1. Checks Azure CLI authentication
 * 2. Starts the azure-logs-test project with `azd app run`
 * 3. Waits for services to be ready and Azure logs to populate
 * 4. Validates that all required UI elements are visible
 * 5. Captures screenshots at various viewports
 * 6. Optimizes images
 * 7. Cleans up all processes
 */

import { chromium, type Browser } from 'playwright';
import type { ChildProcess } from 'child_process';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { SCREENSHOT_CONFIGS } from './screenshot-config.js';
import { captureScreenshot } from './screenshot-capture.js';
import { 
  ensureDir, 
  findAzdAppBinary, 
  optimizeImages,
  startProcess,
  killProcess,
  waitForUrl
} from './screenshot-io.js';

// ES Module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Directories
const SCRIPTS_DIR = __dirname;
const WEB_DIR = path.dirname(SCRIPTS_DIR);
const CLI_DIR = path.join(path.dirname(WEB_DIR), 'cli');
const DEMO_DIR = path.join(CLI_DIR, 'tests', 'projects', 'integration', 'azure-logs-test');
const SCREENSHOTS_DIR = path.join(WEB_DIR, 'public', 'screenshots');

// URLs
const API_URL = 'http://localhost:3000';

// Processes to clean up
const processes: ChildProcess[] = [];

async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function cleanup(): void {
  console.log('\n🧹 Cleaning up processes...');
  processes.forEach((proc) => killProcess(proc));
}

async function checkAzureAuth(): Promise<boolean> {
  console.log('🔐 Checking Azure CLI authentication...');
  try {
    const { execSync } = await import('child_process');
    execSync('az account show', { stdio: 'pipe' });
    console.log('  ✓ Azure CLI authenticated\n');
    return true;
  } catch (error) {
    console.error('  ❌ Azure CLI not authenticated');
    console.error('  Please run: az login');
    return false;
  }
}

async function main(): Promise<void> {
  console.log('═══════════════════════════════════════════════════════════');
  console.log('  📸 azd app Dashboard Screenshot Capture');
  console.log('═══════════════════════════════════════════════════════════\n');

  // Check Azure CLI authentication first
  const isAzureAuthenticated = await checkAzureAuth();
  if (!isAzureAuthenticated) {
    console.error('\n❌ Azure authentication required for azure-logs-test project');
    console.error('   This project uses Azure resources and requires Azure CLI authentication.');
    console.error('   Run "az login" to authenticate, then try again.\n');
    process.exit(1);
  }

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
    const azdAppBin = await findAzdAppBinary(CLI_DIR);
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
    }, processes);

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

    // Wait for Azure logs to populate from Log Analytics
    // Azure resources need time to sync and query data
    console.log('⏳ Waiting for Azure Log Analytics data to populate (15s)...');
    console.log('   Note: Ensure Azure resources are deployed and generating logs\n');
    await sleep(15000);

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

    // Filter to specific screenshot if needed for testing
    const targetScreenshot = process.env.SCREENSHOT_FILTER;
    const screenshotsToCapture = targetScreenshot 
      ? SCREENSHOT_CONFIGS.filter(c => c.name === targetScreenshot)
      : SCREENSHOT_CONFIGS;

    if (targetScreenshot && screenshotsToCapture.length === 0) {
      console.error(`\n❌ Screenshot "${targetScreenshot}" not found in config`);
      process.exit(1);
    }

    // Navigate once to avoid closing websockets between screenshots
    await page.goto(dashboardUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });

    for (const config of screenshotsToCapture) {
      try {
        // Use detected dashboard URL, but skip initial goto
        const configWithUrl = { ...config, url: dashboardUrl, skipGoto: true };
        const result = await captureScreenshot(page, configWithUrl, SCREENSHOTS_DIR);
        results.push({ name: config.name, ...result });
      } catch (e) {
        console.error(`  ❌ Failed: ${config.name}`, e);
        results.push({ name: config.name, success: false, errors: [String(e)] });
      }
    }

    // Step 5: Optimize images
    await optimizeImages(SCREENSHOTS_DIR);

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
