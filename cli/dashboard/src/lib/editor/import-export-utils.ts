/**
 * Import/Export Utilities
 * Helper functions for importing and exporting configurations
 */

import type {
  DiffSection,
  CherryPickSection,
  SecurityWarning,
  MergeStrategy,
} from './import-export-types'

/**
 * Merge configurations based on strategy
 */
export function mergeConfigurations(
  current: Record<string, unknown>,
  imported: Record<string, unknown>,
  strategy: MergeStrategy,
  selectedSections?: string[]
): Record<string, unknown> {
  switch (strategy) {
    case 'replace':
      return imported

    case 'merge':
      return deepMerge(current, imported)

    case 'cherry-pick':
      return cherryPickMerge(current, imported, selectedSections || [])

    default:
      return current
  }
}

/**
 * Deep merge two objects
 */
function deepMerge(target: Record<string, unknown>, source: Record<string, unknown>): Record<string, unknown> {
  const result = { ...target }

  for (const key in source) {
    const sourceValue = source[key]
    const targetValue = result[key]

    if (isObject(sourceValue) && isObject(targetValue)) {
      result[key] = deepMerge(
        targetValue as Record<string, unknown>,
        sourceValue as Record<string, unknown>
      )
    } else if (Array.isArray(sourceValue) && Array.isArray(targetValue)) {
      // Append arrays
      result[key] = [...targetValue, ...sourceValue]
    } else {
      result[key] = sourceValue
    }
  }

  return result
}

/**
 * Cherry-pick specific sections from imported config
 */
function cherryPickMerge(
  current: Record<string, unknown>,
  imported: Record<string, unknown>,
  selectedSections: string[]
): Record<string, unknown> {
  const result = { ...current }

  for (const section of selectedSections) {
    if (section in imported) {
      const importedValue = imported[section]
      const currentValue = result[section]

      if (isObject(importedValue) && isObject(currentValue)) {
        result[section] = {
          ...(currentValue as Record<string, unknown>),
          ...(importedValue as Record<string, unknown>),
        }
      } else {
        result[section] = importedValue
      }
    }
  }

  return result
}

/**
 * Generate diff between current and imported configurations
 */
export function generateDiff(
  current: Record<string, unknown>,
  imported: Record<string, unknown>
): DiffSection[] {
  const diff: DiffSection[] = []
  const allKeys = new Set([...Object.keys(current), ...Object.keys(imported)])

  for (const key of allKeys) {
    const currentValue = current[key]
    const importedValue = imported[key]

    if (!(key in current)) {
      diff.push({
        path: key,
        type: 'added',
        importedValue,
      })
    } else if (!(key in imported)) {
      diff.push({
        path: key,
        type: 'removed',
        currentValue,
      })
    } else if (JSON.stringify(currentValue) !== JSON.stringify(importedValue)) {
      diff.push({
        path: key,
        type: 'changed',
        currentValue,
        importedValue,
      })

      // Recursively diff nested objects
      if (isObject(currentValue) && isObject(importedValue)) {
        const nestedDiff = generateDiff(
          currentValue as Record<string, unknown>,
          importedValue as Record<string, unknown>
        )
        diff.push(
          ...nestedDiff.map(d => ({
            ...d,
            path: `${key}.${d.path}`,
          }))
        )
      }
    } else {
      diff.push({
        path: key,
        type: 'unchanged',
        currentValue,
      })
    }
  }

  return diff
}

/**
 * Extract cherry-pickable sections from configuration
 */
export function extractCherryPickSections(config: Record<string, unknown>): CherryPickSection[] {
  const sections: CherryPickSection[] = []

  // Services
  if (config.services && isObject(config.services)) {
    const services = config.services as Record<string, unknown>
    for (const [name, service] of Object.entries(services)) {
      sections.push({
        id: `service.${name}`,
        name: `Service: ${name}`,
        description: getServiceDescription(service),
        type: 'service',
        selected: false,
      })
    }
  }

  // Resources
  if (config.resources && isObject(config.resources)) {
    const resources = config.resources as Record<string, unknown>
    for (const [name, resource] of Object.entries(resources)) {
      sections.push({
        id: `resource.${name}`,
        name: `Resource: ${name}`,
        description: getResourceDescription(resource),
        type: 'resource',
        selected: false,
      })
    }
  }

  // Hooks
  if (config.hooks && isObject(config.hooks)) {
    sections.push({
      id: 'hooks',
      name: 'Lifecycle Hooks',
      description: 'All lifecycle hooks configuration',
      type: 'hooks',
      selected: false,
    })
  }

  // Pipeline
  if (config.pipeline && isObject(config.pipeline)) {
    sections.push({
      id: 'pipeline',
      name: 'Pipeline Configuration',
      description: 'CI/CD pipeline settings',
      type: 'pipeline',
      selected: false,
    })
  }

  return sections
}

/**
 * Apply cherry-picked selections to merge
 */
