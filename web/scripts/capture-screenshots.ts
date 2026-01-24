/**
 * Screenshot Capture Script
 *
 * Automatically captures screenshots of the azd app dashboard for the marketing website.
 * 
 * Prerequisites:
 * - Azure CLI authenticated (az login)
 * - Azure resources deployed in azure-logs-test project
 * 
 * Run with: 
 *   pnpm screenshots                    # Capture all screenshots
 *   pnpm screenshots:editor             # Capture only editor-main-view
 *   tsx scripts/capture-screenshots.ts [screenshot-name]  # Capture specific screenshot
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
    
    await startProcess(azdAppBin, ['run'], DEMO_DIR, 'azd-app', (line) => {
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

    // Note: We'll create contexts per screenshot to support different color schemes
    // Start with a default context for non-editor screenshots
    const defaultContext = await browser.newContext({
      deviceScaleFactor: 2, // Retina quality
      colorScheme: 'dark', // Default to dark mode for dashboard screenshots
    });

    const page = await defaultContext.newPage();

    const results: { name: string; success: boolean; errors: string[] }[] = [];

    // Filter to specific screenshot if needed for testing
    // Support both command-line argument and environment variable (for cross-platform compatibility)
    // Supports exact match or prefix match (e.g., "editor" or "editor-" matches all "editor-*" screenshots)
    const targetScreenshot = process.argv[2] || process.env.SCREENSHOT_FILTER;
    const screenshotsToCapture = targetScreenshot 
      ? SCREENSHOT_CONFIGS.filter(c => {
          const filter = targetScreenshot.endsWith('-') ? targetScreenshot : targetScreenshot + '-';
          return c.name === targetScreenshot || c.name.startsWith(filter);
        })
      : SCREENSHOT_CONFIGS;

    if (targetScreenshot && screenshotsToCapture.length === 0) {
      console.error(`\n❌ Screenshot "${targetScreenshot}" not found in config`);
      process.exit(1);
    }

    // Navigate once to avoid closing websockets between screenshots
    await page.goto(dashboardUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });

    for (const config of screenshotsToCapture) {
      try {
        const isEditorScreenshot = config.name.startsWith('editor-');
        const editorUrl = dashboardUrl.replace(/\/$/, '') + '/editor';
        
        // Determine variants to capture
        const variants = config.variants || [config.colorScheme || 'dark'];
        
        // Capture each variant
        for (const variant of variants) {
          // Create a context with the required color scheme
          const context = await browser.newContext({
            deviceScaleFactor: 2, // Retina quality
            colorScheme: variant,
          });
          
          // Set localStorage BEFORE any page loads using addInitScript
          // This ensures React's theme provider reads the correct theme on mount
          await context.addInitScript((theme) => {
            // Set localStorage before React initializes
            // Set both editor and dashboard theme keys to ensure compatibility
            try {
              localStorage.setItem('azure-yaml-editor-theme', theme);
              localStorage.setItem('dashboard-theme', theme);
            } catch (e) {
              // Ignore errors
            }
          }, variant);
          
          const screenshotPage = await context.newPage();
          
          try {
            const configWithUrl = { 
              ...config, 
              url: isEditorScreenshot ? editorUrl : dashboardUrl, 
              skipGoto: true // We'll navigate manually
            };
            
            // Navigate to the page - localStorage is already set via addInitScript
            // Use 'load' instead of 'networkidle' because dashboard has ongoing WebSocket connections
            // that prevent networkidle from ever being reached
            await screenshotPage.goto(configWithUrl.url, { waitUntil: 'load', timeout: 60000 });
            
            // Wait for React to hydrate and theme provider to initialize
            // Reduced wait since 'load' already waits for resources
            await screenshotPage.waitForTimeout(1000);
            
            // Force theme application - ensure DOM is updated
            // The theme provider should have applied it, but we ensure it's correct
            await screenshotPage.evaluate((theme) => {
              const root = document.documentElement;
              // Remove existing theme classes
              root.classList.remove('light', 'dark');
              // Add the correct theme class
              root.classList.add(theme);
              // Set data-theme attribute
              root.setAttribute('data-theme', theme);
              // Set CSS color-scheme property
              root.style.colorScheme = theme;
              // Ensure localStorage is still set for both editor and dashboard
              try {
                localStorage.setItem('azure-yaml-editor-theme', theme);
                localStorage.setItem('dashboard-theme', theme);
              } catch (e) {
                // Ignore localStorage errors
              }
            }, variant);
            
            // Wait for theme styles to fully apply
            await screenshotPage.waitForTimeout(500);
            
            // Verify theme is actually applied (for debugging)
            const themeCheck = await screenshotPage.evaluate(() => {
              const root = document.documentElement;
              return {
                dataTheme: root.getAttribute('data-theme'),
                classList: Array.from(root.classList),
                colorScheme: root.style.colorScheme || getComputedStyle(root).colorScheme,
                localStorage: localStorage.getItem('azure-yaml-editor-theme')
              };
            });
            
            if (variant === 'light' && themeCheck.dataTheme !== 'light') {
              console.log(`  ⚠️ Warning: Theme not applied correctly. Expected 'light', got '${themeCheck.dataTheme}'`);
              console.log(`     Classes: ${themeCheck.classList.join(', ')}`);
              console.log(`     localStorage: ${themeCheck.localStorage}`);
              // Force it one more time with a longer wait
              await screenshotPage.evaluate(() => {
                const root = document.documentElement;
                root.classList.remove('light', 'dark');
                root.classList.add('light');
                root.setAttribute('data-theme', 'light');
                root.style.colorScheme = 'light';
                // Trigger a reflow to ensure styles apply
                void root.offsetHeight;
              });
              await screenshotPage.waitForTimeout(1000);
              // Verify again
              const recheck = await screenshotPage.evaluate(() => document.documentElement.getAttribute('data-theme'));
              console.log(`  ✓ Theme after force: ${recheck}`);
            } else if (variant === 'light') {
              console.log(`  ✓ Theme verified: ${themeCheck.dataTheme} mode`);
            }
            
            // When there's only one variant (not using variants array), save without suffix for backward compatibility
            // When using variants array, save with suffix to distinguish them
            const shouldUseSuffix = config.variants && config.variants.length > 1;
            const captureVariant = shouldUseSuffix ? variant : undefined;
            
            const result = await captureScreenshot(screenshotPage, configWithUrl, SCREENSHOTS_DIR, captureVariant);
            // Use the actual filename that was saved
            const resultName = captureVariant ? `${config.name}-${captureVariant}` : config.name;
            results.push({ name: resultName, ...result });
          } finally {
            // Always close the page and context
            await screenshotPage.close();
            await context.close();
          }
        }
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
