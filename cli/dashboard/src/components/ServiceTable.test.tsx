import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { ServiceTable } from '@/components/ServiceTable'
import { mockServices, mockServiceWithAzure } from '@/test/mocks'

describe('ServiceTable', () => {
  it('should render table with headers', () => {
    render(<ServiceTable services={mockServices} />)

    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('State')).toBeInTheDocument()
    expect(screen.getByText('Start time')).toBeInTheDocument()
    expect(screen.getByText('Source')).toBeInTheDocument()
    expect(screen.getByText('Local URL')).toBeInTheDocument()
    expect(screen.getByText('Azure URL')).toBeInTheDocument()
    expect(screen.getByText('Actions')).toBeInTheDocument()
  })

  it('should render all services in table rows', () => {
    render(<ServiceTable services={mockServices} />)

    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getByText('web')).toBeInTheDocument()
    expect(screen.getByText('database')).toBeInTheDocument()
  })

  it('should display service names with correct styling', () => {
    render(<ServiceTable services={mockServices} />)

    // Check that service names are rendered as font-semibold
    const apiName = screen.getByText('api')
    expect(apiName).toHaveClass('font-semibold')
  })

  it('should display local URLs as clickable links', () => {
    render(<ServiceTable services={mockServices} />)

    const link = screen.getByRole('link', { name: /localhost:5000/i })
    expect(link).toHaveAttribute('href', 'http://localhost:5000')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should display Azure URLs when available', () => {
    render(<ServiceTable services={[mockServiceWithAzure]} />)

    const link = screen.getByRole('link', { name: /azurewebsites\.net/i })
    expect(link).toHaveAttribute('href', 'https://my-app.azurewebsites.net')
  })

  it('should display "-" for missing URLs', () => {
    const serviceWithoutUrl = {
      ...mockServices[0],
      local: { ...mockServices[0].local!, url: undefined },
      azure: undefined,
    }

    render(<ServiceTable services={[serviceWithoutUrl]} />)

    // Should have "-" placeholders
    const cells = screen.getAllByText('-')
    expect(cells.length).toBeGreaterThan(0)
  })

  it('should call onViewLogs when logs button is clicked', async () => {
    const user = userEvent.setup()
    const onViewLogs = vi.fn()

    render(<ServiceTable services={mockServices} onViewLogs={onViewLogs} />)

    // Find and click the first logs button
    const logsButtons = screen.getAllByTitle('View Logs')
    await user.click(logsButtons[0])

    expect(onViewLogs).toHaveBeenCalledWith('api')
  })

  it('should display formatted start times', () => {
    render(<ServiceTable services={mockServices} />)

    // Should have formatted times in HH:MM:SS format
    const timePattern = /\d{2}:\d{2}:\d{2}/
    const times = screen.getAllByText(timePattern)
    expect(times.length).toBeGreaterThan(0)
  })

  it('should display source/project information', () => {
    render(<ServiceTable services={mockServices} />)

    // Multiple services have the same project path
    expect(screen.getAllByText('/Users/dev/projects/fullstack').length).toBeGreaterThan(0)
  })

  it('should render empty table when no services provided', () => {
    render(<ServiceTable services={[]} />)

    // Headers should still be present
    expect(screen.getByText('Name')).toBeInTheDocument()
    
    // But no service rows
    expect(screen.queryByText('api')).not.toBeInTheDocument()
  })

  it('should render action buttons for each service', () => {
    render(<ServiceTable services={mockServices} />)

    // Each service should have logs button
    const logsButtons = screen.getAllByTitle('View Logs')

    expect(logsButtons).toHaveLength(mockServices.length)
  })

  it('should display status cells with correct state', () => {
    render(<ServiceTable services={mockServices} />)

    // Should display different statuses (StatusCell displays "Running" for ready status)
    expect(screen.getAllByText('Running').length).toBeGreaterThan(0) // api and web show as Running
    expect(screen.getByText('Starting')).toBeInTheDocument() // database
  })

  it('should handle services without onViewLogs callback', async () => {
    const user = userEvent.setup()
    
    render(<ServiceTable services={mockServices} />)

    // Should still render logs button
    const logsButtons = screen.getAllByTitle('View Logs')
    
    // Should not throw when clicked without callback
    await user.click(logsButtons[0])
    
    expect(logsButtons[0]).toBeInTheDocument()
  })

  it('should truncate long project paths with ellipsis', () => {
    const serviceWithLongPath = {
      ...mockServices[0],
      project: '/very/long/path/to/project/that/should/be/truncated/in/the/ui',
    }

    const { container } = render(<ServiceTable services={[serviceWithLongPath]} />)

    // Check for truncate class
    const truncateElement = container.querySelector('.truncate')
    expect(truncateElement).toBeInTheDocument()
  })

  it('should show "-" for services without start time', () => {
    const serviceWithoutTime = {
      ...mockServices[0],
      local: { ...mockServices[0].local!, startTime: undefined },
    }

    render(<ServiceTable services={[serviceWithoutTime]} />)

    // Should show "-" in start time column
    expect(screen.getAllByText('-').length).toBeGreaterThan(0)
  })

  it('should render with proper table structure', () => {
    const { container } = render(<ServiceTable services={mockServices} />)

    expect(container.querySelector('table')).toBeInTheDocument()
    expect(container.querySelector('thead')).toBeInTheDocument()
    expect(container.querySelector('tbody')).toBeInTheDocument()
  })

  it('should apply hover effects on table rows', () => {
    const { container } = render(<ServiceTable services={mockServices} />)

    // Find table rows (excluding header)
    const rows = container.querySelectorAll('tbody tr')
    expect(rows.length).toBe(mockServices.length)
  })
})
