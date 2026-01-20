/**
 * View Backup Modal Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ViewBackupModal } from './ViewBackupModal'

describe('ViewBackupModal', () => {
  const mockOnClose = vi.fn()
  const mockTimestamp = '2026-01-11T14:30:00Z'
  const mockContent = 'name: my-app\nservices:\n  api:\n    host: containerapp\n    language: node'

  beforeEach(() => {
    vi.clearAllMocks()
    // Mock clipboard
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
      writable: true,
      configurable: true,
    })
    // Mock URL APIs
    global.URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    global.URL.revokeObjectURL = vi.fn()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders when isOpen is true', () => {
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('View Backup')).toBeInTheDocument()
  })

  it('does not render when isOpen is false', () => {
    render(
      <ViewBackupModal
        isOpen={false}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('displays formatted timestamp in description', () => {
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    expect(screen.getByText(/Backup from Jan 11, 2026/)).toBeInTheDocument()
  })

  it('displays backup content', () => {
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    expect(screen.getByText(/name: my-app/)).toBeInTheDocument()
    expect(screen.getByText(/services:/)).toBeInTheDocument()
  })

  it('shows loading state when isLoading is true', () => {
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content=""
        isLoading={true}
      />
    )

    expect(screen.getByText('Loading backup...')).toBeInTheDocument()
  })

  it('copies content to clipboard when copy button clicked', async () => {
    const user = userEvent.setup()
    const writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: writeTextMock },
      writable: true,
      configurable: true,
    })

    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const copyButton = screen.getByRole('button', { name: /Copy/i })
    await user.click(copyButton)

    await waitFor(() => {
      expect(writeTextMock).toHaveBeenCalledWith(mockContent)
    })
  })

  it('shows "Copied!" text after successful copy', async () => {
    const user = userEvent.setup()
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const copyButton = screen.getByRole('button', { name: /Copy/i })
    await user.click(copyButton)

    await waitFor(() => {
      expect(screen.getByText('Copied!')).toBeInTheDocument()
    })
  })

  it('handles copy error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

    const user = userEvent.setup()
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
        forceCopyError={true}
      />
    )

    const copyButton = screen.getByRole('button', { name: /Copy/i })
    await user.click(copyButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to copy to clipboard')
    })

    alertSpy.mockRestore()
  })

  it('downloads backup file when download button clicked', async () => {
    const user = userEvent.setup()
    const originalAppendChild = document.body.appendChild
    const originalRemoveChild = document.body.removeChild
    const appendChildSpy = vi
      .spyOn(document.body, 'appendChild')
      .mockImplementation((node: Node) => originalAppendChild.call(document.body, node))
    const removeChildSpy = vi
      .spyOn(document.body, 'removeChild')
      .mockImplementation((node: Node) => originalRemoveChild.call(document.body, node))

    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const downloadButton = screen.getByRole('button', { name: /Download/i })
    await user.click(downloadButton)

    await waitFor(() => {
      expect(global.URL.createObjectURL).toHaveBeenCalled()
    })

    appendChildSpy.mockRestore()
    removeChildSpy.mockRestore()
  })

  it('handles download error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    
    // Mock error in createObjectURL
    vi.spyOn(global.URL, 'createObjectURL').mockImplementationOnce(() => {
      throw new Error('Download failed')
    })

    const user = userEvent.setup()
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const downloadButton = screen.getByRole('button', { name: /Download/i })
    await user.click(downloadButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to download backup')
    })

    alertSpy.mockRestore()
  })

  it('calls onClose when close button clicked', async () => {
    const user = userEvent.setup()
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const closeButton = screen.getByRole('button', { name: 'Close' })
    await user.click(closeButton)

    expect(mockOnClose).toHaveBeenCalled()
  })

  it('calls onClose when X button clicked', async () => {
    const user = userEvent.setup()
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const xButton = screen.getByLabelText('Close dialog')
    await user.click(xButton)

    expect(mockOnClose).toHaveBeenCalled()
  })

  it('disables buttons when isLoading is true', () => {
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content=""
        isLoading={true}
      />
    )

    const copyButton = screen.getByRole('button', { name: /Copy/i })
    const downloadButton = screen.getByRole('button', { name: /Download/i })

    expect(copyButton).toBeDisabled()
    expect(downloadButton).toBeDisabled()
  })

  it('renders content with correct formatting', () => {
    render(
      <ViewBackupModal
        isOpen={true}
        onClose={mockOnClose}
        timestamp={mockTimestamp}
        content={mockContent}
      />
    )

    const pre = screen.getByText(/name: my-app/).closest('pre')
    expect(pre).toHaveClass('font-mono')
    expect(pre).toHaveClass('whitespace-pre')
  })
})

