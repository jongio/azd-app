/**
 * Add Service Modal
 * Modal dialog for adding new services with three tabs:
 * 1. Well-Known Services - Pre-configured common services
 * 2. Application Service - Custom code services
 * 3. Container Service - Docker container services
 */

import * as React from 'react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { WellKnownServicesTab } from './WellKnownServicesTab'
import { ApplicationServiceTab } from './ApplicationServiceTab'
import { ContainerServiceTab } from './ContainerServiceTab'
import { cn } from '@/lib/utils'
import type { WellKnownService, ServiceFormData } from '@/lib/editor/wellknown-types'

export interface AddServiceModalProps {
  /** Whether modal is open */
  isOpen: boolean
  /** Callback to close modal */
  onClose: () => void
  /** Callback when service is added */
  onAddService: (service: ServiceFormData) => void | Promise<void>
  /** Existing service names for validation */
  existingServiceNames?: string[]
}

/**
 * Add Service Modal Component
 */
export function AddServiceModal({
  isOpen,
  onClose,
  onAddService,
  existingServiceNames = [],
}: AddServiceModalProps) {
  const [activeTab, setActiveTab] = React.useState('wellknown')
  const [selectedWellKnownService, setSelectedWellKnownService] = React.useState<WellKnownService | undefined>()
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  // Reset state when modal closes
  React.useEffect(() => {
    if (!isOpen) {
      setActiveTab('wellknown')
      setSelectedWellKnownService(undefined)
      setIsSubmitting(false)
    }
  }, [isOpen])

  // Handle adding well-known service
  const handleAddWellKnownService = async () => {
    if (!selectedWellKnownService) return

    // Check for duplicate service name
    if (existingServiceNames.includes(selectedWellKnownService.name)) {
      alert(`A service named "${selectedWellKnownService.name}" already exists. Please choose a different name.`)
      return
    }

    try {
      setIsSubmitting(true)

      const serviceData: ServiceFormData = {
        name: selectedWellKnownService.name,
        host: selectedWellKnownService.host,
        image: selectedWellKnownService.image,
        ports: selectedWellKnownService.ports,
        environment: selectedWellKnownService.environment,
        healthcheck: selectedWellKnownService.healthcheck,
      }

      await onAddService(serviceData)
      onClose()
    } catch (error) {
      alert('Failed to add service. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle adding application service
  const handleAddApplicationService = async (data: ServiceFormData) => {
    // Check for duplicate service name
    if (existingServiceNames.includes(data.name)) {
      alert(`A service named "${data.name}" already exists. Please choose a different name.`)
      return
    }

    try {
      setIsSubmitting(true)
      await onAddService(data)
      onClose()
    } catch (error) {
      alert('Failed to add service. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle adding container service
  const handleAddContainerService = async (data: ServiceFormData) => {
    // Check for duplicate service name
    if (existingServiceNames.includes(data.name)) {
      alert(`A service named "${data.name}" already exists. Please choose a different name.`)
      return
    }

    try {
      setIsSubmitting(true)
      await onAddService(data)
      onClose()
    } catch (error) {
      alert('Failed to add service. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="4xl">
      <DialogHeader onClose={onClose}>
        <DialogTitle>Add Service</DialogTitle>
        <DialogDescription>
          Choose a service type and configure your service settings
        </DialogDescription>
      </DialogHeader>

      <DialogContent className="p-0">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <div className="px-6 pt-4 border-b border-slate-200 dark:border-slate-700">
            <TabsList className="w-full justify-start">
              <TabsTrigger value="wellknown">Well-Known Services</TabsTrigger>
              <TabsTrigger value="application">Application Service</TabsTrigger>
              <TabsTrigger value="container">Container Service</TabsTrigger>
            </TabsList>
          </div>

          <div className="px-6 py-4">
            <TabsContent value="wellknown" className="mt-0">
              <WellKnownServicesTab
                onSelectService={setSelectedWellKnownService}
                selectedService={selectedWellKnownService}
              />
            </TabsContent>

            <TabsContent value="application" className="mt-0">
              <ApplicationServiceTab
                onSubmit={handleAddApplicationService}
                isSubmitting={isSubmitting}
              />
            </TabsContent>

            <TabsContent value="container" className="mt-0">
              <ContainerServiceTab
                onSubmit={handleAddContainerService}
                isSubmitting={isSubmitting}
              />
            </TabsContent>
          </div>
        </Tabs>
      </DialogContent>

      {/* Footer only shows for well-known services tab */}
      {activeTab === 'wellknown' && (
        <DialogFooter>
          <div className="flex items-center justify-between w-full gap-3">
            <p className="text-sm text-slate-600 dark:text-slate-400">
              {selectedWellKnownService
                ? `Selected: ${selectedWellKnownService.displayName}`
                : 'Select a service to add'}
            </p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className={cn(
                  'px-4 py-2 rounded-lg text-sm font-semibold',
                  'text-slate-700 dark:text-slate-300',
                  'border border-slate-200 dark:border-slate-700',
                  'hover:bg-slate-100 dark:hover:bg-slate-800',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                  'transition-colors duration-150'
                )}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleAddWellKnownService()}
                disabled={!selectedWellKnownService || isSubmitting}
                className={cn(
                  'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                  'bg-cyan-600 text-white hover:bg-cyan-700',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                  'disabled:opacity-50 disabled:cursor-not-allowed',
                  'transition-colors duration-150'
                )}
              >
                {isSubmitting ? 'Adding Service...' : 'Add Service'}
              </button>
            </div>
          </div>
        </DialogFooter>
      )}
    </Dialog>
  )
}
