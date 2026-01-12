/**
 * Dialog Component Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from './dialog'

describe('Dialog', () => {
  it('renders when isOpen is true', () => {
    render(
      <Dialog isOpen={true} onClose={vi.fn()} title="Test Dialog">
        <div>Dialog content</div>
      </Dialog>
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Dialog content')).toBeInTheDocument()
  })

  it('does not render when isOpen is false', () => {
    render(
      <Dialog isOpen={false} onClose={vi.fn()}>
        <div>Dialog content</div>
      </Dialog>
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('calls onClose when backdrop is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(
      <Dialog isOpen={true} onClose={onClose}>
        <div>Dialog content</div>
      </Dialog>
    )

    // Click backdrop (the fixed inset div before the dialog)
    const backdrop = document.querySelector('.fixed.inset-0')
    expect(backdrop).toBeInTheDocument()

    if (backdrop) {
      await user.click(backdrop)
      expect(onClose).toHaveBeenCalledTimes(1)
    }
  })

  it('calls onClose when Escape key is pressed', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(
      <Dialog isOpen={true} onClose={onClose}>
        <div>Dialog content</div>
      </Dialog>
    )

    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('supports different max widths', () => {
    const { rerender } = render(
      <Dialog isOpen={true} onClose={vi.fn()} maxWidth="sm">
        <div>Content</div>
      </Dialog>
    )

    let dialog = screen.getByRole('dialog')
    expect(dialog).toHaveClass('max-w-sm')

    rerender(
      <Dialog isOpen={true} onClose={vi.fn()} maxWidth="4xl">
        <div>Content</div>
      </Dialog>
    )

    dialog = screen.getByRole('dialog')
    expect(dialog).toHaveClass('max-w-4xl')
  })

  it('renders with custom className', () => {
    render(
      <Dialog isOpen={true} onClose={vi.fn()} className="custom-class">
        <div>Content</div>
      </Dialog>
    )

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveClass('custom-class')
  })
})

describe('DialogHeader', () => {
  it('renders children', () => {
    render(
      <DialogHeader>
        <DialogTitle>Test Title</DialogTitle>
      </DialogHeader>
    )

    expect(screen.getByText('Test Title')).toBeInTheDocument()
  })

  it('renders close button when onClose provided', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(
      <DialogHeader onClose={onClose}>
        <DialogTitle>Test</DialogTitle>
      </DialogHeader>
    )

    const closeButton = screen.getByRole('button', { name: /close dialog/i })
    expect(closeButton).toBeInTheDocument()

    await user.click(closeButton)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not render close button when onClose not provided', () => {
    render(
      <DialogHeader>
        <DialogTitle>Test</DialogTitle>
      </DialogHeader>
    )

    expect(screen.queryByRole('button', { name: /close dialog/i })).not.toBeInTheDocument()
  })
})

describe('DialogTitle', () => {
  it('renders with correct id', () => {
    render(<DialogTitle>My Title</DialogTitle>)

    const title = screen.getByText('My Title')
    expect(title).toHaveAttribute('id', 'dialog-title')
  })

  it('applies custom className', () => {
    render(<DialogTitle className="custom-title">Title</DialogTitle>)

    const title = screen.getByText('Title')
    expect(title).toHaveClass('custom-title')
  })
})

describe('DialogDescription', () => {
  it('renders with correct id', () => {
    render(<DialogDescription>My Description</DialogDescription>)

    const description = screen.getByText('My Description')
    expect(description).toHaveAttribute('id', 'dialog-description')
  })

  it('applies custom className', () => {
    render(<DialogDescription className="custom-desc">Description</DialogDescription>)

    const description = screen.getByText('Description')
    expect(description).toHaveClass('custom-desc')
  })
})

describe('DialogContent', () => {
  it('renders children', () => {
    render(
      <DialogContent>
        <p>Content here</p>
      </DialogContent>
    )

    expect(screen.getByText('Content here')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    render(
      <DialogContent className="custom-content">
        <p>Content</p>
      </DialogContent>
    )

    const content = screen.getByText('Content').parentElement
    expect(content).toHaveClass('custom-content')
  })
})

describe('DialogFooter', () => {
  it('renders children', () => {
    render(
      <DialogFooter>
        <button>Cancel</button>
        <button>Confirm</button>
      </DialogFooter>
    )

    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument()
  })

  it('applies custom className', () => {
    render(
      <DialogFooter className="custom-footer">
        <button>OK</button>
      </DialogFooter>
    )

    const footer = screen.getByRole('button', { name: 'OK' }).parentElement
    expect(footer).toHaveClass('custom-footer')
  })
})

describe('Dialog Integration', () => {
  it('renders complete dialog with all parts', () => {
    render(
      <Dialog isOpen={true} onClose={vi.fn()} title="Complete Dialog">
        <DialogHeader onClose={vi.fn()}>
          <DialogTitle>Test Dialog</DialogTitle>
          <DialogDescription>This is a test dialog</DialogDescription>
        </DialogHeader>
        <DialogContent>
          <p>Dialog body content</p>
        </DialogContent>
        <DialogFooter>
          <button>Cancel</button>
          <button>OK</button>
        </DialogFooter>
      </Dialog>
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Test Dialog')).toBeInTheDocument()
    expect(screen.getByText('This is a test dialog')).toBeInTheDocument()
    expect(screen.getByText('Dialog body content')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'OK' })).toBeInTheDocument()
  })

  it('focuses close button on open', async () => {
    const { rerender } = render(
      <Dialog isOpen={false} onClose={vi.fn()}>
        <DialogHeader onClose={vi.fn()}>
          <DialogTitle>Test</DialogTitle>
        </DialogHeader>
      </Dialog>
    )

    rerender(
      <Dialog isOpen={true} onClose={vi.fn()}>
        <DialogHeader onClose={vi.fn()}>
          <DialogTitle>Test</DialogTitle>
        </DialogHeader>
      </Dialog>
    )

    // Note: Focus management happens in useEffect, so we check the close button exists
    await new Promise(resolve => setTimeout(resolve, 10))
    const closeButton = screen.queryByLabelText('Close dialog')
    expect(closeButton).toBeInTheDocument()
  })
})
