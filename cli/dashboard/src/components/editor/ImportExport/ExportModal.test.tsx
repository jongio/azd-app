/**
 * Export Modal Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ExportModal } from './ExportModal'

describe('ExportModal', () => {
  const mockOnClose = vi.fn()
  const mockConfig = {
    name: 'test-app',
    services: {
      api: {
        host: 'containerapp',
        project: './src/api',
      },
    },
  }

  it('should render when open', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    expect(screen.getByText('Export Configuration')).toBeInTheDocument()
  })

  it('should not render when closed', () => {
    render(
      <ExportModal
        isOpen={false}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    expect(screen.queryByText('Export Configuration')).not.toBeInTheDocument()
  })

  it('should show three format options', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    expect(screen.getByText('YAML')).toBeInTheDocument()
    expect(screen.getByText('JSON')).toBeInTheDocument()
    expect(screen.getByText('Template')).toBeInTheDocument()
  })

  it('should show preview of exported content', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    expect(screen.getByText('Preview')).toBeInTheDocument()
    const preview = screen.getByText(/name: test-app/i)
    expect(preview).toBeInTheDocument()
  })

  it('should show include comments option for YAML format', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    expect(screen.getByText('Include comments')).toBeInTheDocument()
  })

  it('should show minify option for JSON format', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    // Switch to JSON format
    const jsonButton = screen.getByText('JSON')
    fireEvent.click(jsonButton)

    expect(screen.getByText('Minify')).toBeInTheDocument()
  })

  it('should show security warning when include secrets is checked', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={{
          ...mockConfig,
          services: {
            api: {
              host: 'containerapp',
              environment: {
                API_KEY: 'secret-value',
              },
            },
          },
        }}
      />
    )

    // Find the checkbox input for "Include secrets"
    const includeSecretsLabel = screen.getByText('Include secrets').closest('label')
    const includeSecretsCheckbox = includeSecretsLabel?.querySelector('input[type="checkbox"]')
    
    expect(includeSecretsCheckbox).toBeInTheDocument()
    fireEvent.click(includeSecretsCheckbox!)

    expect(screen.getByText('Security Warning')).toBeInTheDocument()
  })

  it('should have download button', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    const downloadButton = screen.getByText('Download')
    expect(downloadButton).toBeInTheDocument()
  })

  it('should have copy button', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    const copyButton = screen.getByText('Copy')
    expect(copyButton).toBeInTheDocument()
  })

  it('should call onClose when close button is clicked', () => {
    render(
      <ExportModal
        isOpen={true}
        onClose={mockOnClose}
        config={mockConfig}
      />
    )

    const closeButton = screen.getByText('Close')
    fireEvent.click(closeButton)

    expect(mockOnClose).toHaveBeenCalled()
  })
})
