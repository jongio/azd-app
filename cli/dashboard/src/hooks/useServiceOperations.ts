/**
 * This file re-exports from the ServiceOperationsContext for backward compatibility.
 * The actual implementation is now in src/contexts/ServiceOperationsContext.tsx
 * to enable shared state across all components.
 */
export {
  useServiceOperations,
  ServiceOperationsProvider,
  type ServiceOperation,
  type OperationState,
  type ServiceOperationResult,
  type BulkOperationResult,
  type ServiceOperationsContextValue,
} from '@/contexts/ServiceOperationsContext'
