/**
 * Component tests for HealthTooltipContent
 * Tests tooltip content layout and rendering
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HealthTooltipContent } from './HealthTooltipContent'
import type { HealthDiagnostic, HealthCheckResult, Service, HealthAction } from '@/types'

describe('HealthTooltipContent', () => {
  const baseService: Service = {
    name: 'api',
    host: 'local',
    local: {
      status: 'running',
      health: 'healthy',
      port: 8080,
      serviceType: 'http',
      serviceMode: 'daemon',
    },
  }

  const mockOnCopy = vi.fn()

  describe('status header', () => {
    it('displays healthy status with correct styling', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Service Health: Healthy/i)).toBeInTheDocument()
      expect(screen.getByText('api')).toBeInTheDocument()
    })

    it('displays unhealthy status with correct styling', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Connection failed',
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Service Health: Unhealthy/i)).toBeInTheDocument()
    })

    it('displays degraded status with correct styling', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'degraded',
        checkType: 'http',
        responseTime: 1500000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Service Health: Degraded/i)).toBeInTheDocument()
    })

    it('displays unknown status with correct styling', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unknown',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Service Health: Unknown/i)).toBeInTheDocument()
    })
  })

  describe('check details section', () => {
    it('displays HTTP check details', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 200,
        responseTime: 12000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 300000000000,
        port: 8080,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/localhost/i)).toBeInTheDocument()
      expect(screen.getByText('200')).toBeInTheDocument()
      expect(screen.getByText(/12ms/i)).toBeInTheDocument()
    })

    it('displays TCP check details', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'database',
        status: 'healthy',
        checkType: 'tcp',
        responseTime: 5000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 180000000000,
        port: 5432,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/TCP/i)).toBeInTheDocument()
      expect(screen.getByText(/5ms/i)).toBeInTheDocument()
    })

    it('displays process check details', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'worker',
        status: 'healthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 120000000000,
        pid: 12345,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/PROCESS/i)).toBeInTheDocument()
    })

    it('displays consecutive failures', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        consecutiveFailures: 5,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Consecutive Failures/i)).toBeInTheDocument()
      expect(screen.getByText('5')).toBeInTheDocument()
    })
  })

  describe('error section', () => {
    it('displays error details when present', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'HTTP 503: Service Unavailable',
        errorDetails: 'Database connection pool exhausted',
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Error Details/i)).toBeInTheDocument()
      expect(screen.getByText(/HTTP 503: Service Unavailable/i)).toBeInTheDocument()
      expect(screen.getByText(/Database connection pool exhausted/i)).toBeInTheDocument()
    })

    it('does not show error section when no error', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.queryByText(/Error Details/i)).not.toBeInTheDocument()
    })
  })

  describe('service info section', () => {
    it('displays service uptime', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 300000000000, // 5 minutes
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Uptime/i)).toBeInTheDocument()
      expect(screen.getByText(/5m/i)).toBeInTheDocument()
    })

    it('displays port when available', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
        port: 8080,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Port/i)).toBeInTheDocument()
      expect(screen.getByText('8080')).toBeInTheDocument()
    })

    it('displays PID when available', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
        pid: 12345,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/PID/i)).toBeInTheDocument()
      expect(screen.getByText('12345')).toBeInTheDocument()
    })

    it('displays service type and mode', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      // Service type displays as 'http' and check type label shows 'Type'
      const typeElements = screen.getAllByText(/http/i)
      expect(typeElements.length).toBeGreaterThan(0)
      expect(screen.getByText(/Mode/i)).toBeInTheDocument()
      expect(screen.getByText(/daemon/i)).toBeInTheDocument()
    })

    it('does not show port when zero', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
        port: 0,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      const { container } = render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      // Port section should not exist
      const portTexts = Array.from(container.querySelectorAll('*')).filter(
        el => el.textContent?.includes('Port:') && el.textContent?.includes('0')
      )
      expect(portTexts.length).toBe(0)
    })
  })

  describe('suggested actions section', () => {
    it('displays suggested actions', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions: HealthAction[] = [
        { label: 'Check service logs', command: 'azd app logs --service api' },
        { label: 'Verify dependencies are running' },
      ]

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: actions,
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Suggested Actions/i)).toBeInTheDocument()
      expect(screen.getByText(/Check service logs/i)).toBeInTheDocument()
      expect(screen.getByText(/azd app logs --service api/i)).toBeInTheDocument()
      expect(screen.getByText(/Verify dependencies are running/i)).toBeInTheDocument()
    })

    it('limits actions to first 5', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions: HealthAction[] = Array.from({ length: 10 }, (_, i) => ({
        label: `Action ${i + 1}`,
      }))

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: actions,
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText('Action 1')).toBeInTheDocument()
      expect(screen.getByText('Action 5')).toBeInTheDocument()
      expect(screen.queryByText('Action 6')).not.toBeInTheDocument()
    })

    it('displays action with documentation link', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions: HealthAction[] = [
        {
          label: 'Read documentation',
          docsUrl: 'https://example.com/docs',
        },
      ]

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: actions,
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Read documentation/i)).toBeInTheDocument()
      const docLink = screen.getByRole('link', { name: /documentation/i })
      expect(docLink).toHaveAttribute('href', 'https://example.com/docs')
      expect(docLink).toHaveAttribute('target', '_blank')
      expect(docLink).toHaveAttribute('rel', 'noopener noreferrer')
    })

    it('does not show actions section when empty', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.queryByText(/Suggested Actions/i)).not.toBeInTheDocument()
    })
  })

  describe('copy button', () => {
    it('renders copy button', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: 'test report',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByRole('button', { name: /copy diagnostics/i })).toBeInTheDocument()
    })

    it('calls onCopy when clicked', async () => {
      const user = userEvent.setup()
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: 'test report',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      const copyButton = screen.getByRole('button', { name: /copy diagnostics/i })
      await user.click(copyButton)

      expect(mockOnCopy).toHaveBeenCalledTimes(1)
    })
  })

  describe('footer', () => {
    it('displays last checked timestamp', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:30:45Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Last checked/i)).toBeInTheDocument()
    })
  })

  describe('dark mode', () => {
    it('applies dark mode classes', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      const { container } = render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      // Should have dark: classes in the markup
      expect(container.innerHTML).toContain('dark:')
    })
  })

  describe('edge cases', () => {
    it('handles missing service local config', () => {
      const service: Service = {
        name: 'api',
        host: 'local',
      }

      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unknown',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic: HealthDiagnostic = {
        service,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Service Health: Unknown/i)).toBeInTheDocument()
    })

    it('handles missing endpoint gracefully', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'tcp',
        responseTime: 5000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.queryByText(/Endpoint/i)).not.toBeInTheDocument()
    })

    it('handles missing uptime', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic: HealthDiagnostic = {
        service: baseService,
        healthStatus,
        suggestedActions: [],
        formattedReport: '',
      }

      render(<HealthTooltipContent diagnostic={diagnostic} onCopy={mockOnCopy} />)

      expect(screen.getByText(/Uptime/i)).toBeInTheDocument()
    })
  })
})
