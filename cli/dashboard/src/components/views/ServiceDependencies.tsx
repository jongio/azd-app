/**
 * ServiceDependencies - Visualizes services grouped by language/technology
 */
import * as React from 'react'
import { ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Service } from '@/types'
import {
  groupServicesByLanguage,
  getLanguageBadgeStyle,
  getStatusIndicator,
  countEnvVars,
  sortGroupsBySize,
  getServiceUrl,
  pluralize,
} from '@/lib/dependencies-utils'

// =============================================================================
// ServiceDependencyCard
// =============================================================================

interface ServiceDependencyCardProps {
  service: Service
  onClick?: () => void
}

function ServiceDependencyCard({ service, onClick }: ServiceDependencyCardProps) {
  const status = getStatusIndicator(service.local?.status || service.status)
  const envCount = countEnvVars(service)
  const url = getServiceUrl(service)

  const handleUrlClick = (e: React.MouseEvent) => {
    e.stopPropagation()
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex flex-col items-start gap-1 p-4 text-left rounded-md border border-border',
        'min-w-[200px] w-full',
        'hover:bg-accent hover:scale-[1.02] transition-all duration-200',
        'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2'
      )}
      aria-label={`${service.name} service - ${service.local?.status || service.status || 'not-running'} - ${service.framework || 'Unknown'} on port ${service.local?.port || 'N/A'}`}
      data-testid={`service-card-${service.name}`}
    >
      {/* Service name with status indicator */}
      <div className="flex items-center gap-2">
        <span
          className={cn(status.color, status.animate)}
          aria-hidden="true"
        >
          {status.icon}
        </span>
        <span className="font-medium text-foreground">{service.name}</span>
      </div>

      {/* Framework */}
      {service.framework && (
        <span className="text-sm text-muted-foreground">{service.framework}</span>
      )}

      {/* Port */}
      {service.local?.port && (
        <span className="text-sm text-muted-foreground">:{service.local.port}</span>
      )}

      {/* Environment variables count */}
      <span className="text-xs text-muted-foreground">
        {envCount} env {pluralize(envCount, 'var')}
      </span>

      {/* URL link */}
      {url && (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={handleUrlClick}
          className={cn(
            'flex items-center gap-1 text-xs text-primary hover:underline',
            'focus:outline-none focus:ring-2 focus:ring-ring'
          )}
          aria-label={`Open ${service.name} at ${url}`}
        >
          {url}
          <ExternalLink className="h-3 w-3" aria-hidden="true" />
        </a>
      )}
    </button>
  )
}

// =============================================================================
// LanguageGroup
// =============================================================================

interface LanguageGroupProps {
  language: string
  services: Service[]
  onServiceClick?: (service: Service) => void
}

function LanguageGroup({ language, services, onServiceClick }: LanguageGroupProps) {
  const badgeStyle = getLanguageBadgeStyle(language)
  const groupId = `group-${language.toLowerCase().replace(/[^a-z0-9]/g, '-')}`

  return (
    <section
      aria-labelledby={groupId}
      className="p-4 rounded-lg border border-border bg-card"
      data-testid={`language-group-${language}`}
    >
      {/* Group header */}
      <div className="flex items-center gap-3 mb-4">
        <span
          className={cn(
            'px-2 py-1 text-xs font-semibold rounded-md',
            badgeStyle.bg,
            badgeStyle.text
          )}
          aria-hidden="true"
        >
          {badgeStyle.abbr}
        </span>
        <h3
          id={groupId}
          className="text-base font-semibold text-foreground"
        >
          {language}
        </h3>
        <span className="text-sm text-muted-foreground">
          ({services.length} {pluralize(services.length, 'service')})
        </span>
      </div>

      {/* Services grid */}
      <div
        role="list"
        className="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        {services.map((service) => (
          <div key={service.name} role="listitem">
            <ServiceDependencyCard
              service={service}
              onClick={onServiceClick ? () => onServiceClick(service) : undefined}
            />
          </div>
        ))}
      </div>
    </section>
  )
}

// =============================================================================
// ServiceDependencies
// =============================================================================

export interface ServiceDependenciesProps {
  services: Service[]
  onServiceClick?: (service: Service) => void
  className?: string
  'data-testid'?: string
}

export function ServiceDependencies({
  services,
  onServiceClick,
  className,
  'data-testid': testId = 'service-dependencies',
}: ServiceDependenciesProps) {
  // Group and sort services
  const groupedServices = React.useMemo(
    () => groupServicesByLanguage(services),
    [services]
  )

  const sortedGroups = React.useMemo(
    () => sortGroupsBySize(groupedServices),
    [groupedServices]
  )

  // Handle empty state
  if (services.length === 0) {
    return (
      <div
        data-testid={testId}
        className={cn('flex flex-col items-center justify-center py-12', className)}
      >
        <p className="text-muted-foreground">No services found</p>
      </div>
    )
  }

  return (
    <section
      aria-labelledby="dependencies-title"
      data-testid={testId}
      className={cn('flex flex-col gap-6', className)}
    >
      <h2 id="dependencies-title" className="sr-only">
        Service Dependencies by Language
      </h2>

      {sortedGroups.map(([language, groupServices]) => (
        <LanguageGroup
          key={language}
          language={language}
          services={groupServices}
          onServiceClick={onServiceClick}
        />
      ))}
    </section>
  )
}
