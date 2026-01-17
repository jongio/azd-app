/**
 * Copy Schema Script
 * 
 * Copies the azure.yaml.json schema from the repository root
 * into the dashboard bundle during build.
 * 
 * This ensures the bundled schema is always up to date with
 * the source schema file.
 */

import { readFileSync, writeFileSync, existsSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

// Paths
const repoRoot = join(__dirname, '../../..')
const sourceSchema = join(repoRoot, 'schemas/v1.1/azure.yaml.json')
const targetSchema = join(__dirname, '../src/lib/schema/bundled-schema.json')

console.log('Copying schema for dashboard bundle...')
console.log(`  Source: ${sourceSchema}`)
console.log(`  Target: ${targetSchema}`)

// Check if source exists
if (!existsSync(sourceSchema)) {
  console.error(`ERROR: Source schema not found at ${sourceSchema}`)
  process.exit(1)
}

try {
  // Read source schema
  const schemaContent = readFileSync(sourceSchema, 'utf-8')
  
  // Validate it's valid JSON
  JSON.parse(schemaContent)
  
  // Write to target
  writeFileSync(targetSchema, schemaContent, 'utf-8')
  
  console.log('✓ Schema copied successfully')
} catch (error) {
  console.error('ERROR: Failed to copy schema:', error)
  process.exit(1)
}
