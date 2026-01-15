/**
 * NavigationItem Component Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FileText } from 'lucide-react'
import { NavigationItem } from './NavigationItem'

describe('NavigationItem', () => {
  const defaultProps = {
    label: 'Test Item',
    depth: 0,
    isActive: false,
    onClick: vi.fn(),
  }

  it('should render label', () => {
    render(<NavigationItem {...defaultProps} />)
    expect(screen.getByText('Test Item')).toBeInTheDocument()
  })

  it('should render icon when provided', () => {
    render(<NavigationItem {...defaultProps} icon={FileText} />)
    const button = screen.getByRole('treeitem')
    expect(button.querySelector('svg')).toBeInTheDocument()
  })

  it('should apply active styles when isActive is true', () => {
    render(<NavigationItem {...defaultProps} isActive={true} />)
    const button = screen.getByRole('treeitem')
    expect(button).toHaveAttribute('aria-current', 'page')
  })

  it('should show expand chevron when hasChildren is true', () => {
    const { container } = render(
      <NavigationItem
        {...defaultProps}
        hasChildren={true}
        isExpanded={false}
        onToggle={vi.fn()}
      />
    )
    
    // Chevron is visual-only (aria-hidden), check for ChevronRight icon
    const treeitem = screen.getByRole('treeitem')
    expect(treeitem).toHaveAttribute('aria-expanded', 'false')
    expect(container.querySelector('svg')).toBeInTheDocument()
  })

  it('should show collapse chevron when expanded', () => {
    const { container } = render(
      <NavigationItem
        {...defaultProps}
        hasChildren={true}
        isExpanded={true}
        onToggle={vi.fn()}
      />
    )
    
    // Chevron is visual-only (aria-hidden), check aria-expanded state
    const treeitem = screen.getByRole('treeitem')
    expect(treeitem).toHaveAttribute('aria-expanded', 'true')
    expect(container.querySelector('svg')).toBeInTheDocument()
  })

  it('should call onClick when clicked', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    
    render(<NavigationItem {...defaultProps} onClick={onClick} />)
    
    // Click the button inside the treeitem
    const button = screen.getByRole('treeitem').querySelector('button')
    expect(button).toBeInTheDocument()
    await user.click(button!)
    expect(onClick).toHaveBeenCalled()
  })

  it('should call onClick when item with children is clicked', async () => {
    const user = userEvent.setup()
    const onToggle = vi.fn()
    const onClick = vi.fn()
    
    render(
      <NavigationItem
        {...defaultProps}
        hasChildren={true}
        isExpanded={false}
        onClick={onClick}
        onToggle={onToggle}
      />
    )
    
    // Current design: single button handles onClick (no separate toggle)
    const button = screen.getByRole('treeitem').querySelector('button')
    await user.click(button!)
    expect(onClick).toHaveBeenCalled()
    // Note: onToggle is not called in current implementation
  })

  it('should display error badge when errorCount > 0', () => {
    render(<NavigationItem {...defaultProps} errorCount={2} />)
    expect(screen.getByLabelText('2 errors')).toBeInTheDocument()
  })

  it('should display warning badge when warningCount > 0', () => {
    render(<NavigationItem {...defaultProps} warningCount={3} />)
    expect(screen.getByLabelText('3 warnings')).toBeInTheDocument()
  })

  it('should display both error and warning badges', () => {
    render(<NavigationItem {...defaultProps} errorCount={1} warningCount={2} />)
    expect(screen.getByLabelText('1 error')).toBeInTheDocument()
    expect(screen.getByLabelText('2 warnings')).toBeInTheDocument()
  })

  it('should apply depth-based indentation', () => {
    const { rerender } = render(<NavigationItem {...defaultProps} depth={0} />)
    let treeitem = screen.getByRole('treeitem')
    let button = treeitem.querySelector('button')
    expect(button).toHaveStyle({ paddingLeft: '12px' })

    rerender(<NavigationItem {...defaultProps} depth={1} />)
    treeitem = screen.getByRole('treeitem')
    button = treeitem.querySelector('button')
    expect(button).toHaveStyle({ paddingLeft: '24px' })

    rerender(<NavigationItem {...defaultProps} depth={2} />)
    treeitem = screen.getByRole('treeitem')
    button = treeitem.querySelector('button')
    expect(button).toHaveStyle({ paddingLeft: '36px' })
  })

  it('should set aria-expanded when hasChildren is true', () => {
    render(
      <NavigationItem
        {...defaultProps}
        hasChildren={true}
        isExpanded={true}
      />
    )
    
    expect(screen.getByRole('treeitem')).toHaveAttribute('aria-expanded', 'true')
  })

  it('should include issues in aria-label', () => {
    render(
      <NavigationItem
        {...defaultProps}
        label="Service API"
        errorCount={1}
        warningCount={2}
      />
    )
    
    const button = screen.getByLabelText('Service API (1 errors, 2 warnings)')
    expect(button).toBeInTheDocument()
  })
})
