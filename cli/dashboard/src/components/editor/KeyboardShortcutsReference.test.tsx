/**
 * KeyboardShortcutsReference Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { KeyboardShortcutsReference } from './KeyboardShortcutsReference'

// Mock the shortcuts registry
const mockShortcuts = [
  {
    id: 'test-1',
    category: 'general',
    description: 'Test shortcut 1',
    keys: ['Ctrl', 'K'],
  },
  {
    id: 'test-2',
    category: 'navigation',
    description: 'Test shortcut 2',
    keys: ['Ctrl', 'N'],
  },
]

const mockRegistry = {
  getAll: vi.fn(() => mockShortcuts),
  formatShortcut: vi.fn((shortcut) => shortcut.keys.join('+')),
}

vi.mock('@/lib/accessibility', () => ({
  getShortcutsRegistry: () => mockRegistry,
}))

describe('KeyboardShortcutsReference', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not render when isOpen is false', () => {
    render(<KeyboardShortcutsReference isOpen={false} onClose={vi.fn()} />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('renders when isOpen is true', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText('Keyboard shortcuts reference')).toBeInTheDocument()
  })

  it('displays title', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument()
  })

  it('displays shortcuts grouped by category', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    
    expect(screen.getByText('general')).toBeInTheDocument()
    expect(screen.getByText('navigation')).toBeInTheDocument()
    expect(screen.getByText('Test shortcut 1')).toBeInTheDocument()
    expect(screen.getByText('Test shortcut 2')).toBeInTheDocument()
  })

  it('calls onClose when close button is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    
    render(<KeyboardShortcutsReference isOpen={true} onClose={onClose} />)
    
    const closeButton = screen.getByLabelText('Close keyboard shortcuts reference')
    await user.click(closeButton)
    
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when Escape key is pressed', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    
    render(<KeyboardShortcutsReference isOpen={true} onClose={onClose} />)
    
    await user.keyboard('{Escape}')
    
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when backdrop is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    
    render(<KeyboardShortcutsReference isOpen={true} onClose={onClose} />)
    
    const backdrop = screen.getByRole('dialog').parentElement
    if (backdrop) {
      await user.click(backdrop)
      expect(onClose).toHaveBeenCalledTimes(1)
    }
  })

  it('does not close when dialog content is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    
    render(<KeyboardShortcutsReference isOpen={true} onClose={onClose} />)
    
    const dialog = screen.getByRole('dialog')
    await user.click(dialog)
    
    expect(onClose).not.toHaveBeenCalled()
  })

  it('focuses first focusable element when opened', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    
    const closeButton = screen.getByLabelText('Close keyboard shortcuts reference')
    expect(document.activeElement).toBe(closeButton)
  })

  it('has proper ARIA attributes', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-label', 'Keyboard shortcuts reference')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
  })

  it('displays formatted keyboard shortcuts', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    
    // Should display formatted shortcuts
    expect(mockRegistry.formatShortcut).toHaveBeenCalled()
  })

  it('displays navigation help section', () => {
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    
    expect(screen.getByText('Navigation')).toBeInTheDocument()
    expect(screen.getByText('Navigate through form fields')).toBeInTheDocument()
  })

  it('handles empty shortcuts gracefully', () => {
    mockRegistry.getAll.mockReturnValueOnce([])
    
    render(<KeyboardShortcutsReference isOpen={true} onClose={vi.fn()} />)
    
    // Should still render dialog
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
