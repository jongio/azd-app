/**
 * ConsoleView - Console/logs view with dark theme and multi-pane layout
 * Follows design spec: cli/dashboard/design/components/console-view.md
 */
import * as React from 'react'
import { cn } from '@/lib/utils'
import { LogsPane, type LogEntry } from '@/components/LogsPane'
import { LogsPaneGrid } from '@/components/LogsPaneGrid'
import { LogsView } from '@/components/LogsView'
import { SettingsDialog } from './SettingsDialog'
import { DiagnosticsModal } from './DiagnosticsModal'
import { AzureSetupGuide } from './AzureSetupGuide'
import { ConsoleToolbar, type TimeRangePreset } from './ConsoleToolbar'
import { ConsoleFilters } from './ConsoleFilters'
import { usePreferences } from '@/hooks/usePreferences'
import { useToast } from '@/components/ui/toast'
import type { SetupStep } from './AzureSetupGuide'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import { useServicesContext } from '@/contexts/ServicesContext'
import { useConsoleFilters } from '@/hooks/useConsoleFilters'
import { useConsoleSyncSettings } from '@/hooks/useConsoleSyncSettings'
import { useAzureConnectionStatus } from '@/hooks/useAzureConnectionStatus'
import type { Service, HealthReportEvent } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface ConsoleViewProps {
  /** Callback when fullscreen changes */
  onFullscreenChange?: (isFullscreen: boolean) => void
  /** Health report for status updates */
  healthReport?: HealthReportEvent | null
  /** Callback when clicking on a service (to open detail panel) */
  onServiceClick?: (service: Service) => void
}

// =============================================================================
// ConsoleView Component
// =============================================================================

