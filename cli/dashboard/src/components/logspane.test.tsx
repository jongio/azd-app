import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, cleanup, waitFor, waitForElementToBeRemoved, screen, act } from '@testing-library/react'
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

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

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

  it('deduplicates embedded timestamps and service prefixes while keeping timezone', async () => {
    fetchMock.mockImplementationOnce(() => {
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
    })

    render(<LogsPane serviceName="appservice-web" onCopy={() => {}} isPaused={false} logMode="azure" />)

    const logLine = await screen.findByText(/Health endpoint hit/)
    const rowText = logLine.parentElement?.textContent ?? ''

    const timestampMatches = rowText.match(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/g) ?? []
    expect(timestampMatches.length).toBe(1)
    expect(rowText).toMatch(/-08:00/)
    const serviceMatches = rowText.match(/appservice-web/g) ?? []
    expect(serviceMatches.length).toBe(1)
  })

  it('uses custom timeRange when provided', async () => {
    const start = new Date('2024-01-01T00:00:00.000Z')
    const end = new Date('2024-01-01T01:00:00.000Z')

    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={false}
        logMode="azure"
        timeRange={{ preset: 'custom', start, end }}
      />
    )

    await waitFor(() => expect(capturedAzureUrls.length).toBeGreaterThan(0))
    
    expect(String(capturedAzureUrls[0])).toContain(`start=${encodeURIComponent(start.toISOString())}`)
    expect(String(capturedAzureUrls[0])).toContain(`end=${encodeURIComponent(end.toISOString())}`)
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

  it('shows a fetching state while Azure logs are loading', async () => {
    const fixedEnd = new Date('2025-12-14T12:00:00.000Z')

    const deferred = createDeferred<Response>()

    fetchMock.mockImplementationOnce(() => deferred.promise)

    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={false}
        logMode="azure"
        timeRange={{ preset: '15m', end: fixedEnd }}
      />
    )

    await screen.findByText('Fetching Azure logs...')

    deferred.resolve({
      ok: true,
      json: () => Promise.resolve({ logs: [] }),
    } as unknown as Response)

    await waitFor(() => {
      expect(screen.getByText('No logs in the selected time range')).toBeInTheDocument()
    })

    const fetchingNode = screen.queryByText('Fetching Azure logs...')
    if (fetchingNode) {
      await waitForElementToBeRemoved(fetchingNode)
    }

    expect(screen.getByRole('log')).toHaveTextContent(
      'No logs were returned between 2025-12-14 11:45:00Z and 2025-12-14 12:00:00Z.'
    )

    expect(
      screen.getByText(/Try changing the timeframe to 24 hours\./)
    ).toBeInTheDocument()
  })

  it('suggests a custom range when timeframe is already 24h', async () => {
    const fixedEnd = new Date('2025-12-14T12:00:00.000Z')

    render(
      <LogsPane
        serviceName="api"
        onCopy={() => {}}
        isPaused={false}
        logMode="azure"
        timeRange={{ preset: '24h', end: fixedEnd }}
      />
    )

    await waitFor(() => expect(capturedAzureUrls.length).toBeGreaterThan(0))

    await waitFor(() => {
      expect(screen.getByText('No logs in the selected time range')).toBeInTheDocument()
    })

    expect(
      screen.getByText(/Try changing the timeframe to a custom range\./)
    ).toBeInTheDocument()
  })

  describe('Azure refresh trigger', () => {
    it('refreshTrigger in useEffect deps causes fetch on state change', async () => {
      // This test verifies the component structure - the refresh mechanism
      // is tested by observing that timeRange changes trigger re-fetch
      const start1 = new Date('2024-01-01T00:00:00.000Z')
      const end1 = new Date('2024-01-01T01:00:00.000Z')

      const { rerender } = render(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          timeRange={{ preset: 'custom', start: start1, end: end1 }}
        />
      )

      const start1Query = `start=${encodeURIComponent(start1.toISOString())}`
      await waitFor(() => {
        expect(capturedAzureUrls.some((url) => String(url).includes(start1Query))).toBe(true)
      })

      // Change timeRange to trigger re-fetch (similar mechanism to refreshTrigger)
      const start2 = new Date('2024-01-02T00:00:00.000Z')
      const end2 = new Date('2024-01-02T01:00:00.000Z')

      rerender(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          timeRange={{ preset: 'custom', start: start2, end: end2 }}
        />
      )

      const start2Query = `start=${encodeURIComponent(start2.toISOString())}`
      await waitFor(() => {
        expect(capturedAzureUrls.some((url) => String(url).includes(start2Query))).toBe(true)
      })
    })

    it('polls again within the updated sync interval when the interval is shortened', async () => {
      vi.useFakeTimers()
      vi.setSystemTime(new Date('2025-01-01T00:00:00.000Z'))

      const { rerender } = render(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          syncInterval={30000}
          azureRealtime={false}
        />
      )

      await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

      rerender(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          syncInterval={5000}
          azureRealtime={false}
        />
      )

      act(() => {
        vi.advanceTimersByTime(5000)
      })

      await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))

      vi.useRealTimers()
    })
  })
})
