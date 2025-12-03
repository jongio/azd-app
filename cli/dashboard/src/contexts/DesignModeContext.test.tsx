import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DesignModeProvider, useDesignModeContext } from './DesignModeContext'

// Test component to access context
function TestComponent() {
  const { designMode, setDesignMode, isClassic, isModern } = useDesignModeContext()
  return (
    <div>
      <div data-testid="mode">{designMode}</div>
      <div data-testid="is-classic">{isClassic ? 'true' : 'false'}</div>
      <div data-testid="is-modern">{isModern ? 'true' : 'false'}</div>
      <button onClick={() => setDesignMode('modern')}>Set Modern</button>
      <button onClick={() => setDesignMode('classic')}>Set Classic</button>
    </div>
  )
}

// Store original values
let originalLocation: Location

describe('DesignModeContext', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear()
    vi.clearAllMocks()

    // Store original location
    originalLocation = window.location

    // Mock window.location
    const mockLocation = {
      ...originalLocation,
      search: '',
      href: 'http://localhost/',
    }
    Object.defineProperty(window, 'location', {
      writable: true,
      value: mockLocation,
    })
  })

  afterEach(() => {
    localStorage.clear()
    // Restore original location
    Object.defineProperty(window, 'location', {
      writable: true,
      value: originalLocation,
    })
  })

  describe('default behavior', () => {
    it('defaults to modern mode when no preference is set', () => {
      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
      expect(screen.getByTestId('is-classic')).toHaveTextContent('false')
      expect(screen.getByTestId('is-modern')).toHaveTextContent('true')
    })

    it('provides setDesignMode function that works', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')

      await user.click(screen.getByText('Set Classic'))

      expect(screen.getByTestId('mode')).toHaveTextContent('classic')
      expect(screen.getByTestId('is-classic')).toHaveTextContent('true')
      expect(screen.getByTestId('is-modern')).toHaveTextContent('false')
    })

    it('can switch back to modern mode', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      await user.click(screen.getByText('Set Classic'))
      expect(screen.getByTestId('mode')).toHaveTextContent('classic')

      await user.click(screen.getByText('Set Modern'))
      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })
  })

  describe('localStorage persistence', () => {
    it('persists mode to localStorage when changed', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      await user.click(screen.getByText('Set Modern'))

      expect(localStorage.getItem('dashboard-design-mode')).toBe('modern')
    })

    it('restores mode from localStorage on mount', () => {
      localStorage.setItem('dashboard-design-mode', 'modern')

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })

    it('ignores invalid localStorage values', () => {
      localStorage.setItem('dashboard-design-mode', 'invalid-mode')

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      // Should default to modern
      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })
  })

  describe('URL parameter parsing', () => {
    it('reads design mode from URL parameter', () => {
      // Set URL parameter
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?design=modern',
          href: 'http://localhost/?design=modern',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })

    it('URL parameter takes precedence over localStorage', () => {
      // Set localStorage to classic
      localStorage.setItem('dashboard-design-mode', 'classic')

      // Set URL parameter to modern
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?design=modern',
          href: 'http://localhost/?design=modern',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      // URL parameter should win
      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })

    it('saves URL parameter preference to localStorage', () => {
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?design=modern',
          href: 'http://localhost/?design=modern',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      // Should save to localStorage for future visits
      expect(localStorage.getItem('dashboard-design-mode')).toBe('modern')
    })

    it('ignores invalid URL parameter values', () => {
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?design=invalid',
          href: 'http://localhost/?design=invalid',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      // Should default to modern
      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })

    it('handles empty URL parameter', () => {
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?design=',
          href: 'http://localhost/?design=',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      // Should default to modern
      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })

    it('handles other URL parameters without affecting design mode', () => {
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?other=value&foo=bar',
          href: 'http://localhost/?other=value&foo=bar',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      // Should default to modern
      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })

    it('handles URL parameter with other parameters', () => {
      Object.defineProperty(window, 'location', {
        writable: true,
        value: {
          ...originalLocation,
          search: '?foo=bar&design=modern&baz=qux',
          href: 'http://localhost/?foo=bar&design=modern&baz=qux',
        },
      })

      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')
    })
  })

  describe('error handling', () => {
    it('throws error when useDesignModeContext is used outside provider', () => {
      // Suppress console.error for this test
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      expect(() => {
        render(<TestComponent />)
      }).toThrow('useDesignModeContext must be used within a DesignModeProvider')

      consoleSpy.mockRestore()
    })

    it('handles localStorage errors gracefully', () => {
      // Mock localStorage to throw
      const getItemSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
        throw new Error('localStorage disabled')
      })

      // Should not throw, should default to modern
      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')

      getItemSpy.mockRestore()
    })
  })

  describe('popstate event handling', () => {
    it('syncs with URL changes on browser navigation', () => {
      render(
        <DesignModeProvider>
          <TestComponent />
        </DesignModeProvider>
      )

      expect(screen.getByTestId('mode')).toHaveTextContent('modern')

      // Simulate browser back/forward navigation
      act(() => {
        Object.defineProperty(window, 'location', {
          writable: true,
          value: {
            ...originalLocation,
            search: '?design=classic',
            href: 'http://localhost/?design=classic',
          },
        })
        window.dispatchEvent(new PopStateEvent('popstate'))
      })

      expect(screen.getByTestId('mode')).toHaveTextContent('classic')
    })
  })
})
