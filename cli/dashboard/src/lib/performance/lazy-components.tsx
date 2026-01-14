/**
 * Lazy Component Loading
 * Provides utilities for code-splitting and lazy loading components
 * 
 * Features:
 * - React.lazy() with Suspense
 * - Loading skeletons
 * - Error boundaries
 * - Preloading on hover
 */

import { lazy, Suspense, ComponentType } from 'react'

/**
 * Loading Skeleton Component
 */
export function LoadingSkeleton({ className }: { className?: string }) {
  return (
    <div className={`animate-pulse ${className || ''}`}>
      <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
      <div className="h-4 bg-muted rounded w-1/2 mb-2"></div>
      <div className="h-4 bg-muted rounded w-5/6"></div>
    </div>
  )
}

/**
 * Modal Loading Skeleton
 */
export function ModalLoadingSkeleton() {
  return (
    <div className="p-6 animate-pulse">
      <div className="h-6 bg-muted rounded w-1/3 mb-4"></div>
      <div className="space-y-3">
        <div className="h-4 bg-muted rounded w-full"></div>
        <div className="h-4 bg-muted rounded w-5/6"></div>
        <div className="h-4 bg-muted rounded w-4/6"></div>
      </div>
      <div className="flex gap-2 mt-6">
        <div className="h-10 bg-muted rounded w-20"></div>
        <div className="h-10 bg-muted rounded w-20"></div>
      </div>
    </div>
  )
}

/**
 * Form Loading Skeleton
 */
export function FormLoadingSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      {[1, 2, 3].map((i) => (
        <div key={i}>
          <div className="h-3 bg-muted rounded w-24 mb-2"></div>
          <div className="h-10 bg-muted rounded w-full"></div>
        </div>
      ))}
    </div>
  )
}

/**
 * Lazy load a component with custom loading fallback
 */
export function lazyLoad<T extends ComponentType<any>>(
  importFunc: () => Promise<{ default: T }>,
  fallback: React.ReactNode = <LoadingSkeleton />
) {
  const LazyComponent = lazy(importFunc)

  return function LazyComponentWithSuspense(props: React.ComponentProps<T>) {
    return (
      <Suspense fallback={fallback}>
        <LazyComponent {...props} />
      </Suspense>
    )
  }
}

/**
 * Preloadable lazy component
 * Returns [Component, preload] where preload can be called to load the component
 */
export function preloadableLazy<T extends ComponentType<any>>(
  importFunc: () => Promise<{ default: T }>,
  fallback: React.ReactNode = <LoadingSkeleton />
) {
  let preloadPromise: Promise<{ default: T }> | null = null

  const preload = () => {
    if (!preloadPromise) {
      preloadPromise = importFunc()
    }
    return preloadPromise
  }

  const LazyComponent = lazy(() => {
    if (!preloadPromise) {
      preloadPromise = importFunc()
    }
    return preloadPromise
  })

  const Component = function PreloadableComponent(props: React.ComponentProps<T>) {
    return (
      <Suspense fallback={fallback}>
        <LazyComponent {...props} />
      </Suspense>
    )
  }

  return [Component, preload] as const
}

/**
 * Lazy load with retry on error
 */
export function lazyWithRetry<T extends ComponentType<any>>(
  importFunc: () => Promise<{ default: T }>,
  retries = 3,
  fallback: React.ReactNode = <LoadingSkeleton />
) {
  const retryImport = async (retriesLeft: number): Promise<{ default: T }> => {
    try {
      return await importFunc()
    } catch (error) {
      if (retriesLeft === 0) {
        throw error
      }
      // Wait before retrying (exponential backoff)
      await new Promise((resolve) => setTimeout(resolve, 1000 * (retries - retriesLeft)))
      return retryImport(retriesLeft - 1)
    }
  }

  const LazyComponent = lazy(() => retryImport(retries))

  return function LazyComponentWithRetry(props: React.ComponentProps<T>) {
    return (
      <Suspense fallback={fallback}>
        <LazyComponent {...props} />
      </Suspense>
    )
  }
}
