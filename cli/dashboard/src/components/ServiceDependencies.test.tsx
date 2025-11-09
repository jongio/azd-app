import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ServiceDependencies } from './ServiceDependencies'
import type { Service } from '@/types'

describe('ServiceDependencies', () => {
  const mockServices: Service[] = [
    {
      name: 'api',
      language: 'TypeScript',
      status: 'running',
      health: 'healthy',
      local: {
        status: 'running',
        health: 'healthy',
        port: 3000,
        url: 'http://localhost:3000',
      },
      environmentVariables: {},
    },
    {
      name: 'worker',
      language: 'TypeScript',
      status: 'running',
      health: 'healthy',
      local: {
        status: 'running',
        health: 'healthy',
        port: 3001,
        url: 'http://localhost:3001',
      },
      environmentVariables: {},
    },
    {
      name: 'database',
      language: 'PostgreSQL',
      status: 'running',
      health: 'healthy',
      local: {
        status: 'running',
        health: 'healthy',
        port: 5432,
      },
      environmentVariables: {},
    },
    {
      name: 'redis',
      language: 'Redis',
      status: 'starting',
      health: 'unknown',
      local: {
        status: 'starting',
        health: 'unknown',
        port: 6379,
      },
      environmentVariables: {},
    },
  ]

  it('should render info banner', () => {
    render(<ServiceDependencies services={mockServices} />)
    expect(screen.getByText('Service Dependencies')).toBeInTheDocument()
    expect(screen.getByText(/Visual representation of your service architecture/)).toBeInTheDocument()
  })

  it('should group services by language', () => {
    render(<ServiceDependencies services={mockServices} />)
    expect(screen.getAllByText('TypeScript').length).toBeGreaterThan(0)
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument()
    expect(screen.getByText('Redis')).toBeInTheDocument()
  })

  it('should display all services', () => {
    render(<ServiceDependencies services={mockServices} />)
    expect(screen.getAllByText('api').length).toBeGreaterThan(0)
    expect(screen.getAllByText('worker').length).toBeGreaterThan(0)
    expect(screen.getAllByText('database').length).toBeGreaterThan(0)
    expect(screen.getAllByText('redis').length).toBeGreaterThan(0)
  })

  it('should display service ports', () => {
    render(<ServiceDependencies services={mockServices} />)
    expect(screen.getAllByText(/3000/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/3001/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/5432/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/6379/).length).toBeGreaterThan(0)
  })

  it('should show correct status colors for healthy services', () => {
    const { container } = render(<ServiceDependencies services={mockServices} />)
    const successIndicators = container.querySelectorAll('.bg-success')
    expect(successIndicators.length).toBeGreaterThan(0)
  })

  it('should show correct status colors for starting services', () => {
    const { container } = render(<ServiceDependencies services={mockServices} />)
    const warningIndicators = container.querySelectorAll('.bg-warning')
    expect(warningIndicators.length).toBeGreaterThan(0)
  })

  it('should handle empty services array', () => {
    render(<ServiceDependencies services={[]} />)
    expect(screen.getByText('Service Dependencies')).toBeInTheDocument()
  })

  it('should handle services without language', () => {
    const servicesWithoutLanguage: Service[] = [
      {
        name: 'unknown-service',
        status: 'running',
        health: 'healthy',
        environmentVariables: {},
      },
    ]
    render(<ServiceDependencies services={servicesWithoutLanguage} />)
    expect(screen.getByText('Unknown')).toBeInTheDocument()
    expect(screen.getByText('unknown-service')).toBeInTheDocument()
  })

  it('should display service framework when available', () => {
    const servicesWithFramework: Service[] = [
      {
        name: 'api',
        language: 'TypeScript',
        framework: 'Express',
        status: 'running',
        health: 'healthy',
        environmentVariables: {},
      },
    ]
    render(<ServiceDependencies services={servicesWithFramework} />)
    expect(screen.getByText('Express')).toBeInTheDocument()
  })

  it('should count environment variables per service', () => {
    const servicesWithEnv: Service[] = [
      {
        name: 'api',
        language: 'TypeScript',
        status: 'running',
        health: 'healthy',
        environmentVariables: {
          VAR1: 'value1',
          VAR2: 'value2',
          VAR3: 'value3',
        },
      },
    ]
    render(<ServiceDependencies services={servicesWithEnv} />)
    expect(screen.getByText(/3 env vars/)).toBeInTheDocument()
  })

  it('should show unhealthy status correctly', () => {
    const unhealthyServices: Service[] = [
      {
        name: 'broken-service',
        language: 'Node.js',
        status: 'error',
        health: 'unhealthy',
        local: {
          status: 'error',
          health: 'unhealthy',
        },
        environmentVariables: {},
      },
    ]
    const { container } = render(<ServiceDependencies services={unhealthyServices} />)
    const errorIndicators = container.querySelectorAll('.bg-destructive')
    expect(errorIndicators.length).toBeGreaterThan(0)
  })

  it('should group multiple services under same language', () => {
    const multipleTypeScriptServices: Service[] = [
      { name: 'api', language: 'TypeScript', status: 'running', health: 'healthy', environmentVariables: {} },
      { name: 'worker', language: 'TypeScript', status: 'running', health: 'healthy', environmentVariables: {} },
      { name: 'scheduler', language: 'TypeScript', status: 'running', health: 'healthy', environmentVariables: {} },
    ]
    render(<ServiceDependencies services={multipleTypeScriptServices} />)
    
    // All three should be under one TypeScript heading
    const typeScriptHeadings = screen.getAllByText('TypeScript')
    expect(typeScriptHeadings).toHaveLength(1)
    expect(screen.getAllByText('api').length).toBeGreaterThan(0)
    expect(screen.getAllByText('worker').length).toBeGreaterThan(0)
    expect(screen.getAllByText('scheduler').length).toBeGreaterThan(0)
  })
})
