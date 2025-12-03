/**
 * ModernHeader - Header component with navigation pills and status summary
 * Follows design spec: cli/dashboard/design/modern/components/header.md
 */
import * as React from 'react'
import { 
  Zap, 
  LayoutGrid, 
  Terminal, 
  Settings2,
  BarChart3,
  HelpCircle,
  Settings,
  Github,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ModernThemeToggle } from './ModernThemeToggle'
import { ModernHealthPill, ModernConnectionStatus } from './ModernStatusIndicator'
import { ModernServiceStatusCard } from './ModernServiceStatusCard'
import type { HealthSummary, Service } from '@/types'

// =============================================================================
// Types
// =============================================================================

export type ModernView = 'resources' | 'console' | 'environment' | 'metrics'

interface NavItem {
  id: ModernView
  label: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number | string
}

export interface ModernHeaderProps {
  /** Project name to display */
  projectName: string
  /** Currently active view */
  activeView: ModernView
  /** Callback when view changes */
  onViewChange: (view: ModernView) => void
  /** Health summary for status pill */
  healthSummary: HealthSummary | null
  /** Whether connected to health stream */
  connected: boolean
  /** Callback for keyboard shortcuts help */
  onShowShortcuts?: () => void
  /** Callback when settings button is clicked */
  onShowSettings?: () => void
  /** Services for detailed status counts (optional - falls back to health pill) */
  services?: Service[]
  /** Whether there are active log errors */
  hasActiveErrors?: boolean
  /** Whether dashboard is loading */
  loading?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// Navigation Items
// =============================================================================

const NAV_ITEMS: NavItem[] = [
  { id: 'console', label: 'Console', icon: Terminal },
  { id: 'resources', label: 'Services', icon: LayoutGrid },
  { id: 'environment', label: 'Environment', icon: Settings2 },
  { id: 'metrics', label: 'Metrics', icon: BarChart3 },
]

// =============================================================================
// ModernNavItem Component
// =============================================================================

interface ModernNavItemProps {
  item: NavItem
  isActive: boolean
  onClick: () => void
}

function ModernNavItem({ item, isActive, onClick }: ModernNavItemProps) {
  const Icon = item.icon

  return (
    <button
      type="button"
      role="tab"
      aria-selected={isActive}
      tabIndex={isActive ? 0 : -1}
      onClick={onClick}
      className={cn(
        'relative flex items-center gap-2 px-4 py-2 rounded-lg',
        'text-sm font-medium whitespace-nowrap',
        'transition-all duration-150 ease-out',
        isActive
          ? 'bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 shadow-sm'
          : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-white/50 dark:hover:bg-slate-800/50'
      )}
    >
      <Icon className={cn(
        'w-4 h-4 shrink-0',
        isActive && 'text-cyan-600 dark:text-cyan-400'
      )} />
      <span className="hidden sm:inline">{item.label}</span>
      {item.badge && (
        <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1.5 flex items-center justify-center text-[10px] font-semibold text-white bg-rose-500 rounded-full animate-scale-in">
          {item.badge}
        </span>
      )}
    </button>
  )
}

// =============================================================================
// ModernHeader Component
// =============================================================================

export function ModernHeader({
  projectName,
  activeView,
  onViewChange,
  healthSummary,
  connected,
  onShowShortcuts,
  onShowSettings,
  services,
  hasActiveErrors = false,
  loading = false,
  className,
}: ModernHeaderProps) {
  const navRef = React.useRef<HTMLDivElement>(null)
  const [isScrolled, setIsScrolled] = React.useState(false)

  // Detect scroll for header elevation
  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 10)
    }
    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  // Keyboard navigation within tabs
  const handleKeyNavigation = (e: React.KeyboardEvent) => {
    const tabs = navRef.current?.querySelectorAll('[role="tab"]')
    if (!tabs) return

    const tabsArray = Array.from(tabs) as HTMLElement[]
    const currentIndex = tabsArray.findIndex(tab => tab === document.activeElement)
    
    let nextIndex = currentIndex

    switch (e.key) {
      case 'ArrowLeft':
        nextIndex = currentIndex > 0 ? currentIndex - 1 : tabsArray.length - 1
        break
      case 'ArrowRight':
        nextIndex = currentIndex < tabsArray.length - 1 ? currentIndex + 1 : 0
        break
      case 'Home':
        nextIndex = 0
        break
      case 'End':
        nextIndex = tabsArray.length - 1
        break
      default:
        return
    }

    e.preventDefault()
    tabsArray[nextIndex]?.focus()
    // Also activate the tab on arrow key navigation
    const viewId = tabsArray[nextIndex]?.getAttribute('data-view') as ModernView
    if (viewId) {
      onViewChange(viewId)
    }
  }

  return (
    <header
      role="banner"
      className={cn(
        'sticky top-0 z-40 h-16 px-4 md:px-6',
        'flex items-center justify-between gap-4',
        'bg-white/85 dark:bg-slate-900/90',
        'backdrop-blur-xl',
        'border-b border-slate-200/60 dark:border-slate-700/60',
        'transition-shadow duration-200',
        isScrolled && 'shadow-md shadow-slate-200/20 dark:shadow-black/20',
        className
      )}
    >
      {/* Brand Zone */}
      <button
        type="button"
        onClick={() => onViewChange('console')}
        className="flex items-center gap-3 min-w-0 hover:opacity-80 transition-opacity cursor-pointer"
        aria-label="Go to Console"
      >
        <div className="w-7 h-7 rounded-lg bg-linear-to-br from-cyan-500 to-cyan-600 flex items-center justify-center shrink-0">
          <Zap className="w-4 h-4 text-white" />
        </div>
        <h1 className="text-lg font-semibold text-slate-900 dark:text-slate-100 truncate tracking-tight">
          {projectName || 'Dashboard'}
        </h1>
        {connected && (
          <span className="relative flex h-2 w-2" aria-hidden="true">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
        )}
      </button>

      {/* Navigation Zone */}
      <nav aria-label="Main navigation" className="hidden md:block">
        <div
          ref={navRef}
          role="tablist"
          aria-orientation="horizontal"
          onKeyDown={handleKeyNavigation}
          className="flex items-center gap-1 p-1 bg-slate-100 dark:bg-slate-800/50 rounded-xl"
        >
          {NAV_ITEMS.map(item => (
            <ModernNavItem
              key={item.id}
              item={item}
              isActive={activeView === item.id}
              onClick={() => onViewChange(item.id)}
            />
          ))}
        </div>
      </nav>

      {/* Mobile Navigation - shown on smaller screens */}
      <nav aria-label="Main navigation" className="md:hidden">
        <div
          ref={navRef}
          role="tablist"
          aria-orientation="horizontal"
          onKeyDown={handleKeyNavigation}
          className="flex items-center gap-0.5 p-1 bg-slate-100 dark:bg-slate-800/50 rounded-xl"
        >
          {NAV_ITEMS.map(item => {
            const Icon = item.icon
            return (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={activeView === item.id}
                data-view={item.id}
                onClick={() => onViewChange(item.id)}
                className={cn(
                  'p-2 rounded-lg transition-all duration-150',
                  activeView === item.id
                    ? 'bg-white dark:bg-slate-800 text-cyan-600 dark:text-cyan-400 shadow-sm'
                    : 'text-slate-500 dark:text-slate-400'
                )}
              >
                <Icon className="w-4 h-4" />
                <span className="sr-only">{item.label}</span>
              </button>
            )
          })}
        </div>
      </nav>

      {/* Utility Zone */}
      <div className="flex items-center gap-2">
        {/* Detailed Service Status Card (when services provided) */}
        {services && services.length > 0 ? (
          <ModernServiceStatusCard
            services={services}
            hasActiveErrors={hasActiveErrors}
            loading={loading}
            onClick={() => onViewChange('console')}
            healthSummary={healthSummary}
            healthConnected={connected}
            className="hidden sm:flex"
          />
        ) : (
          /* Fallback to Health Status Pill - clickable to go to Console */
          healthSummary && (
            <button
              type="button"
              onClick={() => onViewChange('console')}
              className="focus:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2 rounded-full hidden sm:block"
              aria-label="View Console"
            >
              <ModernHealthPill
                total={healthSummary.total}
                healthy={healthSummary.healthy}
                degraded={healthSummary.degraded}
                unhealthy={healthSummary.unhealthy}
                starting={healthSummary.starting}
              />
            </button>
          )
        )}

        {/* Connection Status (shown when disconnected) */}
        {!connected && (
          <ModernConnectionStatus connected={connected} reconnecting={true} />
        )}

        {/* Divider */}
        <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1 hidden sm:block" />

        {/* Theme Toggle - Only shown when ?theme=1 is in URL */}
        {new URLSearchParams(window.location.search).get('theme') === '1' && (
          <ModernThemeToggle />
        )}

        {/* GitHub Link */}
        <a
          href="https://github.com/jongio/azd-app"
          target="_blank"
          rel="noopener noreferrer"
          className="p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors hidden sm:flex"
          aria-label="View on GitHub"
        >
          <Github className="w-[18px] h-[18px]" />
        </a>

        {/* Help */}
        <button
          type="button"
          onClick={onShowShortcuts}
          className="p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          aria-label="Keyboard shortcuts"
          title="Keyboard shortcuts (?)"
        >
          <HelpCircle className="w-[18px] h-[18px]" />
        </button>

        {/* Settings */}
        <button
          type="button"
          onClick={onShowSettings}
          className="p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors hidden sm:flex"
          aria-label="Settings"
          title="Settings"
        >
          <Settings className="w-[18px] h-[18px]" />
        </button>
      </div>
    </header>
  )
}
