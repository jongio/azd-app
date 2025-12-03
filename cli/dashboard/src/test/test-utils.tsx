import { render, type RenderOptions } from '@testing-library/react'
import { ServiceOperationsProvider } from '@/contexts/ServiceOperationsContext'
import type { ReactNode, ReactElement } from 'react'

/**
 * Custom render function that wraps components with all required providers.
 * Use this instead of the default render from @testing-library/react
 * when testing components that use hooks requiring context (e.g., useServiceOperations).
 */

interface AllProvidersProps {
  children: ReactNode
}

function AllProviders({ children }: AllProvidersProps) {
  return (
    <ServiceOperationsProvider>
      {children}
    </ServiceOperationsProvider>
  )
}

function customRender(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: AllProviders, ...options })
}

// Re-export everything from @testing-library/react
export * from '@testing-library/react'

// Override render with our custom render
export { customRender as render }
