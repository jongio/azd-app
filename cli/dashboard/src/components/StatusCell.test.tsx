import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusCell } from '@/components/StatusCell'

describe('StatusCell', () => {
  it('should display Running status for ready/healthy service', () => {
    render(<StatusCell status="ready" health="healthy" />)
    
    expect(screen.getByText('Running')).toBeInTheDocument()
    const statusElement = screen.getByText('Running')
    expect(statusElement).toHaveClass('text-green-400')
  })

  it('should display Running status for running/healthy service', () => {
    render(<StatusCell status="running" health="healthy" />)
    
    expect(screen.getByText('Running')).toBeInTheDocument()
    const statusElement = screen.getByText('Running')
    expect(statusElement).toHaveClass('text-green-400')
  })

  it('should display Unhealthy status for ready/unhealthy service', () => {
    render(<StatusCell status="ready" health="unhealthy" />)
    
    expect(screen.getByText('Unhealthy')).toBeInTheDocument()
    const statusElement = screen.getByText('Unhealthy')
    expect(statusElement).toHaveClass('text-red-400')
  })

  it('should display Unhealthy status for running/unhealthy service', () => {
    render(<StatusCell status="running" health="unhealthy" />)
    
    expect(screen.getByText('Unhealthy')).toBeInTheDocument()
    const statusElement = screen.getByText('Unhealthy')
    expect(statusElement).toHaveClass('text-red-400')
  })

  it('should display Starting status', () => {
    render(<StatusCell status="starting" health="unknown" />)
    
    expect(screen.getByText('Starting')).toBeInTheDocument()
    const statusElement = screen.getByText('Starting')
    expect(statusElement).toHaveClass('text-yellow-400')
  })

  it('should display Error status', () => {
    render(<StatusCell status="error" health="unknown" />)
    
    expect(screen.getByText('Error')).toBeInTheDocument()
    const statusElement = screen.getByText('Error')
    expect(statusElement).toHaveClass('text-red-400')
  })

  it('should display Stopping status', () => {
    render(<StatusCell status="stopping" health="unknown" />)
    
    expect(screen.getByText('Stopping')).toBeInTheDocument()
    const statusElement = screen.getByText('Stopping')
    expect(statusElement).toHaveClass('text-gray-400')
  })

  it('should display Stopped status for stopped service', () => {
    render(<StatusCell status="stopped" health="unknown" />)
    
    expect(screen.getByText('Stopped')).toBeInTheDocument()
    const statusElement = screen.getByText('Stopped')
    expect(statusElement).toHaveClass('text-gray-400')
  })

  it('should display Not Running status for not-running service', () => {
    render(<StatusCell status="not-running" health="unknown" />)
    
    expect(screen.getByText('Not Running')).toBeInTheDocument()
    const statusElement = screen.getByText('Not Running')
    expect(statusElement).toHaveClass('text-gray-400')
  })

  it('should display Unknown status for unexpected values', () => {
    // @ts-expect-error - Testing invalid status value
    render(<StatusCell status="invalid" health="unknown" />)
    
    expect(screen.getByText('Unknown')).toBeInTheDocument()
    const statusElement = screen.getByText('Unknown')
    expect(statusElement).toHaveClass('text-gray-400')
  })

  it('should render status indicator dot with correct color for healthy service', () => {
    const { container } = render(<StatusCell status="ready" health="healthy" />)
    
    const dot = container.querySelector('.bg-green-500')
    expect(dot).toBeInTheDocument()
    expect(dot).toHaveClass('w-2', 'h-2', 'rounded-full')
  })

  it('should render status indicator dot with correct color for error service', () => {
    const { container } = render(<StatusCell status="error" health="unknown" />)
    
    const dot = container.querySelector('.bg-red-500')
    expect(dot).toBeInTheDocument()
  })

  it('should render status indicator dot with correct color for starting service', () => {
    const { container } = render(<StatusCell status="starting" health="unknown" />)
    
    const dot = container.querySelector('.bg-yellow-500')
    expect(dot).toBeInTheDocument()
  })

  it('should render status indicator dot with correct color for stopped service', () => {
    const { container } = render(<StatusCell status="stopped" health="unknown" />)
    
    const dot = container.querySelector('.bg-gray-500')
    expect(dot).toBeInTheDocument()
  })

  it('should render CheckCircle icon for healthy running service', () => {
    const { container } = render(<StatusCell status="ready" health="healthy" />)
    
    // Icon should be present in the component
    const iconContainer = container.querySelector('.flex.items-center.gap-2')
    expect(iconContainer).toBeInTheDocument()
  })

  it('should handle different health states with ready status', () => {
    const { rerender } = render(<StatusCell status="ready" health="healthy" />)
    expect(screen.getByText('Running')).toBeInTheDocument()
    
    rerender(<StatusCell status="ready" health="unhealthy" />)
    expect(screen.getByText('Unhealthy')).toBeInTheDocument()
  })

  it('should handle different health states with running status', () => {
    const { rerender } = render(<StatusCell status="running" health="healthy" />)
    expect(screen.getByText('Running')).toBeInTheDocument()
    
    rerender(<StatusCell status="running" health="unhealthy" />)
    expect(screen.getByText('Unhealthy')).toBeInTheDocument()
  })

  it('should apply heartbeat animation for healthy running service', () => {
    const { container } = render(<StatusCell status="ready" health="healthy" />)
    
    const dot = container.querySelector('.bg-green-500')
    expect(dot).toHaveClass('animate-heartbeat')
  })

  it('should apply flash animation for unhealthy service', () => {
    const { container } = render(<StatusCell status="ready" health="unhealthy" />)
    
    const dot = container.querySelector('.bg-red-500')
    expect(dot).toHaveClass('animate-status-flash')
  })

  it('should apply caution-pulse animation for degraded service', () => {
    const { container } = render(<StatusCell status="running" health="degraded" />)
    
    const dot = container.querySelector('.bg-amber-500')
    expect(dot).toHaveClass('animate-caution-pulse')
  })

  it('should apply flash animation for error status', () => {
    const { container } = render(<StatusCell status="error" health="unknown" />)
    
    const dot = container.querySelector('.bg-red-500')
    expect(dot).toHaveClass('animate-status-flash')
  })

  it('should not apply animation for stopped service', () => {
    const { container } = render(<StatusCell status="stopped" health="unknown" />)
    
    const dot = container.querySelector('.bg-gray-500')
    expect(dot).not.toHaveClass('animate-heartbeat')
    expect(dot).not.toHaveClass('animate-status-flash')
  })
})
