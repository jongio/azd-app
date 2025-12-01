/**
 * ServiceDependencies - Visualizes services grouped by language/technology
 */
import * as React from 'react'
import { ExternalLink, Search, Filter, X } from 'lucide-react'
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
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'

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

      {/* Port - only show if port > 0 */}
      {service.local?.port && service.local.port > 0 && (
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
  // Search and filter state
  const [searchQuery, setSearchQuery] = React.useState('')
  const [selectedLanguages, setSelectedLanguages] = React.useState<Set<string>>(new Set())

  // Get all available languages for filter dropdown
  const availableLanguages = React.useMemo(() => {
    const languages = new Set<string>()
    services.forEach(service => {
      const lang = service.language || 'Other'
      languages.add(lang)
    })
    return Array.from(languages).sort()
  }, [services])

  // Filter services based on search query and selected languages
  const filteredServices = React.useMemo(() => {
    return services.filter(service => {
      // Search filter - match name, framework, or language
      if (searchQuery) {
        const query = searchQuery.toLowerCase()
        const matchesName = service.name.toLowerCase().includes(query)
        const matchesFramework = service.framework?.toLowerCase().includes(query)
        const matchesLanguage = service.language?.toLowerCase().includes(query)
        if (!matchesName && !matchesFramework && !matchesLanguage) {
          return false
        }
      }

      // Language filter
      if (selectedLanguages.size > 0) {
        const lang = service.language || 'Other'
        if (!selectedLanguages.has(lang)) {
          return false
        }
      }

      return true
    })
  }, [services, searchQuery, selectedLanguages])

  // Group and sort filtered services
  const groupedServices = React.useMemo(
    () => groupServicesByLanguage(filteredServices),
    [filteredServices]
  )

  const sortedGroups = React.useMemo(
    () => sortGroupsBySize(groupedServices),
    [groupedServices]
  )

  // Toggle language filter
  const toggleLanguage = (language: string) => {
    setSelectedLanguages(prev => {
      const next = new Set(prev)
      if (next.has(language)) {
        next.delete(language)
      } else {
        next.add(language)
      }
      return next
    })
  }

  // Clear all filters
  const clearFilters = () => {
    setSearchQuery('')
    setSelectedLanguages(new Set())
  }

  const hasActiveFilters = searchQuery.length > 0 || selectedLanguages.size > 0

  // Handle empty state (no services at all)
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

      {/* Search and Filter Bar */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search
            className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none"
            aria-hidden="true"
          />
          <Input
            type="search"
            placeholder="Search services..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
            aria-label="Search services"
            data-testid="service-search-input"
          />
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className={cn(
                'gap-2',
                selectedLanguages.size > 0 && 'border-primary text-primary'
              )}
              data-testid="language-filter-button"
            >
              <Filter className="h-4 w-4" />
              {selectedLanguages.size > 0 ? (
                <span>{selectedLanguages.size} selected</span>
              ) : (
                <span className="sr-only md:not-sr-only">Filter</span>
              )}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            {availableLanguages.map(language => (
              <DropdownMenuCheckboxItem
                key={language}
                checked={selectedLanguages.has(language)}
                onCheckedChange={() => toggleLanguage(language)}
              >
                {language}
              </DropdownMenuCheckboxItem>
            ))}
            {selectedLanguages.size > 0 && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuCheckboxItem
                  checked={false}
                  onCheckedChange={() => setSelectedLanguages(new Set())}
                  className="text-muted-foreground"
                >
                  Clear all
                </DropdownMenuCheckboxItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="gap-1 text-muted-foreground hover:text-foreground"
            data-testid="clear-filters-button"
          >
            <X className="h-4 w-4" />
            Clear
          </Button>
        )}
      </div>

      {/* Results count */}
      {hasActiveFilters && (
        <div
          className="text-sm text-muted-foreground"
          aria-live="polite"
          aria-atomic="true"
        >
          Showing {filteredServices.length} of {services.length} {pluralize(services.length, 'service')}
        </div>
      )}

      {/* Empty filtered state */}
      {filteredServices.length === 0 && hasActiveFilters && (
        <div className="flex flex-col items-center justify-center py-12">
          <p className="text-muted-foreground mb-4">No services match your filters</p>
          <Button variant="outline" size="sm" onClick={clearFilters}>
            Clear filters
          </Button>
        </div>
      )}

      {/* Service groups */}
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
