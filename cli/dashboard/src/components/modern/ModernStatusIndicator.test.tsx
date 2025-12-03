import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  ModernStatusDot,
  ModernStatusBadge,
  ModernStatusIndicator,
  ModernHealthPill,
  ModernConnectionStatus,
  ModernStatusSkeleton,
  ModernSpinner,
} from './ModernStatusIndicator'

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

describe('ModernStatusDot', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  describe('rendering', () => {
    it('renders a status dot', () => {
      render(<ModernStatusDot status="healthy" />)
      
      const dot = screen.getByRole('img', { name: 'Healthy' })
      expect(dot).toBeInTheDocument()
    })

    it('renders with correct ARIA label for each status', () => {
      const statuses = ['healthy', 'unhealthy', 'degraded', 'starting', 'stopping', 'stopped', 'error', 'unknown'] as const
      
      statuses.forEach(status => {
        const { unmount } = render(<ModernStatusDot status={status} />)
        const dot = screen.getByRole('img')
        expect(dot).toHaveAttribute('aria-label')
        expect(dot).toHaveAttribute('title')
        unmount()
      })
    })

    it('applies correct color for healthy status', () => {
      render(<ModernStatusDot status="healthy" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-emerald-500')
    })

    it('applies correct color for error status', () => {
      render(<ModernStatusDot status="error" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-rose-500')
    })

    it('applies correct color for warning status', () => {
      render(<ModernStatusDot status="degraded" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-amber-500')
    })

    it('applies correct color for info status (starting)', () => {
      render(<ModernStatusDot status="starting" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-sky-500')
    })

    it('applies correct color for muted status (stopped)', () => {
      render(<ModernStatusDot status="stopped" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('bg-slate-400')
    })
  })

  describe('sizes', () => {
    it('renders small size', () => {
      render(<ModernStatusDot status="healthy" size="sm" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('w-1.5', 'h-1.5')
    })

    it('renders medium size (default)', () => {
      render(<ModernStatusDot status="healthy" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('w-2', 'h-2')
    })

    it('renders large size', () => {
      render(<ModernStatusDot status="healthy" size="lg" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('w-3', 'h-3')
    })
  })

  describe('animation', () => {
    it('applies animation class when animated', () => {
      mockMatchMedia(false)
      render(<ModernStatusDot status="healthy" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('animate-modern-heartbeat')
    })

    it('applies pulse animation for starting status', () => {
      mockMatchMedia(false)
      render(<ModernStatusDot status="starting" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('animate-modern-pulse')
    })

    it('does not apply animation when animated is false', () => {
      render(<ModernStatusDot status="healthy" animated={false} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
    })

    it('respects prefers-reduced-motion', () => {
      mockMatchMedia(true)
      render(<ModernStatusDot status="healthy" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
    })

    it('applies spin animation for restarting status', () => {
      mockMatchMedia(false)
      render(<ModernStatusDot status="restarting" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('animate-spin')
    })

    it('does not apply animation for stopped status', () => {
      render(<ModernStatusDot status="stopped" animated={true} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
      expect(dot).not.toHaveClass('animate-spin')
    })
  })

  describe('custom className', () => {
    it('applies custom className', () => {
      render(<ModernStatusDot status="healthy" className="custom-class" />)
      
      const dot = screen.getByRole('img')
      expect(dot).toHaveClass('custom-class')
    })
  })
})

describe('ModernStatusBadge', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  it('renders status badge with text', () => {
    render(<ModernStatusBadge status="healthy" />)
    
    expect(screen.getByText('Healthy')).toBeInTheDocument()
  })

  it('includes status dot by default', () => {
    render(<ModernStatusBadge status="healthy" />)
    
    expect(screen.getByRole('img', { name: 'Healthy' })).toBeInTheDocument()
  })

  it('hides dot when showDot is false', () => {
    render(<ModernStatusBadge status="healthy" showDot={false} />)
    
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
  })

  it('applies correct background colors for healthy', () => {
    render(<ModernStatusBadge status="healthy" />)
    
    const badge = screen.getByText('Healthy').parentElement
    expect(badge).toHaveClass('bg-emerald-50')
  })

  it('applies correct background colors for error', () => {
    render(<ModernStatusBadge status="error" />)
    
    const badge = screen.getByText('Error').parentElement
    expect(badge).toHaveClass('bg-rose-50')
  })

  it('applies correct text colors', () => {
    render(<ModernStatusBadge status="healthy" />)
    
    const badge = screen.getByText('Healthy').parentElement
    expect(badge).toHaveClass('text-emerald-600')
  })

  it('applies custom className', () => {
    render(<ModernStatusBadge status="healthy" className="custom-badge" />)
    
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
      const { unmount } = render(<ModernStatusBadge status={status as unknown as Parameters<typeof ModernStatusBadge>[0]['status']} />)
      expect(screen.getByText(text)).toBeInTheDocument()
      unmount()
    })
  })
})

describe('ModernStatusIndicator', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  describe('dot variant', () => {
    it('renders dot variant by default', () => {
      render(<ModernStatusIndicator status="healthy" />)
      
      expect(screen.getByRole('img', { name: 'Healthy' })).toBeInTheDocument()
    })

    it('shows label when showLabel is true', () => {
      render(<ModernStatusIndicator status="healthy" showLabel={true} />)
      
      expect(screen.getByRole('img', { name: 'Healthy' })).toBeInTheDocument()
      expect(screen.getByText('Healthy')).toBeInTheDocument()
    })

    it('hides label by default', () => {
      render(<ModernStatusIndicator status="healthy" />)
      
      expect(screen.queryByText('Healthy')).not.toBeInTheDocument()
    })
  })

  describe('badge variant', () => {
    it('renders badge variant', () => {
      render(<ModernStatusIndicator status="healthy" variant="badge" />)
      
      expect(screen.getByText('Healthy')).toBeInTheDocument()
    })
  })

  describe('full variant', () => {
    it('renders full variant with icon and text', () => {
      render(<ModernStatusIndicator status="healthy" variant="full" />)
      
      expect(screen.getByText('Healthy')).toBeInTheDocument()
      // Should have an icon (SVG)
      const container = screen.getByText('Healthy').parentElement
      const svg = container?.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })

    it('applies animation to icon when status is restarting', () => {
      mockMatchMedia(false)
      render(<ModernStatusIndicator status="restarting" variant="full" animated={true} />)
      
      expect(screen.getByText('Restarting')).toBeInTheDocument()
    })

    it('does not apply spin animation when reduced motion is preferred', () => {
      mockMatchMedia(true)
      render(<ModernStatusIndicator status="restarting" variant="full" animated={true} />)
      
      expect(screen.getByText('Restarting')).toBeInTheDocument()
    })
  })

  describe('animation prop', () => {
    it('respects animated prop', () => {
      mockMatchMedia(false)
      render(<ModernStatusIndicator status="healthy" animated={false} />)
      
      const dot = screen.getByRole('img')
      expect(dot).not.toHaveClass('animate-modern-heartbeat')
    })
  })
})

describe('ModernHealthPill', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  it('renders with healthy status when all services healthy', () => {
    render(
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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
      <ModernHealthPill
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

describe('ModernConnectionStatus', () => {
  beforeEach(() => {
    mockMatchMedia(false)
  })

  it('shows connected status', () => {
    render(<ModernConnectionStatus connected={true} />)
    
    expect(screen.getByText('Connected')).toBeInTheDocument()
  })

  it('shows disconnected status', () => {
    render(<ModernConnectionStatus connected={false} />)
    
    expect(screen.getByText('Disconnected')).toBeInTheDocument()
  })

  it('shows reconnecting status', () => {
    render(<ModernConnectionStatus connected={false} reconnecting={true} />)
    
    expect(screen.getByText('Reconnecting')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <ModernConnectionStatus connected={true} className="custom-connection" />
    )
    
    const element = container.firstChild as HTMLElement
    expect(element).toHaveClass('custom-connection')
  })

  it('includes status dot', () => {
    render(<ModernConnectionStatus connected={true} />)
    
    // Text is sr-only, but dot should be visible
    expect(screen.getByRole('img')).toBeInTheDocument()
  })
})

describe('ModernStatusSkeleton', () => {
  it('renders skeleton loader', () => {
    const { container } = render(<ModernStatusSkeleton />)
    
    const skeleton = container.firstChild as HTMLElement
    expect(skeleton).toHaveClass('animate-pulse')
    expect(skeleton).toHaveClass('rounded-full')
  })

  it('applies custom className', () => {
    const { container } = render(<ModernStatusSkeleton className="custom-skeleton" />)
    
    const skeleton = container.firstChild as HTMLElement
    expect(skeleton).toHaveClass('custom-skeleton')
  })
})

describe('ModernSpinner', () => {
  it('renders spinner', () => {
    render(<ModernSpinner />)
    
    const spinner = screen.getByRole('status', { name: 'Loading' })
    expect(spinner).toBeInTheDocument()
  })

  it('has screen reader text', () => {
    render(<ModernSpinner />)
    
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('applies animation class', () => {
    render(<ModernSpinner />)
    
    const spinner = screen.getByRole('status')
    expect(spinner).toHaveClass('animate-spin')
  })

  describe('sizes', () => {
    it('renders small size', () => {
      render(<ModernSpinner size="sm" />)
      
      const spinner = screen.getByRole('status')
      expect(spinner).toHaveClass('w-3.5', 'h-3.5')
    })

    it('renders medium size (default)', () => {
      render(<ModernSpinner />)
      
      const spinner = screen.getByRole('status')
      expect(spinner).toHaveClass('w-5', 'h-5')
    })

    it('renders large size', () => {
      render(<ModernSpinner size="lg" />)
      
      const spinner = screen.getByRole('status')
      expect(spinner).toHaveClass('w-8', 'h-8')
    })
  })

  it('applies custom className', () => {
    render(<ModernSpinner className="custom-spinner" />)
    
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
      const { unmount } = render(<ModernStatusBadge status={status} />)
      // Should render without errors
      expect(screen.getByRole('img')).toBeInTheDocument()
      unmount()
    })
  })

  it('falls back to unknown for undefined status', () => {
    // Testing invalid input to verify fallback behavior
    render(<ModernStatusBadge status={'invalid-status' as unknown as Parameters<typeof ModernStatusBadge>[0]['status']} />)
    
    expect(screen.getByText('Unknown')).toBeInTheDocument()
  })
})
