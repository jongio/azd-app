/**
 * Navigation Types Tests
 */

import { describe, it, expect } from 'vitest'
import { buildNavigationTree } from './navigation-types'

describe('buildNavigationTree', () => {
  it('should return empty array for null config', () => {
    const result = buildNavigationTree(null)
    expect(result).toEqual([])
  })

  it('should create overview section with properties', () => {
    const config = {
      name: 'my-app',
      resourceGroup: 'rg-dev',
      metadata: { template: 'azd-app' },
    }

    const result = buildNavigationTree(config)
    
    const overview = result.find((n) => n.id === 'overview')
    expect(overview).toBeDefined()
    expect(overview?.children).toHaveLength(3)
    expect(overview?.children?.map((c) => c.id)).toEqual(['name', 'resourceGroup', 'metadata'])
  })

  it('should create services section with service nodes', () => {
    const config = {
      services: {
        api: { host: 'containerapp' },
        web: { host: 'containerapp' },
      },
    }

    const result = buildNavigationTree(config)
    
    const services = result.find((n) => n.id === 'services')
    expect(services).toBeDefined()
    expect(services?.children).toHaveLength(2)
    expect(services?.children?.map((c) => c.id)).toEqual(['api', 'web'])
    expect(services?.collapsible).toBe(true)
  })

  it('should create empty services section when no services exist', () => {
    const config = {
      name: 'my-app',
    }

    const result = buildNavigationTree(config)
    
    const services = result.find((n) => n.id === 'services')
    expect(services).toBeDefined()
    expect(services?.children).toEqual([])
  })

  it('should create resources section with resource nodes', () => {
    const config = {
      resources: {
        storage: { type: 'Microsoft.Storage/storageAccounts' },
        cosmos: { type: 'Microsoft.DocumentDB/databaseAccounts' },
      },
    }

    const result = buildNavigationTree(config)
    
    const resources = result.find((n) => n.id === 'resources')
    expect(resources).toBeDefined()
    expect(resources?.children).toHaveLength(2)
    expect(resources?.children?.map((c) => c.id)).toEqual(['storage', 'cosmos'])
  })

  it('should create hooks section when hooks exist', () => {
    const config = {
      hooks: {
        preprovision: { run: 'npm install' },
      },
    }

    const result = buildNavigationTree(config)
    
    const hooks = result.find((n) => n.id === 'hooks')
    expect(hooks).toBeDefined()
    expect(hooks?.type).toBe('section')
  })

  it('should create pipeline section when pipeline exists', () => {
    const config = {
      pipeline: {
        provider: 'github',
      },
    }

    const result = buildNavigationTree(config)
    
    const pipeline = result.find((n) => n.id === 'pipeline')
    expect(pipeline).toBeDefined()
  })

  it('should create requiredVersions section when it exists', () => {
    const config = {
      requiredVersions: {
        azd: '>=1.0.0',
      },
    }

    const result = buildNavigationTree(config)
    
    const versions = result.find((n) => n.id === 'requiredVersions')
    expect(versions).toBeDefined()
  })

  it('should create state section when it exists', () => {
    const config = {
      state: {
        backend: 'azblob',
      },
    }

    const result = buildNavigationTree(config)
    
    const state = result.find((n) => n.id === 'state')
    expect(state).toBeDefined()
  })

  it('should build complete tree with all sections', () => {
    const config = {
      name: 'my-app',
      resourceGroup: 'rg-dev',
      services: {
        api: { host: 'containerapp' },
      },
      resources: {
        storage: { type: 'Microsoft.Storage/storageAccounts' },
      },
      hooks: {
        preprovision: { run: 'npm install' },
      },
      pipeline: {
        provider: 'github',
      },
      requiredVersions: {
        azd: '>=1.0.0',
      },
      state: {
        backend: 'azblob',
      },
    }

    const result = buildNavigationTree(config)
    
    expect(result).toHaveLength(7) // overview, services, resources, hooks, pipeline, requiredVersions, state
    expect(result.map((n) => n.id)).toEqual([
      'overview',
      'services',
      'resources',
      'hooks',
      'pipeline',
      'requiredVersions',
      'state',
    ])
  })

  it('should filter out missing overview properties', () => {
    const config = {
      name: 'my-app',
      // No resourceGroup or metadata
    }

    const result = buildNavigationTree(config)
    
    const overview = result.find((n) => n.id === 'overview')
    expect(overview?.children).toHaveLength(1)
    expect(overview?.children?.[0].id).toBe('name')
  })

  it('should set correct node types', () => {
    const config = {
      name: 'my-app',
      services: {
        api: { host: 'containerapp' },
      },
    }

    const result = buildNavigationTree(config)
    
    const overview = result.find((n) => n.id === 'overview')
    expect(overview?.type).toBe('section')
    expect(overview?.children?.[0].type).toBe('property')

    const services = result.find((n) => n.id === 'services')
    expect(services?.type).toBe('section')
    expect(services?.children?.[0].type).toBe('item')
  })
})
