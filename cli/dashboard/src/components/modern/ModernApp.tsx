/**
 * ModernApp - Main application shell for Modern mode
 * Follows design spec: cli/dashboard/design/modern/components/layout.md
 */
import * as React from 'react'
import { cn } from '@/lib/utils'
import { Wifi, WifiOff, RefreshCw } from 'lucide-react'
import { ModernHeader, type ModernView } from './ModernHeader'
import { ModernServiceCard } from './ModernServiceCard'
import { ModernServiceTable } from './ModernServiceTable'
import { ModernLogsView } from './ModernLogsView'
import { ModernServiceDetailPanel } from './ModernServiceDetailPanel'
import { ModernSettingsDialog } from './ModernSettingsDialog'
import { ModernEnvironmentPanel } from './ModernEnvironmentPanel'
import { ModernMetricsView } from './ModernMetricsView'
import { KeyboardShortcuts } from '@/components/modals/KeyboardShortcuts'
import type { Service, HealthCheckResult, HealthSummary, HealthReportEvent } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface ModernAppProps {
  /** Project name to display */
  projectName: string
  /** List of services */
  services: Service[]
  /** Whether connected to backend */
  connected: boolean
  /** Health summary for header */
  healthSummary: HealthSummary
  /** Latest health report event for logs/table */
  healthReport: HealthReportEvent | null
  /** Map of service health results */
  healthMap: Map<string, HealthCheckResult>
  /** Error message when health stream fails */
  healthError?: string | null
  /** Function to manually reconnect */
  healthReconnect?: () => void
  /** Additional class names */
  className?: string
}

type ViewMode = 'grid' | 'table'

// =============================================================================
// Empty State Component
// =============================================================================

interface EmptyStateProps {
  connected: boolean
}

function EmptyState({ connected }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-8 text-center">
      <div className="w-16 h-16 mb-6 rounded-full bg-gradient-to-br from-cyan-100 to-teal-100 dark:from-cyan-900/30 dark:to-teal-900/30 flex items-center justify-center">
        <svg
          className="w-8 h-8 text-cyan-600 dark:text-cyan-400"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M21 7.5l-9-5.25L3 7.5m18 0l-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9" />
        </svg>
      </div>
      <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
        {connected ? 'No Services Found' : 'Connecting...'}
      </h3>
      <p className="text-sm text-slate-500 dark:text-slate-400 max-w-md">
        {connected 
          ? 'No services are currently configured in your project. Add services to azure.yaml to get started.'
          : 'Attempting to connect to the dashboard server. Make sure azd is running.'
        }
      </p>
    </div>
  )
}

// =============================================================================
// Services Grid Component
// =============================================================================

interface ServicesGridProps {
  services: Service[]
  healthMap: Map<string, HealthCheckResult>
  onSelectService: (service: Service) => void
}

function ServicesGrid({ services, healthMap, onSelectService }: ServicesGridProps) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 3xl:grid-cols-6 gap-4">
      {services.map((service) => (
        <ModernServiceCard
          key={service.name}
          service={service}
          healthStatus={healthMap.get(service.name)}
          onClick={() => onSelectService(service)}
        />
      ))}
    </div>
  )
}

// =============================================================================
// ModernApp Component
// =============================================================================

// URL path to view mapping
const PATH_TO_VIEW: Record<string, ModernView> = {
  '/console': 'console',
  '/services': 'resources',
  '/environment': 'environment',
  '/metrics': 'metrics',
}

const VIEW_TO_PATH: Record<ModernView, string> = {
  'console': '/console',
  'resources': '/services',
  'environment': '/environment',
  'metrics': '/metrics',
}

function getInitialView(): ModernView {
  const path = window.location.pathname
  return PATH_TO_VIEW[path] || 'console' // Default to console
}

