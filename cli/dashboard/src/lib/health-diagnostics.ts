/**
 * Health diagnostics utilities
 * 
 * Builds diagnostic information from health check results including
 * suggested actions, formatted reports, and error analysis.
 */

import type { HealthCheckResult, Service, HealthDiagnostic, HealthAction } from '@/types'
import { formatResponseTime, formatUptime } from './service-formatters'
import { getCheckTypeDisplay } from './service-health'

/**
 * Build complete health diagnostic from health check result
 */
export function buildHealthDiagnostic(
  healthStatus: HealthCheckResult,
  service: Service
): HealthDiagnostic {
  const actions = getSuggestedActions(healthStatus, service)
  const report = formatDiagnosticReport(healthStatus, service, actions)
  
  return {
    service,
    healthStatus,
    suggestedActions: actions,
    formattedReport: report,
  }
}

/**
 * Generate suggested actions based on health status and error details
 */
export function getSuggestedActions(
  healthStatus: HealthCheckResult,
  service: Service
): HealthAction[] {
  const actions: HealthAction[] = []
  
  // Always suggest viewing logs for unhealthy services
  if (healthStatus.status === 'unhealthy') {
    actions.push({
      label: 'Check service logs',
      icon: 'terminal',
      command: `azd app logs --service ${service.name}`,
    })
  }
  
  // Add specific suggestions based on check type and error
  if (healthStatus.checkType === 'http' && healthStatus.statusCode) {
    actions.push(...getHTTPSpecificActions(healthStatus.statusCode, service))
  } else if (healthStatus.checkType === 'tcp') {
    actions.push(...getTCPSpecificActions(healthStatus, service))
  } else if (healthStatus.checkType === 'process') {
    actions.push(...getProcessSpecificActions(healthStatus, service))
  }
  
  // Add suggestion from backend if available and not redundant
  if (healthStatus.details?.suggestion) {
    const backendSuggestion = healthStatus.details.suggestion as string
    // Only add if not already in actions
    if (!actions.some(a => a.label === backendSuggestion)) {
      actions.push({
        label: backendSuggestion,
      })
    }
  }
  
  // Add degraded-specific actions
  if (healthStatus.status === 'degraded') {
    if (!actions.some(a => a.label.includes('performance'))) {
      actions.push(
        { label: 'Check CPU and memory usage' },
        { label: 'Review application performance metrics' }
      )
    }
  }
  
  // Add consecutive failure context
  if (healthStatus.consecutiveFailures && healthStatus.consecutiveFailures >= 3) {
    actions.push({
      label: `Service has failed ${healthStatus.consecutiveFailures} times consecutively - consider restarting`,
      icon: 'alert-triangle',
    })
  }
  
  return actions
}

/**
 * Get HTTP-specific actions based on status code
 */
export function getHTTPSpecificActions(statusCode: number, service: Service): HealthAction[] {
  const actions: HealthAction[] = []
  
  if (statusCode === 503) {
    actions.push(
      { label: 'Verify all service dependencies are running' },
      { label: 'Check database connection status' },
      { label: 'Review connection pool settings' }
    )
  } else if (statusCode >= 500 && statusCode < 600) {
    actions.push(
      { label: 'Check application logs for errors' },
      { label: 'Review error stack traces' },
      { label: 'Verify recent code deployments' }
    )
  } else if (statusCode === 404) {
    actions.push(
      { 
        label: 'Verify health endpoint configuration',
        command: `Check azure.yaml for ${service.name} health check settings`
      },
      { 
        label: 'View health check documentation',
        docsUrl: 'https://github.com/jongio/azd-app/blob/main/cli/docs/features/health-checks.md'
      }
    )
  } else if (statusCode === 401 || statusCode === 403) {
    actions.push(
      { label: 'Check authentication credentials' },
      { label: 'Verify API keys and tokens' },
      { label: 'Review service permissions' }
    )
  } else if (statusCode === 429) {
    actions.push(
      { label: 'Reduce request rate' },
      { label: 'Check rate limiting quotas' },
      { label: 'Review throttling configuration' }
    )
  } else if (statusCode === 408 || statusCode === 504) {
    actions.push(
      { label: 'Check network connectivity' },
      { label: 'Verify timeout settings' },
      { label: 'Review service response time' }
    )
  }
  
  return actions
}

/**
 * Get TCP-specific actions
 */
