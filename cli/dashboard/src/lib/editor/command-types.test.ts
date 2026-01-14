/**
 * Command Types Tests
 */

import { describe, it, expect } from 'vitest'
import type { Command, CommandAction, CommandSearchResult } from './command-types'

describe('command-types', () => {
  describe('Command', () => {
    it('should define navigation command structure', () => {
      const command: Command = {
        id: 'nav.overview',
        label: 'Go to Overview',
        description: 'View overview',
        category: 'navigation',
        icon: 'ArrowRight',
        keywords: ['overview', 'home'],
        action: { type: 'navigate', path: 'overview' },
      }
      
      expect(command.id).toBe('nav.overview')
      expect(command.category).toBe('navigation')
      expect(command.action.type).toBe('navigate')
    })
    
    it('should define action command structure', () => {
      const handler = () => {}
      const command: Command = {
        id: 'action.add-service',
        label: 'Add Service',
        category: 'action',
        shortcut: 'Cmd+N',
        action: { type: 'execute', handler },
      }
      
      expect(command.id).toBe('action.add-service')
      expect(command.category).toBe('action')
      expect(command.action.type).toBe('execute')
      expect(command.shortcut).toBe('Cmd+N')
    })
    
    it('should define field command structure', () => {
      const command: Command = {
        id: 'field.name',
        label: 'Application Name',
        category: 'field',
        action: { type: 'jump-to-field', fieldPath: 'name' },
      }
      
      expect(command.action.type).toBe('jump-to-field')
    })
    
    it('should define help command structure', () => {
      const command: Command = {
        id: 'help.services',
        label: 'Services Help',
        category: 'help',
        action: { type: 'open-help', topic: 'services' },
      }
      
      expect(command.action.type).toBe('open-help')
    })
  })
  
  describe('CommandAction', () => {
    it('should define navigate action', () => {
      const action: CommandAction = { type: 'navigate', path: 'services' }
      expect(action.type).toBe('navigate')
    })
    
    it('should define execute action', () => {
      const handler = () => {}
      const action: CommandAction = { type: 'execute', handler }
      expect(action.type).toBe('execute')
    })
    
    it('should define jump-to-field action', () => {
      const action: CommandAction = { type: 'jump-to-field', fieldPath: 'services.api.host' }
      expect(action.type).toBe('jump-to-field')
    })
    
    it('should define open-help action', () => {
      const action: CommandAction = { type: 'open-help', topic: 'hooks' }
      expect(action.type).toBe('open-help')
    })
  })
  
  describe('CommandSearchResult', () => {
    it('should include command and score', () => {
      const command: Command = {
        id: 'test',
        label: 'Test',
        category: 'action',
        action: { type: 'execute', handler: () => {} },
      }
      
      const result: CommandSearchResult = {
        command,
        score: 0.85,
      }
      
      expect(result.command).toBe(command)
      expect(result.score).toBe(0.85)
    })
    
    it('should include optional match indices', () => {
      const command: Command = {
        id: 'test',
        label: 'Test',
        category: 'action',
        action: { type: 'execute', handler: () => {} },
      }
      
      const result: CommandSearchResult = {
        command,
        score: 0.9,
        matches: [0, 1, 2, 3],
      }
      
      expect(result.matches).toEqual([0, 1, 2, 3])
    })
  })
})
