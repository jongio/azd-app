import type { Service } from '@/types'

function serviceSearchText(service: Service): string {
  return [
    service.name,
    service.language,
    service.framework,
    service.project,
    service.local?.url,
    service.local?.customUrl,
    service.azure?.url,
    service.azure?.customUrl,
    service.azure?.customDomain,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}

export function filterServicesByQuery(services: Service[], query: string): Service[] {
  const terms = query
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)

  if (terms.length === 0) {
    return services
  }

  return services.filter((service) => {
    const searchText = serviceSearchText(service)
    return terms.every((term) => searchText.includes(term))
  })
}
