/**
 * Editor State Management - Zustand Store
 * 
 * Central state management for the Azure YAML Editor.
 * Manages config, validation, active section, modals, and more.
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import type { ValidationError } from '@/lib/editor/validation-types'
import type { WellKnownService } from '@/lib/editor/wellknown-types'
import { parseYaml, updateYamlField, mergeYamlUpdates, deleteYamlPath } from '@/lib/editor/yaml-utils'
import type { Document } from 'yaml'
import { loadConfig, saveConfig as saveConfigApi } from '@/lib/editor/config-api'
import { validateConfiguration } from '@/lib/editor/validation-engine'

// =============================================================================
// Types
// =============================================================================

export interface EditorState {
  // Configuration
  config: Record<string, unknown> | null
  configYaml: string
  originalYaml: string
  originalDocument: Document | null // YAML document with comments preserved
  isDirty: boolean
  isLoading: boolean
  isSaving: boolean
  error: string | null

  // Navigation
  activeSection: string
  
  // Validation
  validationErrors: ValidationError[]
  
  // UI State
  isPreviewVisible: boolean
  isSidebarCollapsed: boolean
  isHelpPanelOpen: boolean
  helpSection: string | undefined
  
  // Modal State
  isAddServiceModalOpen: boolean
  isResourceConfigModalOpen: boolean
  isHealthCheckModalOpen: boolean
  isHooksConfigModalOpen: boolean
  isBackupManagerOpen: boolean
  isCommandPaletteOpen: boolean
  isImportModalOpen: boolean
  isExportModalOpen: boolean
  isKeyboardShortcutsOpen: boolean
  
  // Modal Context
  selectedService: string | null
  selectedResource: string | null
  
  // Actions - Config Management
  loadConfig: () => Promise<void>
  saveConfig: (schema?: Record<string, unknown>) => Promise<boolean>
  updateConfig: (updates: Record<string, unknown>) => void
  updateField: (path: string, value: unknown) => void
  resetConfig: () => void
  discardChanges: () => void
  
  // Actions - Navigation
  setActiveSection: (section: string) => void
  
  // Actions - Validation
  setValidationErrors: (errors: ValidationError[]) => void
  
  // Actions - UI
  togglePreview: () => void
  toggleSidebar: () => void
  toggleHelpPanel: (section?: string) => void
  
  // Actions - Modals
  openAddServiceModal: () => void
  closeAddServiceModal: () => void
  openResourceConfigModal: (resourceName: string) => void
  closeResourceConfigModal: () => void
  openHealthCheckModal: (serviceName: string) => void
  closeHealthCheckModal: () => void
  openHooksConfigModal: () => void
  closeHooksConfigModal: () => void
  openBackupManager: () => void
  closeBackupManager: () => void
  openCommandPalette: () => void
  closeCommandPalette: () => void
  openImportModal: () => void
  closeImportModal: () => void
  openExportModal: () => void
  closeExportModal: () => void
  openKeyboardShortcuts: () => void
  closeKeyboardShortcuts: () => void
  
  // Actions - Service Management
  addService: (name: string, service: WellKnownService) => void
  removeService: (name: string) => void
  
  // Actions - Resource Management
  addResource: (name: string, type: string) => void
  removeResource: (name: string) => void
  
  // Actions - Import/Export
  importConfig: (yaml: string) => boolean
  exportConfig: () => string
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Load draft from localStorage
 */
function loadDraft(): string | null {
  try {
    return localStorage.getItem('azd-editor-draft')
  } catch {
    return null
  }
}

/**
 * Save draft to localStorage
 */
function saveDraft(yaml: string): void {
  try {
    localStorage.setItem('azd-editor-draft', yaml)
  } catch {
    // Silently fail in private browsing mode
  }
}

/**
 * Clear draft from localStorage
 */
function clearDraft(): void {
  try {
    localStorage.removeItem('azd-editor-draft')
  } catch {
    // Silently fail
  }
}

// =============================================================================
// Zustand Store
// =============================================================================

