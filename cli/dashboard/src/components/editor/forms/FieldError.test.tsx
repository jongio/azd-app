/**
 * FieldError Component Tests
 */

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { FieldError } from './FieldError'

describe('FieldError', () => {
  it('renders error message', () => {
    render(<FieldError message="This field is required" />)
    expect(screen.getByText('This field is required')).toBeInTheDocument()
  })

  it('has alert role for accessibility', () => {
    render(<FieldError message="Error message" />)
    const errorElement = screen.getByRole('alert')
    expect(errorElement).toBeInTheDocument()
  })

  it('has aria-live attribute for screen readers', () => {
    render(<FieldError message="Error message" />)
    const errorElement = screen.getByRole('alert')
    expect(errorElement).toHaveAttribute('aria-live', 'polite')
  })

  it('displays error icon', () => {
    render(<FieldError message="Error message" />)
    // Icon should be present (AlertCircle from lucide-react)
    const errorElement = screen.getByRole('alert')
    expect(errorElement.querySelector('svg')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <FieldError message="Error message" className="custom-error-class" />
    )
    expect(container.firstChild).toHaveClass('custom-error-class')
  })

  it('has destructive text color styling', () => {
    const { container } = render(<FieldError message="Error message" />)
    expect(container.firstChild).toHaveClass('text-destructive')
  })
})
