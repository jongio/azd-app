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
import { parseYaml, stringifyYaml } from '@/lib/editor/yaml-utils'
import { loadConfig, saveConfig } from '@/lib/editor/config-api'

// =============================================================================
// Types
// =============================================================================

export interface EditorState {
  // Configuration
  config: Record<string, unknown> | null
  configYaml: string
  originalYaml: string
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
  saveConfig: () => Promise<boolean>
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
 * Set nested object property by path (e.g., 'services.api.port')
 */
function setNestedProperty(obj: Record<string, unknown>, path: string, value: unknown): Record<string, unknown> {
  const parts = path.split('.')
  const result = { ...obj }
  let current: any = result
  
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i]
    if (!(part in current) || typeof current[part] !== 'object') {
      current[part] = {}
    } else {
      current[part] = { ...current[part] }
    }
    current = current[part]
  }
  
  current[parts[parts.length - 1]] = value
  return result
}

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
            
            // Parse YAML
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
        
        saveConfig: async () => {
          const { configYaml, isSaving } = get()
          
          if (isSaving) return false
          
          set({ isSaving: true, error: null })
          
          try {
            const response = await saveConfig(configYaml)
            
            if (response.success) {
              set({
                originalYaml: configYaml,
                isDirty: false,
                isSaving: false,
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
          const { config } = get()
          const baseConfig = config ?? {}
          
          const newConfig = { ...baseConfig, ...updates }
          const newYaml = stringifyYaml(newConfig, { indent: 2 })
          
          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })
          
          // Save draft
          saveDraft(newYaml)
        },
        
        updateField: (path, value) => {
          const { config } = get()
          const baseConfig = config ?? {}
          
          const newConfig = setNestedProperty(baseConfig, path, value)
          const newYaml = stringifyYaml(newConfig, { indent: 2 })
          
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
          if ((service as Record<string, unknown>).language) {
            serviceConfig.language = (service as Record<string, unknown>).language
          }

          if ((service as Record<string, unknown>).project) {
            serviceConfig.project = (service as Record<string, unknown>).project
          }

          const newConfig = {
            ...baseConfig,
            services: {
              ...services,
              [name]: serviceConfig,
            },
          }

          const newYaml = stringifyYaml(newConfig, { indent: 2 })

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        removeService: (name) => {
          const { config } = get()
          const baseConfig = config ?? {}

          const services = { ...(baseConfig.services as Record<string, unknown> || {}) }
          delete services[name]

          const newConfig = { ...baseConfig, services }
          const newYaml = stringifyYaml(newConfig, { indent: 2 })

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        // Resource Management
        addResource: (name, type) => {
          const { config } = get()
          const baseConfig = config ?? {}
          const resources = (baseConfig.resources as Record<string, unknown>) || {}

          const newConfig = {
            ...baseConfig,
            resources: {
              ...resources,
              [name]: {
                type,
                uses: [],
              },
            },
          }

          const newYaml = stringifyYaml(newConfig, { indent: 2 })

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        removeResource: (name) => {
          const { config } = get()
          const baseConfig = config ?? {}

          const resources = { ...(baseConfig.resources as Record<string, unknown> || {}) }
          delete resources[name]

          const newConfig = { ...baseConfig, resources }
          const newYaml = stringifyYaml(newConfig, { indent: 2 })

          set({
            config: newConfig,
            configYaml: newYaml,
            isDirty: true,
          })

          saveDraft(newYaml)
        },
        
        // Import/Export
        importConfig: (yaml) => {
          const parsed = parseYaml(yaml)
          
          if (!parsed.success) {
            set({ error: `Import failed: ${parsed.error}` })
            return false
          }
          
          set({
            config: parsed.data as Record<string, unknown>,
            configYaml: yaml,
            isDirty: true,
            error: null,
          })
          
          // Save draft
          saveDraft(yaml)
          
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
