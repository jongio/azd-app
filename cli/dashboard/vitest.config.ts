import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

// Handle __dirname for ESM
const __filename: string = fileURLToPath(import.meta.url)
const __dirname: string = path.dirname(__filename)

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
    watch: false,
    testTimeout: 10000,
    // Use forks pool: avoids worker thread timeouts on Windows where thread-based
    // workers can stall under CPU contention (vitest issue #4734).
    pool: 'forks',
    maxWorkers: 4,
    exclude: ['node_modules', 'e2e'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/hooks/useServiceOperations.ts',
        'src/test/**',
        'src/gen/**',
        '**/*.d.ts',
        '**/*.config.*',
      ],
      thresholds: {
        statements: 40,
        branches: 40,
        functions: 40,
        lines: 40,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    } as Record<string, string>,
  },
})
