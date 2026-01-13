/**
 * Custom hook for service URL handling
 * Eliminates prop drilling by centralizing URL logic and badge configuration
 */

import type { Service } from '@/types'
import { 
  getEffectiveLocalUrl, 
  getEffectiveAzureUrl,
  getLocalUrlBadgeConfig,
  getAzureUrlBadgeConfig,
  getLocalUrlIconColor,
  getAzureUrlIconColor,
  getLocalUrlTooltip,
  getAzureUrlTooltip,
  type EffectiveLocalUrl,
  type EffectiveAzureUrl,
  type UrlBadgeConfig,
} from '@/lib/service-url-utils'

export interface ServiceUrlData {
  // Local URLs
  localUrl: string | null
  localSource: EffectiveLocalUrl['source']
  localBadge: UrlBadgeConfig | null
  localIconColor: string
  localTooltip: string | undefined
  effectiveLocal: EffectiveLocalUrl
  
  // Azure URLs
  azureUrl: string | null
  azureSource: EffectiveAzureUrl['source']
  azureBadge: UrlBadgeConfig | null
  azureIconColor: string
  azureTooltip: string | undefined
  effectiveAzure: EffectiveAzureUrl
}

/**
 * Get all URL-related data for a service in a single hook
 * 
 * @param service - Service object containing local and azure info
 * @returns Complete URL data including badges, colors, tooltips
 * 
 * @example
 * ```tsx
 * const { localUrl, localBadge, azureUrl } = useServiceUrls(service)
 * ```
 */
export function useServiceUrls(service: Service | null): ServiceUrlData {
  if (!service) {
    return {
      localUrl: null,
      localSource: null,
      localBadge: null,
      localIconColor: 'text-slate-400',
      localTooltip: undefined,
      effectiveLocal: { url: null, source: null },
      azureUrl: null,
      azureSource: null,
      azureBadge: null,
      azureIconColor: 'text-slate-400',
      azureTooltip: undefined,
      effectiveAzure: { url: null, source: null },
    }
  }

  // Get effective URLs with precedence logic
  const effectiveLocal = getEffectiveLocalUrl(service.local)
  const effectiveAzure = getEffectiveAzureUrl(service.azure)

  // Get badge configurations
  const localBadge = getLocalUrlBadgeConfig(effectiveLocal.source)
  const azureBadge = getAzureUrlBadgeConfig(effectiveAzure.source)

  // Get icon colors
  const localIconColor = getLocalUrlIconColor(effectiveLocal.source)
  const azureIconColor = getAzureUrlIconColor(effectiveAzure.source)

  // Get tooltips
  const localTooltip = getLocalUrlTooltip(effectiveLocal)
  const azureTooltip = getAzureUrlTooltip(effectiveAzure)

  return {
    localUrl: effectiveLocal.url,
    localSource: effectiveLocal.source,
    localBadge,
    localIconColor,
    localTooltip,
    effectiveLocal,
    azureUrl: effectiveAzure.url,
    azureSource: effectiveAzure.source,
    azureBadge,
    azureIconColor,
    azureTooltip,
    effectiveAzure,
  }
}
