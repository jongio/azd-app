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
    // Use threads pool: lighter than forks under CPU contention (avoids fork worker timeouts)
    pool: 'threads',
    poolOptions: {
      threads: {
        // Limit workers to avoid overwhelming the OS under parallel CI contention
        maxThreads: 4,
        minThreads: 1,
      },
    },
    exclude: ['node_modules', 'e2e'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/hooks/useServiceOperations.ts',
        'src/test/',
        'e2e/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/dist/**',
        '**/coverage/**',
      ],
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    } as Record<string, string>,
  },
})