export function ModernApp({
  projectName,
  services,
  connected,
  healthSummary,
  healthReport,
  healthMap,
  healthError,
  healthReconnect,
  className,
}: ModernAppProps) {
  const [activeView, setActiveView] = React.useState<ModernView>(getInitialView)
  const [viewMode, setViewMode] = React.useState<ViewMode>('grid')
  const [selectedService, setSelectedService] = React.useState<Service | null>(null)
  const [isPanelOpen, setIsPanelOpen] = React.useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = React.useState(false)
  const [isShortcutsModalOpen, setIsShortcutsModalOpen] = React.useState(false)

  // Sync URL with view changes
  React.useEffect(() => {
    const newPath = VIEW_TO_PATH[activeView]
    if (window.location.pathname !== newPath) {
      window.history.pushState({}, '', newPath)
    }
  }, [activeView])

  // Handle browser back/forward navigation
  React.useEffect(() => {
    const handlePopState = () => {
      const path = window.location.pathname
      const view = PATH_TO_VIEW[path] || 'console'
      setActiveView(view)
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  // Handle service selection
  const handleSelectService = React.useCallback((service: Service) => {
    setSelectedService(service)
    setIsPanelOpen(true)
  }, [])

  // Handle panel close
  const handleClosePanel = React.useCallback(() => {
    setIsPanelOpen(false)
    // Keep selected service for a moment for smooth animation
    setTimeout(() => {
      setSelectedService(null)
    }, 300)
  }, [])

  // Handle view change
  const handleViewChange = React.useCallback((view: ModernView) => {
    setActiveView(view)
  }, [])

  // Handle settings dialog
  const handleShowSettings = React.useCallback(() => {
    setIsSettingsOpen(true)
  }, [])

  const handleCloseSettings = React.useCallback(() => {
    setIsSettingsOpen(false)
  }, [])

  // Handle keyboard shortcuts modal
  const handleShowShortcuts = React.useCallback(() => {
    setIsShortcutsModalOpen(true)
  }, [])

  const handleCloseShortcuts = React.useCallback(() => {
    setIsShortcutsModalOpen(false)
  }, [])

  // Scroll to top on view change (except console)
  React.useEffect(() => {
    if (activeView !== 'console') {
      const mainElement = document.querySelector('main')
      if (mainElement) {
        mainElement.scrollTo({ top: 0, behavior: 'smooth' })
      }
    }
  }, [activeView])

  // Scroll to top when view mode changes in services view
  React.useEffect(() => {
    if (activeView === 'resources') {
      const mainElement = document.querySelector('main')
      if (mainElement) {
        mainElement.scrollTo({ top: 0, behavior: 'smooth' })
      }
    }
  }, [viewMode, activeView])

  // Global keyboard shortcuts
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't handle shortcuts when in input/textarea
      const target = e.target as HTMLElement
      const tagName = target.tagName.toLowerCase()
      if (tagName === 'input' || tagName === 'textarea' || tagName === 'select' || target.isContentEditable) {
        return
      }

      // Don't handle shortcuts when modal is open (except Escape)
      if ((isShortcutsModalOpen || isSettingsOpen) && e.key !== 'Escape') return

      const key = e.key

      // ? - Show keyboard shortcuts
      if (key === '?') {
        e.preventDefault()
        setIsShortcutsModalOpen(true)
        return
      }

      // Navigation shortcuts (1-4)
      const viewMap: Record<string, ModernView> = {
        '1': 'console',
        '2': 'resources',
        '3': 'environment',
        '4': 'metrics',
      }
      if (viewMap[key]) {
        e.preventDefault()
        setActiveView(viewMap[key])
        return
      }

      // T - Toggle table/grid view (only in resources view)
      if (key.toLowerCase() === 't' && activeView === 'resources') {
        e.preventDefault()
        setViewMode(prev => prev === 'table' ? 'grid' : 'table')
        return
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isShortcutsModalOpen, isSettingsOpen, activeView])

  // Sync selected service with services list (in case it updates)
  React.useEffect(() => {
    if (selectedService && isPanelOpen) {
      const updated = services.find((s) => s.name === selectedService.name)
      if (updated) {
        setSelectedService(updated)
      }
    }
  }, [services, selectedService, isPanelOpen])

  // Get latest health report event (if array)
  // This is no longer needed as we receive a single HealthReportEvent now

  return (
    <div
      className={cn(
        'min-h-screen flex flex-col relative',
        'bg-gradient-to-br from-slate-50 to-slate-100',
        'dark:from-slate-900 dark:to-slate-950',
        'transition-colors duration-300',
        className
      )}
    >
      {/* Connection Lost / Reconnecting Overlay */}
      {healthError && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-white/80 dark:bg-slate-900/90 backdrop-blur-sm transition-opacity duration-300">
          <div className="flex flex-col items-center gap-6 p-8 rounded-2xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-2xl max-w-md mx-4">
            {/* Animated Icon */}
            <div className="relative w-24 h-24 flex items-center justify-center">
              {healthError.includes('Failed to connect') ? (
                <>
                  {/* Disconnected state */}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-24 h-24 rounded-full border-2 border-rose-500/20 animate-pulse" />
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-16 h-16 rounded-full bg-rose-500/10 animate-pulse" />
                  </div>
                  <div className="relative z-10 w-14 h-14 rounded-full bg-rose-500/15 flex items-center justify-center">
                    <WifiOff className="w-7 h-7 text-rose-500" />
                  </div>
                </>
              ) : (
                <>
                  {/* Reconnecting state - animated waves */}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-24 h-24 rounded-full bg-cyan-500/10 animate-ping" style={{ animationDuration: '2s' }} />
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-20 h-20 rounded-full bg-cyan-500/15 animate-ping" style={{ animationDuration: '2s', animationDelay: '0.3s' }} />
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-16 h-16 rounded-full bg-cyan-500/20 animate-ping" style={{ animationDuration: '2s', animationDelay: '0.6s' }} />
                  </div>
                  <div className="relative z-10 w-14 h-14 rounded-full bg-cyan-500/20 flex items-center justify-center">
                    <Wifi className="w-7 h-7 text-cyan-500 animate-pulse" />
                  </div>
                </>
              )}
            </div>

            {/* Message */}
            <div className="text-center">
              <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100 mb-2">
                {healthError.includes('Failed to connect') ? 'Connection Lost' : 'Reconnecting...'}
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400">{healthError}</p>
            </div>

            {/* Retry Button */}
            {healthError.includes('Failed to connect') && healthReconnect && (
              <button
                type="button"
                onClick={healthReconnect}
                className="flex items-center gap-2 px-6 py-3 bg-cyan-500 hover:bg-cyan-600 text-white rounded-lg font-medium transition-all shadow-lg hover:shadow-xl hover:scale-105 active:scale-95"
              >
                <RefreshCw className="w-4 h-4" />
                Retry Connection
              </button>
            )}
          </div>
        </div>
      )}

      {/* Header */}
      <ModernHeader
        projectName={projectName}
        activeView={activeView}
        onViewChange={handleViewChange}
        healthSummary={healthSummary}
        connected={connected}
        onShowSettings={handleShowSettings}
        onShowShortcuts={handleShowShortcuts}
        services={services}
        hasActiveErrors={false}
        loading={!connected && services.length === 0}
      />

      {/* Settings Dialog */}
      <ModernSettingsDialog
        isOpen={isSettingsOpen}
        onClose={handleCloseSettings}
      />

      {/* Main Content */}
      <main className={cn(
        "flex-1 overflow-hidden",
        activeView !== 'console' && "overflow-auto"
      )}>
        {/* Services View */}
        {activeView === 'resources' && (
          <div className="p-6">
            {/* View Toggle and Stats */}
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-4">
                <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                  Services
                </h2>
                <span className="px-2.5 py-1 text-xs font-medium bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-full">
                  {services.length} {services.length === 1 ? 'service' : 'services'}
                </span>
              </div>

              {/* View Mode Toggle */}
              {services.length > 0 && (
                <div className="flex items-center gap-1 p-1 bg-slate-200/80 dark:bg-slate-700/80 rounded-lg">
                  <button
                    type="button"
                    onClick={() => setViewMode('grid')}
                    className={cn(
                      'px-3 py-1.5 text-sm font-medium rounded-md transition-all',
                      viewMode === 'grid'
                        ? 'bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 shadow-sm'
                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
                    )}
                    aria-pressed={viewMode === 'grid'}
                  >
                    Grid
                  </button>
                  <button
                    type="button"
                    onClick={() => setViewMode('table')}
                    className={cn(
                      'px-3 py-1.5 text-sm font-medium rounded-md transition-all',
                      viewMode === 'table'
                        ? 'bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 shadow-sm'
                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
                    )}
                    aria-pressed={viewMode === 'table'}
                  >
                    Table
                  </button>
                </div>
              )}
            </div>

            {/* Services Content */}
            {services.length === 0 ? (
              <EmptyState connected={connected} />
            ) : viewMode === 'grid' ? (
              <ServicesGrid
                services={services}
                healthMap={healthMap}
                onSelectService={handleSelectService}
              />
            ) : (
              <ModernServiceTable
                services={services}
                healthReport={healthReport}
                onServiceClick={handleSelectService}
              />
            )}
          </div>
        )}

        {/* Console View */}
        {activeView === 'console' && (
          <ModernLogsView
            healthReport={healthReport}
            onServiceClick={handleSelectService}
          />
        )}

        {/* Environment View */}
        {activeView === 'environment' && (
          <div className="p-6">
            <div className="mb-6">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                Environment Variables
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                View and manage environment variables across all services
              </p>
            </div>
            <ModernEnvironmentPanel services={services} />
          </div>
        )}

        {/* Metrics View */}
        {activeView === 'metrics' && (
          <div className="p-6">
            <ModernMetricsView 
              services={services} 
              healthReport={healthReport}
            />
          </div>
        )}
      </main>

      {/* Service Detail Panel */}
      <ModernServiceDetailPanel
        service={selectedService}
        isOpen={isPanelOpen}
        onClose={handleClosePanel}
        healthStatus={selectedService ? healthMap.get(selectedService.name) : undefined}
      />

      {/* Keyboard Shortcuts Modal */}
      <KeyboardShortcuts
        isOpen={isShortcutsModalOpen}
        onClose={handleCloseShortcuts}
      />
    </div>
  )
}
