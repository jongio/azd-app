/**
 * Import Modal Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ImportModal } from './ImportModal'

describe('ImportModal', () => {
  const mockOnClose = vi.fn()
  const mockOnImport = vi.fn()
  const mockCurrentConfig = {
    name: 'test-app',
    services: {
      api: { host: 'containerapp' },
    },
  }

  it('should render when open', () => {
    render(
      <ImportModal
        isOpen={true}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    expect(screen.getByText('Import Configuration')).toBeInTheDocument()
  })

  it('should not render when closed', () => {
    render(
      <ImportModal
        isOpen={false}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    expect(screen.queryByText('Import Configuration')).not.toBeInTheDocument()
  })

  it('should show three tabs', () => {
    render(
      <ImportModal
        isOpen={true}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    expect(screen.getByText('Templates')).toBeInTheDocument()
    expect(screen.getByText('File Upload')).toBeInTheDocument()
    expect(screen.getByText('Paste YAML')).toBeInTheDocument()
  })

  it('should call onClose when cancel is clicked', () => {
    render(
      <ImportModal
        isOpen={true}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    const cancelButton = screen.getByText('Cancel')
    fireEvent.click(cancelButton)

    expect(mockOnClose).toHaveBeenCalled()
  })

  it('should disable preview button when no YAML imported', () => {
    render(
      <ImportModal
        isOpen={true}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    const previewButton = screen.getByText('Preview')
    expect(previewButton).toBeDisabled()
  })

  it('should show merge strategy selector when YAML is imported', async () => {
    render(
      <ImportModal
        isOpen={true}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    // Switch to paste tab
    const pasteTab = screen.getByText('Paste YAML')
    fireEvent.click(pasteTab)

    // Paste valid YAML
    const textarea = screen.getByPlaceholderText(/name: my-app/i)
    fireEvent.change(textarea, {
      target: { value: 'name: imported-app\nservices:\n  web:\n    host: containerapp' },
    })

    await waitFor(() => {
      expect(screen.getByText('Merge Strategy')).toBeInTheDocument()
    })
  })

  it('should show parse error for invalid YAML', async () => {
    render(
      <ImportModal
        isOpen={true}
        onClose={mockOnClose}
        onImport={mockOnImport}
        currentConfig={mockCurrentConfig}
      />
    )

    // Switch to paste tab
    const pasteTab = screen.getByText('Paste YAML')
    fireEvent.click(pasteTab)

    // Paste invalid YAML
    const textarea = screen.getByPlaceholderText(/name: my-app/i)
    fireEvent.change(textarea, {
      target: { value: 'invalid: yaml: content:' },
    })

    await waitFor(() => {
      expect(screen.getByText('Parse Error')).toBeInTheDocument()
    })
  })
})
