import { describe, expect, it } from 'vitest'
import { createHttpService, createProcessService } from '@/test/fixtures'
import { filterServicesByQuery } from './service-filter'

describe('filterServicesByQuery', () => {
  const services = [
    createHttpService({
      name: 'web',
      language: 'TypeScript',
      framework: 'React',
      project: 'apps/web',
      port: 3000,
    }),
    createHttpService({
      name: 'api',
      language: 'Python',
      framework: 'FastAPI',
      project: 'src/api',
      port: 8000,
      azure: { url: 'https://api.contoso.example' },
    }),
    createProcessService({
      name: 'worker',
      language: 'Go',
      framework: 'Worker',
      project: 'services/worker',
    }),
  ]

  it('returns all services for an empty query', () => {
    expect(filterServicesByQuery(services, '')).toEqual(services)
    expect(filterServicesByQuery(services, '   ')).toEqual(services)
  })

  it('matches service name, language, framework, project, and URLs', () => {
    expect(filterServicesByQuery(services, 'web')).toEqual([services[0]])
    expect(filterServicesByQuery(services, 'python')).toEqual([services[1]])
    expect(filterServicesByQuery(services, 'react')).toEqual([services[0]])
    expect(filterServicesByQuery(services, 'services/worker')).toEqual([services[2]])
    expect(filterServicesByQuery(services, 'contoso')).toEqual([services[1]])
  })

  it('requires every search term to match the same service', () => {
    expect(filterServicesByQuery(services, 'api fastapi')).toEqual([services[1]])
    expect(filterServicesByQuery(services, 'api react')).toEqual([])
  })
})
