import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { InfoField } from './InfoField'

describe('InfoField', () => {
  it('should render label and value', () => {
    render(<InfoField label="Name" value="Test Service" />)
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('Test Service')).toBeInTheDocument()
  })

  it('should not render when value is undefined', () => {
    const { container } = render(<InfoField label="Name" value={undefined} />)
    expect(container.firstChild).toBeNull()
  })

  it('should not render when value is null', () => {
    const { container } = render(<InfoField label="Name" value={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('should not render when value is empty string', () => {
    const { container } = render(<InfoField label="Name" value="" />)
    expect(container.firstChild).toBeNull()
  })

  it('should not show copy button by default', () => {
    render(<InfoField label="Name" value="Test" />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('should show copy button when copyable is true', () => {
    render(<InfoField label="Name" value="Test" copyable />)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('should not show copy button for empty values even when copyable', () => {
    const { container } = render(<InfoField label="Name" value={undefined} copyable />)
    expect(container.firstChild).toBeNull()
  })

  it('should copy value to clipboard when copy button is clicked', async () => {
    const mockWriteText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText,
      },
    })

    render(<InfoField label="Name" value="Test Value" copyable />)
    const copyButton = screen.getByRole('button')
    
    fireEvent.click(copyButton)
    
    // Wait for async clipboard operation
    await vi.waitFor(() => {
      expect(mockWriteText).toHaveBeenCalledWith('Test Value')
    })
  })

  it('should show check icon after copying', async () => {
    const mockWriteText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText,
      },
    })

    render(<InfoField label="Name" value="Test Value" copyable />)
    const copyButton = screen.getByRole('button')
    
    fireEvent.click(copyButton)
    
    // The icon should change to check (lucide-check class)
    await vi.waitFor(() => {
      const svg = copyButton.querySelector('svg')
      expect(svg).toHaveClass('lucide-check')
    })
  })

  it('should render as a link when value starts with http', () => {
    render(<InfoField label="URL" value="https://example.com" />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should render as link when link prop is true', () => {
    render(<InfoField label="URL" value="example.com" link />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'example.com')
  })

  it('should not render for empty values even when link is provided', () => {
    const { container } = render(<InfoField label="URL" value={undefined} link />)
    expect(container.firstChild).toBeNull()
  })

  it('should render with custom className', () => {
    const { container } = render(<InfoField label="Name" value="Test" className="custom-class" />)
    const element = container.firstChild
    expect(element).toHaveClass('custom-class')
  })
})
