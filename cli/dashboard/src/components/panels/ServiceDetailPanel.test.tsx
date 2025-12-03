import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ServiceDetailPanel } from './ServiceDetailPanel'
import type { Service, HealthCheckResult } from '@/types'

// =============================================================================
// Test Data
// =============================================================================

const createService = (overrides: Partial<Service> = {}): Service => ({
  name: 'api',
  language: 'TypeScript',
  framework: 'Express',
  project: './services/api',
  local: {
    status: 'running',
    health: 'healthy',
    port: 3100,
    url: 'http://localhost:3100',
    pid: 12345,
    startTime: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    lastChecked: new Date().toISOString(),
    healthDetails: {
      checkType: 'http',
      endpoint: '/health',
      responseTime: 45000000, // 45ms in nanoseconds
      statusCode: 200,
      consecutiveFailures: 0,
    },
  },
  azure: {
    resourceName: 'api-containerapp',
    resourceType: 'containerapp',
    url: 'https://api.azurecontainers.io',
    imageName: 'myregistry.azurecr.io/api:latest',
    subscriptionId: 'xxxx-xxxx-xxxx-xxxx',
    resourceGroup: 'rg-myapp-prod',
    location: 'East US',
    containerAppEnvId: '/subscriptions/xxx/containerAppEnv',
    logAnalyticsId: '/subscriptions/xxx/logAnalytics',
  },
  environmentVariables: {
    NODE_ENV: 'production',
    API_KEY: 'secret-key-123',
    DATABASE_URL: 'postgres://localhost/db',
    PORT: '3100',
  },
  ...overrides,
})

const createHealthStatus = (overrides: Partial<HealthCheckResult> = {}): HealthCheckResult => ({
  serviceName: 'api',
  status: 'healthy',
  checkType: 'http',
  endpoint: '/health',
  responseTime: 45000000,
  statusCode: 200,
  timestamp: new Date().toISOString(),
  ...overrides,
})

// =============================================================================
// Helper Function Tests (via panel-utils.ts)
// =============================================================================

import {
  formatUptime,
  formatTimestamp,
  getStatusColor,
  getHealthColor,
  buildAzurePortalUrl,
  isSensitiveKey,
  maskValue,
  formatResourceType,
  getStatusDisplay,
  getHealthDisplay,
  formatCheckType,
  hasAzureDeployment,
} from '@/lib/panel-utils'

