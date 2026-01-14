/**
 * Resource Types - Type definitions for Azure resource configuration
 */

export type ResourceCategory = 'storage' | 'database' | 'messaging' | 'compute' | 'other'

export interface ResourceType {
  /** Resource type identifier (e.g., "Microsoft.Storage/storageAccounts") */
  id: string
  
  /** Display name for UI */
  displayName: string
  
  /** Description */
  description: string
  
  /** Category for grouping */
  category: ResourceCategory
  
  /** Icon identifier or emoji */
  icon?: string
  
  /** Whether this type supports containers/queues/topics */
  supportsContainers?: boolean
  
  /** Whether this type supports databases */
  supportsDatabases?: boolean
}

export interface ResourceTemplate {
  /** Template identifier */
  id: string
  
  /** Display name */
  name: string
  
  /** Description */
  description: string
  
  /** Resource type this template is for */
  resourceType: string
  
  /** Icon */
  icon?: string
  
  /** Template configuration */
  config: Partial<ResourceConfig>
}

export interface ResourceConfig {
  /** Resource name (unique identifier) */
  name: string
  
  /** Resource type */
  type: string
  
  /** Services or resources this resource depends on */
  uses?: string[]
  
  /** Whether this is a pre-existing resource */
  existing?: boolean
  
  // Storage Account specific
  containers?: string[]
  
  // Cosmos DB specific
  databases?: CosmosDatabase[]
  
  // Event Hubs specific
  hubs?: string[]
  
  // Service Bus specific
  queues?: string[]
  topics?: string[]
}

export interface CosmosDatabase {
  /** Database name */
  name: string
  
  /** Containers in this database */
  containers?: CosmosContainer[]
}

export interface CosmosContainer {
  /** Container name */
  name: string
  
  /** Partition key path */
  partitionKey: string
}

export interface ResourceFormData {
  /** Resource name */
  name: string
  
  /** Resource type */
  type: string
  
  /** Dependencies */
  uses: string[]
  
  /** Is existing */
  existing: boolean
  
  /** Type-specific fields */
  containers?: string[]
  databases?: CosmosDatabase[]
  hubs?: string[]
  queues?: string[]
  topics?: string[]
}

/**
 * Common Azure resource types
 */
export const RESOURCE_TYPES: ResourceType[] = [
  {
    id: 'Microsoft.Storage/storageAccounts',
    displayName: 'Storage Account',
    description: 'Azure Storage (Blob, Queue, Table, File)',
    category: 'storage',
    icon: '💾',
    supportsContainers: true,
  },
  {
    id: 'Microsoft.DocumentDB/databaseAccounts',
    displayName: 'Cosmos DB',
    description: 'Azure Cosmos DB (NoSQL database)',
    category: 'database',
    icon: '🗄️',
    supportsDatabases: true,
  },
  {
    id: 'Microsoft.EventHub/namespaces',
    displayName: 'Event Hubs',
    description: 'Event streaming platform',
    category: 'messaging',
    icon: '📨',
  },
  {
    id: 'Microsoft.ServiceBus/namespaces',
    displayName: 'Service Bus',
    description: 'Message broker service',
    category: 'messaging',
    icon: '🚌',
  },
  {
    id: 'Microsoft.Web/sites',
    displayName: 'App Service',
    description: 'Web app hosting service',
    category: 'compute',
    icon: '🌐',
  },
  {
    id: 'Microsoft.KeyVault/vaults',
    displayName: 'Key Vault',
    description: 'Secrets management service',
    category: 'other',
    icon: '🔐',
  },
  {
    id: 'Microsoft.Sql/servers',
    displayName: 'SQL Server',
    description: 'Azure SQL Database',
    category: 'database',
    icon: '🗃️',
  },
  {
    id: 'Microsoft.ContainerRegistry/registries',
    displayName: 'Container Registry',
    description: 'Docker container registry',
    category: 'compute',
    icon: '🐳',
  },
]

/**
 * Resource templates for common scenarios
 */
