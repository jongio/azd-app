/**
 * Lazy-loaded Modal Components
 * Code-split large modals to reduce initial bundle size
 */

import { lazyLoad, preloadableLazy, ModalLoadingSkeleton } from '@/lib/performance'

// Lazy load modals with loading skeleton
export const LazyAddServiceModal = lazyLoad(
  () => import('./modals/AddServiceModal').then(m => ({ default: m.AddServiceModal })),
  <ModalLoadingSkeleton />
)

export const LazyResourceConfigModal = lazyLoad(
  () => import('./modals/ResourceConfigModal').then(m => ({ default: m.ResourceConfigModal })),
  <ModalLoadingSkeleton />
)

export const LazyHealthCheckModal = lazyLoad(
  () => import('./modals/HealthCheckModal').then(m => ({ default: m.HealthCheckModal })),
  <ModalLoadingSkeleton />
)

export const LazyBackupManager = lazyLoad(
  () => import('./BackupManager').then(m => ({ default: m.BackupManager })),
  <ModalLoadingSkeleton />
)

// Preloadable modals (with hover preloading)
export const [PreloadableAddServiceModal, preloadAddServiceModal] = preloadableLazy(
  () => import('./modals/AddServiceModal').then(m => ({ default: m.AddServiceModal })),
  <ModalLoadingSkeleton />
)

export const [PreloadableResourceConfigModal, preloadResourceConfigModal] = preloadableLazy(
  () => import('./modals/ResourceConfigModal').then(m => ({ default: m.ResourceConfigModal })),
  <ModalLoadingSkeleton />
)

export const [PreloadableHealthCheckModal, preloadHealthCheckModal] = preloadableLazy(
  () => import('./modals/HealthCheckModal').then(m => ({ default: m.HealthCheckModal })),
  <ModalLoadingSkeleton />
)

// Export preload functions for hover events
export const preloadModals = {
  addService: preloadAddServiceModal,
  resourceConfig: preloadResourceConfigModal,
  healthCheck: preloadHealthCheckModal,
}
