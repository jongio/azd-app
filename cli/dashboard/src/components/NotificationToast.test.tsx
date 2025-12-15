import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { NotificationToast } from './NotificationToast'

describe('NotificationToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2025-01-01T00:00:00.000Z'))
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      cb(0)
      return 0
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  const renderToast = (overrides?: Partial<ComponentProps<typeof NotificationToast>>) => {
    const onDismiss = vi.fn()
    const onClick = vi.fn()

    const props: ComponentProps<typeof NotificationToast> = {
      id: 'n1',
      title: 'Title',
      message: 'Message',
      severity: 'info',
      timestamp: new Date('2025-01-01T00:00:00.000Z'),
      onDismiss,
      onClick,
      ...overrides,
    }

    render(<NotificationToast {...props} />)
    return { onDismiss, onClick }
  }

  it('renders title/message and relative time', () => {
    renderToast({ timestamp: new Date('2024-12-31T23:59:50.000Z') })

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Title')).toBeInTheDocument()
    expect(screen.getByText('Message')).toBeInTheDocument()
    expect(screen.getByText('Just now')).toBeInTheDocument()
  })

  it('formats relative time for minutes, hours, and days', () => {
    const { rerender } = render(
      <NotificationToast
        id="n1"
        title="Title"
        message="Message"
        severity="info"
        timestamp={new Date('2024-12-31T23:55:00.000Z')}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.getByText('5 minutes ago')).toBeInTheDocument()

    rerender(
      <NotificationToast
        id="n1"
        title="Title"
        message="Message"
        severity="info"
        timestamp={new Date('2024-12-31T22:00:00.000Z')}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.getByText('2 hours ago')).toBeInTheDocument()

    rerender(
      <NotificationToast
        id="n1"
        title="Title"
        message="Message"
        severity="info"
        timestamp={new Date('2024-12-28T00:00:00.000Z')}
        onDismiss={vi.fn()}
      />
    )
    expect(screen.getByText('4 days ago')).toBeInTheDocument()
  })

  it('auto-dismisses after the default timeout', () => {
    const { onDismiss } = renderToast({ autoDismiss: true, onClick: undefined })

    act(() => {
      vi.advanceTimersByTime(5000)
    })

    expect(onDismiss).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(onDismiss).toHaveBeenCalledWith('n1')
  })

  it('calls onClick and then dismisses when the toast is clicked', () => {
    const { onDismiss, onClick } = renderToast({ autoDismiss: false })

    fireEvent.click(screen.getByRole('alert'))
    expect(onClick).toHaveBeenCalledWith('n1')

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(onDismiss).toHaveBeenCalledWith('n1')
  })

  it('dismisses when the close button is clicked without triggering onClick', () => {
    const { onDismiss, onClick } = renderToast({ autoDismiss: false })

    fireEvent.click(screen.getByRole('button', { name: 'Dismiss notification' }))
    expect(onClick).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(onDismiss).toHaveBeenCalledWith('n1')
  })

  it('dismisses when Escape is pressed on the close button', () => {
    const { onDismiss } = renderToast({ autoDismiss: false, onClick: undefined })

    const closeButton = screen.getByRole('button', { name: 'Dismiss notification' })
    fireEvent.keyDown(closeButton, { key: 'Escape' })

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(onDismiss).toHaveBeenCalledWith('n1')
  })

  it('pauses auto-dismiss on hover and resumes on mouse leave', () => {
    const { onDismiss } = renderToast({ autoDismiss: true, onClick: undefined })
    const toast = screen.getByRole('alert')

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    fireEvent.mouseEnter(toast)

    act(() => {
      vi.advanceTimersByTime(6000)
    })

    expect(onDismiss).not.toHaveBeenCalled()

    fireEvent.mouseLeave(toast)

    act(() => {
      vi.advanceTimersByTime(5000)
      vi.advanceTimersByTime(300)
    })

    expect(onDismiss).toHaveBeenCalledWith('n1')
  })

  it('does not render a progress bar when autoDismiss is false', () => {
    renderToast({ autoDismiss: false })
    expect(document.querySelector('div.bg-muted')).not.toBeInTheDocument()
  })

  it('uses longer default timeout for critical severity', () => {
    const { onDismiss } = renderToast({ severity: 'critical', autoDismiss: true, onClick: undefined })

    act(() => {
      vi.advanceTimersByTime(9000)
    })
    expect(onDismiss).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(1000)
      vi.advanceTimersByTime(300)
    })
    expect(onDismiss).toHaveBeenCalledWith('n1')
  })

  it('handles already-expired remaining time by scheduling an immediate dismiss', () => {
    const onDismiss = vi.fn()
    const { rerender } = render(
      <NotificationToast
        id="n1"
        title="Title"
        message="Message"
        severity="info"
        timestamp={new Date('2025-01-01T00:00:00.000Z')}
        onDismiss={onDismiss}
        autoDismiss={true}
        dismissTimeout={5000}
      />
    )

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Force a rerun of the effect where elapsed > timeout (remaining <= 0).
    rerender(
      <NotificationToast
        id="n1"
        title="Title"
        message="Message"
        severity="info"
        timestamp={new Date('2025-01-01T00:00:00.000Z')}
        onDismiss={onDismiss}
        autoDismiss={true}
        dismissTimeout={1}
      />
    )

    act(() => {
      vi.advanceTimersByTime(0)
      vi.advanceTimersByTime(300)
    })

    expect(onDismiss).toHaveBeenCalledWith('n1')
  })
})
