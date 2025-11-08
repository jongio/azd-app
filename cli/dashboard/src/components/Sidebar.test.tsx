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
    expect(resourcesButton).toHaveClass('text-purple-400')
    expect(resourcesButton).toHaveClass('bg-purple-500/15')
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
    
    // Inactive items should have gray text color
    expect(consoleButton).toHaveClass('text-gray-500')
  })

  it('should render all 7 navigation items', () => {
    const onViewChange = vi.fn()
    render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(7)
  })

  it('should update active state when activeView prop changes', () => {
    const onViewChange = vi.fn()
    const { rerender } = render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    // Resources should be active
    let resourcesButton = screen.getByRole('button', { name: /resources/i })
    expect(resourcesButton).toHaveClass('text-purple-400')

    // Change to console
    rerender(<Sidebar activeView="console" onViewChange={onViewChange} />)

    // Console should now be active
    const consoleButton = screen.getByRole('button', { name: /console/i })
    expect(consoleButton).toHaveClass('text-purple-400')

    // Resources should no longer be active
    resourcesButton = screen.getByRole('button', { name: /resources/i })
    expect(resourcesButton).toHaveClass('text-gray-500')
  })

  it('should render with proper layout styling', () => {
    const onViewChange = vi.fn()
    const { container } = render(<Sidebar activeView="resources" onViewChange={onViewChange} />)

    const sidebar = container.firstChild
    expect(sidebar).toHaveClass('w-20')
    expect(sidebar).toHaveClass('bg-[#0d0d0d]')
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
})
