/**
 * Resource Management Integration Example
 * 
 * This example demonstrates how to integrate ResourceConfigModal with:
 * - QuickActionsBar for one-click resource addition
 * - NavigationSidebar for resource listing
 * - Validation engine for circular dependency detection
 */

import * as React from 'react'
import { ResourceConfigModal } from '@/components/editor/modals'
import { QuickActionsBar } from '@/components/editor/QuickActionsBar'
import type { ResourceConfig } from '@/lib/editor/resource-types'
import type { WellKnownService } from '@/lib/editor/wellknown-types'

interface ResourceManagementExampleProps {
  /** Current azure.yaml configuration */
  config: {
    services?: Record<string, unknown>
    resources?: Record<string, ResourceConfig>
  }
  
  /** Callback when resource is added/updated */
  onResourceChange: (resources: Record<string, ResourceConfig>) => void
}

export function ResourceManagementExample({
  config,
  onResourceChange,
}: ResourceManagementExampleProps) {
  const [isModalOpen, setIsModalOpen] = React.useState(false)
  const [editingResource, setEditingResource] = React.useState<ResourceConfig | undefined>()

  // Get list of available services and resources for dependency selection
  const availableServices = React.useMemo(
    () => Object.keys(config.services || {}),
    [config.services]
  )

  const availableResources = React.useMemo(
    () => Object.keys(config.resources || {}),
    [config.resources]
  )

  const existingResourceNames = React.useMemo(
    () => Object.keys(config.resources || {}),
    [config.resources]
  )

  // Handle adding new resource
  const handleAddResource = () => {
    setEditingResource(undefined)
    setIsModalOpen(true)
  }

  // Handle editing existing resource
  const handleEditResource = (resourceName: string) => {
    const resource = config.resources?.[resourceName]
    if (resource) {
      setEditingResource(resource)
      setIsModalOpen(true)
    }
  }

  // Handle saving resource
  const handleSaveResource = (resource: ResourceConfig) => {
    const updatedResources = {
      ...(config.resources || {}),
      [resource.name]: resource,
    }
    onResourceChange(updatedResources)
    setIsModalOpen(false)
  }

  // Handle deleting resource
  const handleDeleteResource = (resourceName: string) => {
    const updatedResources = { ...(config.resources || {}) }
    delete updatedResources[resourceName]
    onResourceChange(updatedResources)
  }

  // Mock well-known services for QuickActionsBar
  // In real implementation, these would come from the backend API
  const wellKnownServices: WellKnownService[] = [
    {
      name: 'azurite',
      displayName: 'Azurite',
      description: 'Azure Storage Emulator',
      category: 'storage',
      icon: '💾',
      host: 'containerapp',
      image: 'mcr.microsoft.com/azure-storage/azurite:latest',
      ports: ['10000:10000', '10001:10001', '10002:10002'],
      environment: {},
    },
    {
      name: 'cosmos',
      displayName: 'Cosmos DB',
      description: 'Azure Cosmos DB Emulator',
      category: 'database',
      icon: '🗄️',
      host: 'containerapp',
      image: 'mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest',
      ports: ['8081:8081'],
      environment: {
        AZURE_COSMOS_EMULATOR_PARTITION_COUNT: '10',
        AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE: 'true',
      },
    },
    {
      name: 'redis',
      displayName: 'Redis',
      description: 'Redis Cache',
      category: 'cache',
      icon: '🔴',
      host: 'containerapp',
      image: 'redis:alpine',
      ports: ['6379:6379'],
      environment: {},
    },
    {
      name: 'postgres',
      displayName: 'PostgreSQL',
      description: 'PostgreSQL Database',
      category: 'database',
      icon: '🐘',
      host: 'containerapp',
      image: 'postgres:16-alpine',
      ports: ['5432:5432'],
      environment: {
        POSTGRES_PASSWORD: 'postgres',
        POSTGRES_DB: 'app',
      },
    },
  ]

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      {/* Main Content */}
      <div className="container mx-auto px-4 py-8">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-3xl font-bold text-slate-900 dark:text-slate-100 mb-6">
            Resource Management Example
          </h1>

          {/* Add Resource Button */}
          <div className="mb-6">
            <button
              onClick={handleAddResource}
              className="px-4 py-2 bg-cyan-600 text-white rounded-lg hover:bg-cyan-700 transition-colors"
            >
              + Add Resource
            </button>
          </div>

          {/* Resources List */}
          <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
            <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100 mb-4">
              Resources
            </h2>

            {Object.keys(config.resources || {}).length === 0 ? (
              <p className="text-slate-500 dark:text-slate-400 text-sm">
                No resources configured. Click "Add Resource" to get started.
              </p>
            ) : (
              <div className="space-y-3">
                {Object.entries(config.resources || {}).map(([name, resource]) => (
                  <div
                    key={name}
                    className="flex items-center justify-between p-4 rounded-lg border border-slate-200 dark:border-slate-700"
                  >
                    <div className="flex-1">
                      <div className="font-semibold text-slate-900 dark:text-slate-100">
                        {name}
                      </div>
                      <div className="text-sm text-slate-600 dark:text-slate-400">
                        {resource.type}
                      </div>
                      {resource.uses && resource.uses.length > 0 && (
                        <div className="text-xs text-slate-500 dark:text-slate-500 mt-1">
                          Uses: {resource.uses.join(', ')}
                        </div>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleEditResource(name)}
                        className="px-3 py-1 text-sm text-cyan-600 hover:text-cyan-700"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDeleteResource(name)}
                        className="px-3 py-1 text-sm text-red-600 hover:text-red-700"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Configuration Preview */}
          <div className="mt-6 bg-slate-800 rounded-lg p-4">
            <h3 className="text-sm font-semibold text-slate-300 mb-2">
              Configuration Preview (YAML)
            </h3>
            <pre className="text-xs text-slate-300 overflow-x-auto">
              {`resources:
${Object.entries(config.resources || {})
  .map(([name, resource]) => {
    let yaml = `  ${name}:
    type: ${resource.type}`
    
    if (resource.uses && resource.uses.length > 0) {
      yaml += `
    uses:
${resource.uses.map(u => `      - ${u}`).join('\n')}`
    }
    
    if (resource.existing) {
      yaml += `
    existing: true`
    }
    
    if (resource.containers && resource.containers.length > 0) {
      yaml += `
    containers:
${resource.containers.map(c => `      - ${c}`).join('\n')}`
    }
    
    if (resource.hubs && resource.hubs.length > 0) {
      yaml += `
    hubs:
${resource.hubs.map(h => `      - ${h}`).join('\n')}`
    }
    
    if (resource.queues && resource.queues.length > 0) {
      yaml += `
    queues:
${resource.queues.map(q => `      - ${q}`).join('\n')}`
    }
    
    if (resource.topics && resource.topics.length > 0) {
      yaml += `
    topics:
${resource.topics.map(t => `      - ${t}`).join('\n')}`
    }
    
    return yaml
  })
  .join('\n')}`}
            </pre>
          </div>
        </div>
      </div>

      {/* Quick Actions Bar */}
      <QuickActionsBar
        services={wellKnownServices}
        onAddService={(service) => {
          console.log('Quick add service:', service)
          // In real implementation, this would add the service to config
        }}
      />

      {/* Resource Config Modal */}
      <ResourceConfigModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSave={handleSaveResource}
        initialConfig={editingResource}
        availableServices={availableServices}
        availableResources={availableResources}
        existingResourceNames={existingResourceNames}
        currentConfig={config}
      />
    </div>
  )
}

/**
 * Usage Example:
 * 
 * ```tsx
 * import { ResourceManagementExample } from './ResourceManagementExample'
 * 
 * function App() {
 *   const [config, setConfig] = useState({
 *     services: {
 *       api: { host: 'containerapp', project: './api' },
 *       web: { host: 'containerapp', project: './web' },
 *     },
 *     resources: {},
 *   })
 * 
 *   return (
 *     <ResourceManagementExample
 *       config={config}
 *       onResourceChange={(resources) => {
 *         setConfig({ ...config, resources })
 *       }}
 *     />
 *   )
 * }
 * ```
 * 
 * Features Demonstrated:
 * 
 * 1. **Add Resource**: Click "+ Add Resource" to open modal
 *    - Select resource type from visual grid
 *    - Choose template or configure manually
 *    - Add dependencies with circular detection
 *    - Configure type-specific fields (containers, hubs, etc.)
 * 
 * 2. **Edit Resource**: Click "Edit" on any resource
 *    - Load existing configuration
 *    - Modify dependencies
 *    - Update type-specific fields
 * 
 * 3. **Delete Resource**: Click "Delete" on any resource
 *    - Removes resource from configuration
 * 
 * 4. **Circular Dependency Detection**:
 *    - Try creating: api -> storage -> api
 *    - Modal will show error and prevent save
 * 
 * 5. **Integration with QuickActionsBar**:
 *    - Quick buttons for common services
 *    - One-click service addition
 * 
 * 6. **Real-time YAML Preview**:
 *    - Shows generated YAML as resources are added
 *    - Validates configuration structure
 */
