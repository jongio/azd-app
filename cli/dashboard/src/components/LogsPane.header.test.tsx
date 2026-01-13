import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsPane } from './LogsPane'
import type { Service } from '@/types'

// Minimal WebSocket stub
class MockWebSocket {
  static readonly OPEN = 1
  static readonly CONNECTING = 0
  readyState = MockWebSocket.OPEN
  onmessage: ((event: MessageEvent<string>) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  close = vi.fn()
  constructor(public url: string) {}
}

const originalWebSocket = globalThis.WebSocket
const originalFetch = globalThis.fetch
const originalOpen = globalThis.open

vi.mock('@/hooks/useServiceOperations', () => ({
  useServiceOperations: () => ({
    startService: vi.fn().mockResolvedValue(true),
    stopService: vi.fn().mockResolvedValue(true),
    restartService: vi.fn().mockResolvedValue(true),
    isOperationInProgress: vi.fn().mockReturnValue(false),
    getOperationState: vi.fn().mockReturnValue('idle'),
    canPerformAction: vi.fn().mockReturnValue(false),
    error: null,
    // bulk helpers used elsewhere
    startAll: vi.fn(),
    stopAll: vi.fn(),
    restartAll: vi.fn(),
    isBulkOperationInProgress: vi.fn().mockReturnValue(false),
  }),
}))

describe('LogsPane header actions', () => {
  beforeEach(() => {
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ logs: [] }),
    }) as unknown as typeof fetch

    globalThis.open = vi.fn() as unknown as typeof globalThis.open
  })

  afterEach(() => {
    cleanup()
    globalThis.WebSocket = originalWebSocket
    globalThis.fetch = originalFetch
    globalThis.open = originalOpen
  })

  it('shows process + health badges and supports Open/Details buttons', async () => {
    const user = userEvent.setup()
    const onShowDetails = vi.fn()

    const service: Service = {
      name: 'api',
      host: 'containerapp',
      azure: { url: 'https://example.azure.com' },
      local: { status: 'running', health: 'healthy', port: 3000, url: 'http://localhost:3000' },
    }

    render(
      <LogsPane
        serviceName="api"
        service={service}
        serviceHealth="healthy"
        onCopy={() => {}}
        isPaused={false}
        logMode="azure"
        onShowDetails={onShowDetails}
      />
    )

    // Wait until initial fetch happens to avoid act warnings.
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalled())

    expect(screen.getByTitle('Process state: running')).toBeInTheDocument()
    expect(screen.getByTitle('Service health: healthy (from health checks)')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /open custom url in new tab/i }))
  expect(globalThis.open).toHaveBeenCalledWith('https://example.azure.com', '_blank', 'noopener,noreferrer')

    await user.click(screen.getByRole('button', { name: /show service details panel/i }))
    expect(onShowDetails).toHaveBeenCalledTimes(1)
  })

  it('opens the local URL when viewing local logs even if an Azure URL exists', async () => {
    const user = userEvent.setup()
    const localUrl = 'http://localhost:4000'
    const service: Service = {
      name: 'web',
      host: 'containerapp',
      azure: { url: 'https://example.azure.com' },
      local: { status: 'running', health: 'healthy', port: 4000, url: localUrl },
    }

    render(
      <LogsPane
        serviceName="web"
        service={service}
        serviceHealth="healthy"
        onCopy={() => {}}
        isPaused={false}
        logMode="local"
      />
    )

    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /open service in new tab/i }))
    expect(globalThis.open).toHaveBeenCalledWith(localUrl, '_blank', 'noopener,noreferrer')
  })
})