describe('panel-utils', () => {
  describe('formatUptime', () => {
    it('returns N/A for undefined', () => {
      expect(formatUptime(undefined)).toBe('N/A')
    })

    it('returns N/A for future timestamps', () => {
      const future = new Date(Date.now() + 10000).toISOString()
      expect(formatUptime(future)).toBe('N/A')
    })

    it('formats seconds only', () => {
      const recent = new Date(Date.now() - 30000).toISOString()
      const result = formatUptime(recent)
      expect(result).toMatch(/^\d+s$/)
    })

    it('formats minutes and seconds', () => {
      const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString()
      const result = formatUptime(fiveMinutesAgo)
      expect(result).toMatch(/^\d+m \d+s$/)
    })

    it('formats hours, minutes and seconds', () => {
      const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
      const result = formatUptime(twoHoursAgo)
      expect(result).toMatch(/^\d+h \d+m \d+s$/)
    })
  })

  describe('formatTimestamp', () => {
    it('returns N/A for undefined', () => {
      expect(formatTimestamp(undefined)).toBe('N/A')
    })

    it('returns N/A for invalid timestamp', () => {
      expect(formatTimestamp('invalid')).toBe('N/A')
    })

    it('formats valid timestamp', () => {
      const timestamp = '2024-12-01T10:30:00Z'
      const result = formatTimestamp(timestamp)
      expect(result).not.toBe('N/A')
    })
  })

  describe('getStatusColor', () => {
    it('returns green for running', () => {
      expect(getStatusColor('running')).toContain('green')
    })

    it('returns green for ready', () => {
      expect(getStatusColor('ready')).toContain('green')
    })

    it('returns yellow for starting', () => {
      expect(getStatusColor('starting')).toContain('yellow')
    })

    it('returns yellow for stopping', () => {
      expect(getStatusColor('stopping')).toContain('yellow')
    })

    it('returns gray for stopped', () => {
      expect(getStatusColor('stopped')).toContain('gray')
    })

    it('returns red for error', () => {
      expect(getStatusColor('error')).toContain('red')
    })

    it('returns gray for undefined', () => {
      expect(getStatusColor(undefined)).toContain('gray')
    })
  })

  describe('getHealthColor', () => {
    it('returns green for healthy', () => {
      expect(getHealthColor('healthy')).toContain('green')
    })

    it('returns yellow for degraded', () => {
      expect(getHealthColor('degraded')).toContain('yellow')
    })

    it('returns red for unhealthy', () => {
      expect(getHealthColor('unhealthy')).toContain('red')
    })

    it('returns gray for unknown', () => {
      expect(getHealthColor('unknown')).toContain('gray')
    })

    it('returns gray for undefined', () => {
      expect(getHealthColor(undefined)).toContain('gray')
    })
  })

  describe('buildAzurePortalUrl', () => {
    it('returns null without required fields', () => {
      const service = createService({ azure: undefined })
      expect(buildAzurePortalUrl(service)).toBeNull()
    })

    it('returns null with partial azure info', () => {
      const service = createService({
        azure: { resourceName: 'test' },
      })
      expect(buildAzurePortalUrl(service)).toBeNull()
    })

    it('builds URL for container app', () => {
      const service = createService()
      const url = buildAzurePortalUrl(service)
      expect(url).toContain('portal.azure.com')
      expect(url).toContain('Microsoft.App/containerApps')
    })

    it('builds URL for app service', () => {
      const service = createService({
        azure: {
          ...createService().azure,
          resourceType: 'appservice',
        },
      })
      const url = buildAzurePortalUrl(service)
      expect(url).toContain('Microsoft.Web/sites')
    })
  })

  describe('isSensitiveKey', () => {
    it('detects password', () => {
      expect(isSensitiveKey('PASSWORD')).toBe(true)
      expect(isSensitiveKey('db_password')).toBe(true)
    })

    it('detects secret', () => {
      expect(isSensitiveKey('SECRET_KEY')).toBe(true)
      expect(isSensitiveKey('client_secret')).toBe(true)
    })

    it('detects api key', () => {
      expect(isSensitiveKey('API_KEY')).toBe(true)
      expect(isSensitiveKey('apikey')).toBe(true)
    })

    it('detects token', () => {
      expect(isSensitiveKey('ACCESS_TOKEN')).toBe(true)
      expect(isSensitiveKey('auth_token')).toBe(true)
    })

    it('detects connection string', () => {
      expect(isSensitiveKey('CONNECTION_STRING')).toBe(true)
      expect(isSensitiveKey('db_connection_string')).toBe(true)
    })

    it('returns false for non-sensitive', () => {
      expect(isSensitiveKey('NODE_ENV')).toBe(false)
      expect(isSensitiveKey('PORT')).toBe(false)
      expect(isSensitiveKey('DEBUG')).toBe(false)
    })
  })

  describe('maskValue', () => {
    it('masks short values completely', () => {
      expect(maskValue('abc')).toBe('•••')
    })

    it('masks long values to 16 characters', () => {
      expect(maskValue('this-is-a-very-long-secret-key')).toBe('•'.repeat(16))
    })
  })

  describe('formatResourceType', () => {
    it('returns Unknown for undefined', () => {
      expect(formatResourceType(undefined)).toBe('Unknown')
    })

    it('formats container app', () => {
      expect(formatResourceType('containerapp')).toBe('Container App')
    })

    it('formats app service', () => {
      expect(formatResourceType('appservice')).toBe('App Service')
    })

    it('formats function', () => {
      expect(formatResourceType('function')).toBe('Function App')
    })

    it('returns original for unknown type', () => {
      expect(formatResourceType('custom')).toBe('custom')
    })
  })

  describe('getStatusDisplay', () => {
    it('returns running display', () => {
      const display = getStatusDisplay('running')
      expect(display.text).toBe('Running')
      expect(display.indicator).toBe('●')
    })

    it('returns stopped display', () => {
      const display = getStatusDisplay('stopped')
      expect(display.text).toBe('Stopped')
      expect(display.indicator).toBe('◉')
    })

    it('returns error display', () => {
      const display = getStatusDisplay('error')
      expect(display.text).toBe('Error')
      expect(display.indicator).toBe('⚠')
    })
  })

  describe('getHealthDisplay', () => {
    it('returns healthy display', () => {
      const display = getHealthDisplay('healthy')
      expect(display.text).toBe('Healthy')
      expect(display.indicator).toBe('●')
    })

    it('returns unhealthy display', () => {
      const display = getHealthDisplay('unhealthy')
      expect(display.text).toBe('Unhealthy')
      expect(display.indicator).toBe('●')
    })
  })

  describe('formatCheckType', () => {
    it('returns Unknown for undefined', () => {
      expect(formatCheckType(undefined)).toBe('Unknown')
    })

    it('formats http check', () => {
      expect(formatCheckType('http')).toBe('HTTP')
    })

    it('formats tcp check', () => { expect(formatCheckType('tcp')).toBe('TCP')
    })
  })

  describe('hasAzureDeployment', () => {
    it('returns true when resourceName exists', () => {
      const service = createService()
      expect(hasAzureDeployment(service)).toBe(true)
    })

    it('returns true when url exists', () => {
      const service = createService({
        azure: { url: 'https://example.com' },
      })
      expect(hasAzureDeployment(service)).toBe(true)
    })

    it('returns false when no azure info', () => {
      const service = createService({ azure: undefined })
      expect(hasAzureDeployment(service)).toBe(false)
    })
  })
})

