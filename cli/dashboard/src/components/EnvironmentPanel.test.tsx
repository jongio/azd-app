import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  EnvironmentPanel,
  isSensitiveVariable,
  aggregateEnvironmentVariables,
  filterEnvironmentVariables,
} from './EnvironmentPanel'
import type { Service } from '@/types'

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  vi.useRealTimers()
})

// Test data
const createMockServices = (): Service[] => [
  {
    name: 'api',
    language: 'javascript',
    environmentVariables: {
      DATABASE_URL: 'postgres://localhost:5432/db',
      API_KEY: 'sk-secret-123',
      NODE_ENV: 'development',
      PORT: '3000',
    },
  },
  {
    name: 'web',
    language: 'typescript',
    environmentVariables: {
      NODE_ENV: 'development',
      NEXT_PUBLIC_API_URL: 'http://localhost:3000',
    },
  },
  {
    name: 'worker',
    language: 'python',
    environmentVariables: {
      DATABASE_URL: 'postgres://localhost:5432/db',
      SECRET_TOKEN: 'token-abc-123',
    },
  },
]

// =============================================================================
// Unit Tests for Helper Functions
// =============================================================================

describe('isSensitiveVariable', () => {
  it('detects sensitive variable names', () => {
    expect(isSensitiveVariable('API_KEY')).toBe(true)
    expect(isSensitiveVariable('api_key')).toBe(true)
    expect(isSensitiveVariable('SECRET_TOKEN')).toBe(true)
    expect(isSensitiveVariable('DATABASE_PASSWORD')).toBe(true)
    expect(isSensitiveVariable('AUTH_TOKEN')).toBe(true)
    expect(isSensitiveVariable('PRIVATE_KEY')).toBe(true)
    expect(isSensitiveVariable('CONNECTION_STRING')).toBe(true)
    expect(isSensitiveVariable('MY_CREDENTIAL')).toBe(true)
  })

  it('returns false for non-sensitive variable names', () => {
    expect(isSensitiveVariable('NODE_ENV')).toBe(false)
    expect(isSensitiveVariable('PORT')).toBe(false)
    expect(isSensitiveVariable('DATABASE_URL')).toBe(false)
    expect(isSensitiveVariable('LOG_LEVEL')).toBe(false)
    expect(isSensitiveVariable('HOST')).toBe(false)
  })
})

describe('aggregateEnvironmentVariables', () => {
  it('aggregates variables from multiple services', () => {
    const services = createMockServices()
    const result = aggregateEnvironmentVariables(services)

    // Should have 6 unique variables:
    // API_KEY, DATABASE_URL, NEXT_PUBLIC_API_URL, NODE_ENV, PORT, SECRET_TOKEN
    expect(result).toHaveLength(6)
  })

  it('groups same variable names across services', () => {
    const services = createMockServices()
    const result = aggregateEnvironmentVariables(services)

    const nodeEnv = result.find(v => v.name === 'NODE_ENV')
    expect(nodeEnv).toBeDefined()
    expect(nodeEnv?.services).toContain('api')
    expect(nodeEnv?.services).toContain('web')
    expect(nodeEnv?.services).toHaveLength(2)
  })

  it('sorts variables alphabetically by name', () => {
    const services = createMockServices()
    const result = aggregateEnvironmentVariables(services)

    const names = result.map(v => v.name)
    expect(names).toEqual([...names].sort())
  })

  it('correctly identifies sensitive variables', () => {
    const services = createMockServices()
    const result = aggregateEnvironmentVariables(services)

    const apiKey = result.find(v => v.name === 'API_KEY')
    const secretToken = result.find(v => v.name === 'SECRET_TOKEN')
    const nodeEnv = result.find(v => v.name === 'NODE_ENV')

    expect(apiKey?.isSensitive).toBe(true)
    expect(secretToken?.isSensitive).toBe(true)
    expect(nodeEnv?.isSensitive).toBe(false)
  })

  it('handles services with no environment variables', () => {
    const services: Service[] = [
      { name: 'empty-service' },
      ...createMockServices(),
    ]
    const result = aggregateEnvironmentVariables(services)

    // Should still work, just skip services without env vars
    expect(result).toHaveLength(6)
  })

  it('returns empty array for no services', () => {
    const result = aggregateEnvironmentVariables([])
    expect(result).toEqual([])
  })
})

