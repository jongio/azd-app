import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { NotificationStack } from './NotificationStack'
import type { Notification } from './NotificationStack'

describe('NotificationStack', () => {
  const mockNotifications: Notification[] = [
    {
      id: '1',
      title: 'Service Started',
      message: 'API service is now running',
      severity: 'info',
      timestamp: new Date('2024-01-01T12:00:00'),
    },
    {
      id: '2',
      title: 'Warning',
      message: 'High memory usage detected',
      severity: 'warning',
      timestamp: new Date('2024-01-01T12:01:00'),
    },
    {
      id: '3',
      title: 'Critical Error',
      message: 'Database connection failed',
      severity: 'critical',
      timestamp: new Date('2024-01-01T12:02:00'),
    },
  ]

  const mockOnDismiss = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders visible notifications', () => {
    render(
      <NotificationStack
        notifications={mockNotifications}
        onDismiss={mockOnDismiss}
        maxVisible={3}
      />
    )

    expect(screen.getByText('Service Started')).toBeInTheDocument()
    expect(screen.getByText('Warning')).toBeInTheDocument()
    expect(screen.getByText('Critical Error')).toBeInTheDocument()
  })

  it('limits visible notifications to maxVisible', () => {
    const manyNotifications = Array.from({ length: 10 }, (_, i) => ({
      id: `${i}`,
      title: `Notification ${i}`,
      message: `Message ${i}`,
      severity: 'info' as const,
      timestamp: new Date(),
    }))

    render(
      <NotificationStack
        notifications={manyNotifications}
        onDismiss={mockOnDismiss}
        maxVisible={3}
      />
    )

    expect(screen.getByText('Notification 0')).toBeInTheDocument()
    expect(screen.getByText('Notification 1')).toBeInTheDocument()
    expect(screen.getByText('Notification 2')).toBeInTheDocument()
    expect(screen.queryByText('Notification 3')).not.toBeInTheDocument()
  })

  it('shows overflow indicator when notifications exceed maxVisible', () => {
    const manyNotifications = Array.from({ length: 5 }, (_, i) => ({
      id: `${i}`,
      title: `Notification ${i}`,
      message: `Message ${i}`,
      severity: 'info' as const,
      timestamp: new Date(),
    }))

    render(
      <NotificationStack
        notifications={manyNotifications}
        onDismiss={mockOnDismiss}
        maxVisible={3}
      />
    )

    expect(screen.getByText('+2 more notifications')).toBeInTheDocument()
  })

  it('does not show overflow indicator when notifications fit within maxVisible', () => {
    render(
      <NotificationStack
        notifications={mockNotifications.slice(0, 2)}
        onDismiss={mockOnDismiss}
        maxVisible={3}
      />
    )

    expect(screen.queryByText(/more notification/)).not.toBeInTheDocument()
  })

  it('renders with top-right position by default', () => {
    const { container } = render(
      <NotificationStack
        notifications={mockNotifications}
        onDismiss={mockOnDismiss}
      />
    )

    const stack = container.querySelector('[role="region"]')
    expect(stack).toHaveClass('top-4', 'right-4')
  })

  it('renders with custom position', () => {
    const { container } = render(
      <NotificationStack
        notifications={mockNotifications}
        onDismiss={mockOnDismiss}
        position="bottom-center"
      />
    )

    const stack = container.querySelector('[role="region"]')
    expect(stack).toHaveClass('bottom-4', 'left-1/2', '-translate-x-1/2')
  })

  it('applies custom className', () => {
    const { container } = render(
      <NotificationStack
        notifications={mockNotifications}
        onDismiss={mockOnDismiss}
        className="custom-class"
      />
    )

    expect(container.querySelector('.custom-class')).toBeInTheDocument()
  })

  it('has accessibility region with label', () => {
    render(
      <NotificationStack
        notifications={mockNotifications}
        onDismiss={mockOnDismiss}
      />
    )

    const region = screen.getByRole('region', { name: 'Notifications' })
    expect(region).toBeInTheDocument()
    expect(region).toHaveAttribute('aria-live', 'polite')
    expect(region).toHaveAttribute('aria-atomic', 'false')
  })
})
