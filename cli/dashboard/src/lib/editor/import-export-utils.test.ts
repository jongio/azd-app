/**
 * Import/Export Utilities Tests
 */

import { describe, it, expect } from 'vitest'
import {
  mergeConfigurations,
  generateDiff,
  extractCherryPickSections,
  applyCherryPick,
  detectSecurityWarnings,
  convertToTemplate,
} from './import-export-utils'

describe('mergeConfigurations', () => {
  const current = {
    name: 'current-app',
    services: {
      api: { host: 'containerapp', project: './api' },
    },
  }

  const imported = {
    name: 'imported-app',
    services: {
      web: { host: 'containerapp', project: './web' },
    },
  }

  it('should replace entire config with replace strategy', () => {
    const result = mergeConfigurations(current, imported, 'replace')
    expect(result).toEqual(imported)
  })

  it('should merge configs with merge strategy', () => {
    const result = mergeConfigurations(current, imported, 'merge')
    expect(result.name).toBe('imported-app')
    expect(result.services).toHaveProperty('api')
    expect(result.services).toHaveProperty('web')
  })

  it('should cherry-pick selected sections', () => {
    const result = mergeConfigurations(current, imported, 'cherry-pick', ['services'])
    expect(result.name).toBe('current-app') // Not changed
    expect(result.services).toHaveProperty('web') // Added
  })
})

describe('generateDiff', () => {
  it('should detect added keys', () => {
    const current = { name: 'app' }
    const imported = { name: 'app', version: '1.0' }
    const diff = generateDiff(current, imported)
    
    const addedDiff = diff.find(d => d.path === 'version')
    expect(addedDiff?.type).toBe('added')
  })

  it('should detect removed keys', () => {
    const current = { name: 'app', version: '1.0' }
    const imported = { name: 'app' }
    const diff = generateDiff(current, imported)
    
    const removedDiff = diff.find(d => d.path === 'version')
    expect(removedDiff?.type).toBe('removed')
  })

  it('should detect changed values', () => {
    const current = { name: 'old-name' }
    const imported = { name: 'new-name' }
    const diff = generateDiff(current, imported)
    
    const changedDiff = diff.find(d => d.path === 'name')
    expect(changedDiff?.type).toBe('changed')
    expect(changedDiff?.currentValue).toBe('old-name')
    expect(changedDiff?.importedValue).toBe('new-name')
  })

  it('should detect unchanged values', () => {
    const current = { name: 'app' }
    const imported = { name: 'app' }
    const diff = generateDiff(current, imported)
    
    const unchangedDiff = diff.find(d => d.path === 'name')
    expect(unchangedDiff?.type).toBe('unchanged')
  })
})

describe('extractCherryPickSections', () => {
  it('should extract service sections', () => {
    const config = {
      services: {
        api: { host: 'containerapp' },
        web: { host: 'containerapp' },
      },
    }
    
    const sections = extractCherryPickSections(config)
    expect(sections).toHaveLength(2)
    expect(sections[0].type).toBe('service')
    expect(sections[0].id).toBe('service.api')
    expect(sections[1].id).toBe('service.web')
  })

  it('should extract resource sections', () => {
    const config = {
      resources: {
        storage: { type: 'Microsoft.Storage/storageAccounts' },
      },
    }
    
    const sections = extractCherryPickSections(config)
    expect(sections).toHaveLength(1)
    expect(sections[0].type).toBe('resource')
    expect(sections[0].id).toBe('resource.storage')
  })

  it('should extract hooks section', () => {
    const config = {
      hooks: {
        postdeploy: { run: 'echo done' },
      },
    }
    
    const sections = extractCherryPickSections(config)
    expect(sections).toHaveLength(1)
    expect(sections[0].type).toBe('hooks')
    expect(sections[0].id).toBe('hooks')
  })
})

describe('applyCherryPick', () => {
  const current = {
    name: 'current',
    services: {
      api: { host: 'containerapp' },
    },
  }

  const imported = {
    name: 'imported',
    services: {
      web: { host: 'containerapp' },
    },
  }

  it('should apply selected service', () => {
    const selections = [
      { id: 'service.web', name: 'Service: web', description: '', type: 'service' as const, selected: true },
    ]
    
    const result = applyCherryPick(current, imported, selections)
    expect(result.services).toHaveProperty('api')
    expect(result.services).toHaveProperty('web')
    expect(result.name).toBe('current')
  })

  it('should not apply unselected sections', () => {
    const selections = [
      { id: 'service.web', name: 'Service: web', description: '', type: 'service' as const, selected: false },
    ]
    
    const result = applyCherryPick(current, imported, selections)
    expect(result.services).toHaveProperty('api')
    expect(result.services).not.toHaveProperty('web')
  })
})

describe('detectSecurityWarnings', () => {
  it('should warn when including secrets', () => {
    const config = {
      services: {
        api: {
          environment: {
            API_KEY: 'secret-value',
          },
        },
      },
    }
    
    const warnings = detectSecurityWarnings(config, true)
    expect(warnings.length).toBeGreaterThan(0)
    expect(warnings[0].type).toBe('secrets')
    expect(warnings[0].requiresConfirmation).toBe(true)
  })

  it('should not warn when not including secrets', () => {
    const config = {
      services: {
        api: {
          environment: {
            API_KEY: 'secret-value',
          },
        },
      },
    }
    
    const warnings = detectSecurityWarnings(config, false)
    expect(warnings.length).toBe(0)
  })
})

describe('convertToTemplate', () => {
  it('should replace string values with placeholders', () => {
    const config = {
      name: 'my-app',
      services: {
        api: {
          host: 'containerapp',
          ports: ['8080'],
        },
      },
    }
    
    const template = convertToTemplate(config)
    expect(template.name).toBe('${NAME}')
  })

  it('should preserve structure', () => {
    const config = {
      services: {
        api: { host: 'containerapp' },
      },
    }
    
    const template = convertToTemplate(config)
    expect(template).toHaveProperty('services')
    expect(template.services).toHaveProperty('api')
  })
})
