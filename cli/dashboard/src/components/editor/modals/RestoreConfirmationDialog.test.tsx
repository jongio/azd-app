/**
 * Restore Confirmation Dialog Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RestoreConfirmationDialog } from './RestoreConfirmationDialog'

describe('RestoreConfirmationDialog', () => {
  const mockOnClose = vi.fn()
  const mockOnConfirm = vi.fn()
  const mockTimestamp = '2026-01-11T14:30:00Z'

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders when isOpen is true', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getAllByText('Restore Backup')).toHaveLength(2) // Title and button
    expect(screen.getByText('Confirm backup restoration')).toBeInTheDocument()
  })

  it('does not render when isOpen is false', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={false}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('displays formatted timestamp in heading', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.getByText(/Restore backup from Jan 11, 2026/)).toBeInTheDocument()
  })

  it('displays warning about current config backup', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.getByText(/Your current configuration will be backed up first/)).toBeInTheDocument()
  })

  it('calls onConfirm and onClose when confirm button clicked', async () => {
    const user = userEvent.setup()
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const confirmButton = screen.getByRole('button', { name: 'Restore Backup' })
    await user.click(confirmButton)

    await waitFor(() => {
      expect(mockOnConfirm).toHaveBeenCalled()
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('calls onClose when cancel button clicked', async () => {
    const user = userEvent.setup()
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    await user.click(cancelButton)

    expect(mockOnClose).toHaveBeenCalled()
    expect(mockOnConfirm).not.toHaveBeenCalled()
  })

  it('calls onClose when X button clicked', async () => {
    const user = userEvent.setup()
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const xButton = screen.getByLabelText('Close dialog')
    await user.click(xButton)

    expect(mockOnClose).toHaveBeenCalled()
    expect(mockOnConfirm).not.toHaveBeenCalled()
  })

  it('disables buttons when isRestoring is true', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
        isRestoring={true}
      />
    )

    const confirmButton = screen.getByRole('button', { name: 'Restoring...' })
    const cancelButton = screen.getByRole('button', { name: 'Cancel' })

    expect(confirmButton).toBeDisabled()
    expect(cancelButton).toBeDisabled()
  })

  it('shows "Restoring..." text when isRestoring is true', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
        isRestoring={true}
      />
    )

    expect(screen.getByText('Restoring...')).toBeInTheDocument()
  })

  it('handles async onConfirm function', async () => {
    const asyncConfirm = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()

    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={asyncConfirm}
        timestamp={mockTimestamp}
      />
    )

    const confirmButton = screen.getByRole('button', { name: 'Restore Backup' })
    await user.click(confirmButton)

    await waitFor(() => {
      expect(asyncConfirm).toHaveBeenCalled()
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('handles onConfirm error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const errorConfirm = vi.fn().mockRejectedValue(new Error('Restore failed'))
    const user = userEvent.setup()

    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={errorConfirm}
        timestamp={mockTimestamp}
      />
    )

    const confirmButton = screen.getByRole('button', { name: 'Restore Backup' })
    await user.click(confirmButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to restore backup. Please try again.')
      expect(mockOnClose).not.toHaveBeenCalled() // Should not close on error
    })

    alertSpy.mockRestore()
  })

  it('displays info icon with correct styling', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const iconContainer = screen.getByText(/Restore backup from/).closest('div')?.previousSibling as HTMLElement | null
    const innerDiv = iconContainer?.querySelector('div')
    expect(innerDiv).toHaveClass('bg-cyan-100', 'dark:bg-cyan-900/30')
  })

  it('applies correct button styling', () => {
    render(
      <RestoreConfirmationDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const confirmButton = screen.getByRole('button', { name: 'Restore Backup' })
    expect(confirmButton).toHaveClass('bg-cyan-600')

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    expect(cancelButton).toHaveClass('border-slate-200')
  })
})
