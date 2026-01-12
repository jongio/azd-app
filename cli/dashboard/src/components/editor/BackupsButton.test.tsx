/**
 * Backups Button Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BackupsButton } from './BackupsButton'

describe('BackupsButton', () => {
  it('renders with text and icon', () => {
    render(<BackupsButton onClick={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Manage backups' })).toBeInTheDocument()
    expect(screen.getByText('Backups')).toBeInTheDocument()
  })

  it('calls onClick when clicked', async () => {
    const onClick = vi.fn()
    const user = userEvent.setup()

    render(<BackupsButton onClick={onClick} />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('is disabled when disabled prop is true', () => {
    render(<BackupsButton onClick={vi.fn()} disabled={true} />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    expect(button).toBeDisabled()
  })

  it('is not disabled by default', () => {
    render(<BackupsButton onClick={vi.fn()} />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    expect(button).not.toBeDisabled()
  })

  it('applies custom className', () => {
    render(<BackupsButton onClick={vi.fn()} className="custom-class" />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    expect(button).toHaveClass('custom-class')
  })

  it('has correct accessibility attributes', () => {
    render(<BackupsButton onClick={vi.fn()} />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    expect(button).toHaveAttribute('aria-label', 'Manage backups')
  })

  it('has correct default styling', () => {
    render(<BackupsButton onClick={vi.fn()} />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    expect(button).toHaveClass('px-4', 'py-2', 'rounded-lg')
  })

  it('does not call onClick when disabled', async () => {
    const onClick = vi.fn()
    const user = userEvent.setup()

    render(<BackupsButton onClick={onClick} disabled={true} />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    expect(onClick).not.toHaveBeenCalled()
  })
})
