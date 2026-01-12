/**
 * Well-Known Services API
 * API client for fetching well-known service definitions
 */

import type { WellKnownService } from '@/lib/editor/wellknown-types'

/**
 * Fetch all well-known services
 */
export async function fetchWellKnownServices(): Promise<WellKnownService[]> {
  const response = await fetch('/api/editor/wellknown')
  
  if (!response.ok) {
    throw new Error(`Failed to fetch well-known services: HTTP ${response.status}: ${response.statusText}`)
  }
  
  const data = await response.json()
  return data.services || []
}

/**
 * Fetch specific well-known service by name
 */
export async function fetchWellKnownService(name: string): Promise<WellKnownService | null> {
  const response = await fetch(`/api/editor/wellknown/${name}`)
  
  if (!response.ok) {
    if (response.status === 404) {
      return null
    }
    throw new Error(`Failed to fetch well-known service ${name}: HTTP ${response.status}: ${response.statusText}`)
  }
  
  return await response.json()
}