export function getTCPSpecificActions(healthStatus: HealthCheckResult, service: Service): HealthAction[] {
  const actions: HealthAction[] = []
  const error = healthStatus.error?.toLowerCase() || ''
  
  if (error.includes('connection refused')) {
    actions.push(
      { label: 'Verify service is running' },
      { label: 'Check if port is correct' },
      { label: `Restart service: azd app restart --service ${service.name}` }
    )
  } else if (error.includes('timeout')) {
    actions.push(
      { label: 'Check network connectivity' },
      { label: 'Verify firewall rules' },
      { label: 'Review DNS resolution' }
    )
  } else if (error.includes('port') || error.includes('bind')) {
    actions.push(
      { label: 'Verify port is not already in use' },
      { label: 'Check port configuration in azure.yaml' }
    )
  }
  
  return actions
}

/**
 * Get process-specific actions
 */
export function getProcessSpecificActions(healthStatus: HealthCheckResult, service: Service): HealthAction[] {
  const actions: HealthAction[] = []
  const error = healthStatus.error?.toLowerCase() || ''
  
  if (error.includes('not running') || error.includes('not found')) {
    actions.push(
      { label: 'Verify service start command' },
      { label: 'Check service logs for startup errors' },
      { label: `Start service: azd app start --service ${service.name}` }
    )
  } else if (error.includes('crashed') || error.includes('exit')) {
    actions.push(
      { label: 'Review crash logs for exit code' },
      { label: 'Check for runtime errors' },
      { label: 'Verify dependencies are installed' }
    )
  } else if (error.includes('pattern') || error.includes('output')) {
    actions.push(
      { label: 'Check health check pattern configuration' },
      { label: 'Verify service startup sequence' },
      { label: 'Allow more time for service initialization' }
    )
  }
  
  return actions
}

/**
 * Format diagnostic report as markdown for copy-to-clipboard
 */
export function formatDiagnosticReport(
  healthStatus: HealthCheckResult,
  service: Service,
  actions: HealthAction[]
): string {
  const timestamp = new Date().toISOString()
  
  // Build markdown report
  let report = `# Service Health Diagnostic Report\n`
  report += `**Service**: ${service.name}\n`
  report += `**Status**: ${healthStatus.status}\n`
  report += `**Timestamp**: ${timestamp}\n\n`
  
  // Health check details
  report += `## Health Check\n`
  report += `- **Type**: ${getCheckTypeDisplay(healthStatus.checkType).toUpperCase()}\n`
  if (healthStatus.endpoint) {
    report += `- **Endpoint**: ${healthStatus.endpoint}\n`
  }
  if (healthStatus.statusCode) {
    report += `- **Status Code**: ${healthStatus.statusCode}\n`
  }
  report += `- **Response Time**: ${formatResponseTime(healthStatus.responseTime)}\n`
  if (healthStatus.consecutiveFailures && healthStatus.consecutiveFailures > 0) {
    report += `- **Consecutive Failures**: ${healthStatus.consecutiveFailures}\n`
  }
  if (healthStatus.lastSuccessTime) {
    report += `- **Last Success**: ${new Date(healthStatus.lastSuccessTime).toLocaleString()}\n`
  }
  report += `\n`
  
  // Error information
  if (healthStatus.error) {
    report += `## Error\n`
    report += `${healthStatus.error}\n`
    if (healthStatus.errorDetails) {
      report += `\n${healthStatus.errorDetails}\n`
    }
    report += `\n`
  }
  
  // Service information
  report += `## Service Info\n`
  report += `- **Uptime**: ${formatUptime(healthStatus.uptime)}\n`
  if (healthStatus.port && healthStatus.port > 0) {
    report += `- **Port**: ${healthStatus.port}\n`
  }
  if (healthStatus.pid) {
    report += `- **PID**: ${healthStatus.pid}\n`
  }
  if (service.local?.serviceType) {
    report += `- **Type**: ${service.local.serviceType}\n`
  }
  if (service.local?.serviceMode) {
    report += `- **Mode**: ${service.local.serviceMode}\n`
  }
  report += `\n`
  
  // Suggested actions
  if (actions.length > 0) {
    report += `## Suggested Actions\n`
    actions.forEach((action, i) => {
      report += `${i + 1}. ${action.label}`
      if (action.command) {
        report += ` → \`${action.command}\``
      }
      if (action.docsUrl) {
        report += ` → [Documentation](${action.docsUrl})`
      }
      report += `\n`
    })
    report += `\n`
  }
  
  report += `---\n`
  report += `Generated by azd app health\n`
  
  return report
}
