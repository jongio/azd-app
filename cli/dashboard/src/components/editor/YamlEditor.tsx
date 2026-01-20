/**
 * YAML Editor - Main Integration Component
 *
 * Wires together navigation, quick actions, preview, validation, and modal workflows.
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import * as React from 'react'
import { SchemaProvider, useSchema } from '@/contexts/SchemaContext'
import { useEditorState } from './useEditorState'
import { ErrorBoundary } from './ErrorHandling/ErrorBoundary'
import { YamlEditorLayout } from './YamlEditorLayout'
import { YamlEditorHeader } from './YamlEditorHeader'
import { NavigationSidebar } from './NavigationSidebar'
import { PreviewPane, type ValidationMarker } from './PreviewPane'
import { ValidationSummaryPanel } from './ValidationSummaryPanel'
import { QuickActionsBar } from './QuickActionsBar'
import { HelpPanel } from './HelpPanel'
import { CommandPalette } from './CommandPalette'
import { ImportModal } from './ImportExport/ImportModal'
import { ExportModal } from './ImportExport/ExportModal'
import { AddServiceModal } from './modals/AddServiceModal'
import { DeleteServiceDialog } from './modals/DeleteServiceDialog'
import { BackupManager } from './BackupManager'
import { buildNavigationTree, type NavigationNode, type ValidationIssue } from '@/lib/editor/navigation-types'
import { validateConfiguration } from '@/lib/editor/validation-engine'
import { useCachedWellKnownServices } from '@/lib/performance'
import { shouldHandleShortcut } from '@/lib/shortcuts-utils'
import { stringifyYaml } from '@/lib/editor/yaml-utils'
import { cn } from '@/lib/utils'
import { SchemaDrivenForm } from './SchemaDrivenForm'
import type { Command } from '@/lib/editor/command-types'
import type { WellKnownService } from '@/lib/editor/wellknown-types'
// Note: Editor now uses dashboard's theme system via document.documentElement classes
// No separate ThemeProvider needed - theme is managed globally

export interface YamlEditorProps {
  /** Initial configuration (optional) */
  initialConfig?: Record<string, unknown>

  /** Callback when configuration changes */
  onChange?: (config: Record<string, unknown>) => void

  /** Callback when save is triggered */
  onSave?: (config: Record<string, unknown>) => Promise<void>
}

