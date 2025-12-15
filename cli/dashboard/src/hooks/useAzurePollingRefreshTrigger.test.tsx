import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, cleanup, act } from '@testing-library/react'
import type { LogMode } from '@/components/ModeToggle'
import { useAzurePollingRefreshTrigger } from './useAzurePollingRefreshTrigger'

function Harness({
  syncInterval,
  isPaused,
  logMode,
  azureRealtime,
}: Readonly<{
  syncInterval?: number
  isPaused: boolean
  logMode: LogMode
  azureRealtime: boolean
}>) {
  const { secondsUntilRefresh, refreshTrigger } = useAzurePollingRefreshTrigger({
    syncInterval,
    isPaused,
    logMode,
    azureRealtime,
  })
  return (
    <div>
      <div data-testid="seconds">{secondsUntilRefresh}</div>
      <div data-testid="trigger">{refreshTrigger}</div>
    </div>
  )
}

describe('useAzurePollingRefreshTrigger', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2025-01-01T00:00:00.000Z'))
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it('increments refreshTrigger after the interval elapses', () => {
    render(<Harness syncInterval={5000} isPaused={false} logMode="azure" azureRealtime={false} />)

    expect(screen.getByTestId('trigger').textContent).toBe('0')

    // Advance just under 5s
    act(() => {
      vi.advanceTimersByTime(4999)
    })
    expect(screen.getByTestId('trigger').textContent).toBe('0')

    // Cross the boundary
    act(() => {
      vi.advanceTimersByTime(2)
    })
    expect(screen.getByTestId('trigger').textContent).toBe('1')
  })

  it('stops countdown and triggering when paused', () => {
    const { rerender } = render(
      <Harness syncInterval={5000} isPaused={false} logMode="azure" azureRealtime={false} />
    )

    act(() => {
      vi.advanceTimersByTime(5002)
    })
    expect(screen.getByTestId('trigger').textContent).toBe('1')

    rerender(<Harness syncInterval={5000} isPaused={true} logMode="azure" azureRealtime={false} />)
    expect(screen.getByTestId('seconds').textContent).toBe('0')

    act(() => {
      vi.advanceTimersByTime(10000)
    })
    expect(screen.getByTestId('trigger').textContent).toBe('1')
  })

  it('catches up and triggers when time jumps forward (simulates tab throttling)', () => {
    render(<Harness syncInterval={5000} isPaused={false} logMode="azure" azureRealtime={false} />)

    // Jump forward without waiting 1s ticks.
    act(() => {
      vi.setSystemTime(new Date('2025-01-01T00:00:12.000Z'))

      // Simulate tab becoming visible again.
      Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // At least one refresh should have triggered.
    expect(Number(screen.getByTestId('trigger').textContent)).toBeGreaterThanOrEqual(1)
  })
})