describe('filterEnvironmentVariables', () => {
  const services = createMockServices()
  const variables = aggregateEnvironmentVariables(services)

  it('filters by search query (name match)', () => {
    const result = filterEnvironmentVariables(variables, 'DATABASE', null)
    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('DATABASE_URL')
  })

  it('filters by search query (value match)', () => {
    const result = filterEnvironmentVariables(variables, 'postgres', null)
    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('DATABASE_URL')
  })

  it('is case-insensitive', () => {
    const result = filterEnvironmentVariables(variables, 'node', null)
    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('NODE_ENV')
  })

  it('filters by service', () => {
    const result = filterEnvironmentVariables(variables, '', 'api')
    expect(result.every(v => v.services.includes('api'))).toBe(true)
  })

  it('combines search and service filters', () => {
    const result = filterEnvironmentVariables(variables, 'NODE', 'web')
    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('NODE_ENV')
    expect(result[0].services).toContain('web')
  })

  it('returns all variables when no filters applied', () => {
    const result = filterEnvironmentVariables(variables, '', null)
    expect(result).toHaveLength(variables.length)
  })

  it('returns empty array when no matches', () => {
    const result = filterEnvironmentVariables(variables, 'nonexistent', null)
    expect(result).toEqual([])
  })
})

// =============================================================================
// Component Tests
// =============================================================================

describe('EnvironmentPanel', () => {
  it('renders with title', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(screen.getByText('Environment Variables')).toBeInTheDocument()
  })

  it('displays variable count', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(screen.getByText('6 variables')).toBeInTheDocument()
  })

  it('renders all environment variables in table', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(screen.getByText('API_KEY')).toBeInTheDocument()
    expect(screen.getByText('DATABASE_URL')).toBeInTheDocument()
    expect(screen.getByText('NODE_ENV')).toBeInTheDocument()
    expect(screen.getByText('PORT')).toBeInTheDocument()
  })

  it('masks sensitive values by default', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // Sensitive values should be masked
    expect(screen.getAllByText('••••••••••••')).toHaveLength(2) // API_KEY and SECRET_TOKEN

    // Non-sensitive values should be visible
    expect(screen.getByText('development')).toBeInTheDocument()
    expect(screen.getByText('3000')).toBeInTheDocument()
  })

  it('shows sensitive indicator (lock icon) for sensitive variables', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const lockIcons = screen.getAllByRole('img', { name: 'Sensitive value' })
    expect(lockIcons.length).toBeGreaterThan(0)
  })

  it('displays service badges for each variable', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // NODE_ENV is used by api and web
    const apiBadges = screen.getAllByText('api')
    const webBadges = screen.getAllByText('web')
    expect(apiBadges.length).toBeGreaterThan(0)
    expect(webBadges.length).toBeGreaterThan(0)
  })

  it('renders empty state when no services', () => {
    render(<EnvironmentPanel services={[]} />)

    expect(screen.getByText('No Environment Variables')).toBeInTheDocument()
    expect(
      screen.getByText("Services haven't defined any environment variables.")
    ).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const services = createMockServices()
    render(
      <EnvironmentPanel
        services={services}
        className="custom-class"
        data-testid="env-panel"
      />
    )

    const panel = screen.getByTestId('env-panel')
    expect(panel).toHaveClass('custom-class')
  })
})

describe('EnvironmentPanel - Show/Hide Toggle', () => {
  it('reveals all values when toggle clicked', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // Initially sensitive values are masked
    expect(screen.getAllByText('••••••••••••')).toHaveLength(2)

    // Click show values toggle
    const toggleButton = screen.getByRole('button', {
      name: 'Show sensitive values',
    })
    await user.click(toggleButton)

    // All values should now be visible
    expect(screen.queryByText('••••••••••••')).not.toBeInTheDocument()
    expect(screen.getByText('sk-secret-123')).toBeInTheDocument()
    expect(screen.getByText('token-abc-123')).toBeInTheDocument()
  })

  it('toggles button label between Show/Hide', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // Initially shows "Show Values"
    expect(screen.getByText('Show Values')).toBeInTheDocument()

    // Click toggle
    await user.click(screen.getByRole('button', { name: 'Show sensitive values' }))

    // Now shows "Hide Values"
    expect(screen.getByText('Hide Values')).toBeInTheDocument()
  })

  it('has correct aria-pressed state', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const toggleButton = screen.getByRole('button', {
      name: 'Show sensitive values',
    })
    expect(toggleButton).toHaveAttribute('aria-pressed', 'false')

    await user.click(toggleButton)

    expect(
      screen.getByRole('button', { name: 'Hide sensitive values' })
    ).toHaveAttribute('aria-pressed', 'true')
  })
})

describe('EnvironmentPanel - Search', () => {
  it('filters variables by name', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'DATABASE')

    // Wait for debounce
    await waitFor(() => {
      expect(screen.getByText('DATABASE_URL')).toBeInTheDocument()
      expect(screen.queryByText('API_KEY')).not.toBeInTheDocument()
    })
  })

  it('filters variables by value', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'development')

    await waitFor(() => {
      expect(screen.getByText('NODE_ENV')).toBeInTheDocument()
    })
  })

  it('updates count when filtering', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(screen.getByText('6 variables')).toBeInTheDocument()

    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'NODE')

    await waitFor(() => {
      expect(screen.getByText('1 of 6 variables')).toBeInTheDocument()
    })
  })

  it('shows empty state when no results', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'nonexistent')

    await waitFor(() => {
      expect(screen.getByText('No Results Found')).toBeInTheDocument()
    })
  })

  it('clears search on Escape key', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'test')
    expect(searchInput).toHaveValue('test')

    await user.keyboard('{Escape}')
    expect(searchInput).toHaveValue('')
  })
})

