/**
 * Integration tests for Editor API
 * These tests verify frontend/backend communication with the actual Go server
 * 
 * NOTE: These tests require the dashboard server to be running.
 * Run with: pnpm test:integration
 */
import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import {
  loadConfig,
  saveConfig,
  validateConfig,
  listBackups,
  getBackup,
  restoreBackup,
  deleteBackup,
  createManualBackup,
} from './config-api'
import { spawn, type ChildProcess } from 'child_process'
import { join } from 'path'
import { rm, mkdir, writeFile } from 'fs/promises'

// Skip integration tests unless explicitly enabled
const INTEGRATION_ENABLED = process.env.TEST_INTEGRATION === 'true'
const describeIntegration = INTEGRATION_ENABLED ? describe : describe.skip

// Test configuration
const TEST_PROJECT_DIR = join(process.cwd(), '.test-project')
const SERVER_PORT = 3333 // Use different port to avoid conflicts
const API_BASE = `http://localhost:${SERVER_PORT}`

// Server process
let serverProcess: ChildProcess | null = null

/**
 * Start the dashboard server for integration testing
 */
async function startServer(): Promise<void> {
  // Create test project directory
  await rm(TEST_PROJECT_DIR, { recursive: true, force: true })
  await mkdir(TEST_PROJECT_DIR, { recursive: true })

  // Create initial azure.yaml
  const initialYaml = `name: test-app
metadata:
  template: azd-init@1.0.0
services:
  api:
    host: containerapp
    language: typescript
    project: ./src/api
`
  await writeFile(join(TEST_PROJECT_DIR, 'azure.yaml'), initialYaml)

  return new Promise((resolve, reject) => {
    // Determine build output path
    const isDev = process.env.NODE_ENV === 'development'
    const serverBinary = isDev
      ? join(process.cwd(), '..', 'bin', 'azd.exe') // Development build
      : join(process.cwd(), '..', 'azd.exe')        // Production build

    // Start server
    serverProcess = spawn(
      serverBinary,
      ['monitor', '--port', SERVER_PORT.toString(), '--cwd', TEST_PROJECT_DIR],
      {
        stdio: 'pipe',
        env: { ...process.env },
      }
    )

    let serverOutput = ''

    // Wait for server ready message
    const timeout = setTimeout(() => {
      reject(new Error(`Server failed to start within 30s. Output:\n${serverOutput}`))
    }, 30000)

    serverProcess.stdout?.on('data', (data) => {
      const output = data.toString()
      serverOutput += output

      // Check for ready message
      if (output.includes('Server started') || output.includes(`localhost:${SERVER_PORT}`)) {
        clearTimeout(timeout)
        // Give server extra time to initialize
        setTimeout(resolve, 1000)
      }
    })

    serverProcess.stderr?.on('data', (data) => {
      serverOutput += data.toString()
    })

    serverProcess.on('error', (error) => {
      clearTimeout(timeout)
      reject(new Error(`Failed to start server: ${error.message}`))
    })

    serverProcess.on('exit', (code) => {
      if (code !== 0 && code !== null) {
        clearTimeout(timeout)
        reject(new Error(`Server exited with code ${code}. Output:\n${serverOutput}`))
      }
    })
  })
}

/**
 * Stop the dashboard server
 */
async function stopServer(): Promise<void> {
  if (serverProcess) {
    serverProcess.kill('SIGTERM')
    
    // Wait for graceful shutdown
    await new Promise<void>((resolve) => {
      const timeout = setTimeout(() => {
        serverProcess?.kill('SIGKILL')
        resolve()
      }, 5000)

      serverProcess?.on('exit', () => {
        clearTimeout(timeout)
        resolve()
      })
    })

    serverProcess = null
  }

  // Cleanup test project
  await rm(TEST_PROJECT_DIR, { recursive: true, force: true })
}

// =============================================================================
// Integration Tests
// =============================================================================

