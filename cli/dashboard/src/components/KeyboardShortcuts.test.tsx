import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { KeyboardShortcuts } from './KeyboardShortcuts'

describe('KeyboardShortcuts', () => {
  const mockOnClose = vi.fn()

  it('should render nothing when not open', () => {
    const { container } = render(
      <KeyboardShortcuts isOpen={false} onClose={mockOnClose} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('should render modal when open', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument()
  })

  it('should display navigation shortcuts', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('Go to Resources view')).toBeInTheDocument()
    expect(screen.getByText('Go to Console view')).toBeInTheDocument()
    expect(screen.getByText('Go to Metrics view')).toBeInTheDocument()
  })

  it('should display action shortcuts', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('Refresh all services')).toBeInTheDocument()
    expect(screen.getByText('Clear console logs')).toBeInTheDocument()
    expect(screen.getByText('Export logs')).toBeInTheDocument()
  })

  it('should display view shortcuts', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('Toggle table/grid view')).toBeInTheDocument()
    expect(screen.getByText('Show keyboard shortcuts')).toBeInTheDocument()
  })

  it('should close when clicking X button', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    const closeButton = screen.getByRole('button')
    fireEvent.click(closeButton)
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it('should close when clicking backdrop', () => {
    const { container } = render(
      <KeyboardShortcuts isOpen={true} onClose={mockOnClose} />
    )
    const backdrop = container.querySelector('.fixed.inset-0')
    if (backdrop) {
      fireEvent.click(backdrop)
      expect(mockOnClose).toHaveBeenCalled()
    }
  })

  it('should close on Escape key', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(mockOnClose).toHaveBeenCalled()
  })

  it('should display shortcut categories', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    expect(screen.getAllByText(/Navigation/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Actions/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Views/).length).toBeGreaterThan(0)
  })

  it('should render keyboard key badges', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    // Check for some key badges
    const badges = screen.getAllByText('1')
    expect(badges.length).toBeGreaterThan(0)
  })

  it('should display compound shortcuts correctly', () => {
    render(<KeyboardShortcuts isOpen={true} onClose={mockOnClose} />)
    // Ctrl+F shortcut should be displayed
    expect(screen.getAllByText('Ctrl').length).toBeGreaterThan(0)
    expect(screen.getByText('Focus search')).toBeInTheDocument()
  })
})
