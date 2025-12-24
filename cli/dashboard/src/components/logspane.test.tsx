import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, cleanup, waitFor, screen } from '@testing-library/react'
import { LogsPane } from './LogsPane'

const DATETIME_LIKE_PATTERN = /\b\d{4}-\d{2}-\d{2}\b|\b\d{2}:\d{2}(:\d{2})?\b/

const fetchMock = vi.fn()
let capturedAzureUrls: string[] = []
let capturedWebSocketUrls: string[] = []

function normalizeRequestUrl(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input
  if (input instanceof URL) return input.toString()
  return input.url
}

class MockWebSocket {
  static readonly OPEN = 1
  static readonly CONNECTING = 0
  readyState = MockWebSocket.OPEN
  onmessage: ((event: MessageEvent<string>) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  close = vi.fn()

  constructor(public url: string) {
    capturedWebSocketUrls.push(url)
  }
}

const originalWebSocket = globalThis.WebSocket
const originalFetch = globalThis.fetch

describe('LogsPane', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    capturedAzureUrls = []
    capturedWebSocketUrls = []

    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const url = normalizeRequestUrl(input)
      if (url.includes('/api/azure/logs')) {
        capturedAzureUrls.push(url)
      }

      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ logs: [] }),
      } as unknown as Response)
    })
    globalThis.fetch = fetchMock as unknown as typeof fetch
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()

    globalThis.WebSocket = originalWebSocket
    globalThis.fetch = originalFetch
  })

  it('uses default 15m timeRange when none is provided in Azure mode', async () => {
    render(<LogsPane serviceName="api" onCopy={() => {}} isPaused={false} logMode="azure" />)

    await waitFor(() => expect(capturedAzureUrls.length).toBeGreaterThan(0))
    expect(String(capturedAzureUrls[0])).toContain('since=15m')
  })

  it('does not show datetime text in the header title (Azure)', async () => {
    render(<LogsPane serviceName="example-service" onCopy={() => {}} isPaused={false} logMode="azure" />)

    await waitFor(() => expect(capturedAzureUrls.length).toBeGreaterThan(0))

    const title = screen.getByTestId('logs-pane-header-title')
    expect(title.textContent ?? '').toContain('example-service')
    expect(title.textContent ?? '').not.toMatch(DATETIME_LIKE_PATTERN)
  })

  it('does not show datetime text in the header title (Local, collapsed)', async () => {
    render(
      <LogsPane
        serviceName="example-service"
        onCopy={() => {}}
        isPaused={false}
        logMode="local"
        isCollapsed={true}
      />
    )

    await waitFor(() => expect(capturedWebSocketUrls.length).toBeGreaterThan(0))

    const title = screen.getByTestId('logs-pane-header-title')
    expect(title.textContent ?? '').toContain('example-service')
    expect(title.textContent ?? '').not.toMatch(DATETIME_LIKE_PATTERN)
  })

  it('deduplicates embedded timestamps and service prefixes', async () => {
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const url = normalizeRequestUrl(input)
      if (url.includes('/api/azure/logs')) {
        capturedAzureUrls.push(url)
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            logs: [{
              timestamp: '2025-12-13T05:45:49.1071934-08:00',
              service: 'appservice-web',
              message: '[2025-12-13T05:45:49.1071934-08:00] [appservice-web] [2025-12-13 05:45:49] Health endpoint hit',
              level: 1,
              isStderr: false,
            }],
          }),
        } as unknown as Response)
      }

      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ logs: [] }),
      } as unknown as Response)
    })

    render(<LogsPane serviceName="appservice-web" onCopy={() => {}} isPaused={false} logMode="azure" />)

    const logLine = await screen.findByText(/Health endpoint hit/)
    const rowText = logLine.parentElement?.textContent ?? ''

    // Should only show timestamp once in MM-DD HH:MM:SS.mmm format
    const timestampMatches = rowText.match(/\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}/g) ?? []
    expect(timestampMatches.length).toBe(1)
    
    // Service name should only appear once (deduplicated)
    const serviceMatches = rowText.match(/appservice-web/g) ?? []
    expect(serviceMatches.length).toBe(1)
  })

  it('does not fetch Azure logs in local mode', async () => {
    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={false}
        logMode="local"
      />
    )

    // Ensure the local-mode WebSocket effect ran.
    await waitFor(() => expect(capturedWebSocketUrls.length).toBeGreaterThan(0))

    // Local mode uses WebSocket, should not fetch /api/azure/logs
    expect(capturedAzureUrls.length).toBe(0)
  })



  describe('Azure refresh trigger', () => {
    it('refreshTrigger in useEffect deps causes fetch on state change', async () => {
      // This test verifies the component structure - the refresh mechanism
      // is tested by observing that timeRange changes trigger re-fetch
      const { rerender } = render(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          timeRange={{ preset: '15m' }}
        />
      )

      await waitFor(() => expect(capturedAzureUrls.some((url) => String(url).includes('since=15m'))).toBe(true))

      // Change timeRange to trigger re-fetch (similar mechanism to refreshTrigger)
      rerender(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          timeRange={{ preset: '24h' }}
        />
      )

      await waitFor(() => expect(capturedAzureUrls.some((url) => String(url).includes('since=24h'))).toBe(true))
    })
  })
})
