/**
 * RecoveryModal Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RecoveryModal, type DraftData } from './RecoveryModal'

describe('RecoveryModal', () => {
  const currentConfig = {
    name: 'current-app',
    services: {
      api: { project: './api' },
    },
  }

  const draftData: DraftData = {
    config: {
      name: 'draft-app',
      services: {
        api: { project: './api' },
        web: { project: './web' },
      },
    },
    timestamp: Date.now() - 60000, // 1 minute ago
    dirty: true,
  }

  const defaultProps = {
    isOpen: true,
    draft: draftData,
    currentConfig,
    onRestore: vi.fn(),
    onDiscard: vi.fn(),
    onCancel: vi.fn(),
  }

  it('should not render when closed', () => {
    const { container } = render(<RecoveryModal {...defaultProps} isOpen={false} />)
    expect(container.firstChild).toBeNull()
  })

  it('should not render when draft is null', () => {
    const { container } = render(<RecoveryModal {...defaultProps} draft={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('should render when open with draft', () => {
    render(<RecoveryModal {...defaultProps} />)
    expect(screen.getByText('Unsaved Changes Detected')).toBeInTheDocument()
  })

  it('should show draft age', () => {
    render(<RecoveryModal {...defaultProps} />)
    expect(screen.getByText(/A draft was saved/)).toBeInTheDocument()
    expect(screen.getByText(/ago/)).toBeInTheDocument()
  })

  it('should show draft timestamp', () => {
    render(<RecoveryModal {...defaultProps} />)
    expect(screen.getByText(/Draft saved:/)).toBeInTheDocument()
  })

  it('should display current and draft configs', () => {
    render(<RecoveryModal {...defaultProps} />)

    expect(screen.getByText('Current Configuration')).toBeInTheDocument()
    expect(screen.getByText('Draft Configuration')).toBeInTheDocument()

    // Check for config names
    expect(screen.getByText(/"current-app"/)).toBeInTheDocument()
    expect(screen.getByText(/"draft-app"/)).toBeInTheDocument()
  })

  it('should call onRestore when Restore button clicked', async () => {
    const user = userEvent.setup()
    const onRestore = vi.fn()

    render(<RecoveryModal {...defaultProps} onRestore={onRestore} />)

    const restoreButton = screen.getByText('Restore Draft')
    await user.click(restoreButton)

    expect(onRestore).toHaveBeenCalledTimes(1)
  })

  it('should call onDiscard when Discard button clicked', async () => {
    const user = userEvent.setup()
    const onDiscard = vi.fn()

    render(<RecoveryModal {...defaultProps} onDiscard={onDiscard} />)

    const discardButton = screen.getByText('Discard Draft')
    await user.click(discardButton)

    expect(onDiscard).toHaveBeenCalledTimes(1)
  })

  it('should call onCancel when Cancel button clicked', async () => {
    const user = userEvent.setup()
    const onCancel = vi.fn()

    render(<RecoveryModal {...defaultProps} onCancel={onCancel} />)

    const cancelButton = screen.getByText('Cancel')
    await user.click(cancelButton)

    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('should call onCancel on backdrop click', async () => {
    const user = userEvent.setup()
    const onCancel = vi.fn()

    render(<RecoveryModal {...defaultProps} onCancel={onCancel} />)

    const backdrop = screen.getByRole('dialog')
    await user.click(backdrop)

    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('should call onCancel on Escape key', async () => {
    const user = userEvent.setup()
    const onCancel = vi.fn()

    render(<RecoveryModal {...defaultProps} onCancel={onCancel} />)

    const dialog = screen.getByRole('dialog')
    dialog.focus()
    await user.keyboard('{Escape}')

    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('should show information about actions', () => {
    render(<RecoveryModal {...defaultProps} />)

    expect(screen.getByText('What will happen:')).toBeInTheDocument()
    expect(screen.getByText(/Restore Draft:/)).toBeInTheDocument()
    expect(screen.getByText(/Discard Draft:/)).toBeInTheDocument()
    expect(screen.getByText(/Cancel:/)).toBeInTheDocument()
  })

  it('should truncate long configs', () => {
    const longConfig = {
      name: 'x'.repeat(1000),
      data: 'y'.repeat(1000),
    }

    const draft: DraftData = {
      config: longConfig,
      timestamp: Date.now(),
      dirty: true,
    }

    render(<RecoveryModal {...defaultProps} draft={draft} currentConfig={longConfig} />)

    // Should show truncation indicator
    expect(screen.getAllByText(/\.\.\./)).toHaveLength(2) // One for current, one for draft
  })

  it('should have correct ARIA attributes', () => {
    render(<RecoveryModal {...defaultProps} />)

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby', 'recovery-modal-title')
  })

  it('should not call onCancel on content click', async () => {
    const user = userEvent.setup()
    const onCancel = vi.fn()

    render(<RecoveryModal {...defaultProps} onCancel={onCancel} />)

    const title = screen.getByText('Unsaved Changes Detected')
    await user.click(title)

    expect(onCancel).not.toHaveBeenCalled()
  })
})
