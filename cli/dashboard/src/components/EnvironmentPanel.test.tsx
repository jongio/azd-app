import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EnvironmentPanel } from './EnvironmentPanel'
import type { Service } from '@/types'

const mockServices: Service[] = [
  {
    name: 'api',
    framework: 'Flask',
    language: 'Python',
    local: {
      status: 'ready',
      health: 'healthy',
      url: 'http://localhost:5000',
      port: 5000,
    },
    environmentVariables: {
      'DATABASE_URL': 'postgresql://localhost/db',
      'API_KEY': 'secret-key-123',
      'DEBUG': 'true',
    },
  },
  {
    name: 'web',
    framework: 'Express',
    language: 'Node.js',
    local: {
      status: 'ready',
      health: 'healthy',
      url: 'http://localhost:3000',
      port: 3000,
    },
    environmentVariables: {
      'NODE_ENV': 'development',
      'API_KEY': 'secret-key-123',
      'PORT': '3000',
    },
  },
]

describe('EnvironmentPanel', () => {
  it('should render environment variables from services', () => {
    render(<EnvironmentPanel services={mockServices} />)
    
    expect(screen.getByText('DATABASE_URL')).toBeInTheDocument()
    expect(screen.getByText('API_KEY')).toBeInTheDocument()
    expect(screen.getByText('DEBUG')).toBeInTheDocument()
    expect(screen.getByText('NODE_ENV')).toBeInTheDocument()
    expect(screen.getByText('PORT')).toBeInTheDocument()
  })

  it('should show service filter dropdown', () => {
    render(<EnvironmentPanel services={mockServices} />)
    
    const select = screen.getByRole('combobox')
    expect(select).toBeInTheDocument()
  })

  it('should show search input', () => {
    render(<EnvironmentPanel services={mockServices} />)
    
    const searchInput = screen.getByPlaceholderText('Search environment variables...')
    expect(searchInput).toBeInTheDocument()
  })

  it('should show summary with variable count', () => {
    render(<EnvironmentPanel services={mockServices} />)
    
    expect(screen.getByText(/Showing \d+ of \d+ variables/)).toBeInTheDocument()
  })

  it('should display service names for each variable', () => {
    render(<EnvironmentPanel services={mockServices} />)
    
    // API_KEY is shared between both services
    const apiServiceTags = screen.getAllByText('api')
    expect(apiServiceTags.length).toBeGreaterThan(0)
    
    const webServiceTags = screen.getAllByText('web')
    expect(webServiceTags.length).toBeGreaterThan(0)
  })

  it('should render empty state when no services', () => {
    render(<EnvironmentPanel services={[]} />)
    
    expect(screen.getByText('No environment variables available')).toBeInTheDocument()
  })
})
