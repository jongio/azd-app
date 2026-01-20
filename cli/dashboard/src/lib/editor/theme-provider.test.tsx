/**
 * Tests for Theme Provider
 * Task 21: Visual Design and Styling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider, useTheme, ThemeToggle } from './theme-provider'

// Test component that uses the theme hook
function TestComponent() {
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme()
  
  return (
    <div>
      <div data-testid="theme">{theme}</div>
      <div data-testid="resolved-theme">{resolvedTheme}</div>
      <button onClick={() => setTheme('light')} data-testid="set-light">
        Light
      </button>
      <button onClick={() => setTheme('dark')} data-testid="set-dark">
        Dark
      </button>
      <button onClick={() => setTheme('system')} data-testid="set-system">
        System
      </button>
      <button onClick={toggleTheme} data-testid="toggle">
        Toggle
      </button>
    </div>
  )
}

describe('ThemeProvider', () => {
  beforeEach(() => {
    // Clear localStorage
    localStorage.clear()
    
    // Reset document classes and attributes
    document.documentElement.className = ''
    document.documentElement.removeAttribute('data-theme')
    document.documentElement.style.colorScheme = ''
    
    // Mock matchMedia
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query) => ({
        matches: query === '(prefers-color-scheme: dark)' ? false : true,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Initialization', () => {
    it('should default to system theme', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      expect(screen.getByTestId('theme')).toHaveTextContent('system')
      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('light')
    })

    it('should apply theme to document on mount', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      expect(document.documentElement.classList.contains('light')).toBe(true)
      expect(document.documentElement.getAttribute('data-theme')).toBe('light')
      expect(document.documentElement.style.colorScheme).toBe('light')
    })

    it('should load theme from localStorage', () => {
      localStorage.setItem('azure-yaml-editor-theme', 'dark')

      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      expect(screen.getByTestId('theme')).toHaveTextContent('dark')
      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')
    })

    it('should use custom storage key', () => {
      // Clear all keys first
      localStorage.clear()
      localStorage.setItem('custom-key', 'dark')

      const { rerender } = render(
        <ThemeProvider storageKey="custom-key">
          <TestComponent />
        </ThemeProvider>
      )

      // Force re-render to ensure state is settled
      rerender(
        <ThemeProvider storageKey="custom-key">
          <TestComponent />
        </ThemeProvider>
      )

      expect(screen.getByTestId('theme')).toHaveTextContent('dark')
      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')
    })
  })

  describe('Theme Setting', () => {
    it('should set light theme', async () => {
      const user = userEvent.setup()
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      await user.click(screen.getByTestId('set-light'))

      await waitFor(() => {
        expect(screen.getByTestId('theme')).toHaveTextContent('light')
        expect(screen.getByTestId('resolved-theme')).toHaveTextContent('light')
        expect(document.documentElement.classList.contains('light')).toBe(true)
        expect(localStorage.getItem('azure-yaml-editor-theme')).toBe('light')
      })
    })

    it('should set dark theme', async () => {
      const user = userEvent.setup()
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      await user.click(screen.getByTestId('set-dark'))

      await waitFor(() => {
        expect(screen.getByTestId('theme')).toHaveTextContent('dark')
        expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')
        expect(document.documentElement.classList.contains('dark')).toBe(true)
        expect(localStorage.getItem('azure-yaml-editor-theme')).toBe('dark')
      })
    })

    it('should set system theme', async () => {
      const user = userEvent.setup()
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      await user.click(screen.getByTestId('set-system'))

      await waitFor(() => {
        expect(screen.getByTestId('theme')).toHaveTextContent('system')
        expect(screen.getByTestId('resolved-theme')).toHaveTextContent('light')
        expect(localStorage.getItem('azure-yaml-editor-theme')).toBe('system')
      })
    })

    it('should remove old theme class when changing themes', async () => {
      const user = userEvent.setup()
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      await user.click(screen.getByTestId('set-dark'))
      expect(document.documentElement.classList.contains('dark')).toBe(true)
      expect(document.documentElement.classList.contains('light')).toBe(false)

      await user.click(screen.getByTestId('set-light'))
      expect(document.documentElement.classList.contains('light')).toBe(true)
      expect(document.documentElement.classList.contains('dark')).toBe(false)
    })
  })

  describe('Theme Toggle', () => {
    it('should toggle from light to dark', async () => {
      const user = userEvent.setup()
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      // Start with light
      await user.click(screen.getByTestId('set-light'))
      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('light')

      // Toggle to dark
      await user.click(screen.getByTestId('toggle'))
      await waitFor(() => {
        expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')
        expect(screen.getByTestId('theme')).toHaveTextContent('dark')
      })
    })

    it('should toggle from dark to light', async () => {
      const user = userEvent.setup()
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      // Start with dark
      await user.click(screen.getByTestId('set-dark'))
      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')

      // Toggle to light
      await user.click(screen.getByTestId('toggle'))
      await waitFor(() => {
        expect(screen.getByTestId('resolved-theme')).toHaveTextContent('light')
        expect(screen.getByTestId('theme')).toHaveTextContent('light')
      })
    })
  })

  describe('System Theme Detection', () => {
    it('should detect dark system preference', () => {
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation((query) => ({
          matches: query === '(prefers-color-scheme: dark)',
          media: query,
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
        })),
      })

      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')
    })

    it('should listen for system theme changes when theme is system', async () => {
      const listeners: ((e: MediaQueryListEvent) => void)[] = []
      
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation(() => ({
          matches: false,
          media: '(prefers-color-scheme: dark)',
          addEventListener: vi.fn((_, handler) => {
            listeners.push(handler)
          }),
          removeEventListener: vi.fn(),
        })),
      })

      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      )

      expect(screen.getByTestId('resolved-theme')).toHaveTextContent('light')

      // Simulate system theme change to dark
      listeners.forEach((listener) => {
        listener({ matches: true } as MediaQueryListEvent)
      })

      await waitFor(() => {
        expect(screen.getByTestId('resolved-theme')).toHaveTextContent('dark')
        expect(document.documentElement.classList.contains('dark')).toBe(true)
      })
    })
  })

  describe('Error Handling', () => {
    it('should handle localStorage errors gracefully', async () => {
      // Mock localStorage.setItem to throw an error
      const originalSetItem = localStorage.setItem
      localStorage.setItem = vi.fn(() => {
        throw new Error('localStorage error')
      })

      try {
        const user = userEvent.setup()
        
        render(
          <ThemeProvider>
            <TestComponent />
          </ThemeProvider>
        )

        // Click to change theme
        await user.click(screen.getByTestId('set-dark'))

        // Theme should still change even if localStorage fails
        await waitFor(() => {
          expect(screen.getByTestId('theme')).toHaveTextContent('dark')
        })
      } finally {
        // Restore
        localStorage.setItem = originalSetItem
      }
    })

    it('should throw error when useTheme is used outside provider', () => {
      // Suppress error boundary console errors
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      expect(() => {
        render(<TestComponent />)
      }).toThrow('useTheme must be used within a ThemeProvider')

      consoleSpy.mockRestore()
    })
  })
})

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
    document.documentElement.removeAttribute('data-theme')
    
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation(() => ({
        matches: false,
        media: '(prefers-color-scheme: dark)',
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    })
  })

  it('should render toggle button', () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    )

    const button = screen.getByRole('button')
    expect(button).toBeInTheDocument()
    expect(button).toHaveAttribute('aria-label')
  })

  it('should show moon icon in light mode', () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    )

    // Moon icon for dark mode option
    const svg = screen.getByRole('button').querySelector('svg')
    expect(svg).toBeInTheDocument()
    expect(svg?.querySelector('path')?.getAttribute('d')).toContain('12.79')
  })

  it('should show sun icon in dark mode', async () => {
    localStorage.setItem('azure-yaml-editor-theme', 'dark')
    
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    )

    // Sun icon for light mode option
    const svg = screen.getByRole('button').querySelector('svg')
    expect(svg).toBeInTheDocument()
    expect(svg?.querySelector('circle')).toBeInTheDocument()
  })

  it('should toggle theme on click', async () => {
    const user = userEvent.setup()
    
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    )

    expect(document.documentElement.classList.contains('light')).toBe(true)

    await user.click(screen.getByRole('button'))

    await waitFor(() => {
      expect(document.documentElement.classList.contains('dark')).toBe(true)
    })
  })

  it('should show label when showLabel is true', () => {
    render(
      <ThemeProvider>
        <ThemeToggle showLabel />
      </ThemeProvider>
    )

    expect(screen.getByText('Dark Mode')).toBeInTheDocument()
  })

  it('should apply custom className', () => {
    render(
      <ThemeProvider>
        <ThemeToggle className="custom-class" />
      </ThemeProvider>
    )

    expect(screen.getByRole('button')).toHaveClass('custom-class')
  })
})
