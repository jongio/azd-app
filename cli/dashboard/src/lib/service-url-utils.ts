/**
 * Service URL utilities - multi-field URL precedence logic
 * 
 * Implements the URL precedence rules from the service-url-fields spec:
 * - Local: customUrl > url
 * - Azure: customDomain (user) > customDomain (SDK) > customUrl > url
 */

import type { LocalServiceInfo, AzureServiceInfo } from '@/types'

// =============================================================================
// Constants
// =============================================================================

/**
 * Tailwind color classes for URL badges
 * Centralized to ensure consistency and easy theme updates
 */
const BADGE_COLORS = {
  PURPLE: 'bg-purple-50 dark:bg-purple-500/10 border-purple-200 dark:border-purple-500/30 text-purple-700 dark:text-purple-300',
  AMBER: 'bg-amber-50 dark:bg-amber-500/10 border-amber-200 dark:border-amber-500/30 text-amber-700 dark:text-amber-300',
  CYAN: 'bg-cyan-50 dark:bg-cyan-500/10 border-cyan-200 dark:border-cyan-500/30 text-cyan-700 dark:text-cyan-300',
} as const

/**
 * Icon color classes for URL sources
 */
const ICON_COLORS = {
  PURPLE: 'text-purple-600 dark:text-purple-400',
  AMBER: 'text-amber-600 dark:text-amber-400',
  CYAN: 'text-cyan-600 dark:text-cyan-400',
  DEFAULT: 'text-slate-400',
} as const

/**
 * Badge labels for different URL sources
 */
const BADGE_LABELS = {
  CUSTOM_URL: 'Custom URL',
  CUSTOM_DOMAIN: 'Custom Domain',
  CUSTOM_DOMAIN_AZURE: 'Custom Domain (Azure)',
  LOCAL_URL: 'Local URL',
  DEPLOYMENT_URL: 'Deployment URL',
} as const

/**
 * Badge descriptions for tooltips
 */
const BADGE_DESCRIPTIONS = {
  USER_CONFIGURED_LOCAL: 'User-configured custom local URL',
  AUTO_DISCOVERED_LOCAL: 'Auto-discovered local URL',
  USER_CONFIGURED_DOMAIN: 'User-configured custom domain (highest priority)',
  AZURE_SDK_DOMAIN: 'Custom domain discovered from Azure Portal',
  USER_CONFIGURED_URL: 'User-configured custom Azure URL',
  AUTO_DISCOVERED_AZURE: 'Auto-discovered Azure deployment URL',
} as const

/**
 * Regex pattern to detect unbound ports (port 0)
 * Matches URLs ending in :0 or :0/
 * Used to filter out URLs from services that haven't been assigned a port yet
 */
const UNBOUND_PORT_PATTERN = /:0\/?$/

// =============================================================================
// Types
// =============================================================================

export interface UrlBadgeConfig {
  color: string
  label: string
  description: string
}

export interface EffectiveLocalUrl {
  url: string | null
  source: 'customUrl' | 'url' | null
  defaultUrl?: string
}

export interface EffectiveAzureUrl {
  url: string | null
  source: 'customDomain-user' | 'customDomain-sdk' | 'customUrl' | 'url' | null
  defaultUrl?: string
  allUrls?: {
    url?: string
    customUrl?: string
    customDomain?: string
    customDomainSource?: 'user' | 'azure-sdk'
  }
}

// =============================================================================
// Validation Utilities
// =============================================================================

/**
 * Validates if a string is a well-formed URL
 * @param url - String to validate
 * @returns true if valid URL, false otherwise
 */