describe('EnvironmentPanel - Service Filter', () => {
  it('filters by selected service', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const serviceSelect = screen.getByRole('combobox', {
      name: 'Filter by service',
    })
    await user.selectOptions(serviceSelect, 'web')

    // Only variables used by 'web' service should appear
    expect(screen.getByText('NODE_ENV')).toBeInTheDocument()
    expect(screen.getByText('NEXT_PUBLIC_API_URL')).toBeInTheDocument()
    expect(screen.queryByText('API_KEY')).not.toBeInTheDocument()
    expect(screen.queryByText('DATABASE_URL')).not.toBeInTheDocument()
  })

  it('shows all services option', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const serviceSelect = screen.getByRole('combobox', {
      name: 'Filter by service',
    })
    expect(serviceSelect).toHaveValue('all')
  })

  it('lists all available services in dropdown', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(screen.getByRole('option', { name: 'All Services' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'api' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'web' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'worker' })).toBeInTheDocument()
  })
})

describe('EnvironmentPanel - Copy Functionality', () => {
  it('shows copied state after clicking copy button', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // Find and click copy button for NODE_ENV
    const copyButtons = screen.getAllByRole('button', { name: /Copy .* value/i })
    const nodeEnvCopyButton = copyButtons.find(btn =>
      btn.getAttribute('aria-label')?.includes('NODE_ENV')
    )
    expect(nodeEnvCopyButton).toBeDefined()

    await user.click(nodeEnvCopyButton!)

    // The button should now show the copied state
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /NODE_ENV copied/i })).toBeInTheDocument()
    })
  })

  it('can copy masked sensitive values', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // Find copy button for API_KEY (which is masked)
    const copyButtons = screen.getAllByRole('button', { name: /Copy .* value/i })
    const apiKeyCopyButton = copyButtons.find(btn =>
      btn.getAttribute('aria-label')?.includes('API_KEY')
    )
    expect(apiKeyCopyButton).toBeDefined()

    await user.click(apiKeyCopyButton!)

    // Should show copied state
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /API_KEY copied/i })).toBeInTheDocument()
    })
  })

  it('resets copied state after timeout', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const copyButton = screen.getAllByRole('button', { name: /Copy .* value/i })[0]
    await user.click(copyButton)

    // Wait for the copied state
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /copied/i })).toBeInTheDocument()
    })

    // Wait for the 2 second timeout to reset
    await waitFor(
      () => {
        expect(screen.queryByRole('button', { name: /copied/i })).not.toBeInTheDocument()
      },
      { timeout: 3000 }
    )
  }, 5000)
})

describe('EnvironmentPanel - Clear Filters', () => {
  it('shows Clear Filters button when filters active', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'nonexistent')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Clear Filters' })).toBeInTheDocument()
    })
  })

  it('clears filters when Clear Filters clicked', async () => {
    const user = userEvent.setup()
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    // Apply filter
    const searchInput = screen.getByPlaceholderText('Search variables...')
    await user.type(searchInput, 'nonexistent')

    // Click clear
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Clear Filters' })).toBeInTheDocument()
    })
    await user.click(screen.getByRole('button', { name: 'Clear Filters' }))

    // Should show all variables again
    await waitFor(() => {
      expect(screen.getByText('6 variables')).toBeInTheDocument()
    })
  })
})

describe('EnvironmentPanel - Accessibility', () => {
  it('has accessible table structure', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(screen.getByRole('table', { name: 'Environment variables' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Variable' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Value' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Services' })).toBeInTheDocument()
  })

  it('has accessible search input', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const searchInput = screen.getByRole('searchbox', {
      name: 'Search environment variables',
    })
    expect(searchInput).toBeInTheDocument()
    expect(searchInput).toHaveAccessibleDescription('Search by variable name or value')
  })

  it('has accessible service filter', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    expect(
      screen.getByRole('combobox', { name: 'Filter by service' })
    ).toBeInTheDocument()
  })

  it('announces filter results via aria-live', () => {
    const services = createMockServices()
    render(<EnvironmentPanel services={services} />)

    const liveRegion = screen.getByText('6 variables')
    expect(liveRegion).toHaveAttribute('aria-live', 'polite')
    expect(liveRegion).toHaveAttribute('aria-atomic', 'true')
  })

  it('empty state has status role', () => {
    render(<EnvironmentPanel services={[]} />)

    const emptyState = screen.getByRole('status')
    expect(emptyState).toHaveTextContent('No Environment Variables')
  })
})
