/**
 * useBicepTemplate - Hook to fetch unified Bicep template for diagnostic settings
 */
import * as React from 'react'

// =============================================================================
// Types
// =============================================================================

/**
 * Parameter definition for the Bicep template
 */
export interface BicepTemplateParameter {
  name: string
  description: string
  example: string
}

/**
 * Integration instructions for the Bicep template
 */
export interface BicepTemplateInstructions {
  summary: string
  steps: string[]
}

/**
 * API response from /api/azure/bicep-template
 */
export interface BicepTemplateResponse {
  template: string
  services: string[]
  instructions: BicepTemplateInstructions
  parameters: BicepTemplateParameter[]
}

/**
 * Hook state
 */
export interface UseBicepTemplateResult {
  /** Loading state */
  isLoading: boolean
  /** Error message if fetch failed */
  error: string | null
  /** Bicep template code */
  template: string | null
  /** List of services included in template */
  services: string[]
  /** Integration instructions */
  instructions: BicepTemplateInstructions | null
  /** Template parameters */
  parameters: BicepTemplateParameter[]
  /** Fetch the template (called automatically on mount, can be called manually to retry) */
  fetchTemplate: () => Promise<void>
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to fetch unified Bicep template for diagnostic settings.
 * 
 * Makes an API call to /api/azure/bicep-template to get a consolidated
 * Bicep template for all detected services.
 * 
 * @example
 * ```tsx
 * const { isLoading, template, services, instructions } = useBicepTemplate()
 * 
 * if (isLoading) return <Spinner />
 * if (template) {
 *   return <CodeBlock code={template} language="bicep" />
 * }
 * ```
 */
export function useBicepTemplate(): UseBicepTemplateResult {
  const [isLoading, setIsLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [template, setTemplate] = React.useState<string | null>(null)
  const [services, setServices] = React.useState<string[]>([])
  const [instructions, setInstructions] = React.useState<BicepTemplateInstructions | null>(null)
  const [parameters, setParameters] = React.useState<BicepTemplateParameter[]>([])

  // Abort controller for cleanup
  const abortControllerRef = React.useRef<AbortController | null>(null)

  // Fetch Bicep template
  const fetchTemplate = React.useCallback(async () => {
    // Cancel any in-flight request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    const controller = new AbortController()
    abortControllerRef.current = controller

    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch('/api/azure/bicep-template', {
        signal: controller.signal,
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`API returned ${response.status}: ${errorText || response.statusText}`)
      }

      const data = (await response.json()) as BicepTemplateResponse

      setTemplate(data.template || null)
      setServices(data.services || [])
      setInstructions(data.instructions || null)
      setParameters(data.parameters || [])
      setError(null)
    } catch (err) {
      // Ignore abort errors
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }

      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch Bicep template'
      setError(errorMessage)
      setTemplate(null)
      setServices([])
      setInstructions(null)
      setParameters([])
    } finally {
      setIsLoading(false)

      // Clear the abort controller if this is still our request
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [])

  // Initial fetch on mount
  React.useEffect(() => {
    void fetchTemplate()
  }, [fetchTemplate])

  // Cleanup on unmount
  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [])

  return {
    isLoading,
    error,
    template,
    services,
    instructions,
    parameters,
    fetchTemplate,
  }
}
