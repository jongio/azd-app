/**
 * Re-export useServices from ServicesContext for backward compatibility.
 * New components should use useServicesContext() directly for access to additional helpers.
 * @deprecated Import from '@/contexts/ServicesContext' instead
 */
export { useServices, useServicesContext } from '@/contexts/ServicesContext'
