/**
 * Resource Configuration Modal
 * Modal dialog for adding/editing Azure resources with support for:
 * - Resource type selection
 * - Dependency management
 * - Circular dependency detection
 * - Resource templates
 * - Type-specific configuration
 */

import * as React from 'react'
import { useForm } from 'react-hook-form'
import { Settings, AlertTriangle, CheckCircle2 } from 'lucide-react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { ResourceTypeSelector } from './ResourceTypeSelector'
import { ResourceTemplateSelector } from './ResourceTemplateSelector'
import { DependencySelector } from './DependencySelector'
import type {
  ResourceFormData,
  ResourceConfig,
  ResourceType,
  ResourceTemplate,
} from '@/lib/editor/resource-types'
import {
  formDataToResource,
  resourceToFormData,
  getResourceNameError,
  getResourceType,
} from '@/lib/editor/resource-types'
import { validateCircularDependencies } from '@/lib/editor/validation-engine'

export interface ResourceConfigModalProps {
  /** Whether modal is open */
  isOpen: boolean
  
  /** Callback to close modal */
  onClose: () => void
  
  /** Callback when resource is saved */
  onSave: (resource: ResourceConfig) => void | Promise<void>
  
  /** Existing resource config (for editing) */
  initialConfig?: ResourceConfig
  
  /** Available services for dependency selection */
  availableServices?: string[]
  
  /** Available resources for dependency selection */
  availableResources?: string[]
  
  /** Existing resource names for validation */
  existingResourceNames?: string[]
  
  /** Current configuration for circular dependency checking */
  currentConfig?: Record<string, unknown>
}

/**
 * Resource Configuration Modal Component
 */
