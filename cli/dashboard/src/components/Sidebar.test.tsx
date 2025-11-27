import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Sidebar } from '@/components/Sidebar'

describe('Sidebar', () => {
  it('should render all navigation items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    expect(screen.getByText('Resources')).toBeInTheDocument()
    expect(screen.getByText('Console')).toBeInTheDocument()
    expect(screen.getByText('Structured')).toBeInTheDocument()
    expect(screen.getByText('Traces')).toBeInTheDocument()
    expect(screen.getByText('Metrics')).toBeInTheDocument()
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

    // Click Traces
    await user.click(screen.getByRole('button', { name: /traces/i }))
    expect(onViewChange).toHaveBeenCalledWith('traces')

    // Click Metrics
    await user.click(screen.getByRole('button', { name: /metrics/i }))
    expect(onViewChange).toHaveBeenCalledWith('metrics')

    // Click Structured
    await user.click(screen.getByRole('button', { name: /structured/i }))
    expect(onViewChange).toHaveBeenCalledWith('structured')
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

    const labels = ['Resources', 'Console', 'Structured', 'Traces', 'Metrics']
    
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
    
    // Should have error ring class
    expect(consoleButton).toHaveClass('ring-2')
    expect(consoleButton).toHaveClass('ring-red-500/50')
    
    // Should have pulsing red dot indicator
    const errorDot = consoleButton.querySelector('.animate-pulse')
    expect(errorDot).toBeInTheDocument()
    expect(errorDot).toHaveClass('bg-red-500')
  })

  it('should not show error indicator on Console nav when hasActiveErrors is false', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={false} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Should NOT have error ring or pulsing dot
    expect(consoleButton).not.toHaveClass('ring-2')
    const errorDot = consoleButton.querySelector('.animate-pulse')
    expect(errorDot).not.toBeInTheDocument()
  })

  it('should not show error indicator on Console nav when it is active', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="console" onViewChange={onViewChange} hasActiveErrors={true} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    
    // Should NOT have error ring when active
    expect(consoleButton).not.toHaveClass('ring-2')
    expect(consoleButton).not.toHaveClass('ring-red-500/50')
    
    // But should still have the pulsing dot
    const errorDot = consoleButton.querySelector('.animate-pulse')
    expect(errorDot).toBeInTheDocument()
  })

  it('should only show error indicator on Console nav, not other nav items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} />)

    const resourcesButton = screen.getByRole('button', { name: /resources/i })
    const structuredButton = screen.getByRole('button', { name: /structured/i })
    const tracesButton = screen.getByRole('button', { name: /traces/i })
    const metricsButton = screen.getByRole('button', { name: /metrics/i })
    
    // None of these should have error indicators
    expect(resourcesButton.querySelector('.animate-pulse')).not.toBeInTheDocument()
    expect(structuredButton.querySelector('.animate-pulse')).not.toBeInTheDocument()
    expect(tracesButton.querySelector('.animate-pulse')).not.toBeInTheDocument()
    expect(metricsButton.querySelector('.animate-pulse')).not.toBeInTheDocument()
  })

  it('should have title attribute on error indicator', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} hasActiveErrors={true} />)

    const consoleButton = screen.getByRole('button', { name: /console/i })
    const errorDot = consoleButton.querySelector('.animate-pulse')
    
    expect(errorDot).toHaveAttribute('title', 'Active errors detected')
  })
})
