import * as React from 'react'
import {
  Activity,
  CheckCircle,
  AlertCircle,
  RefreshCw,
  Trash2,
  Download,
  Terminal
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  countRunningServices,
  countHealthyServices,
  countErrorServices,
  pluralize
} from '@/lib/service-stats'
import type { Service } from '@/types'

// ============================================================================
// Types
// ============================================================================

export interface QuickActionsProps {
  /** Services data for computing stats */
  services: Service[]
  /** Callback to refresh all services */
  onRefresh?: () => void
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

export interface StatCardProps {
  /** Card title */
  title: string
  /** Statistic value */
  value: number
  /** Card icon */
  icon: React.ComponentType<{ className?: string; 'aria-hidden'?: boolean | 'true' | 'false' }>
  /** Card color variant */
  variant: 'primary' | 'success' | 'error'
  /** Subtitle text (e.g., "services") */
  subtitle?: string
}

// ============================================================================
// StatCard Component
// ============================================================================

function StatCard({ title, value, icon: Icon, variant, subtitle }: StatCardProps) {
  const variantStyles = {
    primary: {
      card: 'border-primary/20 bg-primary/5',
      icon: 'text-primary bg-primary/10',
    },
    success: {
      card: 'border-green-500/20 bg-green-500/5',
      icon: 'text-green-500 bg-green-500/10',
    },
    error: {
      card: value > 0
        ? 'border-destructive/30 bg-destructive/10'
        : 'border-destructive/20 bg-destructive/5',
      icon: value > 0
        ? 'text-destructive bg-destructive/10 animate-pulse'
        : 'text-destructive bg-destructive/10',
    },
  }

  const styles = variantStyles[variant]

  return (
    <div
      className={`rounded-lg border p-6 ${styles.card}`}
      role="status"
      aria-label={`${value} ${title.toLowerCase()} ${pluralize(value, 'service', 'services')}`}
      data-testid={`stat-card-${variant}`}
    >
      <div className="flex items-center gap-3 mb-4">
        <div className={`p-2 rounded-lg ${styles.icon}`}>
          <Icon className="h-6 w-6" aria-hidden="true" />
        </div>
        <span className="text-sm font-medium text-muted-foreground">{title}</span>
      </div>
      <div className="text-3xl font-bold text-foreground">{value}</div>
      {subtitle && (
        <div className="text-sm text-muted-foreground mt-1">{subtitle}</div>
      )}
    </div>
  )
}

// ============================================================================
// QuickActions Component
// ============================================================================

export function QuickActions({
  services,
  onRefresh,
  className = '',
  'data-testid': testId = 'quick-actions',
}: QuickActionsProps) {
  const [isRefreshing, setIsRefreshing] = React.useState(false)

  // Compute statistics
  const runningCount = countRunningServices(services)
  const healthyCount = countHealthyServices(services)
  const errorCount = countErrorServices(services)

  const handleRefresh = React.useCallback(() => {
    if (isRefreshing) return
    setIsRefreshing(true)
    try {
      onRefresh?.()
    } finally {
      // Show loading state briefly for visual feedback
      setTimeout(() => setIsRefreshing(false), 500)
    }
  }, [isRefreshing, onRefresh])

  const handleClearLogs = React.useCallback(() => {
    // Dispatch event for LogsMultiPaneView to handle
    window.dispatchEvent(new CustomEvent('clear-all-logs'))
  }, [])

  const handleExportLogs = React.useCallback(() => {
    // Dispatch event for LogsMultiPaneView to handle
    window.dispatchEvent(new CustomEvent('export-all-logs'))
  }, [])

  const handleOpenTerminal = React.useCallback(() => {
    // Dispatch event for backend/parent to handle
    window.dispatchEvent(new CustomEvent('open-terminal'))
  }, [])

  return (
    <section
      aria-labelledby="quick-actions-title"
      className={`space-y-6 ${className}`}
      data-testid={testId}
    >
      <h2 id="quick-actions-title" className="sr-only">
        Quick Actions Dashboard
      </h2>

      {/* Stats Section */}
      <section aria-labelledby="stats-title">
        <h3 id="stats-title" className="text-lg font-semibold text-foreground mb-4">
          Service Statistics
        </h3>
        <div
          className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"
          data-testid="stats-grid"
        >
          <StatCard
            title="Running"
            value={runningCount}
            icon={Activity}
            variant="primary"
            subtitle={pluralize(runningCount, 'service', 'services')}
          />
          <StatCard
            title="Healthy"
            value={healthyCount}
            icon={CheckCircle}
            variant="success"
            subtitle={pluralize(healthyCount, 'service', 'services')}
          />
          <StatCard
            title="Errors"
            value={errorCount}
            icon={AlertCircle}
            variant="error"
            subtitle={pluralize(errorCount, 'service', 'services')}
          />
        </div>
      </section>

      {/* Actions Section */}
      <section aria-labelledby="actions-title">
        <h3 id="actions-title" className="text-lg font-semibold text-foreground mb-4">
          Global Actions
        </h3>
        <div
          role="group"
          aria-label="Action buttons"
          className="flex flex-wrap gap-3"
          data-testid="actions-group"
        >
          <Button
            variant="secondary"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="gap-2"
            data-testid="refresh-all-btn"
            aria-label={isRefreshing ? 'Refreshing all services' : 'Refresh all services'}
          >
            <RefreshCw
              className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`}
              aria-hidden="true"
            />
            {isRefreshing ? 'Refreshing...' : 'Refresh All'}
          </Button>

          <Button
            variant="secondary"
            onClick={handleClearLogs}
            className="gap-2"
            data-testid="clear-logs-btn"
            aria-label="Clear all logs"
          >
            <Trash2 className="h-4 w-4" aria-hidden="true" />
            Clear Logs
          </Button>

          <Button
            variant="secondary"
            onClick={handleExportLogs}
            className="gap-2"
            data-testid="export-logs-btn"
            aria-label="Export all logs"
          >
            <Download className="h-4 w-4" aria-hidden="true" />
            Export Logs
          </Button>

          <Button
            variant="secondary"
            onClick={handleOpenTerminal}
            className="gap-2"
            data-testid="open-terminal-btn"
            aria-label="Open terminal"
          >
            <Terminal className="h-4 w-4" aria-hidden="true" />
            Open Terminal
          </Button>
        </div>
      </section>
    </section>
  )
}