describeIntegration('Editor API Integration', () => {
  beforeAll(async () => {
    await startServer()
    
    // Override API base URL for tests
    process.env.VITE_API_BASE = API_BASE
  }, 60000) // Allow 60s for server startup

  afterAll(async () => {
    await stopServer()
  }, 10000)

  describe('Configuration Load/Save', () => {
    it('should load existing azure.yaml configuration', async () => {
      const result = await loadConfig()

      expect(result).toBeDefined()
      expect(result.content).toBeDefined()
      expect(result.content).toContain('name: test-app')
      expect(result.content).toContain('services:')
      expect(result.path).toContain('azure.yaml')
      expect(result.lastModified).toBeDefined()
    })

    it('should save configuration and create backup', async () => {
      const newContent = `name: test-app
metadata:
  template: azd-init@1.0.0
services:
  api:
    host: containerapp
    language: typescript
    project: ./src/api
  web:
    host: appservice
    language: typescript
    project: ./src/web
`

      const result = await saveConfig(newContent)

      expect(result.success).toBe(true)
      expect(result.written).toBe(true)
      expect(result.backup).toBeDefined()
      expect(result.errors).toHaveLength(0)

      // Verify content was saved by loading it
      const loaded = await loadConfig()
      expect(loaded.content).toBe(newContent)
    })

    it('should preserve line endings (LF) across save/load', async () => {
      const contentWithLF = 'name: line-ending-test\nservices:\n  api:\n    host: containerapp\n'

      await saveConfig(contentWithLF)
      const loaded = await loadConfig()

      // Content should be identical including line endings
      expect(loaded.content).toBe(contentWithLF)
      expect(loaded.content.includes('\r\n')).toBe(false) // No CRLF
    })

    it('should reject invalid YAML with validation errors', async () => {
      const invalidYaml = 'this is: not: valid: yaml: : :'

      await expect(saveConfig(invalidYaml)).rejects.toThrow()
    })

    it('should handle large configuration files', async () => {
      // Generate large config with 50 services
      const services = Array.from({ length: 50 }, (_, i) => 
        `  service-${i}:\n    host: containerapp\n    language: typescript\n    project: ./src/service-${i}\n`
      ).join('')

      const largeContent = `name: large-app\nmetadata:\n  template: azd-init@1.0.0\nservices:\n${services}`

      const result = await saveConfig(largeContent)
      expect(result.success).toBe(true)

      const loaded = await loadConfig()
      expect(loaded.content).toBe(largeContent)
    })
  })

  describe('Validation', () => {
    it('should validate correct azure.yaml configuration', async () => {
      const validYaml = `name: validation-test
metadata:
  template: azd-init@1.0.0
services:
  api:
    host: containerapp
    language: typescript
    project: ./src/api
`

      const result = await validateConfig(validYaml)

      expect(result).toBeDefined()
      expect(result.valid).toBe(true)
      expect(result.errors).toHaveLength(0)
    })

    it('should detect schema violations', async () => {
      const invalidYaml = `name: invalid-schema
services:
  api:
    invalid_field_not_in_schema: true
    host: not-a-valid-host-type
`

      const result = await validateConfig(invalidYaml)

      expect(result.valid).toBe(false)
      expect(result.errors.length).toBeGreaterThan(0)
    })

    it('should detect missing required fields', async () => {
      const missingName = `services:
  api:
    host: containerapp
`

      const result = await validateConfig(missingName)

      expect(result.valid).toBe(false)
      expect(result.errors.some(e => e.message.toLowerCase().includes('name'))).toBe(true)
    })
  })

  describe('Backup Operations', () => {
    it('should create manual backup', async () => {
      const backupName = 'manual-test-backup'
      const result = await createManualBackup(backupName)

      expect(result.success).toBe(true)
      expect(result.timestamp).toBeDefined()
      expect(result.path).toContain(backupName)
    })

    it('should list backups sorted by timestamp (newest first)', async () => {
      // Create multiple backups
      await createManualBackup('backup-1')
      await new Promise(resolve => setTimeout(resolve, 100))
      await createManualBackup('backup-2')
      await new Promise(resolve => setTimeout(resolve, 100))
      await createManualBackup('backup-3')

      const result = await listBackups()

      expect(result.backups.length).toBeGreaterThan(0)
      
      // Verify sorting (newest first)
      for (let i = 0; i < result.backups.length - 1; i++) {
        const current = new Date(result.backups[i].timestamp)
        const next = new Date(result.backups[i + 1].timestamp)
        expect(current.getTime()).toBeGreaterThanOrEqual(next.getTime())
      }
    })

    it('should retrieve backup content', async () => {
      const backupResult = await createManualBackup('content-test')
      const timestamp = backupResult.timestamp

      const backup = await getBackup(timestamp)

      expect(backup.content).toBeDefined()
      expect(backup.content).toContain('name:')
      expect(backup.timestamp).toBe(timestamp)
    })

    it('should restore backup and create safety backup', async () => {
      // Save current state
      const beforeContent = `name: before-restore\nservices:\n  api:\n    host: containerapp\n`
      await saveConfig(beforeContent)

      // Create backup
      const backupResult = await createManualBackup('restore-test')
      const backupTimestamp = backupResult.timestamp

      // Modify configuration
      const modifiedContent = `name: modified\nservices:\n  web:\n    host: appservice\n`
      await saveConfig(modifiedContent)

      // Restore backup
      const restoreResult = await restoreBackup(backupTimestamp)

      expect(restoreResult.success).toBe(true)
      expect(restoreResult.restoredFrom).toBe(backupTimestamp)
      expect(restoreResult.backupCreated).toBeDefined() // Safety backup created

      // Verify content was restored
      const loaded = await loadConfig()
      expect(loaded.content).toBe(beforeContent)
    })

    it('should delete backup', async () => {
      // Create backup to delete
      const backupResult = await createManualBackup('delete-test')
      const timestamp = backupResult.timestamp

      // Delete it
      await deleteBackup(timestamp)

      // Verify it's gone
      await expect(getBackup(timestamp)).rejects.toThrow()
    })

    it('should enforce backup count limit', async () => {
      // Create many backups (assuming limit is 10)
      for (let i = 0; i < 15; i++) {
        await createManualBackup(`limit-test-${i}`)
        await new Promise(resolve => setTimeout(resolve, 50))
      }

      const result = await listBackups()

      // Should not exceed limit (typically 10)
      expect(result.backups.length).toBeLessThanOrEqual(10)
    })
  })

  describe('Error Handling', () => {
    it('should handle network timeout gracefully', async () => {
      // Stop server to simulate network failure
      await stopServer()

      await expect(loadConfig()).rejects.toThrow()

      // Restart for remaining tests
      await startServer()
    }, 90000)

    it('should handle corrupted configuration file', async () => {
      // Write corrupted file directly (simulating external corruption)
      await writeFile(join(TEST_PROJECT_DIR, 'azure.yaml'), '\x00\x01\x02CORRUPTED')

      // Should fail gracefully
      await expect(loadConfig()).rejects.toThrow()

      // Restore valid config
      await writeFile(
        join(TEST_PROJECT_DIR, 'azure.yaml'),
        'name: restored\nservices:\n  api:\n    host: containerapp\n'
      )
    })

    it('should handle concurrent save operations', async () => {
      // Issue multiple saves simultaneously
      const saves = Array.from({ length: 5 }, (_, i) => 
        saveConfig(`name: concurrent-${i}\nservices:\n  api:\n    host: containerapp\n`)
      )

      const results = await Promise.all(saves)

      // All should succeed (or all but last might be rejected)
      const successful = results.filter(r => r.success)
      expect(successful.length).toBeGreaterThan(0)
    })
  })

  describe('Well-Known Services API', () => {
    it('should fetch well-known services definitions', async () => {
      const response = await fetch(`${API_BASE}/api/editor/well-known-services`)
      expect(response.ok).toBe(true)

      const data = await response.json()
      
      expect(data).toBeDefined()
      expect(Array.isArray(data.services) || typeof data === 'object').toBe(true)
    })
  })

  describe('Schema API', () => {
    it('should fetch azure.yaml schema', async () => {
      const response = await fetch(`${API_BASE}/api/editor/schema`)
      expect(response.ok).toBe(true)

      const schema = await response.json()

      expect(schema.$schema).toBeDefined()
      expect(schema.type).toBe('object')
      expect(schema.properties).toBeDefined()
      expect(schema.properties.name).toBeDefined()
      expect(schema.properties.services).toBeDefined()
    })
  })
})
