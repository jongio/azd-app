/**
 * Connection Strings Panel Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { act } from 'react'
import { ConnectionStringsPanel } from './ConnectionStringsPanel'
import type { WellKnownService } from '@/lib/editor/wellknown-types'

const mockService: WellKnownService = {
  name: 'azurite',
  displayName: 'Azurite (Azure Storage Emulator)',
  description: 'Local Azure Storage emulator',
  category: 'storage',
  icon: '📦',
  host: 'containerapp',
  image: 'mcr.microsoft.com/azure-storage/azurite:latest',
  ports: ['10000:10000', '10001:10001', '10002:10002'],
  connectionStrings: {
    blob: 'DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;...',
    queue: 'DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;...',
    table: 'DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;...',
    default: 'UseDevelopmentStorage=true',
  },
  docsUrl: 'https://learn.microsoft.com/azure/storage/common/storage-use-azurite',
}

const mockServiceWithSingleConnection: WellKnownService = {
  name: 'redis',
  displayName: 'Redis Cache',
  description: 'In-memory data structure store',
  category: 'cache',
  icon: '🔴',
  host: 'containerapp',
  image: 'redis:7-alpine',
  ports: ['6379:6379'],
  connectionStrings: {
    default: 'redis://localhost:6379',
  },
  docsUrl: 'https://redis.io/docs/',
}

const mockServiceWithoutConnectionStrings: WellKnownService = {
  name: 'test',
  displayName: 'Test Service',
  description: 'Test service without connections',
  category: 'other',
  host: 'containerapp',
  image: 'test:latest',
  ports: ['8080:8080'],
}

describe('ConnectionStringsPanel', () => {
  // Mock clipboard API
  let mockWriteText: ReturnType<typeof vi.fn>

  beforeEach(() => {
    mockWriteText = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue(undefined)
    ;(globalThis as any).__initialWriteText = mockWriteText
  })

  afterEach(() => {
    vi.restoreAllMocks()
    delete (globalThis as any).__initialWriteText
  })

  it('renders connection strings for service', () => {
    render(<ConnectionStringsPanel service={mockService} />)

    expect(screen.getByText('Connection Strings')).toBeInTheDocument()
    expect(screen.getByText(mockService.displayName)).toBeInTheDocument()

    // Should show all connection string keys
    expect(screen.getByText(/blob Connection String/i)).toBeInTheDocument()
    expect(screen.getByText(/queue Connection String/i)).toBeInTheDocument()
    expect(screen.getByText(/table Connection String/i)).toBeInTheDocument()
    expect(screen.getByText(/^Connection String$/i)).toBeInTheDocument() // 'default' becomes 'Connection String'
  })

  it('displays connection string values', () => {
    render(<ConnectionStringsPanel service={mockService} />)

    // Use getAllByText since some values might be duplicated (they're unique in our test but good practice)
    const values = Object.values(mockService.connectionStrings!)
    values.forEach((value) => {
      const elements = screen.queryAllByText(value)
      expect(elements.length).toBeGreaterThan(0)
    })
  })

  it('copies connection string to clipboard when copy button clicked', async () => {
    const user = userEvent.setup()
    render(<ConnectionStringsPanel service={mockService} />)

    const copyButtons = screen.getAllByRole('button', { name: /Copy.*connection string/i })
    await user.click(copyButtons[0])

    expect(mockWriteText).toHaveBeenCalledTimes(1)
    expect(mockWriteText).toHaveBeenCalledWith(mockService.connectionStrings!.blob)
  })

  it('shows "Copied!" feedback after successful copy', async () => {
    const user = userEvent.setup()
    render(<ConnectionStringsPanel service={mockService} />)

    const copyButton = screen.getByLabelText(/Copy blob connection string/i)
    await user.click(copyButton)

    await waitFor(() => {
      expect(screen.getByText('Copied!')).toBeInTheDocument()
    })
  })

  it('resets "Copied!" feedback after 2 seconds', async () => {
    vi.useFakeTimers()
    render(<ConnectionStringsPanel service={mockService} />)

    const copyButton = screen.getByLabelText(/Copy blob connection string/i)
    fireEvent.click(copyButton)

    expect(screen.getByText('Copied!')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    vi.useRealTimers()

    await waitFor(() => {
      expect(screen.queryByText('Copied!')).not.toBeInTheDocument()
    })
  })

  it('handles clipboard write failure gracefully', async () => {
    mockWriteText.mockRejectedValueOnce(new Error('Clipboard error'))

    render(<ConnectionStringsPanel service={mockService} />)

    const copyButton = screen.getByLabelText(/Copy blob connection string/i)
    fireEvent.click(copyButton)

    // Just verify the function was called - error handling is silent
    await waitFor(() => {
      expect(mockWriteText).toHaveBeenCalled()
    })
  })

  it('renders documentation link when docsUrl is provided', () => {
    render(<ConnectionStringsPanel service={mockService} />)

    const docsLink = screen.getByRole('link', { name: /View.*Documentation/i })
    expect(docsLink).toBeInTheDocument()
    expect(docsLink).toHaveAttribute('href', mockService.docsUrl)
    expect(docsLink).toHaveAttribute('target', '_blank')
    expect(docsLink).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('does not render documentation link when docsUrl is not provided', () => {
    const serviceWithoutDocs = { ...mockService, docsUrl: undefined }
    render(<ConnectionStringsPanel service={serviceWithoutDocs} />)

    expect(screen.queryByRole('link', { name: /Documentation/i })).not.toBeInTheDocument()
  })

  it('renders usage hint', () => {
    render(<ConnectionStringsPanel service={mockService} />)

    expect(screen.getByText(/💡 Tip:/i)).toBeInTheDocument()
    expect(screen.getByText(/Use these connection strings/i)).toBeInTheDocument()
  })

  it('returns null when service has no connection strings', () => {
    const { container } = render(<ConnectionStringsPanel service={mockServiceWithoutConnectionStrings} />)

    expect(container.firstChild).toBeNull()
  })

  it('returns null when connectionStrings is empty object', () => {
    const serviceWithEmptyConnections = {
      ...mockService,
      connectionStrings: {},
    }
    const { container } = render(<ConnectionStringsPanel service={serviceWithEmptyConnections} />)

    expect(container.firstChild).toBeNull()
  })

  it('handles single connection string correctly', () => {
    render(<ConnectionStringsPanel service={mockServiceWithSingleConnection} />)

    // 'default' key should be rendered as 'Connection String'
    expect(screen.getByText(/^Connection String$/i)).toBeInTheDocument()
    expect(screen.getByText(mockServiceWithSingleConnection.connectionStrings!.default)).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(<ConnectionStringsPanel service={mockService} className="custom-class" />)

    const panel = container.firstChild as HTMLElement
    expect(panel).toHaveClass('custom-class')
  })

  it('capitalizes connection string labels correctly', () => {
    render(<ConnectionStringsPanel service={mockService} />)

    // 'default' should become 'Connection String'
    expect(screen.getByText(/^Connection String$/i)).toBeInTheDocument()

    // Other keys should be capitalized (e.g., 'blob' -> 'Blob Connection String')
    expect(screen.getByText(/blob Connection String/i)).toBeInTheDocument()
    expect(screen.getByText(/queue Connection String/i)).toBeInTheDocument()
  })

  it('supports copying multiple different connection strings', async () => {
    render(<ConnectionStringsPanel service={mockService} />)

    const copyButtons = screen.getAllByRole('button', { name: /Copy.*connection string/i })

    // Copy blob connection string
    fireEvent.click(copyButtons[0])
    expect(mockWriteText).toHaveBeenCalledWith(mockService.connectionStrings!.blob)

    // Copy queue connection string
    fireEvent.click(copyButtons[1])
    expect(mockWriteText).toHaveBeenCalledWith(mockService.connectionStrings!.queue)

    expect(mockWriteText).toHaveBeenCalledTimes(2)
  })

  it('displays connection strings with proper code formatting', () => {
    render(<ConnectionStringsPanel service={mockService} />)

    const codeElements = screen.getAllByText(/DefaultEndpointsProtocol|UseDevelopmentStorage/)
    codeElements.forEach((code) => {
      expect(code.tagName).toBe('CODE')
    })
  })
})
