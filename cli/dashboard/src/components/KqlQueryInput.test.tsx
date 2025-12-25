/**
 * KqlQueryInput Component Tests
 * 
 * Tests the KQL query input component including editing, execution,
 * reset functionality, and keyboard shortcuts.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { KqlQueryInput, type KqlQueryInputProps } from './KqlQueryInput'

describe('KqlQueryInput', () => {
  const defaultQuery = 'ContainerAppConsoleLogs_CL | where service == "api"'
  const defaultProps: KqlQueryInputProps = {
    value: defaultQuery,
    onChange: vi.fn(),
    onRunQuery: vi.fn(),
    defaultQuery,
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Basic Rendering', () => {
    it('renders collapsed by default', () => {
      render(<KqlQueryInput {...defaultProps} />)
      
      expect(screen.getByRole('button', { name: /Advanced Query/i })).toBeInTheDocument()
      expect(screen.queryByRole('textbox', { name: /KQL query input/i })).not.toBeInTheDocument()
    })

    it('renders expanded when isCollapsed is false', () => {
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toBeInTheDocument()
      expect(textarea).toHaveValue(defaultQuery)
    })

    it('shows Modified badge when query differs from default', () => {
      const customQuery = 'ContainerAppConsoleLogs_CL | where error'
      render(<KqlQueryInput {...defaultProps} value={customQuery} />)
      
      expect(screen.getByText('Modified')).toBeInTheDocument()
    })

    it('does not show Modified badge when query matches default', () => {
      render(<KqlQueryInput {...defaultProps} value={defaultQuery} defaultQuery={defaultQuery} />)
      
      expect(screen.queryByText('Modified')).not.toBeInTheDocument()
    })

    it('does not show Modified badge when query is empty', () => {
      render(<KqlQueryInput {...defaultProps} value="" />)
      
      expect(screen.queryByText('Modified')).not.toBeInTheDocument()
    })
  })

  describe('Collapse/Expand Functionality', () => {
    it('expands section when header is clicked', async () => {
      const user = userEvent.setup()
      render(<KqlQueryInput {...defaultProps} />)
      
      const header = screen.getByRole('button', { name: /Advanced Query/i })
      await user.click(header)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toBeInTheDocument()
    })

    it('collapses section when header is clicked again', async () => {
      const user = userEvent.setup()
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      const header = screen.getByRole('button', { name: /Advanced Query/i })
      await user.click(header)
      
      expect(screen.queryByRole('textbox', { name: /KQL query input/i })).not.toBeInTheDocument()
    })

    it('calls onCollapsedChange when controlled', async () => {
      const onCollapsedChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <KqlQueryInput 
          {...defaultProps} 
          isCollapsed={true} 
          onCollapsedChange={onCollapsedChange} 
        />
      )
      
      const header = screen.getByRole('button', { name: /Advanced Query/i })
      await user.click(header)
      
      expect(onCollapsedChange).toHaveBeenCalledWith(false)
    })

    it('header has correct aria-expanded state', async () => {
      const user = userEvent.setup()
      render(<KqlQueryInput {...defaultProps} />)
      
      const header = screen.getByRole('button', { name: /Advanced Query/i })
      expect(header).toHaveAttribute('aria-expanded', 'false')
      
      await user.click(header)
      expect(header).toHaveAttribute('aria-expanded', 'true')
    })
  })

  describe('Query Editing', () => {
    it('calls onChange when query text is edited', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onChange={onChange} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      await user.clear(textarea)
      await user.type(textarea, 'New query')
      
      expect(onChange).toHaveBeenCalled()
      expect(onChange).toHaveBeenLastCalledWith('New query')
    })

    it('displays placeholder text when empty', () => {
      render(<KqlQueryInput {...defaultProps} value="" isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toHaveAttribute('placeholder', expect.stringContaining('ContainerAppConsoleLogs_CL'))
    })

    it('accepts custom placeholder', () => {
      const customPlaceholder = 'Enter your custom KQL query'
      render(
        <KqlQueryInput 
          {...defaultProps} 
          value="" 
          placeholder={customPlaceholder} 
          isCollapsed={false} 
        />
      )
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toHaveAttribute('placeholder', customPlaceholder)
    })
  })

  describe('Run Query Functionality', () => {
    it('shows Run Query button when expanded', () => {
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      const runButton = screen.getByRole('button', { name: /Run Query/i })
      expect(runButton).toBeInTheDocument()
    })

    it('calls onRunQuery when Run Query button is clicked', async () => {
      const onRunQuery = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onRunQuery={onRunQuery} isCollapsed={false} />)
      
      const runButton = screen.getByRole('button', { name: /Run Query/i })
      await user.click(runButton)
      
      expect(onRunQuery).toHaveBeenCalledOnce()
    })

    it('calls onRunQuery with Ctrl+Enter', async () => {
      const onRunQuery = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onRunQuery={onRunQuery} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      await user.click(textarea)
      await user.keyboard('{Control>}{Enter}{/Control}')
      
      expect(onRunQuery).toHaveBeenCalledOnce()
    })

    it('calls onRunQuery with Cmd+Enter on Mac', async () => {
      const onRunQuery = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onRunQuery={onRunQuery} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      await user.click(textarea)
      await user.keyboard('{Meta>}{Enter}{/Meta}')
      
      expect(onRunQuery).toHaveBeenCalledOnce()
    })

    it('disables Run Query button when disabled', () => {
      render(<KqlQueryInput {...defaultProps} disabled={true} isCollapsed={false} />)
      
      const runButton = screen.getByRole('button', { name: /Run Query/i })
      expect(runButton).toBeDisabled()
    })

    it('does not call onRunQuery when disabled and Ctrl+Enter pressed', async () => {
      const onRunQuery = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onRunQuery={onRunQuery} disabled={true} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      await user.click(textarea)
      await user.keyboard('{Control>}{Enter}{/Control}')
      
      expect(onRunQuery).not.toHaveBeenCalled()
    })
  })

  describe('Reset Functionality', () => {
    it('shows Reset button when expanded', () => {
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      expect(resetButton).toBeInTheDocument()
    })

    it('calls onChange with defaultQuery when Reset is clicked', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      const customQuery = 'Custom query'
      
      render(
        <KqlQueryInput 
          {...defaultProps} 
          value={customQuery} 
          onChange={onChange} 
          isCollapsed={false} 
        />
      )
      
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      await user.click(resetButton)
      
      expect(onChange).toHaveBeenCalledWith(defaultQuery)
    })

    it('disables Reset button when query matches default', () => {
      render(<KqlQueryInput {...defaultProps} value={defaultQuery} isCollapsed={false} />)
      
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      expect(resetButton).toBeDisabled()
    })

    it('disables Reset button when query is empty', () => {
      render(<KqlQueryInput {...defaultProps} value="" isCollapsed={false} />)
      
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      expect(resetButton).toBeDisabled()
    })

    it('disables Reset button when disabled prop is true', () => {
      render(
        <KqlQueryInput 
          {...defaultProps} 
          value="custom" 
          disabled={true} 
          isCollapsed={false} 
        />
      )
      
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      expect(resetButton).toBeDisabled()
    })

    it('focuses textarea after reset', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <KqlQueryInput 
          {...defaultProps} 
          value="custom" 
          onChange={onChange} 
          isCollapsed={false} 
        />
      )
      
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      await user.click(resetButton)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toHaveFocus()
    })
  })

  describe('Disabled State', () => {
    it('disables textarea when disabled', () => {
      render(<KqlQueryInput {...defaultProps} disabled={true} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toBeDisabled()
    })

    it('disables all buttons when disabled', () => {
      render(<KqlQueryInput {...defaultProps} disabled={true} isCollapsed={false} />)
      
      const runButton = screen.getByRole('button', { name: /Run Query/i })
      const resetButton = screen.getByRole('button', { name: /Reset/i })
      
      expect(runButton).toBeDisabled()
      expect(resetButton).toBeDisabled()
    })
  })

  describe('Accessibility', () => {
    it('has accessible section with proper controls', () => {
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      const section = screen.getByRole('region', { hidden: true })
      expect(section).toHaveAttribute('id', 'kql-query-section')
    })

    it('header controls the section', () => {
      render(<KqlQueryInput {...defaultProps} />)
      
      const header = screen.getByRole('button', { name: /Advanced Query/i })
      expect(header).toHaveAttribute('aria-controls', 'kql-query-section')
    })

    it('shows keyboard shortcut hint', () => {
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      expect(screen.getByText(/Ctrl\+Enter/i)).toBeInTheDocument()
    })

    it('has accessible labels for all interactive elements', () => {
      render(<KqlQueryInput {...defaultProps} isCollapsed={false} />)
      
      expect(screen.getByRole('textbox', { name: /KQL query input/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Run Query/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Reset/i })).toBeInTheDocument()
    })
  })

  describe('Custom Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <KqlQueryInput {...defaultProps} className="custom-kql-class" />
      )
      
      const element = container.querySelector('.custom-kql-class')
      expect(element).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    it('handles multiline queries', async () => {
      const onChange = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onChange={onChange} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      await user.clear(textarea)
      await user.type(textarea, 'Line 1{Enter}Line 2{Enter}Line 3')
      
      expect(onChange).toHaveBeenLastCalledWith('Line 1\nLine 2\nLine 3')
    })

    it('prevents Enter key from submitting when Ctrl/Cmd not pressed', async () => {
      const onRunQuery = vi.fn()
      const user = userEvent.setup()
      
      render(<KqlQueryInput {...defaultProps} onRunQuery={onRunQuery} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      await user.click(textarea)
      await user.keyboard('{Enter}')
      
      expect(onRunQuery).not.toHaveBeenCalled()
    })

    it('handles very long queries', () => {
      const longQuery = 'ContainerAppConsoleLogs_CL'.repeat(100)
      render(<KqlQueryInput {...defaultProps} value={longQuery} isCollapsed={false} />)
      
      const textarea = screen.getByRole('textbox', { name: /KQL query input/i })
      expect(textarea).toHaveValue(longQuery)
    })
  })
})
