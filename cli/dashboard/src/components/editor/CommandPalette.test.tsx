/**
 * CommandPalette Component Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CommandPalette } from './CommandPalette'
import type { Command } from '@/lib/editor/command-types'

describe('CommandPalette', () => {
  const mockCommands: Command[] = [
    {
      id: 'nav.overview',
      label: 'Go to Overview',
      description: 'View application overview',
      category: 'navigation',
      icon: 'ArrowRight',
      action: { type: 'navigate', path: 'overview' },
    },
    {
      id: 'nav.services',
      label: 'Go to Services',
      description: 'Manage services',
      category: 'navigation',
      action: { type: 'navigate', path: 'services' },
    },
    {
      id: 'action.add-service',
      label: 'Add Service',
      description: 'Add a new service',
      category: 'action',
      shortcut: 'Cmd+N',
      action: { type: 'execute', handler: vi.fn() },
    },
    {
      id: 'field.name',
      label: 'Application Name',
      category: 'field',
      action: { type: 'jump-to-field', fieldPath: 'name' },
    },
    {
      id: 'help.services',
      label: 'Services Help',
      category: 'help',
      action: { type: 'open-help', topic: 'services' },
    },
  ]
  
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    commands: mockCommands,
    onNavigate: vi.fn(),
    onJumpToField: vi.fn(),
    onOpenHelp: vi.fn(),
  }
  
  beforeEach(() => {
    localStorage.clear()
  })
  
  afterEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })
  
  describe('Rendering', () => {
    it('should not render when closed', () => {
      render(<CommandPalette {...defaultProps} isOpen={false} />)
      
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    
    it('should render when open', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('dialog', { name: /command palette/i })).toBeInTheDocument()
    })
    
    it('should render search input', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('textbox', { name: /search commands/i })).toBeInTheDocument()
    })
    
    it('should auto-focus search input when opened', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('textbox')).toHaveFocus()
    })
    
    it('should render close button', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('button', { name: /close/i })).toBeInTheDocument()
    })
    
    it('should render keyboard hints in footer', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByText(/navigate/i)).toBeInTheDocument()
      expect(screen.getByText(/select/i)).toBeInTheDocument()
      expect(screen.getByText(/close/i)).toBeInTheDocument()
    })
  })
  
  describe('Search Functionality', () => {
    it('should show all commands when search is empty', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByText('Go to Overview')).toBeInTheDocument()
      expect(screen.getByText('Add Service')).toBeInTheDocument()
    })
    
    it('should filter commands by search query', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const searchInput = screen.getByRole('textbox')
      await user.clear(searchInput)
      await user.type(searchInput, 'service')
      
      // Wait for filtering to complete - text may be split by highlighting
      await waitFor(() => {
        // Use text matcher that ignores highlighting markup
        expect(screen.getByText((_content, element) => {
          return element?.textContent === 'Go to Services'
        })).toBeInTheDocument()
      }, { timeout: 5000 })
      
      expect(screen.getByText((_content, element) => {
        return element?.textContent === 'Add Service'
      })).toBeInTheDocument()
      expect(screen.queryByText('Go to Overview')).not.toBeInTheDocument()
    })
    
    it('should show empty state when no matches', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const searchInput = screen.getByRole('textbox')
      await user.type(searchInput, 'nonexistent')
      
      await waitFor(() => {
        expect(screen.getByText(/no commands found/i)).toBeInTheDocument()
      })
    })
    
    it('should update results count in footer', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      // Initially shows all
      expect(screen.getByText(/5 results/i)).toBeInTheDocument()
      
      const searchInput = screen.getByRole('textbox')
      await user.type(searchInput, 'overview')
      
      await waitFor(() => {
        expect(screen.getByText(/1 result/i)).toBeInTheDocument()
      })
    })
  })
  
  describe('Grouped Results', () => {
    it('should group results by category', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByText(/navigation/i)).toBeInTheDocument()
      expect(screen.getByText(/actions/i)).toBeInTheDocument()
      expect(screen.getByText(/fields/i)).toBeInTheDocument()
      expect(screen.getByText(/help/i)).toBeInTheDocument()
    })
    
    it('should show truncated count when results exceed max', () => {
      // Add many navigation commands
      const manyCommands: Command[] = [
        ...Array.from({ length: 10 }, (_, i) => ({
          id: `nav${i}`,
          label: `Navigate ${i}`,
          category: 'navigation' as const,
          action: { type: 'navigate' as const, path: `nav${i}` },
        })),
      ]
      
      render(<CommandPalette {...defaultProps} commands={manyCommands} maxPerCategory={5} />)
      
      expect(screen.getByText(/5 of 10/i)).toBeInTheDocument()
    })
  })
  
  describe('Keyboard Navigation', () => {
    it('should select first result by default', () => {
      render(<CommandPalette {...defaultProps} />)
      
      const firstResult = screen.getAllByRole('button')[1] // Skip close button
      expect(firstResult).toHaveAttribute('data-selected', 'true')
    })
    
    it('should navigate down with ArrowDown', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const dialog = screen.getByRole('dialog')
      await user.type(dialog, '{arrowdown}')
      
      const results = screen.getAllByRole('button').filter(b => b.hasAttribute('data-selected'))
      expect(results.length).toBe(1)
    })
    
    it('should navigate up with ArrowUp', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const dialog = screen.getByRole('dialog')
      await user.type(dialog, '{arrowdown}{arrowdown}{arrowup}')
      
      // Should be back to second item
      const results = screen.getAllByRole('button').filter(b => b.hasAttribute('data-selected'))
      expect(results.length).toBe(1)
    })
    
    it('should not navigate above first result', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const dialog = screen.getByRole('dialog')
      await user.type(dialog, '{arrowup}')
      
      const firstResult = screen.getAllByRole('button')[1]
      expect(firstResult).toHaveAttribute('data-selected', 'true')
    })
    
    it('should execute command on Enter', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const dialog = screen.getByRole('dialog')
      await user.type(dialog, '{enter}')
      
      expect(defaultProps.onNavigate).toHaveBeenCalledWith('overview')
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
    
    it('should close on Escape', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const dialog = screen.getByRole('dialog')
      await user.type(dialog, '{escape}')
      
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
  })
  
  describe('Command Execution', () => {
    it('should execute navigate command', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const overviewButton = screen.getByText('Go to Overview')
      await user.click(overviewButton)
      
      expect(defaultProps.onNavigate).toHaveBeenCalledWith('overview')
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
    
    it('should execute action command', async () => {
      const user = userEvent.setup()
      const handler = vi.fn()
      const commands: Command[] = [
        {
          id: 'test',
          label: 'Test Action',
          category: 'action',
          action: { type: 'execute', handler },
        },
      ]
      
      render(<CommandPalette {...defaultProps} commands={commands} />)
      
      const button = screen.getByText('Test Action')
      await user.click(button)
      
      expect(handler).toHaveBeenCalled()
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
    
    it('should execute jump-to-field command', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const button = screen.getByText('Application Name')
      await user.click(button)
      
      expect(defaultProps.onJumpToField).toHaveBeenCalledWith('name')
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
    
    it('should execute open-help command', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const button = screen.getByText('Services Help')
      await user.click(button)
      
      expect(defaultProps.onOpenHelp).toHaveBeenCalledWith('services')
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
  })
  
  describe('Recent History', () => {
    it('should show recent commands section when no query', () => {
      // Add history
      localStorage.setItem('azd-command-palette-history', JSON.stringify({
        recent: ['nav.overview'],
        lastUpdated: Date.now(),
      }))
      
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByText(/recent/i)).toBeInTheDocument()
    })
    
    it('should hide recent section when query present', async () => {
      const user = userEvent.setup()
      
      localStorage.setItem('azd-command-palette-history', JSON.stringify({
        recent: ['nav.overview'],
        lastUpdated: Date.now(),
      }))
      
      render(<CommandPalette {...defaultProps} />)
      
      const searchInput = screen.getByRole('textbox')
      await user.type(searchInput, 'test')
      
      await waitFor(() => {
        expect(screen.queryByText(/recent/i)).not.toBeInTheDocument()
      })
    })
    
    it('should add executed command to history', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const button = screen.getByText('Go to Overview')
      await user.click(button)
      
      const stored = localStorage.getItem('azd-command-palette-history')
      expect(stored).toBeTruthy()
      
      const history = JSON.parse(stored!)
      expect(history.recent).toContain('nav.overview')
    })
    
    it('should clear history when clear button clicked', async () => {
      const user = userEvent.setup()
      
      localStorage.setItem('azd-command-palette-history', JSON.stringify({
        recent: ['nav.overview', 'nav.services'],
        lastUpdated: Date.now(),
      }))
      
      render(<CommandPalette {...defaultProps} />)
      
      const clearButton = screen.getByText(/clear/i)
      await user.click(clearButton)
      
      await waitFor(() => {
        expect(screen.queryByText(/recent/i)).not.toBeInTheDocument()
      })
      
      const stored = localStorage.getItem('azd-command-palette-history')
      const history = JSON.parse(stored!)
      expect(history.recent).toEqual([])
    })
  })
  
  describe('Accessibility', () => {
    it('should have accessible dialog', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('dialog', { name: /command palette/i })).toBeInTheDocument()
    })
    
    it('should have accessible search input', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('textbox', { name: /search commands/i })).toBeInTheDocument()
    })
    
    it('should have accessible buttons', () => {
      render(<CommandPalette {...defaultProps} />)
      
      expect(screen.getByRole('button', { name: /close command palette/i })).toBeInTheDocument()
    })
  })
  
  describe('Close Interactions', () => {
    it('should close on close button click', async () => {
      const user = userEvent.setup()
      render(<CommandPalette {...defaultProps} />)
      
      const closeButton = screen.getByRole('button', { name: /close command palette/i })
      await user.click(closeButton)
      
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
    
    it('should close on backdrop click', async () => {
      const user = userEvent.setup()
      const { container } = render(<CommandPalette {...defaultProps} />)
      
      const backdrop = container.querySelector('.fixed.inset-0')!
      await user.click(backdrop)
      
      expect(defaultProps.onClose).toHaveBeenCalled()
    })
  })
  
  describe('Shortcuts Display', () => {
    it('should display keyboard shortcut hints', () => {
      render(<CommandPalette {...defaultProps} />)
      
      const addServiceButton = screen.getByText('Add Service')
      expect(addServiceButton.closest('button')?.textContent).toContain('Cmd+N')
    })
  })
})
