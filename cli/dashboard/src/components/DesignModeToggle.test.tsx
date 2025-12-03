import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DesignModeProvider } from '@/contexts/DesignModeContext'
import { DesignModeToggle, ModernDesignModeToggle } from './DesignModeToggle'

// Store original values
let originalLocation: Location
let originalHistoryReplaceState: typeof window.history.replaceState

// Common setup for all tests
function setupMocks() {
  localStorage.clear()
  vi.clearAllMocks()

  // Store original location
  originalLocation = window.location

  // Mock window.location
  Object.defineProperty(window, 'location', {
    writable: true,
    value: {
      ...originalLocation,
      search: '',
      href: 'http://localhost/',
    },
  })

  // Store and mock history.replaceState
  originalHistoryReplaceState = window.history.replaceState.bind(window.history)
  window.history.replaceState = vi.fn()
}

function teardownMocks() {
  localStorage.clear()
  Object.defineProperty(window, 'location', {
    writable: true,
    value: originalLocation,
  })
  window.history.replaceState = originalHistoryReplaceState
  vi.restoreAllMocks()
}

describe('DesignModeToggle', () => {
  beforeEach(setupMocks)
  afterEach(teardownMocks)

  describe('icon variant (default)', () => {
    it('renders with icon variant by default', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      expect(button).toBeInTheDocument()
      expect(button).toHaveAttribute('aria-label', 'Switch to Classic design')
    })

    it('displays correct tooltip based on current mode', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      
      // Default is modern, so should offer to switch to classic
      expect(button).toHaveAttribute('title', 'Switch to Classic design')

      // Click to toggle
      await user.click(button)

      // Now should offer to switch back to modern
      expect(button).toHaveAttribute('title', 'Switch to Modern design')
    })

    it('toggles design mode on click', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      
      // Initially in modern mode
      expect(button).toHaveAttribute('aria-label', 'Switch to Classic design')

      // Click to toggle to classic
      await user.click(button)
      expect(button).toHaveAttribute('aria-label', 'Switch to Modern design')

      // Click to toggle back to modern
      await user.click(button)
      expect(button).toHaveAttribute('aria-label', 'Switch to Classic design')
    })

    it('updates URL when mode changes', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      await user.click(button)

      expect(window.history.replaceState).toHaveBeenCalled()
    })

    it('announces mode change to screen readers', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      await user.click(button)

      const status = screen.getByRole('status')
      expect(status).toHaveTextContent('Classic design mode enabled')
    })

    it('applies custom className', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle className="custom-class" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      expect(button).toHaveClass('custom-class')
    })

    it('shows correct icon for modern mode', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      // Should show Sparkles icon for modern mode
      const button = screen.getByRole('button')
      const svg = button.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })

    it('shows correct icon for classic mode', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      await user.click(button)

      // Should show Monitor icon for classic mode
      const svg = button.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })
  })

  describe('dropdown variant', () => {
    it('renders with dropdown variant', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      expect(button).toBeInTheDocument()
      expect(button).toHaveAttribute('aria-haspopup', 'listbox')
    })

    it('opens dropdown on click', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      expect(button).toHaveAttribute('aria-expanded', 'false')

      await user.click(button)

      expect(button).toHaveAttribute('aria-expanded', 'true')
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    })

    it('displays both mode options in dropdown', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      await user.click(button)

      expect(screen.getByRole('option', { name: /Classic/i })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: /Modern/i })).toBeInTheDocument()
    })

    it('shows current mode as selected', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      await user.click(button)

      const classicOption = screen.getByRole('option', { name: /Classic/i })
      const modernOption = screen.getByRole('option', { name: /Modern/i })

      expect(classicOption).toHaveAttribute('aria-selected', 'false')
      expect(modernOption).toHaveAttribute('aria-selected', 'true')
    })

    it('changes mode when option is clicked', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      await user.click(button)

      const classicOption = screen.getByRole('option', { name: /Classic/i })
      await user.click(classicOption)

      // Dropdown should close
      expect(screen.queryByRole('listbox')).not.toBeInTheDocument()

      // Open dropdown again to check selection
      await user.click(button)
      expect(screen.getByRole('option', { name: /Classic/i })).toHaveAttribute('aria-selected', 'true')
    })

    it('closes dropdown on Escape key', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      await user.click(button)

      expect(screen.getByRole('listbox')).toBeInTheDocument()

      await user.keyboard('{Escape}')

      expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    })

    it('closes dropdown when clicking outside', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <div data-testid="outside">Outside</div>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      await user.click(button)

      expect(screen.getByRole('listbox')).toBeInTheDocument()

      await user.click(screen.getByTestId('outside'))

      await waitFor(() => {
        expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
      })
    })

    it('announces mode change in dropdown variant', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      await user.click(button)

      const classicOption = screen.getByRole('option', { name: /Classic/i })
      await user.click(classicOption)

      const status = screen.getByRole('status')
      expect(status).toHaveTextContent('Classic design mode enabled')
    })

    it('displays current mode name in button', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle variant="dropdown" />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button', { name: 'Design mode' })
      expect(button).toHaveTextContent('modern')
    })
  })

  describe('accessibility', () => {
    it('has keyboard accessible button', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      button.focus()
      expect(button).toHaveFocus()
    })

    it('has focus-visible styles', () => {
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      expect(button).toHaveClass('focus-visible:outline-none')
      expect(button).toHaveClass('focus-visible:ring-2')
    })

    it('provides screen reader announcements via live region', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const status = screen.getByRole('status')
      expect(status).toHaveAttribute('aria-live', 'polite')
      expect(status).toHaveClass('sr-only')

      const button = screen.getByRole('button')
      await user.click(button)

      expect(status).toHaveTextContent('Classic design mode enabled')
    })
  })

  describe('keyboard interaction', () => {
    it('responds to click interaction', async () => {
      const user = userEvent.setup()
      render(
        <DesignModeProvider>
          <DesignModeToggle />
        </DesignModeProvider>
      )

      const button = screen.getByRole('button')
      await user.click(button)

      // Verify it toggled to classic
      expect(button).toHaveAttribute('aria-label', 'Switch to Modern design')
    })
  })
})

