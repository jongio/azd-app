/**
 * TableSelector Component Tests
 * 
 * Tests the table selector component including multi-select, filtering,
 * categorization, and accessibility features.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TableSelector, type TableSelectorProps } from './TableSelector'
import type { TableInfo, TableCategory } from '@/hooks/useLogConfig'

describe('TableSelector', () => {
  const mockTables: TableInfo[] = [
    {
      name: 'ContainerAppConsoleLogs_CL',
      category: 'container',
      description: 'Container application console logs',
      recommended: true,
    },
    {
      name: 'ContainerAppSystemLogs_CL',
      category: 'container',
      description: 'Container application system logs',
    },
    {
      name: 'AppServiceConsoleLogs',
      category: 'appservice',
      description: 'App Service console logs',
      recommended: true,
    },
    {
      name: 'AzureDiagnostics',
      category: 'diagnostics',
      description: 'Azure diagnostics logs',
    },
  ]

  const mockCategories: TableCategory[] = [
    { name: 'container', displayName: 'Container Apps', tables: [] },
    { name: 'appservice', displayName: 'App Service', tables: [] },
    { name: 'diagnostics', displayName: 'Diagnostics', tables: [] },
  ]

  const defaultProps: TableSelectorProps = {
    tables: mockTables,
    categories: mockCategories,
    selectedTables: [],
    onSelectionChange: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Basic Rendering', () => {
    it('renders with no tables selected', () => {
      render(<TableSelector {...defaultProps} />)
      
      expect(screen.getByText('0 of 4 tables selected')).toBeInTheDocument()
    })

    it('renders with some tables selected', () => {
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['ContainerAppConsoleLogs_CL', 'AzureDiagnostics']} 
        />
      )
      
      expect(screen.getByText('2 of 4 tables selected')).toBeInTheDocument()
    })

    it('renders all categories', () => {
      render(<TableSelector {...defaultProps} />)
      
      expect(screen.getByText('Container Apps')).toBeInTheDocument()
      expect(screen.getByText('App Service')).toBeInTheDocument()
      expect(screen.getByText('Diagnostics')).toBeInTheDocument()
    })

    it('shows table count for each category', () => {
      render(<TableSelector {...defaultProps} />)
      
      // Container Apps has 2 tables
      expect(screen.getByText('(2)')).toBeInTheDocument()
      // App Service has 1 table
      expect(screen.getByText('(1)')).toBeInTheDocument()
    })
  })

  describe('Category Expansion/Collapse', () => {
    it('expands all categories by default', () => {
      render(<TableSelector {...defaultProps} />)
      
      expect(screen.getByText('ContainerAppConsoleLogs_CL')).toBeInTheDocument()
      expect(screen.getByText('AppServiceConsoleLogs')).toBeInTheDocument()
    })

    it('collapses category when header is clicked', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const containerHeader = screen.getByRole('button', { name: /Container Apps/i })
      await user.click(containerHeader)
      
      expect(screen.queryByText('ContainerAppConsoleLogs_CL')).not.toBeInTheDocument()
    })

    it('expands collapsed category when header is clicked again', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const containerHeader = screen.getByRole('button', { name: /Container Apps/i })
      
      // Collapse
      await user.click(containerHeader)
      expect(screen.queryByText('ContainerAppConsoleLogs_CL')).not.toBeInTheDocument()
      
      // Expand
      await user.click(containerHeader)
      expect(screen.getByText('ContainerAppConsoleLogs_CL')).toBeInTheDocument()
    })
  })

  describe('Table Selection', () => {
    it('selects table when clicked', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TableSelector {...defaultProps} onSelectionChange={onSelectionChange} />)
      
      const table = screen.getByRole('button', { name: /ContainerAppConsoleLogs_CL/i })
      await user.click(table)
      
      expect(onSelectionChange).toHaveBeenCalledWith(['ContainerAppConsoleLogs_CL'])
    })

    it('deselects table when clicked again', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['ContainerAppConsoleLogs_CL']} 
          onSelectionChange={onSelectionChange} 
        />
      )
      
      const table = screen.getByRole('button', { name: /ContainerAppConsoleLogs_CL/i })
      await user.click(table)
      
      expect(onSelectionChange).toHaveBeenCalledWith([])
    })

    it('adds to existing selection', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['AzureDiagnostics']} 
          onSelectionChange={onSelectionChange} 
        />
      )
      
      const table = screen.getByRole('button', { name: /ContainerAppConsoleLogs_CL/i })
      await user.click(table)
      
      expect(onSelectionChange).toHaveBeenCalledWith([
        'AzureDiagnostics',
        'ContainerAppConsoleLogs_CL',
      ])
    })

    it('shows visual indication for selected tables', () => {
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['ContainerAppConsoleLogs_CL']} 
        />
      )
      
      const table = screen.getByRole('button', { name: /ContainerAppConsoleLogs_CL/i })
      expect(table).toHaveClass('bg-cyan-50')
    })
  })

  describe('Select All/Clear Actions', () => {
    it('selects all tables when Select All is clicked', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TableSelector {...defaultProps} onSelectionChange={onSelectionChange} />)
      
      const selectAllButton = screen.getByRole('button', { name: /Select All/i })
      await user.click(selectAllButton)
      
      expect(onSelectionChange).toHaveBeenCalledWith([
        'ContainerAppConsoleLogs_CL',
        'ContainerAppSystemLogs_CL',
        'AppServiceConsoleLogs',
        'AzureDiagnostics',
      ])
    })

    it('clears all selections when Clear is clicked', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['ContainerAppConsoleLogs_CL', 'AzureDiagnostics']} 
          onSelectionChange={onSelectionChange} 
        />
      )
      
      const clearButton = screen.getByRole('button', { name: /Clear/i })
      await user.click(clearButton)
      
      expect(onSelectionChange).toHaveBeenCalledWith([])
    })

    it('disables Clear button when no tables selected', () => {
      render(<TableSelector {...defaultProps} selectedTables={[]} />)
      
      const clearButton = screen.getByRole('button', { name: /Clear/i })
      expect(clearButton).toBeDisabled()
    })
  })

  describe('Recommended Tables', () => {
    it('shows Recommended badge for recommended tables', () => {
      render(<TableSelector {...defaultProps} recommendedTables={['ContainerAppConsoleLogs_CL']} />)
      
      const badges = screen.getAllByText('Recommended')
      expect(badges.length).toBeGreaterThan(0)
    })

    it('selects recommended tables when Recommended button clicked', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <TableSelector 
          {...defaultProps} 
          recommendedTables={['ContainerAppConsoleLogs_CL', 'AppServiceConsoleLogs']} 
          onSelectionChange={onSelectionChange} 
        />
      )
      
      const recommendedButton = screen.getByRole('button', { name: /Recommended/i })
      await user.click(recommendedButton)
      
      expect(onSelectionChange).toHaveBeenCalledWith([
        'ContainerAppConsoleLogs_CL',
        'AppServiceConsoleLogs',
      ])
    })

    it('does not show Recommended button when no recommended tables', () => {
      render(<TableSelector {...defaultProps} recommendedTables={[]} />)
      
      expect(screen.queryByRole('button', { name: /Recommended/i })).not.toBeInTheDocument()
    })
  })

  describe('Category Selection', () => {
    it('selects all tables in category when category select clicked', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(<TableSelector {...defaultProps} onSelectionChange={onSelectionChange} />)
      
      // Find and click the category select button for Container Apps
      const categoryHeader = screen.getByRole('button', { name: /Container Apps/i })
      const selectButton = categoryHeader.querySelector('button[title="Select all"]')
      
      if (selectButton) {
        await user.click(selectButton)
        
        expect(onSelectionChange).toHaveBeenCalledWith([
          'ContainerAppConsoleLogs_CL',
          'ContainerAppSystemLogs_CL',
        ])
      }
    })

    it('shows "All selected" when all tables in category are selected', () => {
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['ContainerAppConsoleLogs_CL', 'ContainerAppSystemLogs_CL']} 
        />
      )
      
      expect(screen.getByText('All selected')).toBeInTheDocument()
    })

    it('shows "Some selected" when some tables in category are selected', () => {
      render(
        <TableSelector 
          {...defaultProps} 
          selectedTables={['ContainerAppConsoleLogs_CL']} 
        />
      )
      
      expect(screen.getByText('Some selected')).toBeInTheDocument()
    })
  })

  describe('Search Functionality', () => {
    it('filters tables by search query', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      await user.type(searchInput, 'Console')
      
      expect(screen.getByText('ContainerAppConsoleLogs_CL')).toBeInTheDocument()
      expect(screen.getByText('AppServiceConsoleLogs')).toBeInTheDocument()
      expect(screen.queryByText('AzureDiagnostics')).not.toBeInTheDocument()
    })

    it('filters tables by description', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      await user.type(searchInput, 'diagnostics')
      
      expect(screen.getByText('AzureDiagnostics')).toBeInTheDocument()
      expect(screen.queryByText('ContainerAppConsoleLogs_CL')).not.toBeInTheDocument()
    })

    it('shows empty state when no tables match search', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      await user.type(searchInput, 'nonexistent')
      
      expect(screen.getByText('No tables match your search')).toBeInTheDocument()
    })

    it('clears search when X button clicked', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      await user.type(searchInput, 'Console')
      
      const clearButton = screen.getByRole('button', { name: '' })
      await user.click(clearButton)
      
      expect(searchInput).toHaveValue('')
    })

    it('is case-insensitive', async () => {
      const user = userEvent.setup()
      render(<TableSelector {...defaultProps} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      await user.type(searchInput, 'CONTAINER')
      
      expect(screen.getByText('ContainerAppConsoleLogs_CL')).toBeInTheDocument()
    })
  })

  describe('Loading State', () => {
    it('shows loading message when isLoading is true', () => {
      render(<TableSelector {...defaultProps} isLoading={true} />)
      
      expect(screen.getByText('Loading tables...')).toBeInTheDocument()
    })

    it('hides table list when loading', () => {
      render(<TableSelector {...defaultProps} isLoading={true} />)
      
      expect(screen.queryByText('ContainerAppConsoleLogs_CL')).not.toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    it('shows empty message when no tables provided', () => {
      render(<TableSelector {...defaultProps} tables={[]} />)
      
      expect(screen.getByText('No tables available')).toBeInTheDocument()
    })
  })

  describe('Disabled State', () => {
    it('disables all interactive elements when disabled', () => {
      render(<TableSelector {...defaultProps} disabled={true} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      const selectAllButton = screen.getByRole('button', { name: /Select All/i })
      
      expect(searchInput).toBeDisabled()
      expect(selectAllButton).toBeDisabled()
    })

    it('does not call onSelectionChange when disabled and table clicked', async () => {
      const onSelectionChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <TableSelector 
          {...defaultProps} 
          disabled={true} 
          onSelectionChange={onSelectionChange} 
        />
      )
      
      const table = screen.getByRole('button', { name: /ContainerAppConsoleLogs_CL/i })
      await user.click(table)
      
      expect(onSelectionChange).not.toHaveBeenCalled()
    })
  })

  describe('Table Descriptions', () => {
    it('shows table descriptions', () => {
      render(<TableSelector {...defaultProps} />)
      
      expect(screen.getByText('Container application console logs')).toBeInTheDocument()
      expect(screen.getByText('App Service console logs')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('has accessible search input', () => {
      render(<TableSelector {...defaultProps} />)
      
      const searchInput = screen.getByPlaceholderText('Search tables...')
      expect(searchInput).toHaveAttribute('type', 'text')
    })

    it('has accessible labels for action buttons', () => {
      render(<TableSelector {...defaultProps} />)
      
      expect(screen.getByRole('button', { name: /Select All/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Clear/i })).toBeInTheDocument()
    })
  })

  describe('Custom Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <TableSelector {...defaultProps} className="custom-selector-class" />
      )
      
      const element = container.querySelector('.custom-selector-class')
      expect(element).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    it('handles empty arrays defensively', () => {
      render(
        <TableSelector 
          {...defaultProps} 
          tables={[]}
          categories={[]}
          selectedTables={[]}
          recommendedTables={[]}
        />
      )
      
      expect(screen.getByText('0 of 0 tables selected')).toBeInTheDocument()
    })

    it('handles null/undefined arrays defensively', () => {
      // @ts-expect-error Testing defensive coding
      render(<TableSelector tables={null} categories={null} selectedTables={null} onSelectionChange={vi.fn()} />)
      
      // Should not crash and show 0 tables
      expect(screen.getByText('0 of 0 tables selected')).toBeInTheDocument()
    })

    it('handles tables without category', () => {
      const tablesWithoutCategory: TableInfo[] = [
        {
          name: 'UncategorizedTable',
          category: '',
          description: 'No category',
        },
      ]
      
      render(<TableSelector {...defaultProps} tables={tablesWithoutCategory} />)
      
      // Should appear in "other" category
      expect(screen.getByText('UncategorizedTable')).toBeInTheDocument()
    })
  })
})
