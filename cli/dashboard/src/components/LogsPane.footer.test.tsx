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

  it('shows paused indicator when paused in Azure mode', async () => {
    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={true}
        logMode="azure"
        azureRealtime={false}
      />
    )

    expect(await screen.findByText(/Paused - log streaming stopped/i)).toBeInTheDocument()
  })

  it('hides footer when collapsed', async () => {
    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={true}
        logMode="azure"
        azureRealtime={false}
        isCollapsed={true}
      />
    )

    await waitFor(() => {
      expect(screen.queryByText(/Paused/i)).not.toBeInTheDocument()
    })
  })
})