export function ResourceConfigModal({
  isOpen,
  onClose,
  onSave,
  initialConfig,
  availableServices = [],
  availableResources = [],
  existingResourceNames = [],
  currentConfig = {},
}: ResourceConfigModalProps) {
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [selectedType, setSelectedType] = React.useState<ResourceType | undefined>()
  const [showTemplates, setShowTemplates] = React.useState(!initialConfig)
  const [circularDepError, setCircularDepError] = React.useState<string | null>(null)

  const isEditing = !!initialConfig

  // Initialize form with default values
  const defaultValues = React.useMemo<ResourceFormData>(() => {
    if (initialConfig) {
      return resourceToFormData(initialConfig)
    }
    return {
      name: '',
      type: '',
      uses: [],
      existing: false,
      containers: [],
      databases: [],
      hubs: [],
      queues: [],
      topics: [],
    }
  }, [initialConfig])

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
    reset,
    setError,
  } = useForm<ResourceFormData>({
    defaultValues,
  })

  const resourceType = watch('type')
  const resourceName = watch('name')
  const dependencies = watch('uses')

  // Resolve selected type object for templates and display
  const templateType = React.useMemo(() => {
    if (selectedType) {
      return selectedType
    }
    if (resourceType) {
      return getResourceType(resourceType)
    }
    return undefined
  }, [resourceType, selectedType])

  // Update selected type when type changes
  React.useEffect(() => {
    if (resourceType) {
      setSelectedType(getResourceType(resourceType))
    }
  }, [resourceType])

  // Reset form when modal opens/closes
  React.useEffect(() => {
    if (isOpen) {
      reset(defaultValues)
      setShowTemplates(!initialConfig)
      setCircularDepError(null)
    }
  }, [isOpen, defaultValues, initialConfig, reset])

  // Validate circular dependencies when dependencies change
  React.useEffect(() => {
    if (!resourceName || dependencies.length === 0) {
      setCircularDepError(null)
      return
    }

    // Build hypothetical config with this resource
    const testConfig = {
      ...currentConfig,
      resources: {
        ...(currentConfig.resources as Record<string, unknown> || {}),
        [resourceName]: {
          type: resourceType,
          uses: dependencies,
        },
      },
    }

    // Check for circular dependencies
    const errors = validateCircularDependencies(testConfig)
    const circularErrors = errors.filter(e => 
      e.rule === 'circular-dependency' && e.message.includes(resourceName)
    )

    if (circularErrors.length > 0) {
      setCircularDepError(circularErrors[0].message)
    } else {
      setCircularDepError(null)
    }
  }, [resourceName, dependencies, resourceType, currentConfig])

  // Handle resource type selection
  const handleTypeSelect = (type: ResourceType) => {
    setValue('type', type.id)
    setShowTemplates(true)
  }

  // Handle template application
  const handleTemplateApply = (template: ResourceTemplate) => {
    // Apply template defaults
    if (template.config.containers) {
      setValue('containers', template.config.containers)
    }
    if (template.config.databases) {
      setValue('databases', template.config.databases)
    }
    if (template.config.hubs) {
      setValue('hubs', template.config.hubs)
    }
    if (template.config.queues) {
      setValue('queues', template.config.queues)
    }
    if (template.config.topics) {
      setValue('topics', template.config.topics)
    }
    if (template.config.existing !== undefined) {
      setValue('existing', template.config.existing)
    }
    
    setShowTemplates(false)
  }

  // Handle form submission
  const onSubmit = async (data: ResourceFormData) => {
    // Validate resource name
    const nameError = getResourceNameError(data.name)
    if (nameError) {
      setError('name', { message: nameError })
      return
    }

    // Check for duplicate name (skip if editing same resource)
    if (!isEditing || data.name !== initialConfig?.name) {
      if (existingResourceNames.includes(data.name)) {
        setError('name', { 
          message: `A resource named "${data.name}" already exists. Please choose a different name.` 
        })
        return
      }
    }

    // Check for circular dependencies
    if (circularDepError) {
      alert(`Cannot save: ${circularDepError}`)
      return
    }

    try {
      setIsSubmitting(true)
      const config = formDataToResource(data)
      await onSave(config)
      onClose()
    } catch (error) {
      console.error('Failed to save resource:', error)
      alert('Failed to save resource. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  // Get available items for dependency selection
  const availableItems = React.useMemo(() => {
    // Combine services and other resources (excluding current resource)
    const items = [
      ...availableServices,
      ...availableResources.filter(r => r !== resourceName),
    ]
    return items
  }, [availableServices, availableResources, resourceName])

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="3xl">
      <DialogHeader onClose={onClose}>
        <div className="flex items-center gap-2">
          <Settings className="w-5 h-5 text-cyan-600" />
          <DialogTitle>
            {isEditing ? 'Edit Resource' : 'Add Resource Configuration'}
          </DialogTitle>
        </div>
        <DialogDescription>
          Configure Azure resource with dependencies and type-specific settings
        </DialogDescription>
      </DialogHeader>

      <form onSubmit={handleSubmit(onSubmit)}>
        <DialogContent>
          <div className="space-y-6">
            {/* Resource Type Selection */}
            {!isEditing && !resourceType && (
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Resource Type <span className="text-red-500">*</span>
                </label>
                <ResourceTypeSelector
                  onSelect={handleTypeSelect}
                />
              </div>
            )}

            {/* Template Selection (only for new resources with selected type) */}
            {!isEditing && resourceType && showTemplates && templateType && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
                    Resource Template (Optional)
                  </label>
                  <button
                    type="button"
                    onClick={() => setShowTemplates(false)}
                    className="text-xs text-cyan-600 hover:text-cyan-700 font-medium"
                  >
                    Skip template selection →
                  </button>
                </div>
                <ResourceTemplateSelector
                  resourceType={templateType as ResourceType}
                  onSelect={handleTemplateApply}
                  onSkip={() => setShowTemplates(false)}
                />
              </div>
            )}

            {/* Resource Configuration Form (shows when type selected and templates skipped or applied) */}
            {resourceType && !showTemplates && (
              <>
                {/* Resource Name */}
                <div>
                  <label
                    htmlFor="resource-name"
                    className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                  >
                    Resource Name <span className="text-red-500">*</span>
                  </label>
                  <Input
                    id="resource-name"
                    type="text"
                    {...register('name', {
                      required: 'Resource name is required',
                      validate: (value) => {
                        const error = getResourceNameError(value)
                        return error ? error : true
                      },
                    })}
                    placeholder="my-storage"
                    className={cn(errors.name && 'border-red-500')}
                    aria-invalid={errors.name ? 'true' : 'false'}
                    aria-describedby={errors.name ? 'resource-name-error' : 'resource-name-help'}
                    disabled={isEditing}
                  />
                  {errors.name ? (
                    <p id="resource-name-error" className="mt-1 text-xs text-red-500">
                      {errors.name.message}
                    </p>
                  ) : (
                    <p id="resource-name-help" className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      Lowercase letters, numbers, and hyphens only
                    </p>
                  )}
                </div>

                {/* Resource Type Display (for editing) */}
                {isEditing && (
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                      Resource Type
                    </label>
                    <div className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
                      <div className="flex items-center gap-2">
                        <span className="text-xl">{selectedType?.icon}</span>
                        <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
                          {selectedType?.displayName}
                        </span>
                      </div>
                    </div>
                  </div>
                )}

                {/* Dependencies */}
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Dependencies (uses)
                  </label>
                  <DependencySelector
                    available={availableItems}
                    selected={dependencies}
                    onChange={(selected) => setValue('uses', selected)}
                    error={circularDepError}
                  />
                  {circularDepError && (
                    <div className="mt-2 p-3 rounded-lg bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900">
                      <div className="flex gap-2">
                        <AlertTriangle className="w-4 h-4 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
                        <p className="text-xs text-red-900 dark:text-red-100">
                          {circularDepError}
                        </p>
                      </div>
                    </div>
                  )}
                  {!circularDepError && dependencies.length > 0 && (
                    <div className="mt-2 p-3 rounded-lg bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-900">
                      <div className="flex gap-2">
                        <CheckCircle2 className="w-4 h-4 text-green-600 dark:text-green-400 shrink-0 mt-0.5" />
                        <p className="text-xs text-green-900 dark:text-green-100">
                          No circular dependencies detected
                        </p>
                      </div>
                    </div>
                  )}
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Select services or resources this resource depends on
                  </p>
                </div>

                {/* Existing Resource Toggle */}
                <div className="flex items-center gap-2">
                  <input
                    id="resource-existing"
                    type="checkbox"
                    {...register('existing')}
                    className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                  />
                  <label htmlFor="resource-existing" className="text-sm text-slate-700 dark:text-slate-300">
                    This is a pre-existing resource
                  </label>
                </div>

                {/* Type-Specific Fields */}
                {selectedType?.supportsContainers && (
                  <div>
                    <label
                      htmlFor="resource-containers"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Containers (comma-separated)
                    </label>
                    <Input
                      id="resource-containers"
                      type="text"
                      {...register('containers', {
                        setValueAs: (value: string | string[]) => {
                          if (Array.isArray(value)) return value
                          return value ? value.split(',').map(s => s.trim()).filter(Boolean) : []
                        },
                      })}
                      placeholder="uploads, static, backups"
                    />
                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      List of blob containers to create (for Storage Accounts)
                    </p>
                  </div>
                )}

                {selectedType?.id === 'Microsoft.EventHub/namespaces' && (
                  <div>
                    <label
                      htmlFor="resource-hubs"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Event Hubs (comma-separated)
                    </label>
                    <Input
                      id="resource-hubs"
                      type="text"
                      {...register('hubs', {
                        setValueAs: (value: string | string[]) => {
                          if (Array.isArray(value)) return value
                          return value ? value.split(',').map(s => s.trim()).filter(Boolean) : []
                        },
                      })}
                      placeholder="events, telemetry"
                    />
                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      List of Event Hub names to create
                    </p>
                  </div>
                )}

                {selectedType?.id === 'Microsoft.ServiceBus/namespaces' && (
                  <>
                    <div>
                      <label
                        htmlFor="resource-queues"
                        className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                      >
                        Queues (comma-separated)
                      </label>
                      <Input
                        id="resource-queues"
                        type="text"
                        {...register('queues', {
                          setValueAs: (value: string | string[]) => {
                            if (Array.isArray(value)) return value
                            return value ? value.split(',').map(s => s.trim()).filter(Boolean) : []
                          },
                        })}
                        placeholder="messages, jobs"
                      />
                      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        List of Service Bus queues to create
                      </p>
                    </div>

                    <div>
                      <label
                        htmlFor="resource-topics"
                        className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                      >
                        Topics (comma-separated)
                      </label>
                      <Input
                        id="resource-topics"
                        type="text"
                        {...register('topics', {
                          setValueAs: (value: string | string[]) => {
                            if (Array.isArray(value)) return value
                            return value ? value.split(',').map(s => s.trim()).filter(Boolean) : []
                          },
                        })}
                        placeholder="notifications, alerts"
                      />
                      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        List of Service Bus topics to create
                      </p>
                    </div>
                  </>
                )}

                {/* Info Panel */}
                <div className="rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900 p-4">
                  <div className="flex gap-2">
                    <Settings className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
                    <div className="text-sm text-blue-900 dark:text-blue-100">
                      <p className="font-medium mb-1">Resource Tips</p>
                      <ul className="text-xs space-y-1 text-blue-800 dark:text-blue-200">
                        <li>• Use dependencies to express which services need this resource</li>
                        <li>• Circular dependencies are automatically detected and prevented</li>
                        <li>• Mark as "existing" if the resource is already provisioned</li>
                        {selectedType?.supportsContainers && (
                          <li>• Containers will be created automatically during provisioning</li>
                        )}
                      </ul>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        </DialogContent>

        <DialogFooter>
          <div className="flex items-center justify-end gap-3 w-full">
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'px-4 py-2 rounded-lg text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'transition-colors duration-150'
              )}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting || !resourceType || showTemplates || !!circularDepError}
              className={cn(
                'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                'bg-cyan-600 text-white hover:bg-cyan-700',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'transition-colors duration-150'
              )}
            >
              {isSubmitting ? 'Saving...' : isEditing ? 'Save Resource' : 'Add Resource'}
            </button>
          </div>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
