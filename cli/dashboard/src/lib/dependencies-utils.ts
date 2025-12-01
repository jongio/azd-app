/**
 * Helper functions for ServiceDependencies component
 */
import type { Service } from '@/types'

/** Grouped services by language */
export type GroupedServices = Record<string, Service[]>

/** Style configuration for language badges */
export interface LanguageBadgeStyle {
  bg: string
  text: string
  abbr: string
}

/** Status indicator configuration */
export interface StatusIndicator {
  icon: string
  color: string
  animate: string
}

/**
 * Group services by their language/technology
 */
export function groupServicesByLanguage(services: Service[]): GroupedServices {
  return services.reduce((groups, service) => {
    const language = normalizeLanguage(service.language || 'Other')
    if (!groups[language]) {
      groups[language] = []
    }
    groups[language].push(service)
    return groups
  }, {} as GroupedServices)
}

/**
 * Normalize language names to consistent display values
 */
export function normalizeLanguage(language: string): string {
  const normalized = language.toLowerCase()
  const languageMap: Record<string, string> = {
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
  }
  return languageMap[normalized] || language
}

/**
 * Get badge styling for a language
 */
export function getLanguageBadgeStyle(language: string): LanguageBadgeStyle {
  const styles: Record<string, LanguageBadgeStyle> = {
    TypeScript: { bg: 'bg-blue-500/10', text: 'text-blue-500', abbr: 'TS' },
    JavaScript: { bg: 'bg-yellow-500/10', text: 'text-yellow-500', abbr: 'JS' },
    Python: { bg: 'bg-green-500/10', text: 'text-green-500', abbr: 'PY' },
    Go: { bg: 'bg-cyan-500/10', text: 'text-cyan-500', abbr: 'GO' },
    Rust: { bg: 'bg-orange-500/10', text: 'text-orange-500', abbr: 'RS' },
    Java: { bg: 'bg-red-500/10', text: 'text-red-500', abbr: 'JV' },
    'C#': { bg: 'bg-purple-500/10', text: 'text-purple-500', abbr: 'C#' },
    '.NET': { bg: 'bg-purple-500/10', text: 'text-purple-500', abbr: '.N' },
  }
  return styles[language] || { bg: 'bg-gray-500/10', text: 'text-gray-500', abbr: '??' }
}

/**
 * Get status indicator for a service status
 */
export function getStatusIndicator(status?: string): StatusIndicator {
  const indicators: Record<string, StatusIndicator> = {
    running: { icon: '●', color: 'text-green-500', animate: 'animate-pulse' },
    ready: { icon: '●', color: 'text-green-500', animate: '' },
    starting: { icon: '◐', color: 'text-yellow-500', animate: 'animate-spin' },
    stopping: { icon: '◑', color: 'text-yellow-500', animate: '' },
    stopped: { icon: '○', color: 'text-gray-500', animate: '' },
    error: { icon: '⚠', color: 'text-red-500', animate: 'animate-pulse' },
    'not-running': { icon: '○', color: 'text-gray-500', animate: '' },
  }
  return indicators[status || 'not-running'] || indicators['not-running']
}

/**
 * Count environment variables for a service
 */
export function countEnvVars(service: Service): number {
  return Object.keys(service.environmentVariables || {}).length
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
 * Get the local URL for a service
 */
export function getServiceUrl(service: Service): string | null {
  // Check for URL in local info
  if (service.local?.url) {
    return service.local.url
  }
  // Build URL from port if available
  if (service.local?.port) {
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
