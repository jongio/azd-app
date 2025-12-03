import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DesignModeProvider } from '@/contexts/DesignModeContext'
import { useDesignMode, isValidDesignMode } from './useDesignMode'

// Test component to access hook
function TestComponent() {
  const { designMode, setDesignMode, isClassic, isModern, toggleDesignMode } = useDesignMode()
  return (
    <div>
      <div data-testid="mode">{designMode}</div>
      <div data-testid="is-classic">{isClassic ? 'true' : 'false'}</div>
      <div data-testid="is-modern">{isModern ? 'true' : 'false'}</div>
      <button onClick={() => setDesignMode('modern')}>Set Modern</button>
      <button onClick={() => setDesignMode('classic')}>Set Classic</button>
      <button onClick={toggleDesignMode}>Toggle</button>
    </div>
  )
}

// Store original location
let originalLocation: Location

describe('useDesignMode', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()

    originalLocation = window.location
    Object.defineProperty(window, 'location', {
      writable: true,
      value: {
        ...originalLocation,
        search: '',
        href: 'http://localhost/',
      },
    })
  })

  afterEach(() => {
    localStorage.clear()
    Object.defineProperty(window, 'location', {
      writable: true,
      value: originalLocation,
    })
  })

  it('returns current design mode', () => {
    render(
      <DesignModeProvider>
        <TestComponent />
      </DesignModeProvider>
    )

    expect(screen.getByTestId('mode')).toHaveTextContent('modern')
  })

  it('returns isClassic and isModern flags', () => {
    render(
      <DesignModeProvider>
        <TestComponent />
      </DesignModeProvider>
    )

    expect(screen.getByTestId('is-classic')).toHaveTextContent('false')
    expect(screen.getByTestId('is-modern')).toHaveTextContent('true')
  })

  it('provides setDesignMode function', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <TestComponent />
      </DesignModeProvider>
    )

    await user.click(screen.getByText('Set Modern'))
    expect(screen.getByTestId('mode')).toHaveTextContent('modern')

    await user.click(screen.getByText('Set Classic'))
    expect(screen.getByTestId('mode')).toHaveTextContent('classic')
  })

  it('provides toggleDesignMode function', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <TestComponent />
      </DesignModeProvider>
    )

    // Initial: modern
    expect(screen.getByTestId('mode')).toHaveTextContent('modern')

    // Toggle to classic
    await user.click(screen.getByText('Toggle'))
    expect(screen.getByTestId('mode')).toHaveTextContent('classic')

    // Toggle back to modern
    await user.click(screen.getByText('Toggle'))
    expect(screen.getByTestId('mode')).toHaveTextContent('modern')
  })

  it('updates flags when mode changes', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <TestComponent />
      </DesignModeProvider>
    )

    // Initial: modern
    expect(screen.getByTestId('is-classic')).toHaveTextContent('false')
    expect(screen.getByTestId('is-modern')).toHaveTextContent('true')

    // Switch to classic
    await user.click(screen.getByText('Toggle'))

    expect(screen.getByTestId('is-classic')).toHaveTextContent('true')
    expect(screen.getByTestId('is-modern')).toHaveTextContent('false')
  })
})

describe('isValidDesignMode', () => {
  it('returns true for "classic"', () => {
    expect(isValidDesignMode('classic')).toBe(true)
  })

  it('returns true for "modern"', () => {
    expect(isValidDesignMode('modern')).toBe(true)
  })

  it('returns false for invalid string', () => {
    expect(isValidDesignMode('invalid')).toBe(false)
  })

  it('returns false for null', () => {
    expect(isValidDesignMode(null)).toBe(false)
  })

  it('returns false for undefined', () => {
    expect(isValidDesignMode(undefined)).toBe(false)
  })

  it('returns false for number', () => {
    expect(isValidDesignMode(123)).toBe(false)
  })

  it('returns false for object', () => {
    expect(isValidDesignMode({})).toBe(false)
  })

  it('returns false for array', () => {
    expect(isValidDesignMode(['classic'])).toBe(false)
  })
})