export const RESOURCE_TEMPLATES: ResourceTemplate[] = [
  {
    id: 'storage-blob',
    name: 'Blob Storage',
    description: 'Storage account with blob containers',
    resourceType: 'Microsoft.Storage/storageAccounts',
    icon: '💾',
    config: {
      type: 'Microsoft.Storage/storageAccounts',
      containers: ['uploads', 'static'],
    },
  },
  {
    id: 'storage-queue',
    name: 'Queue Storage',
    description: 'Storage account with queues',
    resourceType: 'Microsoft.Storage/storageAccounts',
    icon: '📬',
    config: {
      type: 'Microsoft.Storage/storageAccounts',
    },
  },
  {
    id: 'cosmos-sql',
    name: 'Cosmos DB (SQL API)',
    description: 'Cosmos DB with SQL API database and container',
    resourceType: 'Microsoft.DocumentDB/databaseAccounts',
    icon: '🗄️',
    config: {
      type: 'Microsoft.DocumentDB/databaseAccounts',
      databases: [
        {
          name: 'main',
          containers: [
            {
              name: 'items',
              partitionKey: '/id',
            },
          ],
        },
      ],
    },
  },
  {
    id: 'eventhub-standard',
    name: 'Event Hub (Standard)',
    description: 'Event Hub namespace with default hub',
    resourceType: 'Microsoft.EventHub/namespaces',
    icon: '📨',
    config: {
      type: 'Microsoft.EventHub/namespaces',
      hubs: ['events'],
    },
  },
  {
    id: 'servicebus-queue',
    name: 'Service Bus Queue',
    description: 'Service Bus with queue',
    resourceType: 'Microsoft.ServiceBus/namespaces',
    icon: '🚌',
    config: {
      type: 'Microsoft.ServiceBus/namespaces',
      queues: ['messages'],
    },
  },
  {
    id: 'servicebus-topic',
    name: 'Service Bus Topic',
    description: 'Service Bus with topic/subscription',
    resourceType: 'Microsoft.ServiceBus/namespaces',
    icon: '📢',
    config: {
      type: 'Microsoft.ServiceBus/namespaces',
      topics: ['notifications'],
    },
  },
]

/**
 * Get resource type by ID
 */
export function getResourceType(id: string): ResourceType | undefined {
  return RESOURCE_TYPES.find(rt => rt.id === id)
}

/**
 * Get resource templates by type
 */
export function getTemplatesForType(resourceTypeId: string): ResourceTemplate[] {
  return RESOURCE_TEMPLATES.filter(t => t.resourceType === resourceTypeId)
}

/**
 * Get all templates for a category
 */
export function getTemplatesForCategory(category: ResourceCategory): ResourceTemplate[] {
  const typeIds = RESOURCE_TYPES
    .filter(rt => rt.category === category)
    .map(rt => rt.id)
  
  return RESOURCE_TEMPLATES.filter(t => typeIds.includes(t.resourceType))
}

/**
 * Validate resource name (lowercase, alphanumeric, hyphens)
 */
export function validateResourceName(name: string): boolean {
  return /^[a-z][a-z0-9-]*[a-z0-9]$/.test(name)
}

/**
 * Get resource name validation error message
 */
export function getResourceNameError(name: string): string | null {
  if (!name) {
    return 'Resource name is required'
  }
  if (name.length < 2) {
    return 'Resource name must be at least 2 characters'
  }
  if (!validateResourceName(name)) {
    return 'Resource name must start with a letter and contain only lowercase letters, numbers, and hyphens'
  }
  return null
}

/**
 * Convert form data to resource config
 */
export function formDataToResource(data: ResourceFormData): ResourceConfig {
  const config: ResourceConfig = {
    name: data.name,
    type: data.type,
  }

  if (data.uses && data.uses.length > 0) {
    config.uses = data.uses
  }

  if (data.existing) {
    config.existing = true
  }

  // Add type-specific fields
  if (data.containers && data.containers.length > 0) {
    config.containers = data.containers
  }

  if (data.databases && data.databases.length > 0) {
    config.databases = data.databases
  }

  if (data.hubs && data.hubs.length > 0) {
    config.hubs = data.hubs
  }

  if (data.queues && data.queues.length > 0) {
    config.queues = data.queues
  }

  if (data.topics && data.topics.length > 0) {
    config.topics = data.topics
  }

  return config
}

/**
 * Convert resource config to form data
 */
export function resourceToFormData(config: ResourceConfig): ResourceFormData {
  return {
    name: config.name,
    type: config.type,
    uses: config.uses || [],
    existing: config.existing || false,
    containers: config.containers || [],
    databases: config.databases || [],
    hubs: config.hubs || [],
    queues: config.queues || [],
    topics: config.topics || [],
  }
}
