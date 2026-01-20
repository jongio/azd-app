/**
 * Focus trap utility for modal dialogs
 * Ensures keyboard focus stays within a container
 */

export interface FocusTrapOptions {
  initialFocus?: HTMLElement | null
  onEscape?: () => void
}

/**
 * Get all focusable elements within a container
 */
export function getFocusableElements(container: HTMLElement): HTMLElement[] {
  const selector = [
    'a[href]',
    'button:not([disabled])',
    'textarea:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
  ].join(', ')

  return Array.from(container.querySelectorAll<HTMLElement>(selector)).filter(
    (el) => !el.hasAttribute('aria-hidden')
  )
}

/**
 * Create a focus trap within a container
 * Returns cleanup function to remove trap
 */
export function createFocusTrap(
  container: HTMLElement,
  options: FocusTrapOptions = {}
): () => void {
  const { initialFocus, onEscape } = options

  // Store previously focused element
  const previouslyFocused = document.activeElement as HTMLElement

  // Get all focusable elements
  const getFocusable = () => getFocusableElements(container)

  // Focus initial element or first focusable
  const focusableElements = getFocusable()
  const initialElement = initialFocus || focusableElements[0]
  if (initialElement) {
    initialElement.focus()
  }

  // Handle keyboard events
  function handleKeyDown(e: KeyboardEvent) {
    // Escape key
    if (e.key === 'Escape' && onEscape) {
      e.preventDefault()
      onEscape()
      return
    }

    // Tab key - trap focus
    if (e.key === 'Tab') {
      const focusable = getFocusable()
      if (focusable.length === 0) return

      const firstElement = focusable[0]
      const lastElement = focusable[focusable.length - 1]

      if (e.shiftKey) {
        // Shift+Tab - move backwards
        if (document.activeElement === firstElement) {
          e.preventDefault()
          lastElement.focus()
        }
      } else {
        // Tab - move forwards
        if (document.activeElement === lastElement) {
          e.preventDefault()
          firstElement.focus()
        }
      }
    }
  }

  // Add event listener
  container.addEventListener('keydown', handleKeyDown)

  // Return cleanup function
  return () => {
    container.removeEventListener('keydown', handleKeyDown)
    if (previouslyFocused && previouslyFocused.focus) {
      previouslyFocused.focus()
    }
  }
}

/**
 * React hook for focus trap
 */
export function useFocusTrap(
  ref: React.RefObject<HTMLElement>,
  isActive: boolean,
  options: FocusTrapOptions = {}
) {
  React.useEffect(() => {
    if (!isActive || !ref.current) return

    const cleanup = createFocusTrap(ref.current, options)
    return cleanup
  }, [isActive, ref, options])
}

// For React import
import * as React from 'react'