describe('ModernDesignModeToggle', () => {
  beforeEach(setupMocks)
  afterEach(teardownMocks)

  it('renders correctly', () => {
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    expect(button).toBeInTheDocument()
  })

  it('has modern styling', () => {
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    expect(button).toHaveClass('rounded-lg')
  })

  it('toggles design mode on click', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    
    // Initially in modern mode
    expect(button).toHaveAttribute('aria-label', 'Switch to Classic design')

    await user.click(button)

    expect(button).toHaveAttribute('aria-label', 'Switch to Modern design')
  })

  it('updates URL when mode changes', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    await user.click(button)

    expect(window.history.replaceState).toHaveBeenCalled()
  })

  it('announces mode change to screen readers', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    await user.click(button)

    const status = screen.getByRole('status')
    expect(status).toHaveTextContent('Classic design mode enabled')
  })

  it('applies custom className', () => {
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle className="custom-class" />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    expect(button).toHaveClass('custom-class')
  })

  it('has proper ARIA attributes', () => {
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('aria-label')
    expect(button).toHaveAttribute('title')
  })

  it('can toggle back to modern mode', async () => {
    const user = userEvent.setup()
    render(
      <DesignModeProvider>
        <ModernDesignModeToggle />
      </DesignModeProvider>
    )

    const button = screen.getByRole('button')
    
    // Toggle to classic
    await user.click(button)
    expect(button).toHaveAttribute('aria-label', 'Switch to Modern design')

    // Toggle back to modern
    await user.click(button)
    expect(button).toHaveAttribute('aria-label', 'Switch to Classic design')
  })
})
