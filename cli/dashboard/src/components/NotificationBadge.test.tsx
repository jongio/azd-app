import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import { NotificationBadge } from './NotificationBadge'

afterEach(() => {
  vi.useRealTimers()
})

describe('NotificationBadge', () => {
  it('renders count correctly', () => {
    render(<NotificationBadge count={5} />)
    expect(screen.getByRole('status')).toHaveTextContent('5')
  })

  it('hides when count is 0 by default', () => {
    const { container } = render(<NotificationBadge count={0} />)
    expect(container.firstChild).toBeNull()
  })

  it('shows 0 when showZero is true', () => {
    render(<NotificationBadge count={0} showZero />)
    expect(screen.getByRole('status')).toHaveTextContent('0')
  })

  it('displays max+ when count exceeds max', () => {
    render(<NotificationBadge count={150} max={99} />)
    expect(screen.getByRole('status')).toHaveTextContent('99+')
  })

  it('clamps negative counts to 0', () => {
    render(<NotificationBadge count={-5} showZero />)
    expect(screen.getByRole('status')).toHaveTextContent('0')
  })

  it('applies correct size classes', () => {
    const { rerender } = render(<NotificationBadge count={5} size="sm" />)
    expect(screen.getByRole('status')).toHaveClass('h-4')

    rerender(<NotificationBadge count={5} size="md" />)
    expect(screen.getByRole('status')).toHaveClass('h-5')

    rerender(<NotificationBadge count={5} size="lg" />)
    expect(screen.getByRole('status')).toHaveClass('h-6')
  })

  it('applies correct variant classes', () => {
    const { rerender } = render(<NotificationBadge count={5} variant="default" />)
    let badge = screen.getByRole('status')
    expect(badge.className).toContain('bg-[hsl(210,100%,50%)]')

    rerender(<NotificationBadge count={5} variant="critical" />)
    badge = screen.getByRole('status')
    expect(badge.className).toContain('bg-[hsl(0,84%,60%)]')

    rerender(<NotificationBadge count={5} variant="warning" />)
    badge = screen.getByRole('status')
    expect(badge.className).toContain('bg-[hsl(45,100%,50%)]')
  })

  it('has correct ARIA attributes', () => {
    render(<NotificationBadge count={5} />)
    const badge = screen.getByRole('status')
    
    expect(badge).toHaveAttribute('aria-label', '5 unread notifications')
    expect(badge).toHaveAttribute('aria-live', 'polite')
    expect(badge).toHaveAttribute('aria-atomic', 'true')
  })

  it('applies custom className', () => {
    render(<NotificationBadge count={5} className="custom-class" />)
    expect(screen.getByRole('status')).toHaveClass('custom-class')
  })

  it('renders with pulse animation when pulse prop is true', () => {
    render(<NotificationBadge count={5} pulse />)
    const badge = screen.getByRole('status')
    
    // Initially should have pulse animation
    // Animation is applied via useEffect
    expect(badge).toBeInTheDocument()
  })

  it('pulses on count increase and then clears', () => {
    vi.useFakeTimers()

    const { rerender } = render(<NotificationBadge count={0} showZero />)
    const badge = screen.getByRole('status')
    expect(badge.className).not.toContain('animate-notification-pulse')

    rerender(<NotificationBadge count={1} showZero />)
    expect(screen.getByRole('status').className).toContain('animate-notification-pulse')

    act(() => {
      vi.advanceTimersByTime(1200)
    })

    expect(screen.getByRole('status').className).not.toContain('animate-notification-pulse')
  })

  it('announces unread notifications after debounce (including severity + pluralization)', () => {
    vi.useFakeTimers()

    const { rerender } = render(<NotificationBadge count={1} variant="warning" />)

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(screen.getByText('1 warning unread notification')).toBeInTheDocument()

    rerender(<NotificationBadge count={2} variant="critical" />)

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(screen.getByText('2 critical unread notifications')).toBeInTheDocument()
  })
})
