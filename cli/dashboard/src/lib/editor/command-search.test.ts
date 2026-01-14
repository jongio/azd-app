/**
 * Command Search Tests
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { CommandSearch, groupResultsByCategory, filterRecentCommands } from './command-search'
import type { Command } from './command-types'

describe('command-search', () => {
  const mockCommands: Command[] = [
    {
      id: 'nav.overview',
      label: 'Go to Overview',
      description: 'View application overview',
      category: 'navigation',
      keywords: ['overview', 'home'],
      action: { type: 'navigate', path: 'overview' },
    },
    {
      id: 'nav.services',
      label: 'Go to Services',
      description: 'Manage services',
      category: 'navigation',
      keywords: ['services', 'containers'],
      action: { type: 'navigate', path: 'services' },
    },
    {
      id: 'action.add-service',
      label: 'Add Service',
      description: 'Add a new service',
      category: 'action',
      keywords: ['add', 'create', 'new'],
      action: { type: 'execute', handler: () => {} },
    },
    {
      id: 'field.name',
      label: 'Application Name',
      description: 'Edit app name',
      category: 'field',
      keywords: ['name', 'title'],
      action: { type: 'jump-to-field', fieldPath: 'name' },
    },
    {
      id: 'help.services',
      label: 'Services Documentation',
      description: 'Learn about services',
      category: 'help',
      keywords: ['docs', 'help'],
      action: { type: 'open-help', topic: 'services' },
    },
  ]
  
  let search: CommandSearch
  
  beforeEach(() => {
    search = new CommandSearch(mockCommands)
  })
  
  describe('CommandSearch', () => {
    it('should return all commands for empty query', () => {
      const results = search.search('')
      
      expect(results).toHaveLength(5)
      expect(results.every(r => r.score === 1)).toBe(true)
    })
    
    it('should fuzzy search by label', () => {
      const results = search.search('overview')
      
      expect(results.length).toBeGreaterThan(0)
      expect(results[0].command.id).toBe('nav.overview')
    })
    
    it('should fuzzy search by description', () => {
      const results = search.search('manage')
      
      expect(results.length).toBeGreaterThan(0)
      expect(results[0].command.id).toBe('nav.services')
    })
    
    it('should fuzzy search by keywords', () => {
      const results = search.search('containers')
      
      expect(results.length).toBeGreaterThan(0)
      expect(results[0].command.id).toBe('nav.services')
    })
    
    it('should rank exact matches higher', () => {
      const results = search.search('services')
      
      expect(results.length).toBeGreaterThan(0)
      // Should prioritize "Go to Services" over "Services Documentation"
      expect(results[0].command.id).toBe('nav.services')
    })
    
    it('should handle partial matches', () => {
      const results = search.search('serv')
      
      expect(results.length).toBeGreaterThan(0)
      expect(results.some(r => r.command.id === 'nav.services')).toBe(true)
    })
    
    it('should handle typos with fuzzy matching', () => {
      const results = search.search('servces') // Missing 'i'
      
      expect(results.length).toBeGreaterThan(0)
      expect(results.some(r => r.command.id === 'nav.services')).toBe(true)
    })
    
    it('should return empty array for no matches', () => {
      const results = search.search('xyz123nonexistent')
      
      expect(results).toHaveLength(0)
    })
    
    it('should limit results', () => {
      const results = search.search('', 2)
      
      expect(results.length).toBeLessThanOrEqual(2)
    })
    
    it('should update commands', () => {
      const newCommands: Command[] = [
        {
          id: 'new.command',
          label: 'New Command',
          category: 'action',
          action: { type: 'execute', handler: () => {} },
        },
      ]
      
      search.updateCommands(newCommands)
      const results = search.search('')
      
      expect(results).toHaveLength(1)
      expect(results[0].command.id).toBe('new.command')
    })
  })
  
  describe('groupResultsByCategory', () => {
    it('should group results by category', () => {
      const results = search.search('')
      const grouped = groupResultsByCategory(results)
      
      expect(grouped.has('navigation')).toBe(true)
      expect(grouped.has('action')).toBe(true)
      expect(grouped.has('field')).toBe(true)
      expect(grouped.has('help')).toBe(true)
    })
    
    it('should limit results per category', () => {
      // Add more navigation commands
      const moreCommands: Command[] = [
        ...mockCommands,
        ...Array.from({ length: 10 }, (_, i) => ({
          id: `nav.test${i}`,
          label: `Test ${i}`,
          category: 'navigation' as const,
          action: { type: 'navigate' as const, path: `test${i}` },
        })),
      ]
      
      const testSearch = new CommandSearch(moreCommands)
      const results = testSearch.search('')
      const grouped = groupResultsByCategory(results, 3)
      
      const navResults = grouped.get('navigation')
      expect(navResults).toBeDefined()
      expect(navResults!.length).toBeLessThanOrEqual(3)
    })
    
    it('should sort category results by score', () => {
      const results = search.search('service')
      const grouped = groupResultsByCategory(results)
      
      for (const categoryResults of grouped.values()) {
        for (let i = 1; i < categoryResults.length; i++) {
          expect(categoryResults[i - 1].score).toBeGreaterThanOrEqual(categoryResults[i].score)
        }
      }
    })
  })
  
  describe('filterRecentCommands', () => {
    it('should filter results by recent IDs', () => {
      const results = search.search('')
      const recentIds = ['nav.overview', 'action.add-service']
      
      const filtered = filterRecentCommands(results, recentIds)
      
      expect(filtered).toHaveLength(2)
      expect(filtered.every(r => recentIds.includes(r.command.id))).toBe(true)
    })
    
    it('should return empty array if no matches', () => {
      const results = search.search('')
      const recentIds = ['nonexistent']
      
      const filtered = filterRecentCommands(results, recentIds)
      
      expect(filtered).toHaveLength(0)
    })
    
    it('should handle empty recent IDs', () => {
      const results = search.search('')
      const filtered = filterRecentCommands(results, [])
      
      expect(filtered).toHaveLength(0)
    })
  })
})
