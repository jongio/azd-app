import { useDesignModeContext, type DesignMode } from '@/contexts/DesignModeContext'

/**
 * Hook to access and manage the design mode
 * 
 * @example
 * ```tsx
 * const { designMode, setDesignMode, isModern } = useDesignMode()
 * 
 * // Check current mode
 * if (isModern) {
 *   // render modern layout
 * }
 * 
 * // Switch mode
 * setDesignMode('modern')
 * ```
 */
export function useDesignMode() {
  const { designMode, setDesignMode, isClassic, isModern } = useDesignModeContext()

  return {
    /** Current design mode ('classic' or 'modern') */
    designMode,
    
    /** Function to change the design mode */
    setDesignMode,
    
    /** True if current mode is 'classic' */
    isClassic,
    
    /** True if current mode is 'modern' */
    isModern,
    
    /** Toggle between classic and modern modes */
    toggleDesignMode: () => {
      setDesignMode(designMode === 'classic' ? 'modern' : 'classic')
    },
  }
}

/**
 * Type guard to check if a value is a valid design mode
 */
export function isValidDesignMode(value: unknown): value is DesignMode {
  return value === 'classic' || value === 'modern'
}

export type { DesignMode }
