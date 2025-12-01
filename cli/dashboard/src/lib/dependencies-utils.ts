/**
 * Helper functions for ServiceDependencies component
 */
import type { Service } from '@/types'

// Re-export status indicator from unified service-utils
export { getStatusIndicator, type StatusIndicator } from '@/lib/service-utils'

/** Grouped services by language */
export type GroupedServices = Record<string, Service[]>

/** Style configuration for language badges */
export interface LanguageBadgeStyle {
  bg: string
  text: string
  abbr: string
}

/**
 * Group services by their language/technology
 */
export function groupServicesByLanguage(services: Service[]): GroupedServices {
  const groups = new Map<string, Service[]>()
  for (const service of services) {
    const language = normalizeLanguage(service.language ?? 'Other')
    const existing = groups.get(language)
    if (existing) {
      existing.push(service)
    } else {
      groups.set(language, [service])
    }
  }
  return Object.fromEntries(groups)
}

/**
 * Normalize language names to consistent display values
 */
export function normalizeLanguage(language: string): string {
  const normalized = language.toLowerCase()
  const languageMap = {
    ts: 'TypeScript',
    typescript: 'TypeScript',
    js: 'JavaScript',
    javascript: 'JavaScript',
    py: 'Python',
    python: 'Python',
    go: 'Go',
    golang: 'Go',
    rs: 'Rust',
    rust: 'Rust',
    java: 'Java',
    'c#': 'C#',
    csharp: 'C#',
    dotnet: '.NET',
    '.net': '.NET',
  } as const satisfies Record<string, string>
  return languageMap[normalized as keyof typeof languageMap] ?? language
}

/**
 * Get badge styling for a language
 */
export function getLanguageBadgeStyle(language: string): LanguageBadgeStyle {
  const styles = {
    TypeScript: { bg: 'bg-blue-500/10', text: 'text-blue-500', abbr: 'TS' },
    JavaScript: { bg: 'bg-yellow-500/10', text: 'text-yellow-500', abbr: 'JS' },
    Python: { bg: 'bg-green-500/10', text: 'text-green-500', abbr: 'PY' },
    Go: { bg: 'bg-cyan-500/10', text: 'text-cyan-500', abbr: 'GO' },
    Rust: { bg: 'bg-orange-500/10', text: 'text-orange-500', abbr: 'RS' },
    Java: { bg: 'bg-red-500/10', text: 'text-red-500', abbr: 'JV' },
    'C#': { bg: 'bg-purple-500/10', text: 'text-purple-500', abbr: 'C#' },
    '.NET': { bg: 'bg-purple-500/10', text: 'text-purple-500', abbr: '.N' },
  } as const satisfies Record<string, LanguageBadgeStyle>
  return styles[language as keyof typeof styles] ?? { bg: 'bg-gray-500/10', text: 'text-gray-500', abbr: '??' }
}

/**
 * Count environment variables for a service
 */
export function countEnvVars(service: Service): number {
  return Object.keys(service.environmentVariables ?? {}).length
}

/**
 * Sort groups by service count (descending) then by name (ascending)
 */
export function sortGroupsBySize(groups: GroupedServices): [string, Service[]][] {
  return Object.entries(groups).sort((a, b) => {
    // Sort by count descending
    if (b[1].length !== a[1].length) {
      return b[1].length - a[1].length
    }
    // Then by name ascending
    return a[0].localeCompare(b[0])
  })
}

/**
 * Check if a URL is valid (not localhost:0 or similar invalid ports)
 */
function isValidUrl(url: string): boolean {
  // Filter out URLs with port 0 (e.g., http://localhost:0)
  if (url.match(/:0\/?$/)) {
    return false
  }
  return true
}

/**
 * Get the local URL for a service
 */
export function getServiceUrl(service: Service): string | null {
  // Check for URL in local info and validate it
  if (service.local?.url && isValidUrl(service.local.url)) {
    return service.local.url
  }
  // Build URL from port if available (port must be > 0 to be valid)
  if (service.local?.port && service.local.port > 0) {
    return `http://localhost:${service.local.port}`
  }
  return null
}

/**
 * Pluralize a word based on count
 */
export function pluralize(count: number, singular: string, plural?: string): string {
  return count === 1 ? singular : (plural || `${singular}s`)
}