function isValidUrl(url: string | undefined | null): url is string {
  if (!url || typeof url !== 'string' || url.trim() === '') {
    return false
  }
  
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

/**
 * Checks if URL has an unbound port (port 0)
 * @param url - URL to check
 * @returns true if URL has port 0, false otherwise
 */
function hasUnboundPort(url: string): boolean {
  return UNBOUND_PORT_PATTERN.test(url)
}

// =============================================================================
// URL Precedence Functions
// =============================================================================

/**
 * Get the effective local URL based on precedence rules.
 * 
 * Precedence: local.customUrl > local.url
 * 
 * @param local - Local service information containing URLs
 * @returns Effective URL with source information
 * 
 * @example
 * ```ts
 * const result = getEffectiveLocalUrl({ 
 *   url: 'http://localhost:3000',
 *   customUrl: 'https://custom.dev'
 * })
 * // Returns: { url: 'https://custom.dev', source: 'customUrl', defaultUrl: 'http://localhost:3000' }
 * ```
 */
export function getEffectiveLocalUrl(local?: LocalServiceInfo): EffectiveLocalUrl {
  if (!local) {
    return { url: null, source: null }
  }

  // Precedence 1: customUrl (user-configured, highest priority)
  if (isValidUrl(local.customUrl)) {
    return {
      url: local.customUrl,
      source: 'customUrl',
      ...(local.url && { defaultUrl: local.url }),
    }
  }

  // Precedence 2: url (auto-discovered)
  // Filter out URLs with unbound ports (port 0)
  if (isValidUrl(local.url) && !hasUnboundPort(local.url)) {
    return {
      url: local.url,
      source: 'url',
    }
  }

  return { url: null, source: null }
}

/**
 * Get the effective Azure URL based on precedence rules.
 * 
 * Precedence: customDomain (user) > customDomain (SDK) > customUrl > url
 * 
 * @param azure - Azure service information containing URLs
 * @returns Effective URL with source and all available URLs
 * 
 * @example
 * ```ts
 * const result = getEffectiveAzureUrl({ 
 *   url: 'https://app.azurewebsites.net',
 *   customUrl: 'https://api.example.com',
 *   customDomain: 'example.com',
 *   customDomainSource: 'user'
 * })
 * // Returns: { url: 'example.com', source: 'customDomain-user', allUrls: {...} }
 * ```
 */
export function getEffectiveAzureUrl(azure?: AzureServiceInfo): EffectiveAzureUrl {
  if (!azure) {
    return { url: null, source: null }
  }

  // Precedence 1: customDomain from user configuration (highest)
  if (isValidUrl(azure.customDomain) && azure.customDomainSource === 'user') {
    return {
      url: azure.customDomain,
      source: 'customDomain-user',
      allUrls: {
        url: azure.url,
        customUrl: azure.customUrl,
        customDomain: azure.customDomain,
        customDomainSource: azure.customDomainSource,
      },
    }
  }

  // Precedence 2: customDomain from Azure SDK
  if (isValidUrl(azure.customDomain) && azure.customDomainSource === 'azure-sdk') {
    return {
      url: azure.customDomain,
      source: 'customDomain-sdk',
      ...(azure.customUrl && { defaultUrl: azure.customUrl }),
      ...(!azure.customUrl && azure.url && { defaultUrl: azure.url }),
      allUrls: {
        url: azure.url,
        customUrl: azure.customUrl,
        customDomain: azure.customDomain,
        customDomainSource: azure.customDomainSource,
      },
    }
  }

  // Precedence 3: customUrl (user-configured)
  if (isValidUrl(azure.customUrl)) {
    return {
      url: azure.customUrl,
      source: 'customUrl',
      ...(azure.url && { defaultUrl: azure.url }),
      allUrls: {
        url: azure.url,
        customUrl: azure.customUrl,
        customDomain: azure.customDomain,
        customDomainSource: azure.customDomainSource,
      },
    }
  }

  // Precedence 4: url (auto-discovered from env vars)
  if (isValidUrl(azure.url)) {
    return {
      url: azure.url,
      source: 'url',
      allUrls: {
        url: azure.url,
        customUrl: azure.customUrl,
        customDomain: azure.customDomain,
        customDomainSource: azure.customDomainSource,
      },
    }
  }

  return { url: null, source: null }
}

// =============================================================================
// Badge Configuration Functions
// =============================================================================

/**
 * Get badge configuration for local URL source.
 * 
 * @param source - URL source type
 * @returns Badge configuration with colors, label, and description
 * 
 * Color scheme:
 * - Purple: User-configured URLs
 * - Cyan: Auto-discovered URLs
 */
export function getLocalUrlBadgeConfig(source: EffectiveLocalUrl['source']): UrlBadgeConfig | null {
  if (!source) return null

  const configs: Record<NonNullable<EffectiveLocalUrl['source']>, UrlBadgeConfig> = {
    customUrl: {
      color: BADGE_COLORS.PURPLE,
      label: BADGE_LABELS.CUSTOM_URL,
      description: BADGE_DESCRIPTIONS.USER_CONFIGURED_LOCAL,
    },
    url: {
      color: BADGE_COLORS.CYAN,
      label: BADGE_LABELS.LOCAL_URL,
      description: BADGE_DESCRIPTIONS.AUTO_DISCOVERED_LOCAL,
    },
  }

  return configs[source] ?? null
}

/**
 * Get badge configuration for Azure URL source.
 * 
 * @param source - URL source type
 * @returns Badge configuration with colors, label, and description
 * 
 * Color scheme:
 * - Purple: User-configured URLs and domains
 * - Amber: Azure SDK-discovered custom domains
 * - Cyan: Auto-discovered deployment URLs
 */
export function getAzureUrlBadgeConfig(source: EffectiveAzureUrl['source']): UrlBadgeConfig | null {
  if (!source) return null

  const configs: Record<NonNullable<EffectiveAzureUrl['source']>, UrlBadgeConfig> = {
    'customDomain-user': {
      color: BADGE_COLORS.PURPLE,
      label: BADGE_LABELS.CUSTOM_DOMAIN,
      description: BADGE_DESCRIPTIONS.USER_CONFIGURED_DOMAIN,
    },
    'customDomain-sdk': {
      color: BADGE_COLORS.AMBER,
      label: BADGE_LABELS.CUSTOM_DOMAIN_AZURE,
      description: BADGE_DESCRIPTIONS.AZURE_SDK_DOMAIN,
    },
    customUrl: {
      color: BADGE_COLORS.PURPLE,
      label: BADGE_LABELS.CUSTOM_URL,
      description: BADGE_DESCRIPTIONS.USER_CONFIGURED_URL,
    },
    url: {
      color: BADGE_COLORS.CYAN,
      label: BADGE_LABELS.DEPLOYMENT_URL,
      description: BADGE_DESCRIPTIONS.AUTO_DISCOVERED_AZURE,
    },
  }

  return configs[source] ?? null
}

// =============================================================================
// Icon Color Functions
// =============================================================================

/**
 * Get icon color class for local URL based on source.
 * 
 * @param source - URL source type
 * @returns Tailwind color class string
 */
export function getLocalUrlIconColor(source: EffectiveLocalUrl['source']): string {
  if (!source) return ICON_COLORS.DEFAULT

  const colors: Record<NonNullable<EffectiveLocalUrl['source']>, string> = {
    customUrl: ICON_COLORS.PURPLE,
    url: ICON_COLORS.CYAN,
  }

  return colors[source] ?? ICON_COLORS.DEFAULT
}

/**
 * Get icon color class for Azure URL based on source.
 * 
 * @param source - URL source type
 * @returns Tailwind color class string
 */
export function getAzureUrlIconColor(source: EffectiveAzureUrl['source']): string {
  if (!source) return ICON_COLORS.DEFAULT

  const colors: Record<NonNullable<EffectiveAzureUrl['source']>, string> = {
    'customDomain-user': ICON_COLORS.PURPLE,
    'customDomain-sdk': ICON_COLORS.AMBER,
    customUrl: ICON_COLORS.PURPLE,
    url: ICON_COLORS.CYAN,
  }

  return colors[source] ?? ICON_COLORS.DEFAULT
}

// =============================================================================
// Tooltip Functions
// =============================================================================

/**
 * Build tooltip text for local URL.
 * 
 * @param effectiveUrl - Effective local URL result
 * @returns Tooltip text or undefined if no tooltip needed
 */
export function getLocalUrlTooltip(effectiveUrl: EffectiveLocalUrl): string | undefined {
  if (!effectiveUrl.source) return undefined

  if (effectiveUrl.source === 'customUrl' && effectiveUrl.defaultUrl) {
    return `Custom URL configured (default: ${effectiveUrl.defaultUrl})`
  }

  return undefined
}

/**
 * Build tooltip text for Azure URL.
 * 
 * @param effectiveUrl - Effective Azure URL result
 * @returns Tooltip text or undefined if no tooltip needed
 */
export function getAzureUrlTooltip(effectiveUrl: EffectiveAzureUrl): string | undefined {
  if (!effectiveUrl.source) return undefined

  const tooltipParts = [
    effectiveUrl.source === 'customDomain-user' && 'User-configured custom domain',
    effectiveUrl.source === 'customDomain-user' && effectiveUrl.allUrls?.url && `Deployment: ${effectiveUrl.allUrls.url}`,
    effectiveUrl.source === 'customDomain-sdk' && 'Azure-discovered custom domain',
    effectiveUrl.source === 'customDomain-sdk' && effectiveUrl.defaultUrl && `Fallback: ${effectiveUrl.defaultUrl}`,
    effectiveUrl.source === 'customUrl' && effectiveUrl.defaultUrl && `Custom URL (default: ${effectiveUrl.defaultUrl})`,
  ].filter((part): part is string => Boolean(part))

  return tooltipParts.length > 0 ? tooltipParts.join(' • ') : undefined
}
