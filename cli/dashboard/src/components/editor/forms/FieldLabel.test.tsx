/**
 * FieldLabel Component Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FieldLabel } from './FieldLabel'

describe('FieldLabel', () => {
  it('renders label text', () => {
    render(<FieldLabel htmlFor="test-field" label="Test Field" />)
    expect(screen.getByText('Test Field')).toBeInTheDocument()
  })

  it('associates label with input via htmlFor', () => {
    render(
      <>
        <FieldLabel htmlFor="test-field" label="Test Field" />
        <input id="test-field" />
      </>
    )
    const label = screen.getByText('Test Field')
    expect(label).toHaveAttribute('for', 'test-field')
  })

  it('shows required indicator when required is true', () => {
    render(<FieldLabel htmlFor="test-field" label="Test Field" required />)
    const requiredIndicator = screen.getByLabelText('required')
    expect(requiredIndicator).toBeInTheDocument()
    expect(requiredIndicator).toHaveTextContent('*')
  })

  it('does not show required indicator when required is false', () => {
    render(<FieldLabel htmlFor="test-field" label="Test Field" required={false} />)
    expect(screen.queryByLabelText('required')).not.toBeInTheDocument()
  })

  it('shows help tooltip when description is provided', async () => {
    const user = userEvent.setup()
    render(
      <FieldLabel
        htmlFor="test-field"
        label="Test Field"
        description="This is a helpful description"
      />
    )
    
    const helpButton = screen.getByLabelText('Help')
    expect(helpButton).toBeInTheDocument()
    
    await user.hover(helpButton)
    
    // Wait for tooltip to appear
    await screen.findByText('This is a helpful description', undefined, { timeout: 2000 })
  })

  it('hides help icon when showHelpIcon is false', () => {
    render(
      <FieldLabel
        htmlFor="test-field"
        label="Test Field"
        description="This is a helpful description"
        showHelpIcon={false}
      />
    )
    
    expect(screen.queryByLabelText('Help')).not.toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <FieldLabel htmlFor="test-field" label="Test Field" className="custom-class" />
    )
    expect(container.firstChild).toHaveClass('custom-class')
  })
})
