/**
 * NavigationSearch Component Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { NavigationSearch } from './NavigationSearch'

describe('NavigationSearch', () => {
  const defaultProps = {
    value: '',
    onChange: vi.fn(),
    onClear: vi.fn(),
  }

  it('should render search input', () => {
    render(<NavigationSearch {...defaultProps} />)
    expect(screen.getByPlaceholderText(/Search/i)).toBeInTheDocument()
  })

  it('should display current value', () => {
    render(<NavigationSearch {...defaultProps} value="test query" />)
    expect(screen.getByDisplayValue('test query')).toBeInTheDocument()
  })

  it('should call onChange when typing', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    
    render(<NavigationSearch {...defaultProps} onChange={onChange} />)
    
    const input = screen.getByPlaceholderText(/Search/i)
    await user.type(input, 'api')
    
    expect(onChange).toHaveBeenCalled()
    // Each character triggers onChange
    expect(onChange.mock.calls.length).toBeGreaterThan(0)
  })

  it('should show clear button when value is not empty', () => {
    render(<NavigationSearch {...defaultProps} value="test" />)
    expect(screen.getByLabelText('Clear search')).toBeInTheDocument()
  })

  it('should not show clear button when value is empty', () => {
    render(<NavigationSearch {...defaultProps} value="" />)
    expect(screen.queryByLabelText('Clear search')).not.toBeInTheDocument()
  })

  it('should call onClear when clear button is clicked', async () => {
    const user = userEvent.setup()
    const onClear = vi.fn()
    
    render(<NavigationSearch {...defaultProps} value="test" onClear={onClear} />)
    
    await user.click(screen.getByLabelText('Clear search'))
    expect(onClear).toHaveBeenCalled()
  })

  it('should have search icon', () => {
    const { container } = render(<NavigationSearch {...defaultProps} />)
    const searchIcon = container.querySelector('svg')
    expect(searchIcon).toBeInTheDocument()
  })

  it('should have proper accessibility labels', () => {
    render(<NavigationSearch {...defaultProps} />)
    expect(screen.getByLabelText('Search navigation')).toBeInTheDocument()
  })

  it('should support keyboard shortcut hint in placeholder', () => {
    render(<NavigationSearch {...defaultProps} />)
    expect(screen.getByPlaceholderText(/Ctrl\+F/i)).toBeInTheDocument()
  })
})
