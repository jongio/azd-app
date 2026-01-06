import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DiagnosticsModal } from './DiagnosticsModal'
import type { SetupStep as _SetupStep } from './AzureSetupGuide'

interface HealthCheck {
  name: string
  status: 'pass' | 'warn' | 'fail'
  message: string
  fix?: string
}

interface HealthCheckResponse {
  status: 'healthy' | 'degraded' | 'error'
  checks: HealthCheck[]
  docsUrl: string
  timestamp: string
}

const createMockHealthResponse = (status: 'healthy' | 'degraded' | 'error', checks: HealthCheck[]): HealthCheckResponse => ({
  status,
  checks,
  docsUrl: 'https://docs.example.com',
  timestamp: new Date().toISOString(),
})

const fetchMock = vi.fn()
globalThis.fetch = fetchMock as unknown as typeof fetch

describe('DiagnosticsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('does not render when closed', () => {
    render(<DiagnosticsModal isOpen={false} onClose={vi.fn()} />)
    expect(screen.queryByText('Azure Logs Diagnostics')).not.toBeInTheDocument()
  })

  it('renders when open and fetches health checks', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('healthy', [
        { name: 'Workspace Check', status: 'pass', message: 'Workspace configured' },
      ]),
    })

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(screen.getByText('Azure Logs Diagnostics')).toBeInTheDocument()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/azure/logs/health', expect.any(Object)))
  })

  it('shows loading state while fetching', async () => {
    fetchMock.mockImplementation(() => new Promise(() => {/* never resolves */}))

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(await screen.findByText('Running health checks...')).toBeInTheDocument()
  })

  it('displays health check results', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Workspace Check', status: 'pass', message: 'Workspace configured' },
        { name: 'Auth Check', status: 'fail', message: 'Authentication failed', fix: 'az login' },
      ]),
    })

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    await waitFor(() => {
      expect(screen.getByText('Workspace Check')).toBeInTheDocument()
      expect(screen.getByText('Auth Check')).toBeInTheDocument()
      expect(screen.getByText('Authentication failed')).toBeInTheDocument()
    })
  })

  it('shows error state when fetch fails', async () => {
    fetchMock.mockRejectedValueOnce(new Error('Network error'))

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(await screen.findByText('Failed to fetch diagnostics')).toBeInTheDocument()
    expect(screen.getByText('Network error')).toBeInTheDocument()
  })

  it('calls onClose when close button clicked', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('healthy', []),
    })

    const onClose = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={onClose} />)

    const closeButton = await screen.findByLabelText('Close diagnostics')
    await userEvent.click(closeButton)

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does NOT show Fix Setup button when all checks pass', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('healthy', [
        { name: 'Workspace Check', status: 'pass', message: 'OK' },
        { name: 'Auth Check', status: 'pass', message: 'OK' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    await waitFor(() => expect(screen.getByText('Workspace Check')).toBeInTheDocument())

    expect(screen.queryByText('Fix Setup')).not.toBeInTheDocument()
  })

  it('does NOT show Fix Setup button when onOpenSetupGuide not provided', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Auth Check', status: 'fail', message: 'Failed' },
      ]),
    })

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    await waitFor(() => expect(screen.getByText('Auth Check')).toBeInTheDocument())

    expect(screen.queryByText('Fix Setup')).not.toBeInTheDocument()
  })

  it('shows Fix Setup button when checks fail and callback provided', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Workspace Check', status: 'pass', message: 'OK' },
        { name: 'Auth Check', status: 'fail', message: 'Authentication failed' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    await waitFor(() => expect(screen.getByText('Fix Setup')).toBeInTheDocument())
  })

  it('calls onOpenSetupGuide with correct step for workspace failure', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('error', [
        { name: 'Workspace Configuration', status: 'fail', message: 'Workspace not found' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('workspace')
  })

  it('calls onOpenSetupGuide with correct step for auth failure', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Authentication Check', status: 'fail', message: 'Not authenticated' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('auth')
  })

  it('calls onOpenSetupGuide with correct step for permission failure', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Permission Check', status: 'fail', message: 'Insufficient permissions' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('auth')
  })

  it('calls onOpenSetupGuide with correct step for diagnostic settings failure', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Diagnostic Settings', status: 'fail', message: 'Not configured' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('diagnostic-settings')
  })

  it('calls onOpenSetupGuide with verification step for other failures', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Log Connectivity', status: 'fail', message: 'Cannot connect to logs' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('verification')
  })

  it('prioritizes workspace step when multiple checks fail', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('error', [
        { name: 'Workspace Check', status: 'fail', message: 'No workspace' },
        { name: 'Auth Check', status: 'fail', message: 'Not authenticated' },
        { name: 'Diagnostic Settings', status: 'fail', message: 'Not configured' },
      ]),
    })

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    // Workspace is most foundational, should be prioritized
    expect(onOpenSetupGuide).toHaveBeenCalledWith('workspace')
  })

  it('re-runs diagnostics when Run Diagnostics button clicked', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => createMockHealthResponse('healthy', [
        { name: 'Test Check', status: 'pass', message: 'OK' },
      ]),
    })

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    await waitFor(() => expect(screen.getByText('Run Diagnostics')).toBeInTheDocument())
    
    // Initial fetch
    expect(fetchMock).toHaveBeenCalledTimes(1)

    const runButton = screen.getByText('Run Diagnostics')
    await userEvent.click(runButton)

    // Second fetch
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })

  it('shows correct status badge for degraded state', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('degraded', [
        { name: 'Test', status: 'warn', message: 'Warning' },
      ]),
    })

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(await screen.findByText('Some checks need attention')).toBeInTheDocument()
  })

  it('copies diagnostics report to clipboard', async () => {
    const mockWriteText = vi.fn()
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText,
      },
    })

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => createMockHealthResponse('healthy', [
        { name: 'Test Check', status: 'pass', message: 'All good' },
      ]),
    })

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    const copyButton = await screen.findByText('Copy Report')
    await userEvent.click(copyButton)

    await waitFor(() => {
      expect(mockWriteText).toHaveBeenCalled()
      const copiedText = mockWriteText.mock.calls[0][0] as string
      expect(copiedText).toContain('Azure Logs Diagnostics')
      expect(copiedText).toContain('Test Check')
    })
  })
})