function YamlEditorInner({ initialConfig, onChange, onSave }: YamlEditorProps) {
  const {
    config,
    configYaml,
    isLoading,
    error: loadError,
    loadConfig,
    saveConfig: saveConfigInternal,
    updateField,
    setActiveSection,
    activeSection,
    isPreviewVisible,
    isSidebarCollapsed,
    togglePreview,
    toggleSidebar,
    validationErrors,
    setValidationErrors,
    openAddServiceModal,
    closeAddServiceModal,
    isAddServiceModalOpen,
    isImportModalOpen,
    openImportModal,
    closeImportModal,
    isExportModalOpen,
    openExportModal,
    closeExportModal,
    isCommandPaletteOpen,
    openCommandPalette,
    closeCommandPalette,
    isHelpPanelOpen,
    toggleHelpPanel,
    importConfig,
    addService,
    removeService,
  } = useEditorState()

  const [forceAddServiceOpen, setForceAddServiceOpen] = useState(false)
  const [serviceToDelete, setServiceToDelete] = useState<string | null>(null)
  const [isDeletingService, setIsDeletingService] = useState(false)

  const { rawSchema, schema } = useSchema()
  const { data: wellKnownServices = [] } = useCachedWellKnownServices()

  const selectedServiceName = activeSection.startsWith('services.')
    ? activeSection.split('.')[1]
    : null

  const performSave = useCallback(async () => {
    // Validate and save with schema - schema validation will block save if errors exist
    const success = await saveConfigInternal(rawSchema as Record<string, unknown> | undefined)
    
    if (success && onSave && config) {
      await onSave(config)
    }

    return success
  }, [config, onSave, saveConfigInternal, rawSchema])

  // Hydrate configuration on mount
  useEffect(() => {
    if (initialConfig) {
      const yaml = stringifyYaml(initialConfig, { indent: 2 })
      useEditorState.setState({
        config: initialConfig,
        configYaml: yaml,
        originalYaml: yaml,
        isDirty: false,
        isLoading: false,
        error: null,
      })
    } else {
      void loadConfig()
    }
  }, [initialConfig, loadConfig])

  // Notify parent of changes
  useEffect(() => {
    if (config && onChange) {
      onChange(config)
    }
  }, [config, onChange])

  // Re-run validation whenever config or schema changes
  useEffect(() => {
    if (!config || !rawSchema) return

    try {
      const result = validateConfiguration(config, rawSchema as Record<string, unknown>)
      setValidationErrors([...result.errors, ...result.warnings, ...result.info])
    } catch (error) {
      console.error('Validation failed:', error)
    }
  }, [config, rawSchema, setValidationErrors])

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (!shouldHandleShortcut(event)) return

      const key = event.key.toLowerCase()

      if (event.ctrlKey && key === 's') {
        event.preventDefault()
        void performSave()
        return
      }

      if (event.ctrlKey && key === 'p') {
        event.preventDefault()
        togglePreview()
        return
      }

      if (event.ctrlKey && key === 'b') {
        event.preventDefault()
        toggleSidebar()
        return
      }

      if (event.ctrlKey && key === 'k') {
        event.preventDefault()
        openCommandPalette()
        return
      }

      if (event.ctrlKey && key === 'n') {
        event.preventDefault()
        openAddServiceModal()
        return
      }

      if (event.key === 'F1') {
        event.preventDefault()
        toggleHelpPanel(activeSection)
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [activeSection, performSave, togglePreview, toggleSidebar, openCommandPalette, openAddServiceModal, toggleHelpPanel])

  const handleImport = useCallback((nextConfig: Record<string, unknown>) => {
    const yaml = stringifyYaml(nextConfig, { indent: 2 })
    importConfig(yaml)
    setActiveSection('overview')
  }, [importConfig, setActiveSection])

  const navigationNodes: NavigationNode[] = useMemo(() => buildNavigationTree(config), [config])

  const validationIssueMap = useMemo(() => {
    const map = new Map<string, ValidationIssue[]>()

    validationErrors.forEach((issue) => {
      const key = issue.path || 'overview'
      const list = map.get(key) ?? []
      list.push({ level: issue.level, message: issue.message, path: issue.path })
      map.set(key, list)
    })

    return map
  }, [validationErrors])

  const errors = validationErrors.filter((v) => v.level === 'error')
  const warnings = validationErrors.filter((v) => v.level === 'warning')
  const info = validationErrors.filter((v) => v.level === 'info')

  const commands = useMemo<Command[]>(() => {
    const navCommands: Command[] = navigationNodes.map((node) => ({
      id: `nav-${node.id}`,
      label: `Go to ${node.label}`,
      category: 'navigation',
      action: { type: 'navigate', path: node.id },
    }))

    return [
      ...navCommands,
      {
        id: 'toggle-preview',
        label: 'Toggle preview',
        category: 'action',
        shortcut: 'Ctrl+P',
        action: { type: 'execute', handler: togglePreview },
      },
      {
        id: 'toggle-sidebar',
        label: 'Toggle navigation',
        category: 'action',
        shortcut: 'Ctrl+B',
        action: { type: 'execute', handler: toggleSidebar },
      },
      {
        id: 'add-service',
        label: 'Add service',
        category: 'action',
        shortcut: 'Ctrl+N',
        action: { type: 'execute', handler: openAddServiceModal },
      },
      {
        id: 'open-help',
        label: 'Open help',
        category: 'help',
        action: { type: 'open-help', topic: activeSection },
      },
    ]
  }, [navigationNodes, togglePreview, toggleSidebar, openAddServiceModal, activeSection])

  // CRITICAL: ALL HOOKS MUST BE CALLED BEFORE ANY EARLY RETURNS
  // This ensures React sees the same hook order on every render
  
  // Determine which form to show based on activeSection
  // This useMemo must always be called in the same order, even during loading
  const formPath = useMemo(() => {
    if (activeSection === 'overview' || activeSection === 'name' || activeSection === 'resourceGroup' || activeSection === 'metadata') {
      // For overview fields, show the root schema
      return null
    }
    
    if (activeSection === 'test') {
      return 'test'
    }
    
    if (activeSection.startsWith('services.')) {
      // Extract service path (e.g., "services.api" or "services.api.test")
      return activeSection
    }
    
    // For other top-level sections, use the section name as path
    return activeSection
  }, [activeSection])
  
  // Determine which fields to show for overview - always return an array (never undefined)
  // This useMemo must always be called in the same order, even during loading
  const overviewFields = useMemo(() => {
    if (activeSection === 'name') return ['name']
    if (activeSection === 'resourceGroup') return ['resourceGroup']
    if (activeSection === 'metadata') return ['metadata']
    if (activeSection === 'overview') return ['name', 'resourceGroup', 'metadata']
    return [] // Return empty array instead of undefined for stability
  }, [activeSection])

  // Convert ValidationIssue[] to ValidationMarker[] with approximate line numbers
  const validationMarkers = useMemo<ValidationMarker[]>(() => {
    if (!configYaml || validationErrors.length === 0) {
      return []
    }

    const markers: ValidationMarker[] = []
    const lines = configYaml.split('\n')

    for (const issue of validationErrors) {
      // Try to find the line number by searching for the path in YAML
      let lineNumber = 1
      
      // Simple heuristic: search for the last part of the path in the YAML
      const pathParts = issue.path.split('.')
      const searchKey = pathParts[pathParts.length - 1]
      
      // Search for the key in the YAML lines
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        // Look for the key followed by colon (YAML key pattern)
        if (line.includes(`${searchKey}:`) || line.includes(`"${searchKey}":`) || line.includes(`'${searchKey}':`)) {
          lineNumber = i + 1 // 1-indexed
          break
        }
      }

      markers.push({
        line: lineNumber,
        level: issue.level,
        message: issue.message,
      })
    }

    return markers
  }, [configYaml, validationErrors])

  // NOW we can do early returns - all hooks have been called above
  // Loading state (covers initial render and refetch)
  if (isLoading || (!config && !loadError)) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          <p className="mt-4 text-muted-foreground">Loading Azure YAML Editor...</p>
        </div>
      </div>
    )
  }

  // Error state
  if (loadError) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center max-w-md">
          <p className="text-destructive mb-4">{loadError}</p>
          <button
            onClick={() => void loadConfig()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  const previewPane = (
    <PreviewPane
      data={(config as Record<string, unknown>) || {}}
      yaml={configYaml} // Use YAML string to preserve comments
      isVisible={isPreviewVisible}
      onToggle={togglePreview}
      validationMarkers={validationMarkers}
      onLineClick={() => undefined}
    />
  )

  const overviewContent = (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">Configuration</h2>
          <p className="text-sm text-muted-foreground">Edit your azure.yaml configuration</p>
        </div>
        <BackupManager onRestoreSuccess={() => void loadConfig()} />
      </div>

      {/* Schema-driven form for overview fields and test section */}
      {(activeSection === 'overview' || activeSection === 'name' || activeSection === 'resourceGroup' || activeSection === 'metadata' || activeSection === 'test') ? (
        <SchemaDrivenForm
          key={`${activeSection}-${formPath || 'root'}-${overviewFields.join(',')}`}
          schema={schema}
          path={formPath}
          config={config}
          onUpdateField={updateField}
          fields={overviewFields.length > 0 ? overviewFields : undefined}
        />
      ) : null}

      {/* Default overview when no specific section selected */}
      {activeSection === 'overview' && (
        <>
          <ValidationSummaryPanel
            errors={errors}
            warnings={warnings}
            info={info}
            onItemClick={(path) => setActiveSection(path.split('.')[0] || 'overview')}
          />
        </>
      )}
    </div>
  )

  // Check if we're viewing a service section
  const isViewingService = selectedServiceName && activeSection.startsWith(`services.${selectedServiceName}`)

  const serviceContent = isViewingService ? (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{selectedServiceName}</h2>
          <p className="text-sm text-muted-foreground">
            Configure service settings
          </p>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setServiceToDelete(selectedServiceName)}
            className={cn(
              'px-4 py-2 rounded-md text-sm font-semibold',
              'bg-rose-600 text-white hover:bg-rose-700',
              'focus:outline-none focus:ring-2 focus:ring-rose-500 focus:ring-offset-2'
            )}
          >
            Delete
          </button>
        </div>
      </div>

      {/* Schema-driven form for service configuration */}
      <SchemaDrivenForm
        key={`service-${activeSection}`}
        schema={schema}
        path={activeSection}
        config={config}
        onUpdateField={updateField}
      />
    </div>
  ) : overviewContent

  const mainContent = serviceContent

  return (
    <div className="relative">
      <YamlEditorLayout
        header={(
          <div className="border-b border-border">
            <YamlEditorHeader onOpenAddService={() => {
              openAddServiceModal()
              setForceAddServiceOpen(true)
            }} />
            <div role="tablist" aria-label="Azure YAML Editor views" className="flex gap-3 px-6 pb-2">
              {[
                { id: 'overview', label: 'Overview' },
                { id: 'services', label: 'Services' },
                { id: 'resources', label: 'Resources' },
              ].map((tab) => (
                <button
                  key={tab.id}
                  role="tab"
                  type="button"
                  aria-selected={activeSection === tab.id}
                  className={cn(
                    'text-sm font-medium px-2 py-1 rounded-md',
                    activeSection === tab.id ? 'bg-primary/10 text-primary' : 'text-muted-foreground'
                  )}
                  onClick={() => setActiveSection(tab.id)}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>
        )}
        sidebar={
          <NavigationSidebar
            nodes={navigationNodes}
            activeSection={activeSection}
            validationIssues={validationIssueMap}
            onNavigate={(path) => setActiveSection(path)}
            onAdd={(type, _parentPath) => type === 'service' ? openAddServiceModal() : undefined}
            showAddButtons={!isAddServiceModalOpen}
            isCollapsed={isSidebarCollapsed}
            onToggleCollapse={toggleSidebar}
          />
        }
        content={mainContent}
        preview={previewPane}
        isSidebarCollapsed={isSidebarCollapsed}
        isPreviewVisible={isPreviewVisible}
        footer={!isAddServiceModalOpen && (
          <QuickActionsBar
            services={wellKnownServices}
            onAddService={(service) => addService(service.name, service)}
            onImportConfig={openImportModal}
            onExportConfig={openExportModal}
          />
        )}
      />

      {/* Help Panel */}
      {isHelpPanelOpen && (
        <div className="fixed right-0 top-0 h-full w-96 z-40 shadow-xl border-l border-border bg-background">
          <HelpPanel isOpen={isHelpPanelOpen} onClose={() => toggleHelpPanel()} section={activeSection} mode="sidebar" />
        </div>
      )}

      {/* Command Palette */}
      <CommandPalette
        isOpen={isCommandPaletteOpen}
        onClose={closeCommandPalette}
        commands={commands}
        onNavigate={(path) => setActiveSection(path)}
        onOpenHelp={(topic) => toggleHelpPanel(topic)}
      />

      {/* Modals */}
      <AddServiceModal
        isOpen={isAddServiceModalOpen || forceAddServiceOpen}
        onClose={() => {
          closeAddServiceModal()
          setForceAddServiceOpen(false)
        }}
        onAddService={(service) => {
          const normalized: WellKnownService = {
            name: service.name,
            displayName: service.name,
            description: 'Custom service',
            category: 'other',
            host: service.host,
            image: service.image || '',
            ports: service.ports,
            environment: service.environment as Record<string, string>,
            healthcheck: service.healthcheck,
          }

          addService(service.name, normalized)
        }}
        existingServiceNames={Object.keys((config?.services as Record<string, unknown> | undefined) || {})}
      />

      <ImportModal
        isOpen={isImportModalOpen}
        onClose={closeImportModal}
        onImport={handleImport}
        currentConfig={(config as Record<string, unknown>) || {}}
      />

      <ExportModal
        isOpen={isExportModalOpen}
        onClose={closeExportModal}
        config={(config as Record<string, unknown>) || {}}
      />

      <DeleteServiceDialog
        isOpen={Boolean(serviceToDelete)}
        onClose={() => setServiceToDelete(null)}
        onConfirm={async () => {
          if (!serviceToDelete) return

          try {
            setIsDeletingService(true)
            removeService(serviceToDelete)
            setActiveSection('services')
          } finally {
            setIsDeletingService(false)
            setServiceToDelete(null)
          }
        }}
        serviceName={serviceToDelete ?? ''}
        isDeleting={isDeletingService}
      />

    </div>
  )
}

export function YamlEditor(props: YamlEditorProps) {
  // Sync theme with dashboard on mount - editor uses the same theme system
  React.useEffect(() => {
    // Read theme from dashboard's localStorage (same key as dashboard uses)
    const getDashboardTheme = () => {
      const stored = localStorage.getItem('dashboard-theme')
      return stored === 'dark' ? 'dark' : 'light'
    }
    
    const applyTheme = (theme: 'light' | 'dark') => {
      document.documentElement.classList.toggle('dark', theme === 'dark')
      document.documentElement.setAttribute('data-theme', theme)
    }
    
    // Apply initial theme
    const initialTheme = getDashboardTheme()
    applyTheme(initialTheme)
    
    // Listen for localStorage changes (from dashboard or editor)
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'dashboard-theme' && e.newValue) {
        const newTheme = e.newValue === 'dark' ? 'dark' : 'light'
        applyTheme(newTheme)
      }
    }

    // Observe theme class/attribute changes (covers same-tab changes without polling)
    const observer = new MutationObserver(() => {
      const isCurrentlyDark = document.documentElement.classList.contains('dark')
      const currentTheme: 'light' | 'dark' = isCurrentlyDark ? 'dark' : 'light'

      // Keep localStorage aligned so other tabs/windows can sync
      const stored = localStorage.getItem('dashboard-theme')
      if (stored !== currentTheme) {
        localStorage.setItem('dashboard-theme', currentTheme)
      }
    })

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme'],
    })
    
    window.addEventListener('storage', handleStorageChange)
    
    return () => {
      window.removeEventListener('storage', handleStorageChange)
      observer.disconnect()
    }
  }, [])

  return (
    <ErrorBoundary>
      <SchemaProvider>
        <YamlEditorInner {...props} />
      </SchemaProvider>
    </ErrorBoundary>
  )
}

export default YamlEditor


