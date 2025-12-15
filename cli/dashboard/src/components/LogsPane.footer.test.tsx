import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import { LogsPane } from './LogsPane'

// Keep these tests isolated from real timers/WS.
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

vi.mock('@/hooks/useAzurePollingRefreshTrigger', () => ({
  useAzurePollingRefreshTrigger: () => ({
    secondsUntilRefresh: 5,
    refreshTrigger: 0,
  }),
}))

describe('LogsPane refresh footer', () => {
  beforeEach(() => {
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ logs: [] }),
    }) as unknown as typeof fetch
  })

  afterEach(() => {
    cleanup()
    globalThis.WebSocket = originalWebSocket
    globalThis.fetch = originalFetch
  })

  it('shows refresh countdown when syncInterval is set', async () => {
    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={false}
        logMode="azure"
        syncInterval={5000}
        azureRealtime={false}
      />
    )

    expect(await screen.findByText(/Next refresh in/i)).toBeInTheDocument()
    expect(screen.getByText('5s')).toBeInTheDocument()
  })

  it('hides refresh countdown when collapsed', async () => {
    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={false}
        logMode="azure"
        syncInterval={5000}
        azureRealtime={false}
        isCollapsed={true}
      />
    )

    await waitFor(() => {
      expect(screen.queryByText(/Next refresh in/i)).not.toBeInTheDocument()
    })
  })
})
