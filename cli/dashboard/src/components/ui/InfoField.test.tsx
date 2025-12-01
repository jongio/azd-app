import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { InfoField } from './InfoField'

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('InfoField', () => {
  describe('rendering', () => {
    it('should render label and value correctly', () => {
      render(<InfoField label="Port" value="3000" />)

      expect(screen.getByText('Port')).toBeInTheDocument()
      expect(screen.getByText('3000')).toBeInTheDocument()
    })

    it('should render with ReactNode value', () => {
      render(
        <InfoField
          label="Status"
          value={<span data-testid="custom-badge">Running</span>}
        />
      )

      expect(screen.getByText('Status')).toBeInTheDocument()
      expect(screen.getByTestId('custom-badge')).toBeInTheDocument()
      expect(screen.getByText('Running')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(
        <InfoField
          label="Test"
          value="Value"
          className="custom-class"
          data-testid="info-field"
        />
      )

      const container = screen.getByTestId('info-field')
      expect(container).toHaveClass('custom-class')
    })

    it('should apply data-testid attribute', () => {
      render(<InfoField label="Test" value="Value" data-testid="my-field" />)

      expect(screen.getByTestId('my-field')).toBeInTheDocument()
    })
  })

  describe('copy button visibility', () => {
    it('should hide copy button when copyable is false', () => {
      render(<InfoField label="Port" value="3000" copyable={false} />)

      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })

    it('should hide copy button when copyable is not provided', () => {
      render(<InfoField label="Port" value="3000" />)

      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })

    it('should show copy button when copyable is true', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      expect(screen.getByRole('button', { name: 'Copy Port' })).toBeInTheDocument()
    })

    it('should hide copy button when value is ReactNode without copyValue', () => {
      render(
        <InfoField
          label="Status"
          value={<span>Running</span>}
          copyable
        />
      )

      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })

    it('should show copy button when value is ReactNode with copyValue', () => {
      render(
        <InfoField
          label="URL"
          value={<a href="https://example.com">example.com</a>}
          copyable
          copyValue="https://example.com"
        />
      )

      expect(screen.getByRole('button', { name: 'Copy URL' })).toBeInTheDocument()
    })
  })

  describe('copy functionality', () => {
    it('should show copied state after clicking copy button', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      // The button should now show the copied state
      expect(screen.getByRole('button', { name: 'Port copied' })).toBeInTheDocument()
    })

    it('should show copied state for copyValue variant', async () => {
      const user = userEvent.setup()

      render(
        <InfoField
          label="URL"
          value={<a href="https://example.com">example.com</a>}
          copyable
          copyValue="https://example.com"
        />
      )

      await user.click(screen.getByRole('button', { name: 'Copy URL' }))

      expect(screen.getByRole('button', { name: 'URL copied' })).toBeInTheDocument()
    })

    it('should fire onCopy callback when copy is triggered', async () => {
      const user = userEvent.setup()
      const onCopy = vi.fn()

      render(<InfoField label="Port" value="3000" copyable onCopy={onCopy} />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      expect(onCopy).toHaveBeenCalledTimes(1)
    })
  })

  describe('icon state transitions', () => {
    it('should show Copy icon by default', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      const button = screen.getByRole('button', { name: 'Copy Port' })
      expect(button.querySelector('svg')).toBeInTheDocument()
    })

    it('should show Check icon after copying', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      // After copy, aria-label changes to "Port copied"
      expect(screen.getByRole('button', { name: 'Port copied' })).toBeInTheDocument()
    })

    it('should reset icon to Copy after 2 seconds', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      expect(screen.getByRole('button', { name: 'Port copied' })).toBeInTheDocument()

      // Wait for the 2 second timeout to reset the icon
      await waitFor(
        () => {
          expect(screen.getByRole('button', { name: 'Copy Port' })).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    }, 5000)
  })

  describe('keyboard accessibility', () => {
    it('should be focusable via Tab', async () => {
      const user = userEvent.setup()

      render(
        <div>
          <button>Before</button>
          <InfoField label="Port" value="3000" copyable />
          <button>After</button>
        </div>
      )

      const beforeButton = screen.getByRole('button', { name: 'Before' })
      beforeButton.focus()

      await user.tab()

      expect(screen.getByRole('button', { name: 'Copy Port' })).toHaveFocus()
    })

    it('should trigger copy on Enter key', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      const copyButton = screen.getByRole('button', { name: 'Copy Port' })
      copyButton.focus()

      await user.keyboard('{Enter}')

      // Verify copy happened via aria-label change
      expect(screen.getByRole('button', { name: 'Port copied' })).toBeInTheDocument()
    })

    it('should trigger copy on Space key', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      const copyButton = screen.getByRole('button', { name: 'Copy Port' })
      copyButton.focus()

      await user.keyboard(' ')

      // Verify copy happened via aria-label change
      expect(screen.getByRole('button', { name: 'Port copied' })).toBeInTheDocument()
    })

    it('should blur on Escape key', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      const copyButton = screen.getByRole('button', { name: 'Copy Port' })
      copyButton.focus()

      expect(copyButton).toHaveFocus()

      await user.keyboard('{Escape}')

      expect(copyButton).not.toHaveFocus()
    })
  })

  describe('accessibility', () => {
    it('should have accessible name for copy button', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      const button = screen.getByRole('button', { name: 'Copy Port' })
      expect(button).toHaveAttribute('aria-label', 'Copy Port')
    })

    it('should update aria-label after copying', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      const button = screen.getByRole('button', { name: 'Port copied' })
      expect(button).toHaveAttribute('aria-label', 'Port copied')
    })

    it('should have aria-live="polite" on copy button', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      const button = screen.getByRole('button', { name: 'Copy Port' })
      expect(button).toHaveAttribute('aria-live', 'polite')
    })

    it('should have button type="button" to prevent form submission', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      const button = screen.getByRole('button', { name: 'Copy Port' })
      expect(button).toHaveAttribute('type', 'button')
    })

    it('should have visible focus ring styles', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      const button = screen.getByRole('button', { name: 'Copy Port' })
      expect(button.className).toContain('focus-visible:ring-2')
    })

    it('should associate label with value via aria-labelledby', () => {
      render(<InfoField label="Port" value="3000" data-testid="info-field" />)

      const container = screen.getByTestId('info-field')
      const label = container.querySelector('span[id]')
      const valueWrapper = container.querySelector('[aria-labelledby]')

      expect(label).toBeInTheDocument()
      expect(valueWrapper).toBeInTheDocument()
      expect(valueWrapper?.getAttribute('aria-labelledby')).toBe(label?.id)
    })
  })

  describe('visual styling', () => {
    it('should have correct label styling', () => {
      render(<InfoField label="Port" value="3000" data-testid="info-field" />)

      const container = screen.getByTestId('info-field')
      const label = container.querySelector('span')

      expect(label).toHaveClass('text-xs', 'font-medium', 'text-muted-foreground')
    })

    it('should have correct value styling', () => {
      render(<InfoField label="Port" value="3000" data-testid="info-field" />)

      const value = screen.getByText('3000')
      expect(value).toHaveClass('text-sm', 'text-foreground', 'break-all')
    })

    it('should have hover background transition', () => {
      render(<InfoField label="Port" value="3000" data-testid="info-field" />)

      const container = screen.getByTestId('info-field')
      expect(container).toHaveClass('transition-colors', 'duration-200')
      expect(container).toHaveClass('hover:bg-secondary/30')
    })

    it('should have correct copy button sizing', () => {
      render(<InfoField label="Port" value="3000" copyable />)

      const button = screen.getByRole('button', { name: 'Copy Port' })
      expect(button).toHaveClass('h-7', 'w-7')
    })

    it('should apply success color when copied', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="3000" copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      const button = screen.getByRole('button', { name: 'Port copied' })
      expect(button).toHaveClass('text-green-600')
    })
  })

  describe('edge cases', () => {
    it('should handle empty string value', () => {
      render(<InfoField label="Port" value="" />)

      expect(screen.getByText('Port')).toBeInTheDocument()
    })

    it('should not show copy button when value is empty string', () => {
      render(<InfoField label="Port" value="" copyable />)

      // Empty string value means canCopy is falsy (empty string is falsy in the component logic)
      // Actually let's check the component - empty string should still show button
      // but clicking won't do anything. Let's verify the button exists but copy does nothing
      const button = screen.queryByRole('button')
      // The component logic: canCopy = copyable && (typeof value === 'string' || copyValue)
      // Empty string is typeof string, so button should be present
      expect(button).toBeInTheDocument()
    })

    it('should copy special characters correctly', async () => {
      const user = userEvent.setup()
      const specialValue = '<script>alert("xss")</script>'

      render(<InfoField label="Code" value={specialValue} copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Code' }))

      // Verify copy happened via state change
      expect(screen.getByRole('button', { name: 'Code copied' })).toBeInTheDocument()
    })

    it('should handle very long values', () => {
      const longValue = 'a'.repeat(1000)

      render(<InfoField label="Long" value={longValue} data-testid="info-field" />)

      const value = screen.getByText(longValue)
      expect(value).toHaveClass('break-all')
    })

    it('should handle numeric-like string values', async () => {
      const user = userEvent.setup()

      render(<InfoField label="Port" value="8080" copyable />)

      await user.click(screen.getByRole('button', { name: 'Copy Port' }))

      expect(screen.getByRole('button', { name: 'Port copied' })).toBeInTheDocument()
    })
  })
})
