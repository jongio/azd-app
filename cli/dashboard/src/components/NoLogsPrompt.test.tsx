/**
 * Tests for NoLogsPrompt component
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { userEvent } from '@testing-library/user-event'
import { NoLogsPrompt } from './NoLogsPrompt'

describe('NoLogsPrompt', () => {
  it('should render service name and warning message', () => {
    render(<NoLogsPrompt serviceName="test-service" />)

    expect(screen.getByText('No logs for test-service')).toBeInTheDocument()
    expect(
      screen.getByText(/This could be because diagnostic settings are not configured/i)
    ).toBeInTheDocument()
  })

  it('should display warning icon', () => {
    render(<NoLogsPrompt serviceName="test-service" />)

    const icon = screen.getByRole('status').querySelector('svg')
    expect(icon).toBeInTheDocument()
  })

  it('should render diagnostic button when callback provided', () => {
    const onOpenDiagnostics = vi.fn()
    render(
      <NoLogsPrompt serviceName="test-service" onOpenDiagnostics={onOpenDiagnostics} />
    )

    const button = screen.getByRole('button', { name: /View diagnostic details/i })
    expect(button).toBeInTheDocument()
  })

  it('should not render diagnostic button when callback not provided', () => {
    render(<NoLogsPrompt serviceName="test-service" />)

    expect(
      screen.queryByRole('button', { name: /View diagnostic details/i })
    ).not.toBeInTheDocument()
  })

  it('should call onOpenDiagnostics when button clicked', async () => {
    const user = userEvent.setup()
    const onOpenDiagnostics = vi.fn()
    render(
      <NoLogsPrompt serviceName="test-service" onOpenDiagnostics={onOpenDiagnostics} />
    )

    const button = screen.getByRole('button', { name: /View diagnostic details/i })
    await user.click(button)

    expect(onOpenDiagnostics).toHaveBeenCalledTimes(1)
  })

  it('should have accessible status role', () => {
    render(<NoLogsPrompt serviceName="test-service" />)

    const status = screen.getByRole('status')
    expect(status).toHaveAttribute('aria-label', 'No logs available for test-service')
  })

  it('should mention all possible reasons for no logs', () => {
    render(<NoLogsPrompt serviceName="test-service" />)

    const explanation = screen.getByText(/This could be because/i)
    expect(explanation.textContent).toContain('diagnostic settings are not configured')
    expect(explanation.textContent).toContain('delay in log ingestion')
    expect(explanation.textContent).toContain("hasn't generated any activity")
  })
})