export function ConsoleView({
  onFullscreenChange,
  healthReport,
  onServiceClick,
}: Readonly<ConsoleViewProps>) {
  const { services } = useServicesContext()
  const { preferences, updateUI } = usePreferences()
  const { showToast, ToastContainer } = useToast()
  const { startAll, stopAll, restartAll, isBulkOperationInProgress } = useServiceOperations()

  // Custom hooks for managing state
  const filters = useConsoleFilters(services)
  const syncSettings = useConsoleSyncSettings()
  const azureConnection = useAzureConnectionStatus({
    onAzureRealtimeConfig: syncSettings.maybeInitializeAzureRealtimeFromConfig,
  })

  // Local UI state
  const [isPaused, setIsPaused] = React.useState(false)
  const [isFullscreen, setIsFullscreen] = React.useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = React.useState(false)
  const [globalSearchTerm, setGlobalSearchTerm] = React.useState('')
  const [autoScrollEnabled, setAutoScrollEnabled] = React.useState(true)
  const [clearAllTrigger, setClearAllTrigger] = React.useState(0)
  const [collapsedPanes, setCollapsedPanes] = React.useState<Record<string, boolean>>({})
  const [timeRange, setTimeRange] = React.useState<{ preset: TimeRangePreset }>({ preset: '15m' })
  const [showDiagnostics, setShowDiagnostics] = React.useState(false)
  const [isSetupGuideOpen, setIsSetupGuideOpen] = React.useState(false)
  const [setupGuideInitialStep, setSetupGuideInitialStep] = React.useState<SetupStep | undefined>(undefined)

  const viewMode = preferences.ui.viewMode

  // Setup guide handlers
  const handleOpenSetupGuide = React.useCallback(() => {
    setIsSetupGuideOpen(true)
  }, [])

  const handleOpenSetupGuideWithStep = React.useCallback((step: SetupStep) => {
    setSetupGuideInitialStep(step)
    setIsSetupGuideOpen(true)
  }, [])

  const handleCloseSetupGuide = React.useCallback(() => {
    setIsSetupGuideOpen(false)
  }, [])

  const handleSetupComplete = React.useCallback(() => {
    setIsSetupGuideOpen(false)
    // Switch to Azure mode
    void azureConnection.handleLogModeChange('azure')
    // Refresh Azure status
    void azureConnection.fetchAzureStatus()
    showToast('Azure setup completed successfully', 'success')
  }, [azureConnection, showToast])

  // Notify parent of fullscreen changes
  React.useEffect(() => {
    onFullscreenChange?.(isFullscreen)
  }, [isFullscreen, onFullscreenChange])

  // Fetch Azure status on mount only (not on every service update)
  React.useEffect(() => {
    void azureConnection.fetchAzureStatus()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- Only fetch on mount, not on service updates
  }, [])

  // Keyboard shortcuts
  React.useEffect(() => {
    const isEditableTarget = (target: EventTarget | null): boolean => {
      const el = target as HTMLElement | null
      if (!el) return false
      if (el.isContentEditable) return true
      return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA'
    }

    const isSpaceToggle = (e: KeyboardEvent): boolean => {
      return e.code === 'Space' && !e.ctrlKey && !e.shiftKey && !e.altKey && !e.metaKey
    }

    const isToggleViewMode = (e: KeyboardEvent): boolean => {
      return e.ctrlKey && e.shiftKey && e.code === 'KeyL'
    }

    const isToggleLogModeShortcut = (e: KeyboardEvent): boolean => {
      return e.ctrlKey && e.shiftKey && e.code === 'KeyM'
    }

    const isToggleFullscreen = (e: KeyboardEvent): boolean => {
      return e.key === 'F11' || (e.ctrlKey && e.shiftKey && e.code === 'KeyF')
    }

    const isExitFullscreen = (e: KeyboardEvent): boolean => {
      return e.key === 'Escape' && isFullscreen
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (isSpaceToggle(e)) {
        if (!isEditableTarget(e.target)) {
          e.preventDefault()
          setIsPaused((prev) => !prev)
        }
        return
      }

      if (isToggleViewMode(e)) {
        e.preventDefault()
        updateUI({ viewMode: viewMode === 'grid' ? 'unified' : 'grid' })
        return
      }

      if (isToggleLogModeShortcut(e)) {
        e.preventDefault()
        if (azureConnection.azureEnabled) {
          void azureConnection.handleLogModeChange(azureConnection.logMode === 'local' ? 'azure' : 'local')
        }
        return
      }

      if (isToggleFullscreen(e)) {
        e.preventDefault()
        setIsFullscreen((prev) => !prev)
        return
      }

      if (isExitFullscreen(e)) {
        setIsFullscreen(false)
      }
    }

    globalThis.addEventListener('keydown', handleKeyDown)
    return () => globalThis.removeEventListener('keydown', handleKeyDown)
  }, [viewMode, updateUI, isFullscreen, azureConnection])

  const handleClearAll = () => {
    setClearAllTrigger((prev) => prev + 1)
    showToast('All logs cleared', 'success')
  }

  const togglePaneCollapse = (serviceName: string) => {
    setCollapsedPanes((prev) => ({
      ...prev,
      [serviceName]: !prev[serviceName],
    }))
  }

  const handleCopyPane = React.useCallback((logs: LogEntry[]) => {
    const format = preferences.copy.defaultFormat
    let content: string

    switch (format) {
      case 'json':
        content = JSON.stringify(logs, null, 2)
        break
      case 'csv':
        content = 'Service,Timestamp,Level,Message\n' +
          logs.map(log => `"${log.service}","${log.timestamp}",${log.level},"${log.message.replaceAll('"', '""')}"`).join('\n')
        break
      case 'markdown':
        content = logs.map(log => `**[${log.timestamp}]** \`${log.service}\` ${log.message}`).join('\n\n')
        break
      default: // plaintext
        content = logs.map(log => `[${log.timestamp}] [${log.service}] ${log.message}`).join('\n')
    }

    void navigator.clipboard.writeText(content)
    showToast(`Copied ${logs.length} lines to clipboard`, 'success')
  }, [showToast, preferences.copy.defaultFormat])

  // Filter and sort services
  const selectedServicesList = Array.from(filters.selectedServices).sort((a, b) =>
    a.toLowerCase().localeCompare(b.toLowerCase())
  )

  // Apply state and health filters when user explicitly selects them
  // Note: Panes are never hidden automatically when services change state/health (see docs/specs/log-pane-visibility/spec.md)
  // But when users actively click the filter buttons, we should filter accordingly
  const paneServicesList = selectedServicesList.filter((serviceName) => {
    const service = services.find((s) => s.name === serviceName)
    if (!service) return false

    // Apply state filter (if any states are deselected)
    if (filters.stateFilter.size < 3) { // Not all states selected
      const serviceState = service.status
      
      // If service has no status yet, treat as 'starting' (initializing)
      if (!serviceState) {
        return filters.stateFilter.has('starting')
      }
      
      // Map service status to filterable states
      const isRunning = serviceState === 'running' || serviceState === 'ready' || serviceState === 'watching' || serviceState === 'built'
      const isStopped = serviceState === 'stopped' || serviceState === 'not-started' || serviceState === 'not-running' || serviceState === 'completed' || serviceState === 'failed'
      const isStarting = serviceState === 'starting' || serviceState === 'restarting' || serviceState === 'building' || serviceState === 'stopping'

      const matchesState = 
        (filters.stateFilter.has('running') && isRunning) ||
        (filters.stateFilter.has('stopped') && isStopped) ||
        (filters.stateFilter.has('starting') && isStarting)

      if (!matchesState) return false
    }

    // Apply health filter (if any health statuses are deselected)
    if (filters.healthFilter.size < 4) { // Not all health statuses selected
      const serviceHealthResult = healthReport?.services.find((s) => s.serviceName === serviceName)
      const serviceHealth = serviceHealthResult?.status ?? 'unknown'
      
      if (!filters.healthFilter.has(serviceHealth)) {
        return false
      }
    }

    return true
  })

  let content: React.ReactNode

  if (viewMode === 'grid') {
    if (paneServicesList.length === 0) {
      content = (
        <div className="flex items-center justify-center h-full text-slate-500">
          <div className="text-center">
            <p className="text-lg font-medium">No services selected</p>
            <p className="text-sm mt-2">Select one or more services to view their logs</p>
          </div>
        </div>
      )
    } else {
      content = (
        <LogsPaneGrid columns={2} collapsedPanes={collapsedPanes} autoFit={true}>
          {paneServicesList.map((serviceName) => {
            const service = services.find((s) => s.name === serviceName)
            const serviceHealthResult = healthReport?.services.find(
              (s) => s.serviceName === serviceName
            )
            const serviceHealthStatus = serviceHealthResult?.status
            // Services with host: local always show local logs regardless of global mode
            const effectiveLogMode = service?.host === 'local' ? 'local' : azureConnection.logMode
            // Only show mode switching if the service's mode can actually change
            // Services with host: local never change mode (always local), so no switching indicator
            const effectiveIsModeSwitching = service?.host !== 'local' && azureConnection.isModeSwitching
            return (
              <LogsPane
                key={serviceName}
                serviceName={serviceName}
                port={service?.local?.port}
                url={service?.local?.url}
                service={service}
                onCopy={handleCopyPane}
                isPaused={isPaused}
                globalSearchTerm={globalSearchTerm}
                autoScrollEnabled={autoScrollEnabled}
                clearAllTrigger={clearAllTrigger}
                levelFilter={filters.levelFilter}
                isCollapsed={collapsedPanes[serviceName] ?? false}
                onToggleCollapse={() => togglePaneCollapse(serviceName)}
                serviceHealth={serviceHealthStatus}
                healthCheckResult={serviceHealthResult}
                onShowDetails={
                  service && onServiceClick ? () => onServiceClick(service) : undefined
                }
                logMode={effectiveLogMode}
                isModeSwitching={effectiveIsModeSwitching}
                timeRange={effectiveLogMode === 'azure' ? timeRange : undefined}
                azureRealtime={syncSettings.azureRealtime}
                onOpenDiagnostics={() => setShowDiagnostics(true)}
              />
            )
          })}
        </LogsPaneGrid>
      )
    }
  } else {
    content = (
      <LogsView
        selectedServices={filters.selectedServices}
        levelFilter={filters.levelFilter}
        isPaused={isPaused}
        autoScrollEnabled={autoScrollEnabled}
        globalSearchTerm={globalSearchTerm}
        clearAllTrigger={clearAllTrigger}
        hideControls={true}
        logMode={azureConnection.logMode}
        isModeSwitching={azureConnection.isModeSwitching}
        timeRange={azureConnection.logMode === 'azure' ? timeRange : undefined}
        azureRealtime={syncSettings.azureRealtime}
      />
    )
  }

  return (
    <div
      className={cn(
        'flex flex-col overflow-hidden',
        // Console uses theme-aware colors
        'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200',
        isFullscreen ? 'fixed inset-0 z-50' : 'h-full'
      )}
    >
      <ToastContainer />

      {/* Toolbar */}
      <ConsoleToolbar
        viewMode={viewMode}
        onViewModeChange={(mode) => updateUI({ viewMode: mode })}
        isFullscreen={isFullscreen}
        onFullscreenChange={setIsFullscreen}
        isPaused={isPaused}
        onPauseChange={setIsPaused}
        autoScrollEnabled={autoScrollEnabled}
        onAutoScrollChange={setAutoScrollEnabled}
        searchTerm={globalSearchTerm}
        onSearchChange={setGlobalSearchTerm}
        onClearAll={handleClearAll}
        onOpenSettings={() => setIsSettingsOpen(true)}
        onStartAll={() => void startAll()}
        onStopAll={() => void stopAll()}
        onRestartAll={() => void restartAll()}
        isBulkOperationInProgress={isBulkOperationInProgress()}
        logMode={azureConnection.logMode}
        onLogModeChange={(mode) => void azureConnection.handleLogModeChange(mode)}
        azureEnabled={azureConnection.azureEnabled}
        azureStatus={azureConnection.azureStatus}
        azureConnectionMessage={azureConnection.azureConnectionMessage}
        timeRange={timeRange}
        onTimeRangeChange={(preset) => setTimeRange(prev => (prev.preset === preset ? prev : { preset }))}
        azureRealtime={syncSettings.azureRealtime}
        onAzureRealtimeChange={syncSettings.setAzureRealtime}
        onRunDiagnostics={() => setShowDiagnostics(true)}
        onOpenSetupGuide={handleOpenSetupGuide}
        onOpenSetupGuideWithStep={handleOpenSetupGuideWithStep}
      />

      {/* Filters */}
      <ConsoleFilters
        services={services}
        selectedServices={filters.selectedServices}
        onToggleService={filters.toggleService}
        levelFilter={filters.levelFilter}
        onToggleLevel={filters.toggleLevel}
        stateFilter={filters.stateFilter}
        onToggleState={filters.toggleState}
        healthFilter={filters.healthFilter}
        onToggleHealth={filters.toggleHealth}
        healthReport={healthReport}
      />

      {/* Content - Constrain to remaining viewport height */}
      <div className="flex-1 overflow-hidden min-h-0">
        {content}
      </div>

      {/* Diagnostics Modal */}
      <DiagnosticsModal
        isOpen={showDiagnostics}
        onClose={() => setShowDiagnostics(false)}
        onOpenSetupGuide={(step) => {
          setShowDiagnostics(false)
          setSetupGuideInitialStep(step)
          setIsSetupGuideOpen(true)
        }}
      />

      {/* Azure Setup Guide */}
      <AzureSetupGuide
        isOpen={isSetupGuideOpen}
        onClose={() => {
          handleCloseSetupGuide()
          setSetupGuideInitialStep(undefined)
        }}
        onComplete={handleSetupComplete}
        initialStep={setupGuideInitialStep}
      />

      {/* Settings Dialog */}
      <SettingsDialog
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />
    </div>
  )
}
