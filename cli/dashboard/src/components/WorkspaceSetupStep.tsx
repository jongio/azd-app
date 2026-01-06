/**
 * WorkspaceSetupStep - Step 1 of Azure Logs Setup Guide
 * Guides users through Log Analytics workspace configuration
 */
import * as React from 'react'
import { CheckCircle, AlertTriangle, Circle, RefreshCw, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { CodeBlock } from './shared/CodeBlock'
import { CollapsibleSection } from './shared/CollapsibleSection'

// =============================================================================
// Types
// =============================================================================

/**
 * Props for WorkspaceSetupStep component.
 * 
 * @property onValidationChange - Callback when step validation status changes (enables/disables Next button)
 */
export interface WorkspaceSetupStepProps {
  onValidationChange: (isValid: boolean) => void
}

/**
 * Workspace configuration status returned from the API.
 * Indicates whether Log Analytics workspace is properly configured.
 */
interface WorkspaceState {
  status: 'configured' | 'missing' | 'not-deployed' | 'invalid' | 'error'
  workspaceId?: string
  message: string
  source?: string
}

/**
 * Response from /api/azure/setup-state endpoint.
 */
interface SetupStateResponse {
  workspace: WorkspaceState
  timestamp: string
}

/**
 * Identifiers for collapsible help sections in the workspace setup step.
 */
type HelpSection = 'what-is-analytics' | 'create-workspace' | 'bicep-example' | 'azure-yaml'

// =============================================================================
// Constants
// =============================================================================

const POLL_INTERVAL_MS = 5000 // Poll every 5 seconds

const BICEP_EXAMPLE = `// monitoring.bicep - Create Log Analytics workspace
param location string = resourceGroup().location
param workspaceName string = 'logs-\${resourceGroup().name}'

resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: workspaceName
  location: location
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
  }
}

output logAnalyticsWorkspaceId string = logAnalyticsWorkspace.id
`

const AZURE_YAML_SNIPPET = `# azure.yaml - Configure Log Analytics for your project
logs:
  analytics:
    # Reference workspace from bicep output
    workspace: \${AZURE_LOG_ANALYTICS_WORKSPACE_ID}
`

const HELP_CONTENT = {
  'what-is-analytics': {
    title: 'What is Log Analytics?',
    content: [
      'Azure Log Analytics is a centralized logging solution that collects and analyzes telemetry from your Azure services.',
      'It provides:',
      '• Unified log storage across all your services',
      '• Powerful KQL (Kusto Query Language) for searching and analysis',
      '• Real-time monitoring and alerting',
      '• Integration with Azure Monitor and Application Insights',
      '',
      'All logs from your Container Apps, Functions, and other services flow into a single workspace for easy troubleshooting.',
    ],
  },
  'create-workspace': {
    title: 'Create Workspace',
    content: [
      'You can create a Log Analytics workspace in several ways:',
      '',
      '1. Via Azure Portal:',
      '   • Navigate to Azure Portal → Create a resource',
      '   • Search for "Log Analytics workspace"',
      '   • Follow the creation wizard',
      '',
      '2. Via Azure CLI:',
      '   az monitor log-analytics workspace create \\',
      '     --resource-group <your-rg> \\',
      '     --workspace-name <workspace-name> \\',
      '     --location <region>',
      '',
      '3. Via Bicep (recommended):',
      '   See the Bicep example below for infrastructure-as-code approach.',
    ],
  },
  'bicep-example': {
    title: 'Bicep Example',
    content: [
      'Add this to your infrastructure code to create a workspace:',
    ],
  },
  'azure-yaml': {
    title: 'azure.yaml Configuration',
    content: [
      'Optionally configure the workspace in azure.yaml. If not specified, azd app will automatically use the workspace from your environment:',
    ],
  },
}

// =============================================================================
// Helper Components
// =============================================================================

interface StatusBadgeProps {
  status: WorkspaceState['status']
}

function StatusBadge({ status }: Readonly<StatusBadgeProps>) {
  const config = {
    configured: {
      Icon: CheckCircle,
      label: 'Configured',
      className: 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800',
    },
    'not-deployed': {
      Icon: AlertTriangle,
      label: 'Not Deployed',
      className: 'bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800',
    },
    missing: {
      Icon: Circle,
      label: 'Missing',
      className: 'bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700',
    },
    invalid: {
      Icon: AlertTriangle,
      label: 'Invalid',
      className: 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800',
    },
    error: {
      Icon: AlertTriangle,
      label: 'Error',
      className: 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800',
    },
  }[status]

  const { Icon, label, className } = config

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium border',
        className,
      )}
    >
      <Icon className="w-4 h-4" />
      {label}
    </span>
  )
}

