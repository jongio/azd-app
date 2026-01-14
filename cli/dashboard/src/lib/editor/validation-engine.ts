/**
 * Validation Engine - Multi-stage validation system for azure.yaml
 * 
 * Responsibilities:
 * 1. JSON Schema validation (Ajv)
 * 2. Custom business rules (unique names, port conflicts, circular deps)
 * 3. Validation levels (error/warning/info)
 */

import Ajv, { type ErrorObject } from 'ajv'
import type { ValidationError, ValidationResult, ValidationOptions } from './validation-types'

// Initialize Ajv instance
const ajv = new Ajv({ allErrors: true, verbose: true })

/**
 * Validate configuration against JSON Schema using Ajv
 */
export function validateSchema(
  config: Record<string, unknown>,
  schema: Record<string, unknown>
): ValidationError[] {
  const errors: ValidationError[] = []

  try {
    const validate = ajv.compile(schema)
    const valid = validate(config)

    if (!valid && validate.errors) {
      errors.push(...formatAjvErrors(validate.errors))
    }
  } catch (err) {
    errors.push({
      level: 'error',
      message: err instanceof Error ? err.message : 'Schema validation failed',
      path: '',
      rule: 'schema',
    })
  }

  return errors
}

/**
 * Format Ajv errors into ValidationError format
 */
