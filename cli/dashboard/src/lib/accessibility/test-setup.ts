/**
 * Vitest setup for axe-core accessibility testing
 */

import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'
import { runWCAGAA, assertNoViolations } from './axe-testing'

// Cleanup after each test
afterEach(() => {
  cleanup()
})

/**
 * Custom matcher for accessibility testing
 */
export async function toHaveNoViolations(container: HTMLElement) {
  const result = await runWCAGAA(container)

  try {
    assertNoViolations(result)
    return {
      pass: true,
      message: () => `Expected accessibility violations, but found none`,
    }
  } catch (error) {
    return {
      pass: false,
      message: () => (error as Error).message,
    }
  }
}

// Extend Vitest matchers
declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Vi {
    interface Matchers<R = unknown> {
      toHaveNoViolations(): Promise<R>
    }
  }
}
