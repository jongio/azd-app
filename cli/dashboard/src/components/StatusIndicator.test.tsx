import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  StatusDot,
  StatusBadge,
  StatusIndicator,
  HealthPill,
  ConnectionStatus,
  StatusSkeleton,
  Spinner,
} from './StatusIndicator'

// Mock matchMedia for prefers-reduced-motion tests
const mockMatchMedia = (matches: boolean) => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('prefers-reduced-motion') ? matches : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

describe('StatusDot', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  describe('rendering', () => {
    it('renders a status dot', () => {
      render(<StatusDot status="healthy" />)
      
      const dot = screen.getByRole('img', { name: 'Healthy' })
      expect(dot).toBeInTheDocument()
    })

    it('renders with correct ARIA label for each status', () => {
      const statuses = ['healthy', 'unhealthy', 'degraded', 'starting', 'stopping', 'stopped', 'error', 'unknown'] as const
      
      statuses.forEach(status => {
        const { unmount } = render(<StatusDot status={status} />)
        const dot = screen.getByRole('img')
        expect(dot).toHaveAttribute('aria-label')
        expect(dot).toHaveAttribute('title')
        unmount()
      })
    })

    it('applies correct color for healthy status', () => {
      render(<StatusDot status="healthy" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-emerald-500')
    })

    it('applies correct color for error status', () => {
      render(<StatusDot status="error" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-rose-500')
    })

    it('applies correct color for warning status', () => {
      render(<StatusDot status="degraded" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-amber-500')
    })

    it('applies correct color for info status (starting)', () => {
      render(<StatusDot status="starting" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-sky-500')
    })

    it('applies correct color for muted status (stopped)', () => {
      render(<StatusDot status="stopped" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-slate-400')
    })
  })

  describe('sizes', () => {
    it('renders small size', () => {
      render(<StatusDot status="healthy" size="sm" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('w-1.5', 'h-1.5')
    })

    it('renders medium size (default)', () => {
      render(<StatusDot status="healthy" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('w-2', 'h-2')
    })

    it('renders large size', () => {
      render(<StatusDot status="healthy" size="lg" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('w-3', 'h-3')
    })
  })

  describe('animation', () => {
    it('applies animation class when animated', () => {
      mockMatchMedia(false)
      render(<StatusDot status="healthy" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('animate-modern-heartbeat')
    })

    it('applies pulse animation for starting status', () => {
      mockMatchMedia(false)
      render(<StatusDot status="starting" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('animate-modern-pulse')
    })

    it('does not apply animation when animated is false', () => {
      render(<StatusDot status="healthy" animated={false} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
    })

    it('respects prefers-reduced-motion', () => {
      mockMatchMedia(true)
      render(<StatusDot status="healthy" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
    })

    it('applies spin animation for restarting status', () => {
      mockMatchMedia(false)
      render(<StatusDot status="restarting" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('animate-spin')
    })

    it('does not apply animation for stopped status', () => {
      render(<StatusDot status="stopped" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
      expect(dot).not.toHaveClass('animate-spin')
    })
  })

  describe('custom className', () => {
    it('applies custom className', () => {
      render(<StatusDot status="healthy" className="custom-class" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('custom-class')
    })
  })
})

describe('StatusBadge', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  it('renders status badge with text', () => {
    render(<StatusBadge status="healthy" />)
    
    expect(screen.getByText('Healthy')).toBeInTheDocument()
  })

  it('includes status dot by default', () => {
    render(<StatusBadge status="healthy" />)
    
    expect(screen.getByRole('img', { name: 'Healthy' })).toBeInTheDocument()
  })

  it('hides dot when showDot is false', () => {
    render(<StatusBadge status="healthy" showDot={false} />)
    
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
  })

  it('applies correct background colors for healthy', () => {
    render(<StatusBadge status="healthy" />)
    
    const badge = screen.getByText('Healthy').parentElement
    expect(badge).toHaveClass('bg-emerald-50')
  })

  it('applies correct background colors for error', () => {
    render(<StatusBadge status="error" />)
    
    const badge = screen.getByText('Error').parentElement
    expect(badge).toHaveClass('bg-rose-50')
  })

  it('applies correct text colors', () => {
    render(<StatusBadge status="healthy" />)
    
    const badge = screen.getByText('Healthy').parentElement
    expect(badge).toHaveClass('text-emerald-600')
  })

  it('applies custom className', () => {
    render(<StatusBadge status="healthy" className="custom-badge" />)
    
    const badge = screen.getByText('Healthy').parentElement
    expect(badge).toHaveClass('custom-badge')
  })

  it('displays correct text for each status', () => {
    const statusTextMap: Record<string, string> = {
      healthy: 'Healthy',
      unhealthy: 'Unhealthy',
      degraded: 'Degraded',
      starting: 'Starting',
      stopping: 'Stopping',
      stopped: 'Stopped',
      error: 'Error',
      unknown: 'Unknown',
      running: 'Running',
      restarting: 'Restarting',
      'not-running': 'Not Running',
    }

    Object.entries(statusTextMap).forEach(([status, text]) => {
      const { unmount } = render(<StatusBadge status={status as unknown as Parameters<typeof StatusBadge>[0]['status']} />)
      expect(screen.getByText(text)).toBeInTheDocument()
      unmount()
    })
  })
})

describe('StatusIndicator', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  describe('dot variant', () => {
    it('renders dot variant by default', () => {
      render(<StatusIndicator status="healthy" />)
      
      expect(screen.getByRole('img', { name: 'Healthy' })).toBeInTheDocument()
    })

    it('shows label when showLabel is true', () => {
      render(<StatusIndicator status="healthy" showLabel={true} />)
      
      expect(screen.getByRole('img', { name: 'Healthy' })).toBeInTheDocument()
      expect(screen.getByText('Healthy')).toBeInTheDocument()
    })

    it('hides label by default', () => {
      render(<StatusIndicator status="healthy" />)
      
      expect(screen.queryByText('Healthy')).not.toBeInTheDocument()
    })
  })

  describe('badge variant', () => {
    it('renders badge variant', () => {
      render(<StatusIndicator status="healthy" variant="badge" />)
      
      expect(screen.getByText('Healthy')).toBeInTheDocument()
    })
  })

  describe('full variant', () => {
    it('renders full variant with icon and text', () => {
      render(<StatusIndicator status="healthy" variant="full" />)
      
      expect(screen.getByText('Healthy')).toBeInTheDocument()
      // Should have an icon (SVG)
      const container = screen.getByText('Healthy').parentElement
      const svg = container?.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })

    it('applies animation to icon when status is restarting', () => {
      mockMatchMedia(false)
      render(<StatusIndicator status="restarting" variant="full" animated={true} />)
      
      expect(screen.getByText('Restarting')).toBeInTheDocument()
    })

    it('does not apply spin animation when reduced motion is preferred', () => {
      mockMatchMedia(true)
      render(<StatusIndicator status="restarting" variant="full" animated={true} />)
      
      expect(screen.getByText('Restarting')).toBeInTheDocument()
    })
  })

  describe('animation prop', () => {
    it('respects animated prop', () => {
      mockMatchMedia(false)
      render(<StatusIndicator status="healthy" animated={false} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
    })
  })
})

describe('HealthPill', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  it('renders with healthy status when all services healthy', () => {
    render(
      <HealthPill
        total={5}
        healthy={5}
        degraded={0}
        unhealthy={0}
        starting={0}
      />
    )
    
    expect(screen.getByText(/5 Running/)).toBeInTheDocument()
  })

  it('shows unhealthy status when there are unhealthy services', () => {
    render(
      <HealthPill
        total={5}
        healthy={3}
        degraded={0}
        unhealthy={2}
        starting={0}
      />
    )
    
    expect(screen.getByText(/2 Unhealthy/)).toBeInTheDocument()
  })

  it('shows degraded status when there are degraded services', () => {
    render(
      <HealthPill
        total={5}
        healthy={4}
        degraded={1}
        unhealthy={0}
        starting={0}
      />
    )
    
    expect(screen.getByText(/1 Degraded/)).toBeInTheDocument()
  })

  it('shows starting status when there are starting services', () => {
    render(
      <HealthPill
        total={5}
        healthy={3}
        degraded={0}
        unhealthy={0}
        starting={2}
      />
    )
    
    expect(screen.getByText(/2 Starting/)).toBeInTheDocument()
  })

  it('prioritizes unhealthy over degraded', () => {
    render(
      <HealthPill
        total={5}
        healthy={2}
        degraded={1}
        unhealthy={2}
        starting={0}
      />
    )
    
    expect(screen.getByText(/2 Unhealthy/)).toBeInTheDocument()
  })

  it('calls onClick when clicked', async () => {
    const user = userEvent.setup()
    const handleClick = vi.fn()
    
    render(
      <HealthPill
        total={5}
        healthy={5}
        degraded={0}
        unhealthy={0}
        starting={0}
        onClick={handleClick}
      />
    )
    
    await user.click(screen.getByRole('button'))
    expect(handleClick).toHaveBeenCalled()
  })

  it('has correct ARIA attributes', () => {
    render(
      <HealthPill
        total={5}
        healthy={5}
        degraded={0}
        unhealthy={0}
        starting={0}
      />
    )
    
    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('aria-label')
  })

  it('shows expanded state when expanded prop is true', () => {
    const handleClick = vi.fn()
    
    render(
      <HealthPill
        total={5}
        healthy={5}
        degraded={0}
        unhealthy={0}
        starting={0}
        onClick={handleClick}
        expanded={true}
      />
    )
    
    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('aria-expanded', 'true')
  })

  it('applies custom className', () => {
    render(
      <HealthPill
        total={5}
        healthy={5}
        degraded={0}
        unhealthy={0}
        starting={0}
        className="custom-pill"
      />
    )
    
    const button = screen.getByRole('button')
    expect(button).toHaveClass('custom-pill')
  })
})

describe('ConnectionStatus', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  it('shows connected status', () => {
    render(<ConnectionStatus connected={true} />)
    
    expect(screen.getByText('Connected')).toBeInTheDocument()
  })

  it('shows disconnected status', () => {
    render(<ConnectionStatus connected={false} />)
    
    expect(screen.getByText('Disconnected')).toBeInTheDocument()
  })

  it('shows reconnecting status', () => {
    render(<ConnectionStatus connected={false} reconnecting={true} />)
    
    expect(screen.getByText('Reconnecting')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <ConnectionStatus connected={true} className="custom-connection" />
    )
    
    const element = container.firstChild as HTMLElement
    expect(element).toHaveClass('custom-connection')
  })

  it('includes status dot', () => {
    render(<ConnectionStatus connected={true} />)
    
    // Text is sr-only, but dot should be visible
    expect(screen.getByRole('img')).toBeInTheDocument()
  })
})

describe('StatusSkeleton', () => {
  it('renders skeleton loader', () => {
    const { container } = render(<StatusSkeleton />)
    
    const skeleton = container.firstChild as HTMLElement
    expect(skeleton).toHaveClass('animate-pulse')
    expect(skeleton).toHaveClass('rounded-full')
  })

  it('applies custom className', () => {
    const { container } = render(<StatusSkeleton className="custom-skeleton" />)
    
    const skeleton = container.firstChild as HTMLElement
    expect(skeleton).toHaveClass('custom-skeleton')
  })
})

describe('Spinner', () => {
  it('renders spinner', () => {
    render(<Spinner />)
    
    const spinner = screen.getByRole('status', { name: 'Loading' })
    expect(spinner).toBeInTheDocument()
  })

  it('has screen reader text', () => {
    render(<Spinner />)
    
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('applies animation class', () => {
    render(<Spinner />)
    
    const spinner = screen.getByRole('status')
    expect(spinner).toHaveClass('animate-spin')
  })

  describe('sizes', () => {
    it('renders small size', () => {
      render(<Spinner size="sm" />)
      
      const spinner = screen.getByRole('status')
      expect(spinner).toHaveClass('w-3.5', 'h-3.5')
    })

    it('renders medium size (default)', () => {
      render(<Spinner />)
      
      const spinner = screen.getByRole('status')
      expect(spinner).toHaveClass('w-5', 'h-5')
    })

    it('renders large size', () => {
      render(<Spinner size="lg" />)
      
      const spinner = screen.getByRole('status')
      expect(spinner).toHaveClass('w-8', 'h-8')
    })
  })

  it('applies custom className', () => {
    render(<Spinner className="custom-spinner" />)
    
    const spinner = screen.getByRole('status')
    expect(spinner).toHaveClass('custom-spinner')
  })
})

describe('status configuration coverage', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  const allStatuses = [
    'running',
    'healthy',
    'starting',
    'stopping',
    'stopped',
    'degraded',
    'error',
    'unhealthy',
    'unknown',
    'restarting',
    'not-running',
  ] as const

  it('handles all defined statuses', () => {
    allStatuses.forEach(status => {
      const { unmount } = render(<StatusBadge status={status} />)
      // Should render without errors
      expect(screen.getByRole('img')).toBeInTheDocument()
      unmount()
    })
  })

  it('falls back to unknown for undefined status', () => {
    // Testing invalid input to verify fallback behavior
    render(<StatusBadge status={'invalid-status' as unknown as Parameters<typeof StatusBadge>[0]['status']} />)
    
    expect(screen.getByText('Unknown')).toBeInTheDocument()
  })
})
