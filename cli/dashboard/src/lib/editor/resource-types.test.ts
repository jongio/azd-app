/**
 * Resource Types Tests
 */

import { describe, it, expect } from 'vitest'
import {
  RESOURCE_TYPES,
  RESOURCE_TEMPLATES,
  getResourceType,
  getTemplatesForType,
  getTemplatesForCategory,
  validateResourceName,
  getResourceNameError,
  formDataToResource,
  resourceToFormData,
  type ResourceFormData,
  type ResourceConfig,
} from './resource-types'

describe('Resource Types', () => {
  describe('RESOURCE_TYPES', () => {
    it('should contain all major Azure resource types', () => {
      expect(RESOURCE_TYPES.length).toBeGreaterThan(0)
      
      const typeIds = RESOURCE_TYPES.map(t => t.id)
      expect(typeIds).toContain('Microsoft.Storage/storageAccounts')
      expect(typeIds).toContain('Microsoft.DocumentDB/databaseAccounts')
      expect(typeIds).toContain('Microsoft.EventHub/namespaces')
      expect(typeIds).toContain('Microsoft.ServiceBus/namespaces')
    })

    it('should have valid categories', () => {
      const validCategories = ['storage', 'database', 'messaging', 'compute', 'other']
      
      for (const type of RESOURCE_TYPES) {
        expect(validCategories).toContain(type.category)
      }
    })

    it('should have display names and descriptions', () => {
      for (const type of RESOURCE_TYPES) {
        expect(type.displayName).toBeTruthy()
        expect(type.description).toBeTruthy()
      }
    })
  })

  describe('RESOURCE_TEMPLATES', () => {
    it('should contain common resource templates', () => {
      expect(RESOURCE_TEMPLATES.length).toBeGreaterThan(0)
      
      const templateIds = RESOURCE_TEMPLATES.map(t => t.id)
      expect(templateIds).toContain('storage-blob')
      expect(templateIds).toContain('cosmos-sql')
      expect(templateIds).toContain('eventhub-standard')
      expect(templateIds).toContain('servicebus-queue')
    })

    it('should have valid resource types', () => {
      const validTypeIds = RESOURCE_TYPES.map(t => t.id)
      
      for (const template of RESOURCE_TEMPLATES) {
        expect(validTypeIds).toContain(template.resourceType)
      }
    })

    it('should have configuration objects', () => {
      for (const template of RESOURCE_TEMPLATES) {
        expect(template.config).toBeDefined()
        expect(template.config.type).toBeTruthy()
      }
    })
  })

  describe('getResourceType', () => {
    it('should return resource type by ID', () => {
      const type = getResourceType('Microsoft.Storage/storageAccounts')
      expect(type).toBeDefined()
      expect(type?.displayName).toBe('Storage Account')
    })

    it('should return undefined for invalid ID', () => {
      const type = getResourceType('Invalid.Type/invalid')
      expect(type).toBeUndefined()
    })
  })

  describe('getTemplatesForType', () => {
    it('should return templates for Storage Account', () => {
      const templates = getTemplatesForType('Microsoft.Storage/storageAccounts')
      expect(templates.length).toBeGreaterThan(0)
      expect(templates.every(t => t.resourceType === 'Microsoft.Storage/storageAccounts')).toBe(true)
    })

    it('should return templates for Cosmos DB', () => {
      const templates = getTemplatesForType('Microsoft.DocumentDB/databaseAccounts')
      expect(templates.length).toBeGreaterThan(0)
      expect(templates[0].id).toBe('cosmos-sql')
    })

    it('should return empty array for type with no templates', () => {
      const templates = getTemplatesForType('Microsoft.KeyVault/vaults')
      expect(templates).toEqual([])
    })
  })

  describe('getTemplatesForCategory', () => {
    it('should return templates for storage category', () => {
      const templates = getTemplatesForCategory('storage')
      expect(templates.length).toBeGreaterThan(0)
      expect(templates.every(t => {
        const type = getResourceType(t.resourceType)
        return type?.category === 'storage'
      })).toBe(true)
    })

    it('should return templates for messaging category', () => {
      const templates = getTemplatesForCategory('messaging')
      expect(templates.length).toBeGreaterThan(0)
    })

    it('should return empty array for category with no templates', () => {
      const templates = getTemplatesForCategory('compute')
      expect(templates).toEqual([])
    })
  })

  describe('validateResourceName', () => {
    it('should accept valid resource names', () => {
      expect(validateResourceName('my-storage')).toBe(true)
      expect(validateResourceName('storage123')).toBe(true)
      expect(validateResourceName('app-db-01')).toBe(true)
    })

    it('should reject names starting with non-letter', () => {
      expect(validateResourceName('123storage')).toBe(false)
      expect(validateResourceName('-storage')).toBe(false)
    })

    it('should reject names ending with hyphen', () => {
      expect(validateResourceName('storage-')).toBe(false)
    })

    it('should reject names with uppercase letters', () => {
      expect(validateResourceName('MyStorage')).toBe(false)
      expect(validateResourceName('STORAGE')).toBe(false)
    })

    it('should reject names with special characters', () => {
      expect(validateResourceName('my_storage')).toBe(false)
      expect(validateResourceName('my.storage')).toBe(false)
      expect(validateResourceName('my storage')).toBe(false)
    })

    it('should reject single character names', () => {
      expect(validateResourceName('a')).toBe(false)
    })
  })

  describe('getResourceNameError', () => {
    it('should return error for empty name', () => {
      expect(getResourceNameError('')).toBe('Resource name is required')
    })

    it('should return error for too short name', () => {
      expect(getResourceNameError('a')).toBe('Resource name must be at least 2 characters')
    })

    it('should return error for invalid format', () => {
      const error = getResourceNameError('Invalid Name!')
      expect(error).toContain('lowercase letters, numbers, and hyphens')
    })

    it('should return null for valid name', () => {
      expect(getResourceNameError('my-storage')).toBeNull()
      expect(getResourceNameError('storage123')).toBeNull()
    })
  })

  describe('formDataToResource', () => {
    it('should convert basic form data to resource config', () => {
      const formData: ResourceFormData = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: [],
        existing: false,
        containers: [],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      }

      const config = formDataToResource(formData)

      expect(config).toEqual({
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
      })
    })

    it('should include dependencies', () => {
      const formData: ResourceFormData = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: ['api', 'web'],
        existing: false,
        containers: [],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      }

      const config = formDataToResource(formData)

      expect(config.uses).toEqual(['api', 'web'])
    })

    it('should include existing flag', () => {
      const formData: ResourceFormData = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: [],
        existing: true,
        containers: [],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      }

      const config = formDataToResource(formData)

      expect(config.existing).toBe(true)
    })

    it('should include type-specific fields', () => {
      const formData: ResourceFormData = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: [],
        existing: false,
        containers: ['uploads', 'static'],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      }

      const config = formDataToResource(formData)

      expect(config.containers).toEqual(['uploads', 'static'])
    })

    it('should omit empty arrays', () => {
      const formData: ResourceFormData = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: [],
        existing: false,
        containers: [],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      }

      const config = formDataToResource(formData)

      expect(config).not.toHaveProperty('containers')
      expect(config).not.toHaveProperty('uses')
    })
  })

  describe('resourceToFormData', () => {
    it('should convert basic resource config to form data', () => {
      const config: ResourceConfig = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
      }

      const formData = resourceToFormData(config)

      expect(formData).toEqual({
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: [],
        existing: false,
        containers: [],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      })
    })

    it('should include all fields from config', () => {
      const config: ResourceConfig = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: ['api'],
        existing: true,
        containers: ['uploads'],
      }

      const formData = resourceToFormData(config)

      expect(formData.uses).toEqual(['api'])
      expect(formData.existing).toBe(true)
      expect(formData.containers).toEqual(['uploads'])
    })

    it('should handle all type-specific fields', () => {
      const config: ResourceConfig = {
        name: 'my-resources',
        type: 'Microsoft.ServiceBus/namespaces',
        queues: ['messages'],
        topics: ['notifications'],
      }

      const formData = resourceToFormData(config)

      expect(formData.queues).toEqual(['messages'])
      expect(formData.topics).toEqual(['notifications'])
    })
  })

  describe('round-trip conversion', () => {
    it('should preserve data through formData -> config -> formData', () => {
      const original: ResourceFormData = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        uses: ['api', 'web'],
        existing: true,
        containers: ['uploads', 'static'],
        databases: [],
        hubs: [],
        queues: [],
        topics: [],
      }

      const config = formDataToResource(original)
      const roundTrip = resourceToFormData(config)

      expect(roundTrip.name).toBe(original.name)
      expect(roundTrip.type).toBe(original.type)
      expect(roundTrip.uses).toEqual(original.uses)
      expect(roundTrip.existing).toBe(original.existing)
      expect(roundTrip.containers).toEqual(original.containers)
    })

    it('should preserve data through config -> formData -> config', () => {
      const original: ResourceConfig = {
        name: 'my-eventhub',
        type: 'Microsoft.EventHub/namespaces',
        hubs: ['events', 'telemetry'],
        uses: ['api'],
      }

      const formData = resourceToFormData(original)
      const roundTrip = formDataToResource(formData)

      expect(roundTrip.name).toBe(original.name)
      expect(roundTrip.type).toBe(original.type)
      expect(roundTrip.hubs).toEqual(original.hubs)
      expect(roundTrip.uses).toEqual(original.uses)
    })
  })
})
