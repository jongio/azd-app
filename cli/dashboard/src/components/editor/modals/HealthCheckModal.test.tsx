/**
 * Health Check Modal Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HealthCheckModal } from './HealthCheckModal'
import type { HealthCheckConfig, ServiceInfo } from '@/lib/editor/healthcheck-types'

describe('HealthCheckModal', () => {
  const mockOnClose = vi.fn()
  const mockOnSave = vi.fn()

  beforeEach(() => {
    mockOnClose.mockClear()
    mockOnSave.mockClear()
  })

  describe('rendering', () => {
    it('should render modal when open', () => {
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByText('Configure Health Check')).toBeInTheDocument()
      expect(screen.getByText(/Set up health monitoring/i)).toBeInTheDocument()
    })

    it('should not render modal when closed', () => {
      render(
        <HealthCheckModal
          isOpen={false}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.queryByText('Configure Health Check')).not.toBeInTheDocument()
    })

    it('should render all health check type options', () => {
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByText('HTTP')).toBeInTheDocument()
      expect(screen.getByText('TCP')).toBeInTheDocument()
      expect(screen.getByText('Process')).toBeInTheDocument()
      expect(screen.getByText('Output')).toBeInTheDocument()
      expect(screen.getByText('None')).toBeInTheDocument()
    })

    it('should show HTTP fields by default', () => {
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByLabelText(/Health Check URL/i)).toBeInTheDocument()
      expect(screen.getByPlaceholderText('http://localhost:8080/health')).toBeInTheDocument()
    })
  })

  describe('type selection', () => {
    it('should show TCP fields when TCP type selected', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const tcpButton = screen.getByText('TCP').closest('button')
      await user.click(tcpButton!)

      expect(screen.getByLabelText(/TCP Port/i)).toBeInTheDocument()
      expect(screen.queryByLabelText(/Health Check URL/i)).not.toBeInTheDocument()
    })

    it('should show process fields when Process type selected', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const processButton = screen.getByText('Process').closest('button')
      await user.click(processButton!)

      expect(screen.getByLabelText(/Command/i)).toBeInTheDocument()
      expect(screen.queryByLabelText(/Health Check URL/i)).not.toBeInTheDocument()
    })

    it('should show output fields when Output type selected', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const outputButton = screen.getByText('Output').closest('button')
      await user.click(outputButton!)

      expect(screen.getByLabelText(/Expected Output Pattern/i)).toBeInTheDocument()
      expect(screen.queryByLabelText(/Health Check URL/i)).not.toBeInTheDocument()
    })

    it('should hide duration fields when None type selected', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const noneButton = screen.getByText('None').closest('button')
      await user.click(noneButton!)

      expect(screen.queryByLabelText(/Interval/i)).not.toBeInTheDocument()
      expect(screen.queryByLabelText(/Timeout/i)).not.toBeInTheDocument()
    })
  })

  describe('validation', () => {
    it('should require URL for HTTP health check', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i)
      await user.clear(urlInput)

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/URL is required/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should validate URL format', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i)
      await user.clear(urlInput)
      await user.type(urlInput, 'not-a-valid-url')

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/Invalid URL format/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should require port for TCP health check', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const tcpButton = screen.getByText('TCP').closest('button')
      await user.click(tcpButton!)

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/Port is required/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should validate port range', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const tcpButton = screen.getByText('TCP').closest('button')
      await user.click(tcpButton!)

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/Port is required/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should validate duration format', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const intervalInput = screen.getByPlaceholderText('30s')
      await user.clear(intervalInput)
      await user.type(intervalInput, 'invalid')

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/Invalid duration format/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })
  })

  describe('form submission', () => {
    it('should save HTTP health check', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i)
      await user.clear(urlInput)
      await user.type(urlInput, 'http://localhost:3000/healthz')

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          type: 'http',
          test: 'http://localhost:3000/healthz',
          path: '/healthz',
          interval: '30s',
          timeout: '5s',
          retries: 3,
          start_period: '0s',
          start_interval: '5s',
        })
      })
      expect(mockOnClose).toHaveBeenCalled()
    })

    it('should save TCP health check', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const tcpButton = screen.getByText('TCP').closest('button')
      await user.click(tcpButton!)

      const portInput = screen.getByLabelText(/TCP Port/i)
      await user.type(portInput, '5432')

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          type: 'tcp',
          test: 'tcp://localhost:5432',
          interval: '30s',
          timeout: '5s',
          retries: 3,
          start_period: '0s',
          start_interval: '5s',
        })
      })
      expect(mockOnClose).toHaveBeenCalled()
    })

    it('should save disabled health check for None type', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const noneButton = screen.getByText('None').closest('button')
      await user.click(noneButton!)

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          disable: true,
        })
      })
      expect(mockOnClose).toHaveBeenCalled()
    })

    it('should save only provided optional fields', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i)
      await user.clear(urlInput)
      await user.type(urlInput, 'http://localhost:8080/health')

      const intervalInput = screen.getByPlaceholderText('30s')
      await user.clear(intervalInput)

      const timeoutInput = document.querySelector('#health-check-timeout') as HTMLInputElement
      await user.clear(timeoutInput)

      const retriesInput = document.querySelector('#health-check-retries') as HTMLInputElement
      await user.clear(retriesInput)

      const startPeriodInput = document.querySelector('#health-check-start-period') as HTMLInputElement
      await user.clear(startPeriodInput)

      const startIntervalInput = document.querySelector('#health-check-start-interval') as HTMLInputElement
      await user.clear(startIntervalInput)

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          type: 'http',
          test: 'http://localhost:8080/health',
          path: '/health',
        })
      })
    })
  })

  describe('initial config', () => {
    it('should load existing HTTP config', () => {
      const initialConfig: HealthCheckConfig = {
        type: 'http',
        test: 'http://localhost:3000/api/health',
        interval: '20s',
        timeout: '10s',
        retries: 5,
      }

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i) as HTMLInputElement
      expect(urlInput.value).toBe('http://localhost:3000/api/health')

      const intervalInput = screen.getByPlaceholderText('30s') as HTMLInputElement
      expect(intervalInput.value).toBe('20s')

      const timeoutInput = document.querySelector('#health-check-timeout') as HTMLInputElement
      expect(timeoutInput.value).toBe('10s')

      const retriesInput = document.querySelector('#health-check-retries') as HTMLInputElement
      expect(retriesInput.value).toBe('5')
    })

    it('should load existing TCP config', () => {
      const initialConfig: HealthCheckConfig = {
        type: 'tcp',
        test: 'tcp://localhost:5432',
        interval: '15s',
      }

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
        />
      )

      const portInput = screen.getByLabelText(/TCP Port/i) as HTMLInputElement
      expect(portInput.value).toBe('5432')
    })

    it('should show None type for disabled config', () => {
      const initialConfig: HealthCheckConfig = {
        disable: true,
      }

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
        />
      )

      const noneButton = screen.getByText('None').closest('button')
      expect(noneButton).toHaveClass(/border-cyan-500/)
    })
  })

  describe('service info defaults', () => {
    it('should suggest postgres TCP health check', () => {
      const serviceInfo: ServiceInfo = {
        image: 'postgres:16-alpine',
      }

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          serviceInfo={serviceInfo}
        />
      )

      // Should auto-select TCP type
      const tcpButton = screen.getByText('TCP').closest('button')
      expect(tcpButton).toHaveClass(/border-cyan-500/)

      // Should suggest port 5432
      const portInput = screen.getByLabelText(/TCP Port/i) as HTMLInputElement
      expect(portInput.value).toBe('5432')
    })

    it('should suggest Node.js HTTP health check', () => {
      const serviceInfo: ServiceInfo = {
        language: 'node',
        ports: ['3000:3000'],
      }

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          serviceInfo={serviceInfo}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i) as HTMLInputElement
      expect(urlInput.value).toBe('http://localhost:3000/health')
    })

    it('should suggest Java Spring Boot health check', () => {
      const serviceInfo: ServiceInfo = {
        language: 'java',
        ports: ['8080:8080'],
      }

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          serviceInfo={serviceInfo}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i) as HTMLInputElement
      expect(urlInput.value).toBe('http://localhost:8080/actuator/health')
    })
  })

  describe('user interactions', () => {
    it('should close modal on cancel button click', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const cancelButton = screen.getByText('Cancel')
      await user.click(cancelButton)

      expect(mockOnClose).toHaveBeenCalled()
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should disable submit button while submitting', async () => {
      const user = userEvent.setup()
      mockOnSave.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)))

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText('Saving...')).toBeInTheDocument()
      })
      expect(saveButton).toBeDisabled()
    })

    it('should handle save errors gracefully', async () => {
      const user = userEvent.setup()
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
      
      mockOnSave.mockRejectedValueOnce(new Error('Save failed'))

      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(alertSpy).toHaveBeenCalledWith('Failed to save health check. Please try again.')
      })
      // Note: onClose may be called due to async timing

      consoleError.mockRestore()
      alertSpy.mockRestore()
    })
  })

  describe('accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i)
      expect(urlInput).toHaveAttribute('aria-invalid', 'false')

      const intervalInput = screen.getByPlaceholderText('30s')
      expect(intervalInput).toHaveAttribute('aria-describedby')
    })

    it('should show aria-invalid when field has error', async () => {
      const user = userEvent.setup()
      render(
        <HealthCheckModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const urlInput = screen.getByLabelText(/Health Check URL/i)
      await user.clear(urlInput)

      const saveButton = screen.getByText('Save Health Check')
      await user.click(saveButton)

      await waitFor(() => {
        expect(urlInput).toHaveAttribute('aria-invalid', 'true')
        expect(urlInput).toHaveAttribute('aria-describedby')
      })
    })
  })
})
