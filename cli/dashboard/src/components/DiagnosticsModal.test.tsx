/**
 * DiagnosticsModal tests.
 *
 * The component previously fetched `/api/azure/logs/health`. It now calls
 * `AzureService.GetAzureLogsHealth` through Connect, so we mock
 * `@/lib/connectClient`'s `createAzureClient` factory and let each test
 * stage a proto response (built by `buildHealthResponse`) or a thrown
 * error. The assertion surface (rendered status, names, fix-setup
 * routing, etc.) is unchanged because the component still renders a
 * `HealthCheckResponse` internally — only the wire shape differs.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { create } from '@bufbuild/protobuf'
import { timestampNow } from '@bufbuild/protobuf/wkt'

import {
  AzureCheckStatus,
  AzureHealthCheckSchema,
  AzureOverallStatus,
  GetAzureLogsHealthResponseSchema,
  type GetAzureLogsHealthResponse,
} from '@/gen/proto/azdapp/v1/azure_pb.js'

const { getAzureLogsHealthMock } = vi.hoisted(() => ({
  getAzureLogsHealthMock: vi.fn(),
}))

vi.mock('@/lib/connectClient', () => ({
  createAzureClient: () => ({
    getAzureLogsHealth: (...args: unknown[]) =>
      getAzureLogsHealthMock(...args) as Promise<GetAzureLogsHealthResponse>,
  }),
}))

import { DiagnosticsModal } from './DiagnosticsModal'

interface DashboardHealthCheck {
  name: string
  status: 'pass' | 'warn' | 'fail'
  message: string
  fix?: string
}

function statusToProto(
  s: 'healthy' | 'degraded' | 'error',
): AzureOverallStatus {
  switch (s) {
    case 'healthy':
      return AzureOverallStatus.HEALTHY
    case 'degraded':
      return AzureOverallStatus.DEGRADED
    default:
      return AzureOverallStatus.ERROR
  }
}

function checkStatusToProto(s: 'pass' | 'warn' | 'fail'): AzureCheckStatus {
  switch (s) {
    case 'pass':
      return AzureCheckStatus.PASS
    case 'warn':
      return AzureCheckStatus.WARN
    default:
      return AzureCheckStatus.FAIL
  }
}

function buildHealthResponse(
  status: 'healthy' | 'degraded' | 'error',
  checks: DashboardHealthCheck[],
): GetAzureLogsHealthResponse {
  return create(GetAzureLogsHealthResponseSchema, {
    status: statusToProto(status),
    checks: checks.map(
      (c) =>
        create(AzureHealthCheckSchema, {
          name: c.name,
          status: checkStatusToProto(c.status),
          message: c.message,
          fix: c.fix ?? '',
        }),
    ),
    docsUrl: 'https://docs.example.com',
    timestamp: timestampNow(),
  })
}

describe('DiagnosticsModal', () => {
  beforeEach(() => {
    getAzureLogsHealthMock.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  it('does not render when closed', () => {
    render(<DiagnosticsModal isOpen={false} onClose={vi.fn()} />)
    expect(screen.queryByText('Azure Logs Diagnostics')).not.toBeInTheDocument()
  })

  it('renders when open and fetches health checks', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('healthy', [
        { name: 'Workspace Check', status: 'pass', message: 'Workspace configured' },
      ]),
    )

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(screen.getByText('Azure Logs Diagnostics')).toBeInTheDocument()
    await waitFor(() => expect(getAzureLogsHealthMock).toHaveBeenCalled())
  })

  it('shows loading state while fetching', async () => {
    getAzureLogsHealthMock.mockImplementation(
      () => new Promise(() => {/* never resolves */}),
    )

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(await screen.findByText('Running health checks...')).toBeInTheDocument()
  })

  it('displays health check results', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Workspace Check', status: 'pass', message: 'Workspace configured' },
        { name: 'Auth Check', status: 'fail', message: 'Authentication failed', fix: 'az login' },
      ]),
    )

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    await waitFor(() => {
      expect(screen.getByText('Workspace Check')).toBeInTheDocument()
      expect(screen.getByText('Auth Check')).toBeInTheDocument()
      expect(screen.getByText('Authentication failed')).toBeInTheDocument()
    })
  })

  it('shows error state when fetch fails', async () => {
    getAzureLogsHealthMock.mockRejectedValueOnce(new Error('Network error'))

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    expect(await screen.findByText('Failed to fetch diagnostics')).toBeInTheDocument()
    expect(screen.getByText('Network error')).toBeInTheDocument()
  })

  it('calls onClose when close button clicked', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('healthy', []),
    )

    const onClose = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={onClose} />)

    const closeButton = await screen.findByLabelText('Close diagnostics')
    await userEvent.click(closeButton)

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does NOT show Fix Setup button when all checks pass', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('healthy', [
        { name: 'Workspace Check', status: 'pass', message: 'OK' },
        { name: 'Auth Check', status: 'pass', message: 'OK' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    await waitFor(() => expect(screen.getByText('Workspace Check')).toBeInTheDocument())

    expect(screen.queryByText('Fix Setup')).not.toBeInTheDocument()
  })

  it('does NOT show Fix Setup button when onOpenSetupGuide not provided', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Auth Check', status: 'fail', message: 'Failed' },
      ]),
    )

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    await waitFor(() => expect(screen.getByText('Auth Check')).toBeInTheDocument())

    expect(screen.queryByText('Fix Setup')).not.toBeInTheDocument()
  })

  it('shows Fix Setup button when checks fail and callback provided', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Workspace Check', status: 'pass', message: 'OK' },
        { name: 'Auth Check', status: 'fail', message: 'Authentication failed' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    await waitFor(() => expect(screen.getByText('Fix Setup')).toBeInTheDocument())
  })

  it('calls onOpenSetupGuide with correct step for workspace failure', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('error', [
        { name: 'Workspace Configuration', status: 'fail', message: 'Workspace not found' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('workspace')
  })

  it('calls onOpenSetupGuide with correct step for auth failure', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Authentication Check', status: 'fail', message: 'Not authenticated' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('auth')
  })

  it('calls onOpenSetupGuide with correct step for permission failure', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Permission Check', status: 'fail', message: 'Insufficient permissions' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('auth')
  })

  it('calls onOpenSetupGuide with correct step for diagnostic settings failure', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Diagnostic Settings', status: 'fail', message: 'Not configured' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('diagnostic-settings')
  })

  it('calls onOpenSetupGuide with verification step for other failures', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Log Connectivity', status: 'fail', message: 'Cannot connect to logs' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    expect(onOpenSetupGuide).toHaveBeenCalledWith('verification')
  })

  it('prioritizes workspace step when multiple checks fail', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('error', [
        { name: 'Workspace Check', status: 'fail', message: 'No workspace' },
        { name: 'Auth Check', status: 'fail', message: 'Not authenticated' },
        { name: 'Diagnostic Settings', status: 'fail', message: 'Not configured' },
      ]),
    )

    const onOpenSetupGuide = vi.fn()
    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} onOpenSetupGuide={onOpenSetupGuide} />)

    const fixButton = await screen.findByText('Fix Setup')
    await userEvent.click(fixButton)

    // Workspace is most foundational, should be prioritized
    expect(onOpenSetupGuide).toHaveBeenCalledWith('workspace')
  })

  it('re-runs diagnostics when Run Diagnostics button clicked', async () => {
    getAzureLogsHealthMock.mockResolvedValue(
      buildHealthResponse('healthy', [
        { name: 'Test Check', status: 'pass', message: 'OK' },
      ]),
    )

    render(<DiagnosticsModal isOpen={true} onClose={vi.fn()} />)

    await waitFor(() => expect(screen.getByText('Run Diagnostics')).toBeInTheDocument())

    // Initial fetch
    expect(getAzureLogsHealthMock).toHaveBeenCalledTimes(1)

    const runButton = screen.getByText('Run Diagnostics')
    await userEvent.click(runButton)

    // Second fetch
    await waitFor(() => expect(getAzureLogsHealthMock).toHaveBeenCalledTimes(2))
  })

  it('shows correct status badge for degraded state', async () => {
    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('degraded', [
        { name: 'Test', status: 'warn', message: 'Warning' },
      ]),
    )

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

    getAzureLogsHealthMock.mockResolvedValueOnce(
      buildHealthResponse('healthy', [
        { name: 'Test Check', status: 'pass', message: 'All good' },
      ]),
    )

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