function formatAjvErrors(ajvErrors: ErrorObject[]): ValidationError[] {
  return ajvErrors.map(err => {
    const path = err.instancePath.replace(/^\//, '').replace(/\//g, '.')
    let message = err.message || 'Validation failed'

    // Improve error messages for common cases
    switch (err.keyword) {
      case 'required':
        message = `${err.params.missingProperty} is required`
        break
      case 'type':
        message = `Must be of type ${err.params.type}`
        break
      case 'enum':
        message = `Must be one of: ${err.params.allowedValues?.join(', ')}`
        break
      case 'pattern':
        message = `Must match pattern: ${err.params.pattern}`
        break
      case 'minimum':
        message = `Must be at least ${err.params.limit}`
        break
      case 'maximum':
        message = `Must be no more than ${err.params.limit}`
        break
      case 'minLength':
        message = `Must be at least ${err.params.limit} characters`
        break
      case 'maxLength':
        message = `Must be no more than ${err.params.limit} characters`
        break
      case 'minItems':
        message = `Must have at least ${err.params.limit} items`
        break
      case 'maxItems':
        message = `Must have no more than ${err.params.limit} items`
        break
    }

    return {
      level: 'error',
      message,
      path: path || (err.params.missingProperty ? err.params.missingProperty : ''),
      rule: err.keyword,
    }
  })
}

/**
 * Validate unique service names
 */
export function validateUniqueServiceNames(
  config: Record<string, unknown>
): ValidationError[] {
  const errors: ValidationError[] = []
  const services = config.services as Record<string, unknown> | undefined

  if (!services || typeof services !== 'object') {
    return errors
  }

  const serviceNames = Object.keys(services)
  const duplicates = findDuplicates(serviceNames)

  if (duplicates.length > 0) {
    errors.push({
      level: 'error',
      message: `Duplicate service names: ${duplicates.join(', ')}`,
      path: 'services',
      rule: 'unique-names',
      context: 'Service names must be unique',
    })
  }

  return errors
}

/**
 * Validate unique resource names
 */
export function validateUniqueResourceNames(
  config: Record<string, unknown>
): ValidationError[] {
  const errors: ValidationError[] = []
  const resources = config.resources as Record<string, unknown> | undefined

  if (!resources || typeof resources !== 'object') {
    return errors
  }

  const resourceNames = Object.keys(resources)
  const duplicates = findDuplicates(resourceNames)

  if (duplicates.length > 0) {
    errors.push({
      level: 'error',
      message: `Duplicate resource names: ${duplicates.join(', ')}`,
      path: 'resources',
      rule: 'unique-names',
      context: 'Resource names must be unique',
    })
  }

  return errors
}

/**
 * Detect port conflicts (same port used by multiple services)
 */
export function validatePortConflicts(
  config: Record<string, unknown>
): ValidationError[] {
  const errors: ValidationError[] = []
  const services = config.services as Record<string, unknown> | undefined

  if (!services || typeof services !== 'object') {
    return errors
  }

  const portMap = buildPortMap(services)

  for (const [port, serviceNames] of portMap.entries()) {
    if (serviceNames.length > 1) {
      errors.push({
        level: 'warning',
        message: `Port ${port} is used by multiple services: ${serviceNames.join(', ')}`,
        path: 'services',
        rule: 'port-conflict',
        context: 'Multiple services using the same port may cause runtime conflicts',
      })
    }
  }

  return errors
}

/**
 * Detect circular dependencies in services and resources
 */
export function validateCircularDependencies(
  config: Record<string, unknown>
): ValidationError[] {
  const errors: ValidationError[] = []
  const services = (config.services as Record<string, unknown>) || {}
  const resources = (config.resources as Record<string, unknown>) || {}

  // Build dependency graph
  const graph = new Map<string, string[]>()

  // Add service dependencies
  for (const [serviceName, service] of Object.entries(services)) {
    if (typeof service !== 'object' || service === null) continue
    
    const deps: string[] = []
    const serviceObj = service as Record<string, unknown>
    
    // Check 'uses' field
    if (Array.isArray(serviceObj.uses)) {
      deps.push(...serviceObj.uses.map(String))
    } else if (typeof serviceObj.uses === 'string') {
      deps.push(serviceObj.uses)
    }
    
    graph.set(serviceName, deps)
  }

  // Add resource dependencies
  for (const [resourceName, resource] of Object.entries(resources)) {
    if (typeof resource !== 'object' || resource === null) continue
    
    const deps: string[] = []
    const resourceObj = resource as Record<string, unknown>
    
    // Check 'uses' field
    if (Array.isArray(resourceObj.uses)) {
      deps.push(...resourceObj.uses.map(String))
    } else if (typeof resourceObj.uses === 'string') {
      deps.push(resourceObj.uses)
    }
    
    graph.set(resourceName, deps)
  }

  // Detect cycles
  const cycles = detectCycles(graph)

  for (const cycle of cycles) {
    errors.push({
      level: 'error',
      message: `Circular dependency detected: ${cycle.join(' → ')}`,
      path: 'services',
      rule: 'circular-dependency',
      context: 'Dependencies must not form a cycle',
    })
  }

  return errors
}

/**
 * Validate recommended fields (info level)
 */
export function validateRecommendedFields(
  config: Record<string, unknown>
): ValidationError[] {
  const errors: ValidationError[] = []
  const services = config.services as Record<string, unknown> | undefined

  if (!services || typeof services !== 'object') {
    return errors
  }

  // Check for missing health checks
  for (const [serviceName, service] of Object.entries(services)) {
    if (typeof service !== 'object' || service === null) continue
    
    const serviceObj = service as Record<string, unknown>
    
    if (!serviceObj.healthcheck && !serviceObj.test) {
      errors.push({
        level: 'info',
        message: `Service '${serviceName}' is missing a health check`,
        path: `services.${serviceName}`,
        rule: 'recommended-healthcheck',
        context: 'Consider adding a health check to monitor service availability',
      })
    }
  }

  return errors
}

/**
 * Main validation function - runs all validation stages
 */
export function validateConfiguration(
  config: Record<string, unknown>,
  schema: Record<string, unknown>,
  options: ValidationOptions = {}
): ValidationResult {
  const {
    full = true,
    includeWarnings = true,
    includeInfo = true,
  } = options

  const allErrors: ValidationError[] = []

  // Stage 1: JSON Schema validation
  if (full) {
    allErrors.push(...validateSchema(config, schema))
  }

  // Stage 2: Custom business rules
  allErrors.push(...validateUniqueServiceNames(config))
  allErrors.push(...validateUniqueResourceNames(config))
  allErrors.push(...validateCircularDependencies(config))

  // Stage 3: Warnings
  if (includeWarnings) {
    allErrors.push(...validatePortConflicts(config))
  }

  // Stage 4: Info/recommendations
  if (includeInfo) {
    allErrors.push(...validateRecommendedFields(config))
  }

  // Separate by level
  const errors = allErrors.filter(e => e.level === 'error')
  const warnings = allErrors.filter(e => e.level === 'warning')
  const info = allErrors.filter(e => e.level === 'info')

  return {
    valid: errors.length === 0,
    errors,
    warnings,
    info,
  }
}

// Helper functions

/**
 * Find duplicate values in an array
 */
function findDuplicates(arr: string[]): string[] {
  const seen = new Set<string>()
  const duplicates = new Set<string>()

  for (const item of arr) {
    if (seen.has(item)) {
      duplicates.add(item)
    }
    seen.add(item)
  }

  return Array.from(duplicates)
}

/**
 * Build a map of ports to service names
 */
function buildPortMap(services: Record<string, unknown>): Map<string, string[]> {
  const portMap = new Map<string, string[]>()

  for (const [serviceName, service] of Object.entries(services)) {
    if (typeof service !== 'object' || service === null) continue
    
    const serviceObj = service as Record<string, unknown>
    const ports = serviceObj.ports

    if (!ports) continue

    // Handle different port formats
    const portList: string[] = []

    if (Array.isArray(ports)) {
      portList.push(...ports.map(String))
    } else if (typeof ports === 'string') {
      portList.push(ports)
    }

    // Extract port numbers from "host:container" format
    for (const portSpec of portList) {
      const match = portSpec.match(/(?:^|:)(\d+)(?:$|:)/)
      if (match) {
        const port = match[1]
        const services = portMap.get(port) || []
        services.push(serviceName)
        portMap.set(port, services)
      }
    }
  }

  return portMap
}

/**
 * Detect cycles in a dependency graph using DFS
 */
function detectCycles(graph: Map<string, string[]>): string[][] {
  const cycles: string[][] = []
  const visited = new Set<string>()
  const recStack = new Set<string>()

  function dfs(node: string, path: string[]): void {
    visited.add(node)
    recStack.add(node)
    path.push(node)

    const neighbors = graph.get(node) || []

    for (const neighbor of neighbors) {
      if (!graph.has(neighbor)) continue // Skip if neighbor doesn't exist

      if (!visited.has(neighbor)) {
        dfs(neighbor, [...path])
      } else if (recStack.has(neighbor)) {
        // Found a cycle
        const cycleStart = path.indexOf(neighbor)
        const cycle = [...path.slice(cycleStart), neighbor]
        cycles.push(cycle)
      }
    }

    recStack.delete(node)
  }

  for (const node of graph.keys()) {
    if (!visited.has(node)) {
      dfs(node, [])
    }
  }

  return cycles
}
