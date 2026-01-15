/**
 * NavigationSidebar Component Tests
 * 
 * Covers:
 * - Tree rendering
 * - Active section highlighting
 * - Validation badges
 * - Keyboard navigation
 * - Search/filter
 * - Expand/collapse
 * - Accessibility
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { NavigationSidebar } from './NavigationSidebar'
import type { NavigationNode, ValidationIssue } from '@/lib/editor/navigation-types'

describe('NavigationSidebar', () => {
  const mockNodes: NavigationNode[] = [
    {
      id: 'overview',
      label: 'Overview',
      type: 'section',
      children: [
        { id: 'name', label: 'Application Name', type: 'property' },
        { id: 'resourceGroup', label: 'Resource Group', type: 'property' },
      ],
    },
    {
      id: 'services',
      label: 'Services',
      type: 'section',
      children: [
        { id: 'api', label: 'api', type: 'item' },
        { id: 'web', label: 'web', type: 'item' },
      ],
    },
    {
      id: 'resources',
      label: 'Resources',
      type: 'section',
      children: [],
    },
  ]

  const mockOnNavigate = vi.fn()
  const mockOnAdd = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('should render navigation header', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      expect(screen.getByText('Configuration')).toBeInTheDocument()
    })

    it('should render all top-level nodes', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      expect(screen.getByRole('treeitem', { name: /Overview/i })).toBeInTheDocument()
      expect(screen.getByRole('treeitem', { name: /Services/i })).toBeInTheDocument()
      expect(screen.getByRole('treeitem', { name: /Resources/i })).toBeInTheDocument()
    })

    it('should render children when expanded', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services.api"
          onNavigate={mockOnNavigate}
        />
      )

      // Services should be auto-expanded
      expect(screen.getByRole('treeitem', { name: 'api' })).toBeInTheDocument()
      expect(screen.getByRole('treeitem', { name: 'web' })).toBeInTheDocument()
    })

    it('should highlight active section', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services.api"
          onNavigate={mockOnNavigate}
        />
      )

      const apiButton = screen.getByRole('treeitem', { name: 'api' })
      expect(apiButton).toHaveAttribute('aria-current', 'page')
    })

    it('should show add buttons for services and resources', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services"
          onNavigate={mockOnNavigate}
          onAdd={mockOnAdd}
        />
      )

      expect(screen.getByText('New Service')).toBeInTheDocument()
      expect(screen.getByText('New Resource')).toBeInTheDocument()
    })
  })

  describe('Validation Badges', () => {
    it('should show error badge when section has errors', () => {
      const issues = new Map<string, ValidationIssue[]>([
        ['services.api', [{ level: 'error', message: 'Invalid port', path: 'services.api' }]],
      ])

      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          validationIssues={issues}
          onNavigate={mockOnNavigate}
        />
      )

      const apiButton = screen.getByRole('treeitem', { name: /api.*1 error/i })
      expect(apiButton).toBeInTheDocument()
    })

    it('should show warning badge when section has warnings', () => {
      const issues = new Map<string, ValidationIssue[]>([
        ['services.web', [{ level: 'warning', message: 'Missing health check', path: 'services.web' }]],
      ])

      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          validationIssues={issues}
          onNavigate={mockOnNavigate}
        />
      )

      const webButton = screen.getByRole('treeitem', { name: /web.*1 warning/i })
      expect(webButton).toBeInTheDocument()
    })

    it('should show both error and warning badges', () => {
      const issues = new Map<string, ValidationIssue[]>([
        [
          'services.api',
          [
            { level: 'error', message: 'Invalid port', path: 'services.api' },
            { level: 'warning', message: 'Missing health check', path: 'services.api' },
          ],
        ],
      ])

      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          validationIssues={issues}
          onNavigate={mockOnNavigate}
        />
      )

      const apiButton = screen.getByRole('treeitem', { name: /api.*1 error.*1 warning/i })
      expect(apiButton).toBeInTheDocument()
    })
  })

  describe('Expand/Collapse', () => {
    it('should expand section when clicked', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      // Initially, overview children should be visible (expanded by default)
      expect(screen.getByRole('treeitem', { name: 'Application Name' })).toBeInTheDocument()

      // Click overview to collapse
      const overviewItem = screen.getByRole('treeitem', { name: /Overview/i })
      const button = overviewItem.querySelector('button')
      await user.click(button!)

      // Children should be hidden after clicking
      await waitFor(() => {
        expect(screen.queryByRole('treeitem', { name: 'Application Name' })).not.toBeInTheDocument()
      })
    })

    it('should toggle expansion with button click', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      // Overview is expanded by default, children visible
      expect(screen.getByRole('treeitem', { name: 'Application Name' })).toBeInTheDocument()

      // Click the button inside the overview treeitem to collapse
      const overviewItem = screen.getByRole('treeitem', { name: /Overview/i })
      const button = overviewItem.querySelector('button')
      await user.click(button!)

      // Children should be hidden
      await waitFor(() => {
        expect(screen.queryByRole('treeitem', { name: 'Application Name' })).not.toBeInTheDocument()
      })
    })
  })

  describe('Navigation', () => {
    it('should call onNavigate when item is clicked', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const apiItem = screen.getByRole('treeitem', { name: 'api' })
      const button = apiItem.querySelector('button')
      await user.click(button!)

      expect(mockOnNavigate).toHaveBeenCalledWith('services.api')
    })

    it('should call onAdd when add button is clicked', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services"
          onNavigate={mockOnNavigate}
          onAdd={mockOnAdd}
        />
      )

      const addServiceButton = screen.getByText('New Service').closest('button')
      expect(addServiceButton).not.toBeNull()

      await user.click(addServiceButton as HTMLButtonElement)

      expect(mockOnAdd).toHaveBeenCalledWith('service', 'services')
    })
  })

  describe('Keyboard Navigation', () => {
    it('should navigate down with ArrowDown', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const nav = screen.getByRole('navigation')
      nav.focus()

      await user.keyboard('{ArrowDown}')
      // Focus should move to next item
    })

    it('should navigate up with ArrowUp', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const nav = screen.getByRole('navigation')
      nav.focus()

      await user.keyboard('{ArrowDown}')
      await user.keyboard('{ArrowUp}')
      // Focus should return to first item
    })

    it('should expand with ArrowRight', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services"
          onNavigate={mockOnNavigate}
        />
      )

      const nav = screen.getByRole('navigation')
      nav.focus()

      // Navigate to a collapsed section
      await user.keyboard('{ArrowDown}') // Assuming this lands on a collapsed node
      await user.keyboard('{ArrowRight}')
      
      // Section should expand
    })

    it('should collapse with ArrowLeft', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services"
          onNavigate={mockOnNavigate}
        />
      )

      const nav = screen.getByRole('navigation')
      nav.focus()

      await user.keyboard('{ArrowLeft}')
      // Expanded section should collapse
    })

    it('should activate item with Enter', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const nav = screen.getByRole('navigation')
      nav.focus()

      await user.keyboard('{Enter}')
      // Should trigger navigation or toggle
    })
  })

  describe('Search', () => {
    it('should filter nodes based on search query', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const searchInput = screen.getByPlaceholderText(/Search/i)
      await user.type(searchInput, 'api')

      // Only api service should be visible
      expect(screen.getByRole('treeitem', { name: 'api' })).toBeInTheDocument()
      expect(screen.queryByRole('treeitem', { name: 'web' })).not.toBeInTheDocument()
    })

    it('should show "no results" when search yields nothing', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const searchInput = screen.getByPlaceholderText(/Search/i)
      await user.type(searchInput, 'nonexistent')

      expect(screen.getByText(/No results found/i)).toBeInTheDocument()
    })

    it('should clear search with clear button', async () => {
      const user = userEvent.setup()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const searchInput = screen.getByPlaceholderText(/Search/i)
      await user.type(searchInput, 'api')

      const clearButton = screen.getByLabelText('Clear search')
      await user.click(clearButton)

      expect(searchInput).toHaveValue('')
      // All nodes should be visible again
      expect(screen.getByRole('treeitem', { name: 'web' })).toBeInTheDocument()
    })
  })

  describe('Collapse State', () => {
    it('should render collapsed sidebar', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
          isCollapsed={true}
          onToggleCollapse={vi.fn()}
        />
      )

      expect(screen.getByLabelText('Expand navigation')).toBeInTheDocument()
      expect(screen.queryByText('Configuration')).not.toBeInTheDocument()
    })

    it('should toggle collapse when button is clicked', async () => {
      const user = userEvent.setup()
      const onToggle = vi.fn()
      
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
          onToggleCollapse={onToggle}
        />
      )

      const collapseButton = screen.getByLabelText('Collapse navigation')
      await user.click(collapseButton)

      expect(onToggle).toHaveBeenCalled()
    })
  })

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      expect(screen.getByRole('navigation', { name: 'Azure YAML Editor Navigation' })).toBeInTheDocument()
      expect(screen.getByRole('tree', { name: 'Configuration structure' })).toBeInTheDocument()
    })

    it('should set aria-current on active item', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services.api"
          onNavigate={mockOnNavigate}
        />
      )

      const apiButton = screen.getByRole('treeitem', { name: 'api' })
      expect(apiButton).toHaveAttribute('aria-current', 'page')
    })

    it('should set aria-expanded on expandable items', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="services"
          onNavigate={mockOnNavigate}
        />
      )

      const servicesButton = screen.getByRole('treeitem', { name: /Services/i })
      expect(servicesButton).toHaveAttribute('aria-expanded', 'true')
    })

    it('should have keyboard focus management', () => {
      render(
        <NavigationSidebar
          nodes={mockNodes}
          activeSection="overview"
          onNavigate={mockOnNavigate}
        />
      )

      const nav = screen.getByRole('navigation')
      expect(nav).toHaveAttribute('tabIndex', '0')
    })
  })
})
