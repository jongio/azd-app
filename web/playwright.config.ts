import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'line',
  // Use platform-independent snapshot naming for cross-platform CI compatibility
  snapshotPathTemplate: '{testDir}/{testFileDir}/{testFileName}-snapshots/{arg}{ext}',
  use: {
    // Note: Trailing slash is required for proper URL resolution with relative paths
    baseURL: 'http://localhost:4321/azd-app/',
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
