/**
 * AuthSetupStep - Step 2 of Azure Logs Setup Guide
 * Guides users through authentication and permission verification
 */
import * as React from 'react'
import { CheckCircle, AlertTriangle, Circle, RefreshCw, Loader2, UserCheck, ShieldAlert, ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import { CodeBlock } from './shared/CodeBlock'
import { CollapsibleSection } from './shared/CollapsibleSection'

// =============================================================================
// Types
// =============================================================================

/**
 * Props for AuthSetupStep component.
 * 
 * @property onValidationChange - Callback when step validation status changes (enables/disables Next button)
 */
export interface AuthSetupStepProps {
  onValidationChange: (isValid: boolean) => void
}

/**
 * Authentication status returned from the API.
 * Indicates whether user is authenticated and has required permissions.
 */
interface AuthState {
  status: 'authenticated' | 'not-authenticated' | 'permission-denied' | 'error'
  principal?: string
  hasLogAnalyticsReader: boolean
  message: string
}

/**
 * Response from /api/azure/setup-state endpoint.
 */
interface SetupStateResponse {
  authentication: AuthState
  timestamp: string
}

/**
 * Identifiers for collapsible help sections in the authentication setup step.
 */
type HelpSection = 'what-is-auth' | 'role-assignment' | 'troubleshooting'

// =============================================================================
// Constants
// =============================================================================

const POLL_INTERVAL_MS = 5000 // Poll every 5 seconds

const ROLE_ASSIGNMENT_COMMAND = `# Assign Log Analytics Reader role to current user
$workspaceId = "<your-workspace-id>"
$currentUser = az ad signed-in-user show --query id -o tsv
az role assignment create \\
  --role "Log Analytics Reader" \\
  --assignee $currentUser \\
  --scope $workspaceId
`

const HELP_CONTENT = {
  'what-is-auth': {
    title: 'Why is authentication required?',
    content: [
      'Azure requires authentication to access Log Analytics data securely.',
      '',
      'The following are needed:',
      '• Valid Azure credentials (via azd auth login)',
      '• Log Analytics Reader role (or equivalent) on the workspace',
      '',
      'Roles that grant log access:',
      '• Log Analytics Reader (recommended)',
      '• Log Analytics Contributor',
      '• Monitoring Reader',
      '• Monitoring Contributor',
      '• Reader (on workspace)',
      '• Contributor (on workspace)',
      '• Owner (on workspace)',
      '',
      'This ensures only authorized users can view logs from your services.',
    ],
  },
  'role-assignment': {
    title: 'How to assign permissions',
    content: [
      'You can assign the Log Analytics Reader role (or equivalent) in several ways:',
      '',
      '1. Via Azure Portal:',
      '   • Navigate to your Log Analytics workspace',
      '   • Select "Access control (IAM)" from the left menu',
      '   • Click "+ Add" → "Add role assignment"',
      '   • Select "Log Analytics Reader" role (or Monitoring Reader)',
      '   • Search for your user and add them',
      '   • Click "Review + assign"',
      '',
      '2. Via Azure CLI:',
      '   Use the command below (replace <your-workspace-id> with your workspace resource ID)',
    ],
  },
  'troubleshooting': {
    title: 'Troubleshooting',
    content: [
      'Common issues and solutions:',
      '',
      '• Not signed in:',
      '  Run: azd auth login',
      '',
      '• Wrong subscription:',
      '  Run: az account set --subscription <subscription-id>',
      '',
      '• Permission denied:',
      '  Your account needs one of these roles on the workspace:',
      '  - Log Analytics Reader (recommended)',
      '  - Log Analytics Contributor',
      '  - Monitoring Reader',
      '  - Reader, Contributor, or Owner',
      '  Ask your Azure administrator or use the role assignment command above.',
      '',
      '• Role assigned but still showing missing:',
      '  Permissions can take up to 5 minutes to propagate. Click "Recheck" after waiting.',
    ],
  },
}

// =============================================================================
// Helper Components
// =============================================================================

interface StatusBadgeProps {
  status: AuthState['status']
  hasPermission?: boolean
}

function StatusBadge({ status, hasPermission }: Readonly<StatusBadgeProps>) {
  // If authenticated, show permission status
  if (status === 'authenticated') {
    const config = hasPermission
      ? {
          Icon: CheckCircle,
          label: 'Authorized',
          className: 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800',
        }
      : {
          Icon: ShieldAlert,
          label: 'Permission Missing',
          className: 'bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800',
        }

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

  // Not authenticated
  const config = {
    'not-authenticated': {
      Icon: Circle,
      label: 'Not Authenticated',
      className: 'bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700',
    },
    'permission-denied': {
      Icon: AlertTriangle,
      label: 'Permission Denied',
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
// AuthSetupStep Component
// =============================================================================

export function AuthSetupStep({ onValidationChange }: Readonly<AuthSetupStepProps>) {
  const [authState, setAuthState] = React.useState<AuthState | null>(null)
  const [isLoading, setIsLoading] = React.useState(true)
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [openSection, setOpenSection] = React.useState<HelpSection | null>(null)

  const pollIntervalRef = React.useRef<number | null>(null)

  // Fetch auth state from API
  const fetchAuthState = React.useCallback(async (isManualRefresh = false) => {
    if (isManualRefresh) {
      setIsRefreshing(true)
    }

    try {
      const response = await fetch('/api/azure/logs/setup-state')
      if (!response.ok) {
        throw new Error(`Failed to fetch setup state: ${response.statusText}`)
      }

      const data = (await response.json()) as SetupStateResponse
      setAuthState(data.authentication)
      setError(null)

      // Update validation state
      const isValid = data.authentication.status === 'authenticated' && data.authentication.hasLogAnalyticsReader
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
    void fetchAuthState()
  }, [fetchAuthState])

  // Set up polling (every 5 seconds when component is mounted)
  React.useEffect(() => {
    pollIntervalRef.current = window.setInterval(() => {
      void fetchAuthState()
    }, POLL_INTERVAL_MS)

    return () => {
      if (pollIntervalRef.current !== null) {
        clearInterval(pollIntervalRef.current)
      }
    }
  }, [fetchAuthState])

  const handleRefresh = () => {
    void fetchAuthState(true)
  }

  const handleToggleSection = (section: HelpSection) => {
    setOpenSection(openSection === section ? null : section)
  }

  if (isLoading) {
    return (
      <div className="p-8 flex flex-col items-center justify-center gap-3">
        <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
        <p className="text-sm text-slate-600 dark:text-slate-400">Checking authentication...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4">
          <p className="text-sm text-red-800 dark:text-red-300 font-medium">Failed to load authentication state</p>
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

  if (!authState) {
    return null
  }

  const isAuthenticated = authState.status === 'authenticated'
  const hasPermission = authState.hasLogAnalyticsReader
  const isFullyAuthorized = isAuthenticated && hasPermission

  return (
    <div className="p-6 space-y-6">
      {/* Status Section */}
      <div>
        <div className="flex items-start justify-between gap-4 mb-3">
          <div>
            <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
              Authentication &amp; Permissions
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400">
              Verify your Azure credentials and Log Analytics access
            </p>
          </div>
          <StatusBadge status={authState.status} hasPermission={hasPermission} />
        </div>

        {/* Authentication Details */}
        {isAuthenticated && (
          <div className="space-y-3">
            {/* User Principal */}
            <div className="rounded-lg bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 p-3">
              <div className="flex items-center gap-2 mb-1">
                <UserCheck className="w-4 h-4 text-emerald-500" />
                <span className="text-xs font-medium text-slate-600 dark:text-slate-400">Signed in as</span>
              </div>
              <p className="text-sm font-mono text-slate-900 dark:text-slate-100 break-all">
                {authState.principal || 'Unknown'}
              </p>
            </div>

            {/* Permission Checklist */}
            <div className="rounded-lg bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 p-3">
              <div className="flex items-start gap-2">
                {hasPermission ? (
                  <CheckCircle className="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                )}
                <div className="flex-1">
                  <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                    Log Analytics Reader Role
                  </p>
                  <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                    {hasPermission
                      ? 'You have the required permission to read logs'
                      : 'This role is required to access Log Analytics data'}
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

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

      {/* Action Required - Not Authenticated */}
      {!isAuthenticated && (
        <div className="rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300 mb-2">
                Action Required: Sign in to Azure
              </p>
              <p className="text-sm text-amber-700 dark:text-amber-400 mb-3">
                You need to authenticate with Azure before accessing Log Analytics.
              </p>
              <CodeBlock
                code="azd auth login"
                language="bash"
              />
              <p className="text-xs text-amber-600 dark:text-amber-500 mt-2">
                After signing in, click "Recheck" to verify your authentication.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Action Required - Missing Permission */}
      {isAuthenticated && !hasPermission && (
        <div className="rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 p-4">
          <div className="flex items-start gap-3">
            <ShieldAlert className="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300 mb-2">
                Action Required: Assign Log Analytics Reader Role
              </p>
              <p className="text-sm text-amber-700 dark:text-amber-400 mb-3">
                You need the "Log Analytics Reader" role to view logs. Ask your administrator or use the command below:
              </p>
              <p className="text-xs text-amber-600 dark:text-amber-500 mb-3">
                Other roles that grant access: Log Analytics Contributor, Monitoring Reader, Monitoring Contributor, Reader, Contributor, or Owner (on workspace)
              </p>
              <CodeBlock
                code={ROLE_ASSIGNMENT_COMMAND}
                language="bash"
              />
              <div className="mt-3 flex items-center gap-2">
                <a
                  href="https://portal.azure.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={cn(
                    'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium',
                    'bg-amber-100 dark:bg-amber-900 text-amber-700 dark:text-amber-300',
                    'hover:bg-amber-200 dark:hover:bg-amber-800',
                    'focus:outline-none focus:ring-2 focus:ring-amber-500',
                    'transition-colors duration-150',
                  )}
                >
                  <ExternalLink className="w-3.5 h-3.5" />
                  Azure Portal
                </a>
                <p className="text-xs text-amber-600 dark:text-amber-500">
                  After assigning the role, click "Recheck" (permissions may take up to 5 minutes to propagate)
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Help Sections */}
      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider">
          Help &amp; Guidance
        </h4>

        {/* What is Auth */}
        <CollapsibleSection
          id="what-is-auth"
          title={HELP_CONTENT['what-is-auth'].title}
          isOpen={openSection === 'what-is-auth'}
          onToggle={() => handleToggleSection('what-is-auth')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300 space-y-2">
            {HELP_CONTENT['what-is-auth'].content.map((line, idx) => (
              <p key={idx} className={line === '' ? 'h-2' : ''}>
                {line}
              </p>
            ))}
          </div>
        </CollapsibleSection>

        {/* Role Assignment */}
        <CollapsibleSection
          id="role-assignment"
          title={HELP_CONTENT['role-assignment'].title}
          isOpen={openSection === 'role-assignment'}
          onToggle={() => handleToggleSection('role-assignment')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300 space-y-2">
            {HELP_CONTENT['role-assignment'].content.map((line, idx) => {
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
            <CodeBlock code={ROLE_ASSIGNMENT_COMMAND} language="bash" />
          </div>
        </CollapsibleSection>

        {/* Troubleshooting */}
        <CollapsibleSection
          id="troubleshooting"
          title={HELP_CONTENT['troubleshooting'].title}
          isOpen={openSection === 'troubleshooting'}
          onToggle={() => handleToggleSection('troubleshooting')}
        >
          <div className="text-sm text-slate-700 dark:text-slate-300 space-y-2">
            {HELP_CONTENT['troubleshooting'].content.map((line, idx) => {
              if (line === '') {
                return <div key={idx} className="h-2" />
              }
              if (line.startsWith('  Run:')) {
                return (
                  <p key={idx} className="font-mono text-xs pl-4 text-slate-600 dark:text-slate-400">
                    {line.replace('  Run: ', '')}
                  </p>
                )
              }
              return <p key={idx}>{line}</p>
            })}
          </div>
        </CollapsibleSection>
      </div>

      {/* Success Message */}
      {isFullyAuthorized && (
        <div className="rounded-lg border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-900/20 p-4">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-emerald-800 dark:text-emerald-300">
                Authentication successful!
              </p>
              <p className="text-sm text-emerald-700 dark:text-emerald-400 mt-1">
                You have the required permissions to access Log Analytics. Click "Next" to continue.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default AuthSetupStep