// =============================================================================
// ServiceDetailPanel Component Tests
// =============================================================================

describe('ServiceDetailPanel', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    mockOnClose.mockClear()
  })

  describe('rendering', () => {
    it('does not render when isOpen is false', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={false}
          onClose={mockOnClose}
        />
      )
      expect(screen.queryByTestId('service-detail-panel')).not.toBeInTheDocument()
    })

    it('does not render when service is null', () => {
      render(
        <ServiceDetailPanel service={null} isOpen={true} onClose={mockOnClose} />
      )
      expect(screen.queryByTestId('service-detail-panel')).not.toBeInTheDocument()
    })

    it('renders when isOpen is true and service exists', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )
      expect(screen.getByTestId('service-detail-panel')).toBeInTheDocument()
    })

    it('renders backdrop when open', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )
      expect(screen.getByTestId('panel-backdrop')).toBeInTheDocument()
    })

    it('displays service name in header', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )
      expect(screen.getByText('api')).toBeInTheDocument()
    })

    it('displays framework in subtitle', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )
      expect(screen.getByText(/Express/)).toBeInTheDocument()
    })
  })

  describe('close behavior', () => {
    it('calls onClose when close button clicked', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByTestId('close-button'))
      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('calls onClose when backdrop clicked', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByTestId('panel-backdrop'))
      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('calls onClose when Escape key pressed', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      fireEvent.keyDown(document, { key: 'Escape' })
      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('tab navigation', () => {
    it('renders all four tabs', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByRole('button', { name: 'Overview' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Local' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Azure' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Environment' })).toBeInTheDocument()
    })

    it('defaults to Overview tab', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      // Overview tab content should be visible by default
      expect(screen.getByTestId('overview-tab-content')).toBeInTheDocument()
    })

    it('switches to Local tab on click', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Local' }))
      expect(screen.getByTestId('local-tab-content')).toBeInTheDocument()
    })

    it('switches to Azure tab on click', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Azure' }))
      expect(screen.getByTestId('azure-tab-content')).toBeInTheDocument()
    })

    it('switches to Environment tab on click', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Environment' }))
      expect(screen.getByTestId('environment-tab-content')).toBeInTheDocument()
    })
  })

  describe('Overview tab', () => {
    it('displays local development section', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByTestId('local-development-section')).toBeInTheDocument()
    })

    it('displays status and health', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      // Text appears multiple times (header and overview section)
      expect(screen.getAllByText(/Running/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/Healthy/).length).toBeGreaterThan(0)
    })

    it('displays Azure deployment section when deployed', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByTestId('azure-deployment-section')).toBeInTheDocument()
      // Check for deployed indicator in the section instead
      expect(screen.getByTestId('azure-deployment-section')).toHaveTextContent(/Deployed/i)
    })

    it('shows not deployed message when no Azure info', () => {
      render(
        <ServiceDetailPanel
          service={createService({ azure: undefined })}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByText('Not deployed to Azure')).toBeInTheDocument()
    })
  })

  describe('Local tab', () => {
    it('displays service details section', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Local' }))
      expect(screen.getByTestId('service-details-section')).toBeInTheDocument()
    })

    it('displays runtime section', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Local' }))
      expect(screen.getByTestId('runtime-section')).toBeInTheDocument()
    })

    it('displays timing section', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Local' }))
      expect(screen.getByTestId('timing-section')).toBeInTheDocument()
    })

    it('displays health details section when available', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Local' }))
      expect(screen.getByTestId('health-details-section')).toBeInTheDocument()
    })
  })

  describe('Azure tab', () => {
    it('displays resource section when deployed', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Azure' }))
      expect(screen.getByTestId('resource-section')).toBeInTheDocument()
    })

    it('displays Azure metadata section', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Azure' }))
      expect(screen.getByTestId('azure-metadata-section')).toBeInTheDocument()
    })

    it('displays Azure portal link', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Azure' }))
      expect(screen.getByText('Open in Azure Portal')).toBeInTheDocument()
    })

    it('shows not deployed message when no Azure info', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService({ azure: undefined })}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Azure' }))
      expect(screen.getByTestId('not-deployed-section')).toBeInTheDocument()
    })
  })

  describe('Environment tab', () => {
    it('displays environment variables', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Environment' }))
      expect(screen.getByText('NODE_ENV')).toBeInTheDocument()
      // Values are now in input fields
      expect(screen.getByDisplayValue('production')).toBeInTheDocument()
    })

    it('masks sensitive values by default', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Environment' }))
      // API_KEY should be masked (values are in input fields)
      expect(screen.queryByDisplayValue('secret-key-123')).not.toBeInTheDocument()
    })

    it('shows empty state when no env vars', async () => {
      const user = userEvent.setup()
      render(
        <ServiceDetailPanel
          service={createService({ environmentVariables: {} })}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      await user.click(screen.getByRole('button', { name: 'Environment' }))
      expect(
        screen.getByText('No environment variables configured')
      ).toBeInTheDocument()
    })
  })

  describe('accessibility', () => {
    it('has dialog role', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })

    it('has aria-modal attribute', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true')
    })

    it('has aria-labelledby pointing to title', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      const dialog = screen.getByRole('dialog')
      expect(dialog).toHaveAttribute('aria-labelledby', 'panel-title')
    })

    it('close button has aria-label', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByTestId('close-button')).toHaveAttribute(
        'aria-label',
        'Close panel'
      )
    })

    it('has tab buttons for navigation', () => {
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
        />
      )

      expect(screen.getByRole('button', { name: 'Overview' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Local' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Azure' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Environment' })).toBeInTheDocument()
    })
  })

  describe('with health status', () => {
    it('uses healthStatus for status display', () => {
      const healthStatus = createHealthStatus({ status: 'unhealthy' })
      render(
        <ServiceDetailPanel
          service={createService()}
          isOpen={true}
          onClose={mockOnClose}
          healthStatus={healthStatus}
        />
      )

      // Text is split across elements, so use partial match in the subtitle
      expect(screen.getByText(/Unhealthy/)).toBeInTheDocument()
    })
  })
})
