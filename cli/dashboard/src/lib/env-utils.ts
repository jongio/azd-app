import type { Service } from '@/types'

/**
 * Represents an aggregated environment variable with all services that use it
 */
export interface AggregatedEnvVar {
  name: string
  value: string
  services: string[]
  isSensitive: boolean
}

/**
 * Patterns to match sensitive variable names
 * Uses substring matching (e.g., 'password' matches 'DATABASE_PASSWORD')
 */
const SENSITIVE_PATTERNS = new Set([
  'password',
  'secret',
  'key',
  'token',
  'api_key',
  'apikey',
  'auth',
  'credential',
  'private',
  'connection_string',
  'connectionstring',
] as const)

/**
 * Determines if an environment variable name indicates a sensitive value
 */
export function isSensitiveVariable(name: string): boolean {
  const lowerName = name.toLowerCase()
  for (const pattern of SENSITIVE_PATTERNS) {
    if (lowerName.includes(pattern)) return true
  }
  return false
}

/**
 * Aggregates environment variables from all services
 * Groups by variable name, collects all services that use each variable
 */
export function aggregateEnvironmentVariables(services: Service[]): AggregatedEnvVar[] {
  const envMap = new Map<string, AggregatedEnvVar>()

  for (const service of services) {
    const envVars = service.environmentVariables ?? {}

    for (const [name, value] of Object.entries(envVars)) {
      const existing = envMap.get(name)

      if (existing) {
        // Variable exists - add service to list
        if (!existing.services.includes(service.name)) {
          existing.services.push(service.name)
        }
      } else {
        // New variable
        envMap.set(name, {
          name,
          value,
          services: [service.name],
          isSensitive: isSensitiveVariable(name),
        })
      }
    }
  }

  // Sort alphabetically by variable name
  return Array.from(envMap.values()).sort((a, b) =>
    a.name.localeCompare(b.name)
  )
}

/**
 * Filters environment variables based on search query and service selection
 */
export function filterEnvironmentVariables(
  variables: AggregatedEnvVar[],
  searchQuery: string,
  selectedService: string | null
): AggregatedEnvVar[] {
  return variables.filter(envVar => {
    // Service filter
    if (selectedService && !envVar.services.includes(selectedService)) {
      return false
    }

    // Search filter (name OR value)
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      const matchesName = envVar.name.toLowerCase().includes(query)
      const matchesValue = envVar.value.toLowerCase().includes(query)
      if (!matchesName && !matchesValue) {
        return false
      }
    }

    return true
  })
}
