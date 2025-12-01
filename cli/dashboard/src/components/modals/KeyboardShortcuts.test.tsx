import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { KeyboardShortcuts } from './KeyboardShortcuts'

describe('KeyboardShortcuts', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('rendering', () => {
    it('should render when open', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })

    it('should not render when closed', () => {
      render(<KeyboardShortcuts {...defaultProps} isOpen={false} />)
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('should have accessible title', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument()
    })

    it('should have aria-modal attribute', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true')
    })

    it('should have aria-labelledby pointing to title', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      const dialog = screen.getByRole('dialog')
      expect(dialog).toHaveAttribute('aria-labelledby', 'shortcuts-title')
      expect(document.getElementById('shortcuts-title')).toHaveTextContent('Keyboard Shortcuts')
    })
  })

  describe('shortcut categories', () => {
    it('should display Navigation category', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Navigation')).toBeInTheDocument()
    })

    it('should display Actions category', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Actions')).toBeInTheDocument()
    })

    it('should display Views category', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Views')).toBeInTheDocument()
    })
  })

  describe('navigation shortcuts', () => {
    it('should display Resources view shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Resources view')).toBeInTheDocument()
      expect(screen.getByText('1')).toBeInTheDocument()
    })

    it('should display Console view shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Console view')).toBeInTheDocument()
      expect(screen.getByText('2')).toBeInTheDocument()
    })

    it('should display Metrics view shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Metrics view')).toBeInTheDocument()
      expect(screen.getByText('3')).toBeInTheDocument()
    })

    it('should display Environment view shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Environment view')).toBeInTheDocument()
      expect(screen.getByText('4')).toBeInTheDocument()
    })

    it('should display Dependencies view shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Dependencies view')).toBeInTheDocument()
      expect(screen.getByText('5')).toBeInTheDocument()
    })
  })

  describe('action shortcuts', () => {
    it('should display Refresh shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Refresh all services')).toBeInTheDocument()
      expect(screen.getByText('R')).toBeInTheDocument()
    })

    it('should display Clear console shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Clear console logs')).toBeInTheDocument()
      expect(screen.getByText('C')).toBeInTheDocument()
    })

    it('should display Export logs shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Export logs')).toBeInTheDocument()
      expect(screen.getByText('E')).toBeInTheDocument()
    })

    it('should display Focus search shortcut with multiple keys', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Focus search input')).toBeInTheDocument()
      expect(screen.getByText('/')).toBeInTheDocument()
      // Should show "or" between alternatives
      expect(screen.getByText('or')).toBeInTheDocument()
    })
  })

  describe('view shortcuts', () => {
    it('should display Toggle view shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Toggle table/grid view')).toBeInTheDocument()
      expect(screen.getByText('T')).toBeInTheDocument()
    })

    it('should display Show shortcuts shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Show keyboard shortcuts')).toBeInTheDocument()
      // Multiple ? symbols (in content and footer)
      expect(screen.getAllByText('?').length).toBeGreaterThan(0)
    })

    it('should display Close dialogs shortcut', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('Close dialogs/modals')).toBeInTheDocument()
      expect(screen.getByText('Esc')).toBeInTheDocument()
    })
  })

  describe('close button', () => {
    it('should render close button with aria-label', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      const closeButton = screen.getByRole('button', { name: 'Close' })
      expect(closeButton).toBeInTheDocument()
    })

    it('should call onClose when close button clicked', async () => {
      const user = userEvent.setup()
      const onClose = vi.fn()
      render(<KeyboardShortcuts {...defaultProps} onClose={onClose} />)
      
      await user.click(screen.getByRole('button', { name: 'Close' }))
      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('backdrop', () => {
    it('should render backdrop', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByTestId('shortcuts-backdrop')).toBeInTheDocument()
    })

    it('should call onClose when backdrop clicked', async () => {
      const user = userEvent.setup()
      const onClose = vi.fn()
      render(<KeyboardShortcuts {...defaultProps} onClose={onClose} />)
      
      await user.click(screen.getByTestId('shortcuts-backdrop'))
      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('escape key', () => {
    it('should call onClose when Escape is pressed', () => {
      const onClose = vi.fn()
      render(<KeyboardShortcuts {...defaultProps} onClose={onClose} />)
      
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('should not call onClose when closed and Escape is pressed', () => {
      const onClose = vi.fn()
      render(<KeyboardShortcuts {...defaultProps} isOpen={false} onClose={onClose} />)
      
      fireEvent.keyDown(document, { key: 'Escape' })
      expect(onClose).not.toHaveBeenCalled()
    })
  })

  describe('footer', () => {
    it('should display help text', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText(/anytime to show this dialog/)).toBeInTheDocument()
    })
  })

  describe('key badges', () => {
    it('should render key badges with kbd element', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      const kbdElements = document.querySelectorAll('kbd')
      expect(kbdElements.length).toBeGreaterThan(0)
    })

    it('should use monospace font for key badges', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      const kbdElements = document.querySelectorAll('kbd')
      expect(kbdElements[0]).toHaveClass('font-mono')
    })
  })

  describe('focus management', () => {
    it('should focus dialog when opened', async () => {
      const { rerender } = render(<KeyboardShortcuts {...defaultProps} isOpen={false} />)
      
      rerender(<KeyboardShortcuts {...defaultProps} isOpen={true} />)
      
      await waitFor(() => {
        const dialog = screen.getByRole('dialog')
        expect(dialog).toHaveFocus()
      })
    })

    it('should have tabIndex on dialog', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      const dialog = screen.getByRole('dialog')
      expect(dialog).toHaveAttribute('tabIndex', '-1')
    })
  })

  describe('tab trapping', () => {
    it('should trap focus within modal on Tab', async () => {
      const user = userEvent.setup()
      render(<KeyboardShortcuts {...defaultProps} />)
      
      const closeButton = screen.getByRole('button', { name: 'Close' })
      closeButton.focus()
      
      // Tab from close button should stay in modal
      await user.tab()
      
      // Focus should still be within the modal
      expect(screen.getByRole('dialog').contains(document.activeElement)).toBe(true)
    })
  })

  describe('multiple key alternatives', () => {
    it('should show "or" between alternative keys', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('or')).toBeInTheDocument()
    })

    it('should show both alternative keys for search', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      expect(screen.getByText('/')).toBeInTheDocument()
      // Ctrl+F or ⌘F depending on platform
      const ctrlF = screen.queryByText('Ctrl+F') || screen.queryByText('⌘F')
      expect(ctrlF).toBeInTheDocument()
    })
  })

  describe('accessibility', () => {
    it('should have keyboard icon with aria-hidden', () => {
      render(<KeyboardShortcuts {...defaultProps} />)
      // Icon should be hidden from screen readers (decorative)
      const icon = document.querySelector('[aria-hidden="true"]')
      expect(icon).toBeInTheDocument()
    })
  })
})
