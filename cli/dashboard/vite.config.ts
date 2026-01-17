import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

// Handle __dirname for ESM
const __filename: string = fileURLToPath(import.meta.url)
const __dirname: string = path.dirname(__filename)

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../src/internal/dashboard/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // Split vendor libraries into separate chunks
          if (id.includes('node_modules')) {
            // React core
            if (id.includes('react') || id.includes('react-dom') || id.includes('scheduler')) {
              return 'vendor-react'
            }
            
            // UI libraries
            if (id.includes('lucide-react') || id.includes('@radix-ui')) {
              return 'vendor-ui'
            }
            
            // Editor-specific heavy dependencies
            if (id.includes('react-syntax-highlighter') || id.includes('prismjs') || id.includes('refractor')) {
              return 'vendor-editor-syntax'
            }
            
            // Form libraries (used heavily in editor)
            if (id.includes('react-hook-form') || id.includes('@hookform')) {
              return 'vendor-editor-forms'
            }
            
            // Schema validation (editor-specific)
            if (id.includes('ajv') || id.includes('json-schema')) {
              return 'vendor-editor-schema'
            }
            
            // YAML parsing (editor-specific)
            if (id.includes('js-yaml') || id.includes('yaml')) {
              return 'vendor-editor-yaml'
            }
            
            // State management
            if (id.includes('zustand')) {
              return 'vendor-state'
            }
            
            // Utility libraries
            if (id.includes('clsx') || id.includes('tailwind-merge') || id.includes('class-variance-authority') || id.includes('ansi-to-html')) {
              return 'vendor-utils'
            }
            
            // Other node_modules
            return 'vendor-other'
          }
          
          // Editor components go into their own chunk
          if (id.includes('/components/editor/') || id.includes('/lib/editor/')) {
            return 'editor-components'
          }
        },
      },
    },
    chunkSizeWarningLimit: 1000, // Increase limit since we're splitting better
  },
})
