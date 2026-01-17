import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'line',
  webServer: {
    command: 'pnpm build && pnpm preview',
    url: 'http://127.0.0.1:4331/azd-app/',
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    cwd: __dirname,
  },
  // Use platform-independent snapshot naming for cross-platform CI compatibility
  snapshotPathTemplate: '{testDir}/{testFileDir}/{testFileName}-snapshots/{arg}{ext}',
  use: {
    // Note: Trailing slash is required for proper URL resolution with relative paths
    baseURL: 'http://127.0.0.1:4331/azd-app/',
    trace: 'on-first-retry',
    headless: true,
  },
  expect: {
    toHaveScreenshot: {
      // Allow for minor rendering differences across platforms
      maxDiffPixelRatio: 0.05,
    },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
