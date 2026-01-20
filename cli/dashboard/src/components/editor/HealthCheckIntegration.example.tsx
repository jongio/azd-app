/**
 * Health Check Integration Example
 * 
 * This file demonstrates how to integrate the HealthCheckModal with:
 * 1. QuickActionsBar - Add "Configure Health Check" button
 * 2. SchemaForm - Add health check editor in service configuration
 * 3. Service Editor - Context-aware health check configuration
 * 
 * Usage Example:
 * 
 * ```tsx
 * import { HealthCheckModal } from '@/components/editor/modals'
 * import { ServiceInfo, HealthCheckConfig } from '@/lib/editor/healthcheck-types'
 * 
 * function ServiceEditor() {
 *   const [healthCheckModalOpen, setHealthCheckModalOpen] = useState(false)
 *   const [currentHealthCheck, setCurrentHealthCheck] = useState<HealthCheckConfig>()
 *   const [serviceInfo, setServiceInfo] = useState<ServiceInfo>()
 *   
 *   const handleSaveHealthCheck = (config: HealthCheckConfig | null) => {
 *     // Update service configuration
 *     updateService({
 *       ...service,
 *       healthcheck: config
 *     })
 *   }
 *   
 *   return (
 *     <>
 *       <button onClick={() => setHealthCheckModalOpen(true)}>
 *         Configure Health Check
 *       </button>
 *       
 *       <HealthCheckModal
 *         isOpen={healthCheckModalOpen}
 *         onClose={() => setHealthCheckModalOpen(false)}
 *         onSave={handleSaveHealthCheck}
 *         initialConfig={currentHealthCheck}
 *         serviceInfo={serviceInfo}
 *       />
 *     </>
 *   )
 * }
 * ```
 */

import * as React from 'react'
import { Activity } from 'lucide-react'
import { HealthCheckModal } from '@/components/editor/modals'
import type { HealthCheckConfig, ServiceInfo } from '@/lib/editor/healthcheck-types'

export interface HealthCheckButtonProps {
  /** Current service information for defaults */
  serviceInfo?: ServiceInfo
  
  /** Current health check configuration */
  currentHealthCheck?: HealthCheckConfig
  
  /** Callback when health check is saved */
  onSave: (config: HealthCheckConfig | null) => void
  
  /** Button variant */
  variant?: 'primary' | 'secondary'
  
  /** Button size */
  size?: 'sm' | 'md' | 'lg'
}

/**
 * Health Check Button Component
 * Reusable button that opens the health check configuration modal
 */
export function HealthCheckButton({
  serviceInfo,
  currentHealthCheck,
  onSave,
  variant = 'secondary',
  size = 'md',
}: HealthCheckButtonProps) {
  const [isModalOpen, setIsModalOpen] = React.useState(false)

  const handleSave = (config: HealthCheckConfig | null) => {
    onSave(config)
    setIsModalOpen(false)
  }

  return (
    <>
      <button
        onClick={() => setIsModalOpen(true)}
        className={`
          inline-flex items-center gap-2 rounded-lg font-medium
          transition-colors duration-150
          focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2
          ${size === 'sm' ? 'px-3 py-1.5 text-xs' : ''}
          ${size === 'md' ? 'px-4 py-2 text-sm' : ''}
          ${size === 'lg' ? 'px-6 py-3 text-base' : ''}
          ${
            variant === 'primary'
              ? 'bg-cyan-600 text-white hover:bg-cyan-700 shadow-sm'
              : 'border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
          }
        `}
      >
        <Activity className="w-4 h-4" />
        Configure Health Check
      </button>

      <HealthCheckModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSave={handleSave}
        initialConfig={currentHealthCheck}
        serviceInfo={serviceInfo}
      />
    </>
  )
}

/**
 * Example: Integration with QuickActionsBar
 * 
 * Add health check button to QuickActionsBar for context-sensitive actions
 */
export function QuickActionsBarWithHealthCheck() {
  // This would be in your actual QuickActionsBar component
  const selectedService = React.useContext(ServiceContext)
  
  return (
    <div className="quick-actions-bar">
      {selectedService && (
        <HealthCheckButton
          serviceInfo={{
            host: selectedService.host,
            ports: selectedService.ports,
            image: selectedService.image,
            language: selectedService.language,
          }}
          currentHealthCheck={selectedService.healthcheck}
          onSave={(config) => {
            // Update service with new health check
            updateService(selectedService.name, {
              ...selectedService,
              healthcheck: config,
            })
          }}
          variant="primary"
        />
      )}
    </div>
  )
}

/**
 * Example: Integration with Service Form
 * 
 * Add health check section to service configuration form
 */
export function ServiceFormWithHealthCheck() {
  const [serviceData, setServiceData] = React.useState<Record<string, unknown>>({})
  const [healthCheckModalOpen, setHealthCheckModalOpen] = React.useState(false)

  return (
    <div className="service-form">
      {/* Other service fields... */}
      
      <div className="form-section">
        <h3 className="text-lg font-semibold mb-2">Health Check</h3>
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-3">
          Configure health monitoring for this service
        </p>
        
        <div className="flex items-center gap-3">
          {serviceData.healthcheck ? (
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-900">
              <Activity className="w-4 h-4 text-green-600 dark:text-green-400" />
              <span className="text-sm text-green-900 dark:text-green-100">
                Health check configured ({serviceData.healthcheck.type})
              </span>
            </div>
          ) : (
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-yellow-50 dark:bg-yellow-950/20 border border-yellow-200 dark:border-yellow-900">
              <Activity className="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
              <span className="text-sm text-yellow-900 dark:text-yellow-100">
                No health check configured
              </span>
            </div>
          )}
          
          <button
            onClick={() => setHealthCheckModalOpen(true)}
            className="px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
          >
            {serviceData.healthcheck ? 'Edit' : 'Configure'}
          </button>
        </div>
      </div>

      <HealthCheckModal
        isOpen={healthCheckModalOpen}
        onClose={() => setHealthCheckModalOpen(false)}
        onSave={(config) => {
          setServiceData({
            ...serviceData,
            healthcheck: config,
          })
        }}
        initialConfig={serviceData.healthcheck}
        serviceInfo={{
          host: serviceData.host,
          ports: serviceData.ports,
          image: serviceData.image,
          language: serviceData.language,
        }}
      />
    </div>
  )
}

// Mock context for examples
const ServiceContext = React.createContext<Record<string, unknown> | null>(null)

// Mock functions for examples
const updateService = (_name: string, _data: Record<string, unknown>) => {
  // Update service logic
}