export function applyCherryPick(
  current: Record<string, unknown>,
  imported: Record<string, unknown>,
  selections: CherryPickSection[]
): Record<string, unknown> {
  const result = { ...current }

  for (const selection of selections) {
    if (!selection.selected) continue

    const [type, name] = selection.id.split('.')

    if (type === 'service') {
      const services = (imported.services as Record<string, unknown>) || {}
      if (name && name in services) {
        const currentServices = (result.services as Record<string, unknown>) || {}
        result.services = {
          ...currentServices,
          [name]: services[name],
        }
      }
    } else if (type === 'resource') {
      const resources = (imported.resources as Record<string, unknown>) || {}
      if (name && name in resources) {
        const currentResources = (result.resources as Record<string, unknown>) || {}
        result.resources = {
          ...currentResources,
          [name]: resources[name],
        }
      }
    } else if (type === 'hooks') {
      result.hooks = imported.hooks
    } else if (type === 'pipeline') {
      result.pipeline = imported.pipeline
    }
  }

  return result
}

/**
 * Detect security warnings in configuration
 */
export function detectSecurityWarnings(
  config: Record<string, unknown>,
  includeSecrets: boolean
): SecurityWarning[] {
  const warnings: SecurityWarning[] = []

  if (includeSecrets) {
    const hasSecrets = detectSecretsInConfig(config)
    if (hasSecrets) {
      warnings.push({
        type: 'secrets',
        message:
          'This export includes secret values. Anyone with access to the exported file will be able to read these secrets.',
        severity: 'warning',
        requiresConfirmation: true,
      })
    }
  }

  return warnings
}

/**
 * Detect secrets in configuration
 */
function detectSecretsInConfig(config: Record<string, unknown>): boolean {
  const secretKeys = ['secret', 'password', 'key', 'token', 'credential', 'apikey']

  function hasSecretKey(obj: unknown): boolean {
    if (!isObject(obj)) return false

    const record = obj as Record<string, unknown>
    
    for (const key of Object.keys(record)) {
      const lowerKey = key.toLowerCase()
      if (secretKeys.some(sk => lowerKey.includes(sk))) {
        return true
      }

      if (isObject(record[key]) && hasSecretKey(record[key])) {
        return true
      }

      if (Array.isArray(record[key])) {
        const arr = record[key] as unknown[]
        if (arr.some(item => isObject(item) && hasSecretKey(item))) {
          return true
        }
      }
    }

    return false
  }

  return hasSecretKey(config)
}

/**
 * Convert configuration to template format (with placeholders)
 */
export function convertToTemplate(config: Record<string, unknown>): Record<string, unknown> {
  return replaceValuesWithPlaceholders(config) as Record<string, unknown>
}

/**
 * Replace actual values with placeholders
 */
function replaceValuesWithPlaceholders(obj: unknown, path = ''): unknown {
  if (!isObject(obj)) {
    if (typeof obj === 'string') {
      // Replace common patterns with placeholders
      if (obj.match(/^https?:\/\//)) return '${URL}'
      if (obj.match(/^\d+$/)) return '${PORT}'
      if (obj.match(/^[a-z0-9-]+$/i) && path.includes('name')) return '${NAME}'
      return '${VALUE}'
    }
    if (typeof obj === 'number') return '${NUMBER}'
    return obj
  }

  const record = obj as Record<string, unknown>
  const result: Record<string, unknown> = {}

  for (const [key, value] of Object.entries(record)) {
    const newPath = path ? `${path}.${key}` : key

    if (isObject(value)) {
      result[key] = replaceValuesWithPlaceholders(value, newPath)
    } else if (Array.isArray(value)) {
      result[key] = value.map(item => replaceValuesWithPlaceholders(item, newPath))
    } else {
      result[key] = replaceValuesWithPlaceholders(value, newPath)
    }
  }

  return result
}

/**
 * Get service description for cherry-pick UI
 */
function getServiceDescription(service: unknown): string {
  if (!isObject(service)) return ''

  const s = service as Record<string, unknown>
  const parts: string[] = []

  if (s.host) parts.push(`host: ${s.host}`)
  if (s.image) parts.push(`image: ${s.image}`)
  if (s.project) parts.push(`project: ${s.project}`)

  return parts.join(', ') || 'Service configuration'
}

/**
 * Get resource description for cherry-pick UI
 */
function getResourceDescription(resource: unknown): string {
  if (!isObject(resource)) return ''

  const r = resource as Record<string, unknown>
  const parts: string[] = []

  if (r.type) parts.push(`type: ${r.type}`)
  if (r.uses) parts.push(`uses: ${Array.isArray(r.uses) ? r.uses.join(', ') : r.uses}`)

  return parts.join(', ') || 'Resource configuration'
}

/**
 * Type guard for objects
 */
function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

/**
 * Format file size
 */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

/**
 * Download file to disk
 */
export function downloadFile(content: string, filename: string, mimeType = 'text/yaml'): void {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * Copy text to clipboard
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch (error) {
    console.error('Failed to copy to clipboard:', error)
    return false
  }
}
