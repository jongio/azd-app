/**
 * Well-Known Services Tab
 * Displays a grid of well-known services that can be added with one click
 */

import * as React from 'react'
import { Loader2, ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import { fetchWellKnownServices } from '@/lib/api/wellknown'
import type { WellKnownService } from '@/lib/editor/wellknown-types'

export interface WellKnownServicesTabProps {
  /** Callback when service is selected */
  onSelectService: (service: WellKnownService) => void
  /** Currently selected service for preview */
  selectedService?: WellKnownService
}

/**
 * Service Card Component
 */
interface ServiceCardProps {
  service: WellKnownService
  isSelected: boolean
  onClick: () => void
}

function ServiceCard({ service, isSelected, onClick }: ServiceCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'group relative flex flex-col items-start gap-3 p-4 rounded-lg border-2 transition-all text-left',
        'hover:shadow-md hover:border-cyan-500',
        'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
        isSelected
          ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-950/30'
          : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800'
      )}
    >
      {/* Icon & Category Badge */}
      <div className="flex items-start justify-between w-full">
        <div className="text-3xl">{service.icon || '📦'}</div>
        <span
          className={cn(
            'text-xs font-medium px-2 py-0.5 rounded-full',
            'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300'
          )}
        >
          {service.category}
        </span>
      </div>

      {/* Name */}
      <div>
        <h3 className="font-semibold text-sm text-slate-900 dark:text-slate-100 mb-1">
          {service.displayName}
        </h3>
        <p className="text-xs text-slate-600 dark:text-slate-400 line-clamp-2">
          {service.description}
        </p>
      </div>

      {/* Tags */}
      <div className="flex flex-wrap gap-1 w-full">
        <span className="text-xs px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300">
          {service.host}
        </span>
        {service.ports && service.ports.length > 0 && (
          <span className="text-xs px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300">
            {service.ports.length} {service.ports.length === 1 ? 'port' : 'ports'}
          </span>
        )}
      </div>

      {/* Selection indicator */}
      {isSelected && (
        <div className="absolute top-2 right-2 w-5 h-5 rounded-full bg-cyan-500 text-white flex items-center justify-center">
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
          </svg>
        </div>
      )}
    </button>
  )
}

/**
 * Service Preview Panel
 */
interface ServicePreviewProps {
  service: WellKnownService
}

function ServicePreview({ service }: ServicePreviewProps) {
  return (
    <div className="space-y-4">
      <div>
        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-2">
          Configuration Preview
        </h4>
        <div className="bg-slate-50 dark:bg-slate-800 rounded-lg p-4 space-y-2 text-sm">
          <div>
            <span className="text-slate-600 dark:text-slate-400">Name:</span>{' '}
            <code className="text-slate-900 dark:text-slate-100">{service.name}</code>
          </div>
          <div>
            <span className="text-slate-600 dark:text-slate-400">Host:</span>{' '}
            <code className="text-slate-900 dark:text-slate-100">{service.host}</code>
          </div>
          <div>
            <span className="text-slate-600 dark:text-slate-400">Image:</span>{' '}
            <code className="text-slate-900 dark:text-slate-100 text-xs">{service.image}</code>
          </div>
          {service.ports && service.ports.length > 0 && (
            <div>
              <span className="text-slate-600 dark:text-slate-400">Ports:</span>{' '}
              <code className="text-slate-900 dark:text-slate-100">{service.ports.join(', ')}</code>
            </div>
          )}
          {service.environment && Object.keys(service.environment).length > 0 && (
            <div>
              <span className="text-slate-600 dark:text-slate-400">Environment:</span>
              <div className="ml-4 mt-1 space-y-1">
                {Object.entries(service.environment).map(([key, value]) => (
                  <div key={key} className="text-xs">
                    <code className="text-slate-900 dark:text-slate-100">
                      {key}={value}
                    </code>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {service.connectionStrings && (
        <div>
          <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-2">
            Connection Strings
          </h4>
          <div className="bg-slate-50 dark:bg-slate-800 rounded-lg p-4 space-y-2 text-sm">
            {Object.entries(service.connectionStrings).map(([key, value]) => (
              <div key={key}>
                <span className="text-slate-600 dark:text-slate-400 capitalize">{key}:</span>
                <code className="block mt-1 text-xs text-slate-900 dark:text-slate-100 break-all">
                  {value}
                </code>
              </div>
            ))}
          </div>
        </div>
      )}

      {service.docsUrl && (
        <a
          href={service.docsUrl}
          target="_blank"
          rel="noopener noreferrer"
          className={cn(
            'inline-flex items-center gap-2 text-sm text-cyan-600 dark:text-cyan-400 hover:underline',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 rounded'
          )}
        >
          View Documentation
          <ExternalLink className="w-3.5 h-3.5" />
        </a>
      )}
    </div>
  )
}

/**
 * Well-Known Services Tab Component
 */
export function WellKnownServicesTab({ onSelectService, selectedService }: WellKnownServicesTabProps) {
  const [services, setServices] = React.useState<WellKnownService[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [categoryFilter, setCategoryFilter] = React.useState<string>('all')

  // Fetch well-known services on mount
  React.useEffect(() => {
    let mounted = true

    const loadServices = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await fetchWellKnownServices()
        if (mounted) {
          setServices(data)
        }
      } catch (err) {
        if (mounted) {
          setError(err instanceof Error ? err.message : 'Failed to load services')
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }

    void loadServices()

    return () => {
      mounted = false
    }
  }, [])

  // Get unique categories
  const categories = React.useMemo(() => {
    const cats = new Set(services.map(s => s.category))
    return ['all', ...Array.from(cats)]
  }, [services])

  // Filter services by category
  const filteredServices = React.useMemo(() => {
    if (categoryFilter === 'all') {
      return services
    }
    return services.filter(s => s.category === categoryFilter)
  }, [services, categoryFilter])

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-cyan-500 mb-3" />
        <p className="text-sm text-slate-600 dark:text-slate-400">Loading services...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="text-red-500 mb-2">⚠️</div>
        <p className="text-sm text-slate-600 dark:text-slate-400">{error}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Category Filter */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">Category:</span>
        {categories.map((category) => (
          <button
            key={category}
            onClick={() => setCategoryFilter(category)}
            className={cn(
              'px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              categoryFilter === category
                ? 'bg-cyan-500 text-white'
                : 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
            )}
          >
            {category}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Services Grid */}
        <div className="space-y-3 max-h-[400px] overflow-y-auto">
          {filteredServices.length === 0 ? (
            <div className="text-center py-8 text-sm text-slate-600 dark:text-slate-400">
              No services found in this category
            </div>
          ) : (
            filteredServices.map((service) => (
              <ServiceCard
                key={service.name}
                service={service}
                isSelected={selectedService?.name === service.name}
                onClick={() => onSelectService(service)}
              />
            ))
          )}
        </div>

        {/* Preview Panel */}
        <div className="border-l border-slate-200 dark:border-slate-700 pl-4">
          {selectedService ? (
            <ServicePreview service={selectedService} />
          ) : (
            <div className="flex items-center justify-center h-full text-sm text-slate-600 dark:text-slate-400">
              Select a service to see configuration preview
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
