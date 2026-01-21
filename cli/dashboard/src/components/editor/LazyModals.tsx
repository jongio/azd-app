/**
 * Lazy-loaded Modal Components
 * Code-split large modals to reduce initial bundle size
 */

/* eslint-disable react-refresh/only-export-components */

import { lazyLoad, ModalLoadingSkeleton } from '@/lib/performance'

// Lazy load modals with loading skeleton
const LazyAddServiceModal = lazyLoad(
  () => import('./modals/AddServiceModal').then(m => ({ default: m.AddServiceModal })),
  <ModalLoadingSkeleton />
)

const LazyResourceConfigModal = lazyLoad(
  () => import('./modals/ResourceConfigModal').then(m => ({ default: m.ResourceConfigModal })),
  <ModalLoadingSkeleton />
)

const LazyHealthCheckModal = lazyLoad(
  () => import('./modals/HealthCheckModal').then(m => ({ default: m.HealthCheckModal })),
  <ModalLoadingSkeleton />
)

const LazyBackupManager = lazyLoad(
  () => import('./BackupManager').then(m => ({ default: m.BackupManager })),
  <ModalLoadingSkeleton />
)

// Named exports to satisfy Fast Refresh
export {
  LazyAddServiceModal,
  LazyResourceConfigModal,
  LazyHealthCheckModal,
  LazyBackupManager
}

// Re-export preloadable modals and utilities from separate file
export {
  PreloadableAddServiceModal,
  PreloadableResourceConfigModal,
  PreloadableHealthCheckModal,
  preloadModals
} from '@/lib/editor/lazy-modal-preload'
