import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, cleanup, waitFor, screen } from '@testing-library/react'
import { LogsPane } from './LogsPane'

const DATETIME_LIKE_PATTERN = /\b\d{4}-\d{2}-\d{2}\b|\b\d{2}:\d{2}(:\d{2})?\b/

// After the Connect-RPC migration Azure logs are fetched through
// `AzureService.GetAzureLogs`, not `GET /api/azure/logs`, so these
// tests capture Connect requests via mocked client factories. We
// preserve the legacy `fetchMock` for any non-logs HTTP surface the
// component still touches (none today, but the stub keeps the test
// resilient to future REST callers that aren't in-scope here).
type AzureRequest = { service?: string; sinceSeconds: bigint; tail: number }
type LocalRequest = { serviceName?: string; tail: number }

let capturedAzureRequests: AzureRequest[] = []
let capturedLocalRequests: LocalRequest[] = []
let azureEntries: unknown[] = []
let localEntries: unknown[] = []

const getAzureLogsMock = vi.fn((req: AzureRequest) => {
  capturedAzureRequests.push(req)
  return Promise.resolve({ entries: azureEntries })
})
const getLogsMock = vi.fn((req: LocalRequest) => {
  capturedLocalRequests.push(req)
  return Promise.resolve({ entries: localEntries })
})

// Preserve every other factory (createLifecycleClient, etc.) so
// transitive consumers like `useCodespaceEnv` keep dialing the real
// default transport. Only the two LogsView callees are stubbed.
vi.mock('@/lib/connectClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/connectClient')>()
  return {
    ...actual,
    createAzureClient: () => ({ getAzureLogs: getAzureLogsMock }),
    createLogsClient: () => ({ getLogs: getLogsMock }),
  }
})

// LogsView translates proto entries into dashboard entries via
// `protoLogEntryToView`. The LogsPane tests provide dashboard-shaped
// fixtures directly, so make the mapper an identity.
vi.mock('@/lib/log-proto', () => ({
  protoLogEntryToView: <T,>(entry: T) => entry,
}))

// Shared log stream is server-streaming Connect now. The LogsPane
// tests don't assert on realtime behaviour, so stub the hook to a
// noop stable value.
vi.mock('@/hooks/useSharedLogStream', () => ({
  useSharedLogStream: () => ({ connectionState: 'connected' as const, droppedCount: 0 }),
}))

// Services context is exercised only for its `serviceNames` shape.
vi.mock('@/contexts/ServicesContext', () => ({
  useServicesContext: () => ({
    services: [],
    serviceNames: ['api', 'web', 'database'],
    loading: false,
    error: null,
    connected: true,
    refetch: vi.fn(),
    getService: vi.fn(),
  }),
  ServicesProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

describe('LogsPane', () => {
  beforeEach(() => {
    capturedAzureRequests = []
    capturedLocalRequests = []
    azureEntries = []
    localEntries = []
    getAzureLogsMock.mockClear()
    getLogsMock.mockClear()
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it('uses default 15m timeRange when none is provided in Azure mode', async () => {
    render(<LogsPane serviceName="api" onCopy={() => {}} isPaused={false} logMode="azure" />)

    await waitFor(() => expect(capturedAzureRequests.length).toBeGreaterThan(0))
    // 15m → 900 seconds on the wire (int64 via protoInt64.parse).
    expect(capturedAzureRequests[0].sinceSeconds).toBe(900n)
  })

  it('does not show datetime text in the header title (Azure)', async () => {
    render(<LogsPane serviceName="example-service" onCopy={() => {}} isPaused={false} logMode="azure" />)

    await waitFor(() => expect(capturedAzureRequests.length).toBeGreaterThan(0))

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

    // Local mode now streams via Connect (LogsService.StreamLocalLogs), not raw
    // WebSocket - the legacy WS bridge is gone. We just need to give effects a
    // tick to mount the streaming hook before checking the title text.
    await waitFor(() => expect(screen.getByTestId('logs-pane-header-title')).toBeTruthy())

    const title = screen.getByTestId('logs-pane-header-title')
    expect(title.textContent ?? '').toContain('example-service')
    expect(title.textContent ?? '').not.toMatch(DATETIME_LIKE_PATTERN)
  })

  it('deduplicates embedded timestamps and service prefixes', async () => {
    azureEntries = [{
      timestamp: '2025-12-13T05:45:49.1071934-08:00',
      service: 'appservice-web',
      message: '[2025-12-13T05:45:49.1071934-08:00] [appservice-web] [2025-12-13 05:45:49] Health endpoint hit',
      level: 1,
      isStderr: false,
    }]

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

    // Allow effects to mount.
    await new Promise((resolve) => setTimeout(resolve, 50))

    // Local mode must not call AzureService.GetAzureLogs.
    expect(capturedAzureRequests.length).toBe(0)
  })

  describe('Azure refresh trigger', () => {
    it('refreshTrigger in useEffect deps causes fetch on state change', async () => {
      // This test verifies the component structure - the refresh mechanism
      // is tested by observing that timeRange changes trigger re-fetch.
      const { rerender } = render(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          timeRange={{ preset: '15m' }}
        />
      )

      await waitFor(() =>
        expect(capturedAzureRequests.some((r) => r.sinceSeconds === 900n)).toBe(true),
      )

      // Change timeRange to trigger re-fetch (similar mechanism to refreshTrigger).
      rerender(
        <LogsPane
          serviceName="api"
          onCopy={() => {}}
          isPaused={false}
          logMode="azure"
          timeRange={{ preset: '24h' }}
        />
      )

      // 24h → 86400 seconds on the wire.
      await waitFor(() =>
        expect(capturedAzureRequests.some((r) => r.sinceSeconds === 86400n)).toBe(true),
      )
    })
  })
})