// Shared components imported above

// =============================================================================
// WorkspaceSetupStep Component
// =============================================================================

export function WorkspaceSetupStep({ onValidationChange }: Readonly<WorkspaceSetupStepProps>) {
  const [workspaceState, setWorkspaceState] = React.useState<WorkspaceState | null>(null)
  const [isLoading, setIsLoading] = React.useState(true)
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [openSection, setOpenSection] = React.useState<HelpSection | null>(null)

  const pollIntervalRef = React.useRef<number | null>(null)

  // Fetch workspace state from API
  const fetchWorkspaceState = React.useCallback(async (isManualRefresh = false) => {
    if (isManualRefresh) {
      setIsRefreshing(true)
    }

    try {
      const response = await fetch('/api/azure/logs/setup-state')
      if (!response.ok) {
        throw new Error(`Failed to fetch setup state: ${response.statusText}`)
      }

      const data = (await response.json()) as SetupStateResponse
      setWorkspaceState(data.workspace)
      setError(null)

      // Update validation state
      const isValid = data.workspace.status === 'configured'
      onValidationChange(isValid)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      setError(errorMessage)
      onValidationChange(false)
    } finally {
      setIsLoading(false)
      if (isManualRefresh) {
        setIsRefreshing(false)
      }
    }
  }, [onValidationChange])

  // Initial fetch
  React.useEffect(() => {
    void fetchWorkspaceState()
  }, [fetchWorkspaceState])

  // Set up polling (every 5 seconds when component is mounted)
  React.useEffect(() => {
    pollIntervalRef.current = window.setInterval(() => {
      void fetchWorkspaceState()
    }, POLL_INTERVAL_MS)

    return () => {
      if (pollIntervalRef.current !== null) {
        clearInterval(pollIntervalRef.current)
      }
    }
  }, [fetchWorkspaceState])

  const handleRefresh = () => {
    void fetchWorkspaceState(true)
  }

  const handleToggleSection = (section: HelpSection) => {
    setOpenSection(openSection === section ? null : section)
  }

  if (isLoading) {
    return (
      <div className="p-8 flex flex-col items-center justify-center gap-3">
        <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
        <p className="text-sm text-slate-600 dark:text-slate-400">Checking workspace configuration...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4">
          <p className="text-sm text-red-800 dark:text-red-300 font-medium">Failed to load workspace state</p>
          <p className="text-sm text-red-600 dark:text-red-400 mt-1">{error}</p>
          <button
            type="button"
            onClick={handleRefresh}
            className={cn(
              'mt-3 inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium',
              'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300',
              'hover:bg-red-200 dark:hover:bg-red-800',
              'focus:outline-none focus:ring-2 focus:ring-red-500',
            )}
          >
            <RefreshCw className="w-3.5 h-3.5" />
            Retry
          </button>
        </div>
      </div>
    )
  }

  if (!workspaceState) {
    return null
  }

  return (
    <div className="p-6 space-y-6">
      {/* Status Section */}
      <div>
        <div className="flex items-start justify-between gap-4 mb-3">
          <div>
            <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
              Log Analytics Workspace
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400">
              A centralized workspace is required to collect logs from your Azure services
            </p>
          </div>
          <StatusBadge status={workspaceState.status} />
        </div>

        {/* Status Message */}
        <div className="rounded-lg bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-4">
          <p className="text-sm text-slate-700 dark:text-slate-300">
            {workspaceState.message}
          </p>
          {workspaceState.source && (
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              Source: {workspaceState.source}
            </p>
          )}
          {workspaceState.workspaceId && (
            <p className="text-xs font-mono text-slate-600 dark:text-slate-400 mt-2 break-all">
              {workspaceState.workspaceId}
            </p>
          )}
        </div>

        {/* Recheck Button */}
        <div className="mt-3">
          <button
            type="button"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className={cn(
              'inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium',
              'text-slate-700 dark:text-slate-300',
              'border border-slate-200 dark:border-slate-700',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'transition-colors duration-150',
            )}
          >
            <RefreshCw className={cn('w-3.5 h-3.5', isRefreshing && 'animate-spin')} />
            {isRefreshing ? 'Checking...' : 'Recheck'}
          </button>
        </div>
      </div>

      {/* Help Sections */}
      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider">
          Setup Guide
        </h4>

        {/* What is Log Analytics */}
        <CollapsibleSection
          id="what-is-analytics"
          title={HELP_CONTENT['what-is-analytics'].title}
          isOpen={openSection === 'what-is-analytics'}
          onToggle={() => handleToggleSection('what-is-analytics')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300 space-y-2">
            {HELP_CONTENT['what-is-analytics'].content.map((line, idx) => (
              <p key={idx} className={line === '' ? 'h-2' : ''}>
                {line}
              </p>
            ))}
          </div>
        </CollapsibleSection>

        {/* Create Workspace */}
        <CollapsibleSection
          id="create-workspace"
          title={HELP_CONTENT['create-workspace'].title}
          isOpen={openSection === 'create-workspace'}
          onToggle={() => handleToggleSection('create-workspace')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300 space-y-2">
            {HELP_CONTENT['create-workspace'].content.map((line, idx) => {
              if (line === '') {
                return <div key={idx} className="h-2" />
              }
              if (line.startsWith('   ')) {
                return (
                  <p key={idx} className="font-mono text-xs pl-4 text-slate-600 dark:text-slate-400">
                    {line.trim()}
                  </p>
                )
              }
              return <p key={idx}>{line}</p>
            })}
          </div>
        </CollapsibleSection>

        {/* Bicep Example */}
        <CollapsibleSection
          id="bicep-example"
          title={HELP_CONTENT['bicep-example'].title}
          isOpen={openSection === 'bicep-example'}
          onToggle={() => handleToggleSection('bicep-example')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300">
            <p>{HELP_CONTENT['bicep-example'].content[0]}</p>
            <CodeBlock code={BICEP_EXAMPLE} language="bicep" />
          </div>
        </CollapsibleSection>

        {/* azure.yaml Config */}
        <CollapsibleSection
          id="azure-yaml"
          title={HELP_CONTENT['azure-yaml'].title}
          isOpen={openSection === 'azure-yaml'}
          onToggle={() => handleToggleSection('azure-yaml')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300">
            <p>{HELP_CONTENT['azure-yaml'].content[0]}</p>
            <CodeBlock code={AZURE_YAML_SNIPPET} language="yaml" />
            <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">
              After adding this to your azure.yaml, run <code className="px-1 py-0.5 rounded bg-slate-200 dark:bg-slate-800 font-mono">azd up</code> to deploy the workspace.
            </p>
          </div>
        </CollapsibleSection>
      </div>

      {/* Success Message */}
      {workspaceState.status === 'configured' && (
        <div className="rounded-lg border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-900/20 p-4">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-emerald-800 dark:text-emerald-300">
                Workspace configured successfully!
              </p>
              <p className="text-sm text-emerald-700 dark:text-emerald-400 mt-1">
                Click "Next" to continue with authentication setup.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default WorkspaceSetupStep
