/**
 * Preload functions for lazy-loaded modals
 * Separated from component file for Fast Refresh compatibility
 */

import { preloadableLazy, ModalLoadingSkeleton } from '@/lib/performance'
import { createElement } from 'react'

// Preloadable modals (with hover preloading)
export const [PreloadableAddServiceModal, preloadAddServiceModal] = preloadableLazy(
  () => import('../components/editor/modals/AddServiceModal').then(m => ({ default: m.AddServiceModal })),
  createElement(ModalLoadingSkeleton)
)

export const [PreloadableResourceConfigModal, preloadResourceConfigModal] = preloadableLazy(
  () => import('../components/editor/modals/ResourceConfigModal').then(m => ({ default: m.ResourceConfigModal })),
  createElement(ModalLoadingSkeleton)
)

export const [PreloadableHealthCheckModal, preloadHealthCheckModal] = preloadableLazy(
  () => import('../components/editor/modals/HealthCheckModal').then(m => ({ default: m.HealthCheckModal })),
  createElement(ModalLoadingSkeleton)
)

// Export preload functions for hover events
export const preloadModals = {
  addService: preloadAddServiceModal,
  resourceConfig: preloadResourceConfigModal,
  healthCheck: preloadHealthCheckModal,
}