export const useEditorState = create<EditorState>()(
  devtools(
    persist(
      (set, get) => ({
        // Initial State
        config: null,
        configYaml: '',
        originalYaml: '',
        originalDocument: null,
        isDirty: false,
        isLoading: false,
        isSaving: false,
        error: null,
        
        activeSection: 'overview',
        
        validationErrors: [],
        
        isPreviewVisible: true,
        isSidebarCollapsed: false,
        isHelpPanelOpen: false,
        helpSection: undefined,
        
        isAddServiceModalOpen: false,
        isResourceConfigModalOpen: false,
        isHealthCheckModalOpen: false,
        isHooksConfigModalOpen: false,
        isBackupManagerOpen: false,
        isCommandPaletteOpen: false,
        isImportModalOpen: false,
        isExportModalOpen: false,
        isKeyboardShortcutsOpen: false,
        
        selectedService: null,
        selectedResource: null,
        
        // Config Management
        loadConfig: async () => {
          set({ isLoading: true, error: null })
          
          try {
            // Check for draft first
            const draft = loadDraft()
            
            // Load from server
            const response = await loadConfig()
            const yaml = response.content
            
            // Parse YAML (preserves comments)
            const parsed = parseYaml(yaml)
            
            if (!parsed.success) {
              throw new Error(`YAML parse error: ${parsed.error}`)
            }
            
            // Use draft if available, otherwise use server response
            const configYaml = draft || yaml
            const configParsed = parseYaml(configYaml)
            
            set({
              config: configParsed.success ? (configParsed.data as Record<string, unknown>) : null,
              configYaml,
              originalYaml: yaml,
              originalDocument: parsed.document || null, // Store document with comments
              isDirty: draft !== null,
              isLoading: false,
            })
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : 'Failed to load configuration',
              isLoading: false,
            })
          }
        },
        
        saveConfig: async (schema?: Record<string, unknown>) => {
          const { configYaml, isSaving, config } = get()
          
          if (isSaving) return false
          
          set({ isSaving: true, error: null })
          
          try {
            // Step 1: Validate against schema if provided
            if (schema) {
              // Parse YAML to get config object for validation
              const parsed = parseYaml(configYaml)
              
              if (!parsed.success || !parsed.data) {
                throw new Error(`Invalid YAML: ${parsed.error || 'Failed to parse YAML'}`)
              }
              
              // Validate against schema
              const validationResult = validateConfiguration(
                parsed.data as Record<string, unknown>,
                schema,
                { full: true, includeWarnings: false, includeInfo: false } // Only check errors
              )
              
              // Block save if there are validation errors
              if (!validationResult.valid && validationResult.errors.length > 0) {
                const errorMessages = validationResult.errors
                  .map(err => {
                    const path = err.path ? `${err.path}: ` : ''
                    return `${path}${err.message}`
                  })
                  .join('; ')
                
                set({
                  error: `Validation failed. Please fix the following errors before saving: ${errorMessages}`,
                  isSaving: false,
                  validationErrors: validationResult.errors,
                })
                return false
              }
              
              // Update validation errors even if save is allowed (warnings/info)
              set({ validationErrors: [...validationResult.errors, ...validationResult.warnings, ...validationResult.info] })
            } else if (config) {
              // If no schema provided but we have config, try to validate with existing validation errors
              // This is a fallback - ideally schema should always be provided
              const { validationErrors } = get()
              const hasErrors = validationErrors.some(err => err.level === 'error')
              
              if (hasErrors) {
                const errorMessages = validationErrors
                  .filter(err => err.level === 'error')
                  .map(err => {
                    const path = err.path ? `${err.path}: ` : ''
                    return `${path}${err.message}`
                  })
                  .join('; ')
                
                set({
                  error: `Validation failed. Please fix the following errors before saving: ${errorMessages}`,
                  isSaving: false,
                })
                return false
              }
            }
            
            // Step 2: Save if validation passed
            const response = await saveConfigApi(configYaml)
            
            if (response.success) {
              set({
                originalYaml: configYaml,
                isDirty: false,
                isSaving: false,
                error: null,
              })
              
              // Clear draft
              clearDraft()
              
              return true
            } else {
              throw new Error(response.errors?.join(', ') || 'Save failed')
            }
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : 'Failed to save configuration',
              isSaving: false,
            })
            return false
          }
        },
        
        updateConfig: (updates) => {
          const { configYaml } = get()
          
          // Preserve comments by merging into current YAML (not originalYaml)
          const newYaml = mergeYamlUpdates(configYaml, updates)
          
          // Parse to get updated config object
          const parsed = parseYaml(newYaml)
          const newConfig = parsed.success ? (parsed.data as Record<string, unknown>) : null
          
          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })
          
          // Save draft
          saveDraft(newYaml)
        },
        
        updateField: (path, value) => {
          const { configYaml } = get()
          
          // Preserve comments by updating field in current YAML (not originalYaml)
          const newYaml = updateYamlField(configYaml, path, value)
          
          // Parse to get updated config object
          const parsed = parseYaml(newYaml)
          const newConfig = parsed.success ? (parsed.data as Record<string, unknown>) : null
          
          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })
          
          // Save draft
          saveDraft(newYaml)
        },
        
        resetConfig: () => {
          const { originalYaml } = get()
          const parsed = parseYaml(originalYaml)
          
          set({
            config: parsed.success ? (parsed.data as Record<string, unknown>) : null,
            configYaml: originalYaml,
            isDirty: false,
          })
          
          // Clear draft
          clearDraft()
        },
        
        discardChanges: () => {
          get().resetConfig()
        },
        
        // Navigation
        setActiveSection: (section) => {
          set({ activeSection: section })
        },
        
        // Validation
        setValidationErrors: (errors) => {
          set({ validationErrors: errors })
        },
        
        // UI Actions
        togglePreview: () => {
          set((state) => ({ isPreviewVisible: !state.isPreviewVisible }))
        },
        
        toggleSidebar: () => {
          set((state) => ({ isSidebarCollapsed: !state.isSidebarCollapsed }))
        },
        
        toggleHelpPanel: (section) => {
          set((state) => ({
            isHelpPanelOpen: !state.isHelpPanelOpen,
            helpSection: section ?? state.helpSection,
          }))
        },
        
        // Modal Actions
        openAddServiceModal: () => set({ isAddServiceModalOpen: true }),
        closeAddServiceModal: () => set({ isAddServiceModalOpen: false }),
        
        openResourceConfigModal: (resourceName) => 
          set({ isResourceConfigModalOpen: true, selectedResource: resourceName }),
        closeResourceConfigModal: () => 
          set({ isResourceConfigModalOpen: false, selectedResource: null }),
        
        openHealthCheckModal: (serviceName) => 
          set({ isHealthCheckModalOpen: true, selectedService: serviceName }),
        closeHealthCheckModal: () => 
          set({ isHealthCheckModalOpen: false, selectedService: null }),
        
        openHooksConfigModal: () => set({ isHooksConfigModalOpen: true }),
        closeHooksConfigModal: () => set({ isHooksConfigModalOpen: false }),
        
        openBackupManager: () => set({ isBackupManagerOpen: true }),
        closeBackupManager: () => set({ isBackupManagerOpen: false }),
        
        openCommandPalette: () => set({ isCommandPaletteOpen: true }),
        closeCommandPalette: () => set({ isCommandPaletteOpen: false }),
        
        openImportModal: () => set({ isImportModalOpen: true }),
        closeImportModal: () => set({ isImportModalOpen: false }),
        
        openExportModal: () => set({ isExportModalOpen: true }),
        closeExportModal: () => set({ isExportModalOpen: false }),
        
        openKeyboardShortcuts: () => set({ isKeyboardShortcutsOpen: true }),
        closeKeyboardShortcuts: () => set({ isKeyboardShortcutsOpen: false }),
        
        // Service Management
        addService: (name, service) => {
          const { config } = get()
          const baseConfig = config ?? {}
          const services = (baseConfig.services as Record<string, unknown>) || {}

          const serviceConfig: Record<string, unknown> = {
            host: service.host || 'containerapp',
          }

          if (service.image) {
            serviceConfig.image = service.image
          }

          if (service.ports && service.ports.length > 0) {
            serviceConfig.ports = service.ports
          }

          if (service.environment && Object.keys(service.environment).length > 0) {
            serviceConfig.environment = service.environment
          }

          if (service.healthcheck) {
            serviceConfig.healthcheck = service.healthcheck
          }

          // Preserve optional metadata if available
          const serviceAny = service as unknown as Record<string, unknown>
          if (serviceAny.language) {
            serviceConfig.language = serviceAny.language
          }

          if (serviceAny.project) {
            serviceConfig.project = serviceAny.project
          }

          const newConfig = {
            ...baseConfig,
            services: {
              ...services,
              [name]: serviceConfig,
            },
          }

          // Try to preserve comments when adding service
          const { configYaml } = get()
          const baseYaml = configYaml
          
          // Use mergeYamlUpdates to preserve comments
          const newYaml = mergeYamlUpdates(baseYaml, {
            services: {
              ...services,
              [name]: serviceConfig,
            },
          })

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        removeService: (name) => {
          const { config, configYaml } = get()
          const baseConfig = config ?? {}

          const services = { ...(baseConfig.services as Record<string, unknown> || {}) }
          delete services[name]

          const newConfig = { ...baseConfig, services }
          const newYaml = deleteYamlPath(configYaml, `services.${name}`)

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        // Resource Management
        addResource: (name, type) => {
          const { config, configYaml } = get()
          const baseConfig = config ?? {}
          const resources = (baseConfig.resources as Record<string, unknown>) || {}

          const resourceConfig: Record<string, unknown> = {
            type,
            uses: [],
          }

          const newConfig = {
            ...baseConfig,
            resources: {
              ...resources,
              [name]: resourceConfig,
            },
          }

          const newYaml = mergeYamlUpdates(configYaml, {
            resources: {
              ...resources,
              [name]: resourceConfig,
            },
          })

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        removeResource: (name) => {
          const { config, configYaml } = get()
          const baseConfig = config ?? {}

          const resources = { ...(baseConfig.resources as Record<string, unknown> || {}) }
          delete resources[name]

          const newConfig = { ...baseConfig, resources }
          const newYaml = deleteYamlPath(configYaml, `resources.${name}`)

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        // Import/Export
        importConfig: (yamlString) => {
          const parsed = parseYaml(yamlString)
          
          if (!parsed.success) {
            set({ error: `Import failed: ${parsed.error}` })
            return false
          }
          
          set({
            config: parsed.data as Record<string, unknown>,
            configYaml: yamlString, // Preserve original YAML with comments
            originalYaml: yamlString,
            originalDocument: parsed.document || null, // Store document with comments
            isDirty: true,
            error: null,
          })
          
          // Save draft
          saveDraft(yamlString)
          
          return true
        },
        
        exportConfig: () => {
          return get().configYaml
        },
      }),
      {
        name: 'azd-editor-state',
        // Only persist UI preferences, not config data
        partialize: (state) => ({
          isPreviewVisible: state.isPreviewVisible,
          isSidebarCollapsed: state.isSidebarCollapsed,
        }),
      }
    )
  )
)
