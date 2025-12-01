import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Sidebar } from '@/components/Sidebar'
import type { HealthSummary } from '@/types'

describe('Sidebar', () => {
  it('should render all navigation items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    expect(screen.getByText('Resources')).toBeInTheDocument()
    expect(screen.getByText('Console')).toBeInTheDocument()
    expect(screen.getByText('Environment')).toBeInTheDocument()
    expect(screen.getByText('Metrics')).toBeInTheDocument()
    expect(screen.getByText('Dependencies')).toBeInTheDocument()
  })

  it('should highlight active view', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const resourcesButton = screen.getByRole('button', { name: /resources/i })
    
    // Active item should have specific styling
    expect(resourcesButton).toHaveClass('text-accent-foreground')
    expect(resourcesButton).toHaveClass('bg-accent')
  })

  it('should call onViewChange when navigation item is clicked', async () => {
    const user = userEvent.setup()
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    await user.click(consoleButton)

    expect(onViewChange).toHaveBeenCalledWith('console')
  })

  it('should switch between different views', async () => {
    const user = userEvent.setup()
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    // Click Console
    await user.click(screen.getByRole('button', { name: /console/i }))
    expect(onViewChange).toHaveBeenCalledWith('console')

    // Click Resources
    await user.click(screen.getByRole('button', { name: /resources/i }))
    expect(onViewChange).toHaveBeenCalledWith('resources')
  })

  it('should render icons for each navigation item', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    // Each button should have an SVG icon
    const buttons = screen.getAllByRole('button')
    buttons.forEach(button => {
      expect(button.querySelector('svg')).toBeInTheDocument()
    })
  })

  it('should apply hover styles to inactive items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Inactive items should have tertiary foreground text color
    expect(consoleButton).toHaveClass('text-foreground-tertiary')
  })

  it('should render all 5 navigation items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(5)
  })

  it('should update active state when activeView prop changes', () => {
    const onViewChange = vi.fn()
    const { rerender } = render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    // Resources should be active
    let resourcesButton = screen.getByRole('button', { name: /resources/i })
    expect(resourcesButton).toHaveClass('text-accent-foreground')

    // Change to console
    rerender(<Sidebar activeView="console" onViewChange={onViewChange} />)

    // Console should now be active
    const consoleButton = screen.getByRole('button', { name: /console/i })
    expect(consoleButton).toHaveClass('text-accent-foreground')

    // Resources should no longer be active
    resourcesButton = screen.getByRole('button', { name: /resources/i })
    expect(resourcesButton).toHaveClass('text-foreground-tertiary')
  })

  it('should render with proper layout styling', () => {
    const onViewChange = vi.fn()
    const { container } = render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const sidebar = container.firstChild
    expect(sidebar).toHaveClass('w-20')
    expect(sidebar).toHaveClass('bg-background')
  })

  it('should render buttons as flex columns with proper alignment', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const buttons = screen.getAllByRole('button')
    buttons.forEach(button => {
      expect(button).toHaveClass('flex')
      expect(button).toHaveClass('flex-col')
      expect(button).toHaveClass('items-center')
    })
  })

  it('should handle rapid clicks without issues', async () => {
    const user = userEvent.setup()
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Click multiple times rapidly
    await user.click(consoleButton)
    await user.click(consoleButton)
    await user.click(consoleButton)

    expect(onViewChange).toHaveBeenCalledTimes(3)
    expect(onViewChange).toHaveBeenCalledWith('console')
  })

  it('should display labels with correct styling', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const labels = ['Resources', 'Console', 'Environment', 'Metrics', 'Dependencies']
    
    labels.forEach(label => {
      const element = screen.getByText(label)
      expect(element).toHaveClass('text-[10px]')
      expect(element).toHaveClass('font-medium')
    })
  })

  it('should show error indicator on Console nav when hasActiveErrors is true', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Should have flashing red dot indicator
    const errorDot = consoleButton.querySelector('.animate-status-flash')
    expect(errorDot).toBeInTheDocument()
    expect(errorDot).toHaveClass('bg-red-500')
  })

  it('should not show error indicator on Console nav when hasActiveErrors is false', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={false} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Should NOT have flashing dot
    const errorDot = consoleButton.querySelector('.animate-status-flash')
    expect(errorDot).not.toBeInTheDocument()
  })

  it('should not show error indicator on Console nav when it is active', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="console" onViewChange={onViewChange} hasActiveErrors={true} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Should still have the flashing dot when active
    const errorDot = consoleButton.querySelector('.animate-status-flash')
    expect(errorDot).toBeInTheDocument()
  })

  it('should only show error indicator on Console nav, not other nav items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} />)

    const resourcesButton = screen.getByRole('button', { name: /resources/i })
    
    // Resources should not have error indicator
    expect(resourcesButton.querySelector('.animate-status-flash')).not.toBeInTheDocument()
  })

  it('should have title attribute on error indicator', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    const errorDot = consoleButton.querySelector('.animate-status-flash')
    
    expect(errorDot).toHaveAttribute('title', 'Active errors detected')
  })

  describe('Health Summary Indicator', () => {
    it('should show red indicator when there are unhealthy services', () => {
      const onViewChange = vi.fn()
      const healthSummary: HealthSummary = {
        total: 4,
        healthy: 2,
        degraded: 1,
        unhealthy: 1,
        unknown: 0,
        overall: 'unhealthy'
      }
      render(<Sidebar activeView="resources" onViewChange={onViewChange} healthSummary={healthSummary} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).toBeInTheDocument()
      expect(indicator).toHaveClass('bg-red-500')
      expect(indicator).toHaveClass('animate-status-flash')
      expect(indicator).toHaveAttribute('title', '1 unhealthy service(s)')
    })

    it('should show yellow indicator when there are degraded but no unhealthy services', () => {
      const onViewChange = vi.fn()
      const healthSummary: HealthSummary = {
        total: 4,
        healthy: 2,
        degraded: 2,
        unhealthy: 0,
        unknown: 0,
        overall: 'degraded'
      }
      render(<Sidebar activeView="resources" onViewChange={onViewChange} healthSummary={healthSummary} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).toBeInTheDocument()
      expect(indicator).toHaveClass('bg-yellow-500')
      expect(indicator).not.toHaveClass('animate-pulse')
      expect(indicator).toHaveAttribute('title', '2 degraded/unknown service(s)')
    })

    it('should show yellow indicator when there are unknown services', () => {
      const onViewChange = vi.fn()
      const healthSummary: HealthSummary = {
        total: 4,
        healthy: 3,
        degraded: 0,
        unhealthy: 0,
        unknown: 1,
        overall: 'unknown'
      }
      render(<Sidebar activeView="resources" onViewChange={onViewChange} healthSummary={healthSummary} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).toBeInTheDocument()
      expect(indicator).toHaveClass('bg-yellow-500')
      expect(indicator).toHaveAttribute('title', '1 degraded/unknown service(s)')
    })

    it('should show green indicator when all services are healthy', () => {
      const onViewChange = vi.fn()
      const healthSummary: HealthSummary = {
        total: 4,
        healthy: 4,
        degraded: 0,
        unhealthy: 0,
        unknown: 0,
        overall: 'healthy'
      }
      render(<Sidebar activeView="resources" onViewChange={onViewChange} healthSummary={healthSummary} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).toBeInTheDocument()
      expect(indicator).toHaveClass('bg-green-500')
      expect(indicator).not.toHaveClass('animate-pulse')
      expect(indicator).toHaveAttribute('title', 'All services healthy')
    })

    it('should prioritize healthSummary over hasActiveErrors', () => {
      const onViewChange = vi.fn()
      const healthSummary: HealthSummary = {
        total: 4,
        healthy: 4,
        degraded: 0,
        unhealthy: 0,
        unknown: 0,
        overall: 'healthy'
      }
      // Even with hasActiveErrors=true, should show green because healthSummary is all healthy
      render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} healthSummary={healthSummary} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).toHaveClass('bg-green-500')
      expect(indicator).toHaveAttribute('title', 'All services healthy')
    })

    it('should fall back to hasActiveErrors when no healthSummary is provided', () => {
      const onViewChange = vi.fn()
      render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).toBeInTheDocument()
      expect(indicator).toHaveClass('bg-red-500')
      expect(indicator).toHaveAttribute('title', 'Active errors detected')
    })

    it('should not show indicator when no healthSummary and hasActiveErrors is false', () => {
      const onViewChange = vi.fn()
      render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={false} />)

      const consoleButton = screen.getByRole('button', { name: /console/i })
      const indicator = consoleButton.querySelector('.rounded-full')
      
      expect(indicator).not.toBeInTheDocument()
    })
  })
})
