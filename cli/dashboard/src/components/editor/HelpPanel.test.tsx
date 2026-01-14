/**
 * HelpPanel Component Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HelpPanel } from './HelpPanel'

describe('HelpPanel', () => {
  it('renders nothing when closed', () => {
    const { container } = render(
      <HelpPanel isOpen={false} onClose={() => {}} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders help panel when open', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} />
    )
    expect(screen.getByText('Azure YAML Editor Help')).toBeInTheDocument()
  })

  it('shows section-specific content', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} section="services" />
    )
    expect(screen.getByText('Services')).toBeInTheDocument()
    expect(screen.getByText(/Services represent the individual components/)).toBeInTheDocument()
  })

  it('shows examples when available', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} section="services" />
    )
    expect(screen.getByText('Examples')).toBeInTheDocument()
    expect(screen.getByText('Basic web service')).toBeInTheDocument()
  })

  it('shows troubleshooting section when available', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} section="services" />
    )
    expect(screen.getByText('Common Issues')).toBeInTheDocument()
    expect(screen.getByText('Service fails to start')).toBeInTheDocument()
  })

  it('shows links when available', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} section="services" />
    )
    expect(screen.getByText('Learn More')).toBeInTheDocument()
    expect(screen.getByText('Service Configuration Reference')).toBeInTheDocument()
  })

  it('calls onClose when close button is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    
    render(
      <HelpPanel isOpen={true} onClose={onClose} />
    )
    
    const closeButton = screen.getByLabelText('Close help panel')
    await user.click(closeButton)
    
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('renders in modal mode', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} mode="modal" />
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('renders in sidebar mode by default', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} />
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('handles unknown sections gracefully', () => {
    render(
      <HelpPanel isOpen={true} onClose={() => {}} section="unknown-section" />
    )
    expect(screen.getByText('Unknown-section')).toBeInTheDocument()
  })
})
