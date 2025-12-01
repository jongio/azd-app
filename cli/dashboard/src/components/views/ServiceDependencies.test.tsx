import { describe, it, expect, vi } from 'vitest'
import { render, screen, within, fireEvent } from '@testing-library/react'
import { ServiceDependencies } from './ServiceDependencies'
import type { Service } from '@/types'

// =============================================================================
// Test Data
// =============================================================================

const createService = (overrides: Partial<Service>): Service => ({
  name: 'test-service',
  language: 'TypeScript',
  framework: 'Express',
  local: {
    status: 'running',
    health: 'healthy',
    port: 3000,
    url: 'http://localhost:3000',
  },
  environmentVariables: { NODE_ENV: 'development' },
  ...overrides,
})

const mockServices: Service[] = [
  createService({
    name: 'api',
    language: 'TypeScript',
    framework: 'Express',
    local: {
      status: 'running',
      health: 'healthy',
      port: 3100,
      url: 'http://localhost:3100',
    },
    environmentVariables: { NODE_ENV: 'production', API_KEY: 'xxx', DB_URL: 'xxx' },
  }),
  createService({
    name: 'web',
    language: 'TypeScript',
    framework: 'Next.js',
    local: {
      status: 'running',
      health: 'healthy',
      port: 3000,
    },
    environmentVariables: { NODE_ENV: 'development' },
  }),
  createService({
    name: 'worker',
    language: 'TypeScript',
    framework: 'Node.js',
    local: {
      status: 'stopped',
      health: 'unknown',
    },
    environmentVariables: {},
  }),
  createService({
    name: 'ml-service',
    language: 'Python',
    framework: 'FastAPI',
    local: {
      status: 'running',
      health: 'healthy',
      port: 8000,
      url: 'http://localhost:8000',
    },
    environmentVariables: { PYTHON_ENV: 'production', MODEL_PATH: '/models' },
  }),
  createService({
    name: 'cache',
    language: 'Go',
    framework: 'Gin',
    local: {
      status: 'starting',
      health: 'unknown',
      port: 6379,
    },
    environmentVariables: { REDIS_URL: 'localhost:6379' },
  }),
]

// =============================================================================
// Helper Function Tests (via dependencies-utils.ts)
// =============================================================================

import {
  groupServicesByLanguage,
  normalizeLanguage,
  getLanguageBadgeStyle,
  getStatusIndicator,
  countEnvVars,
  sortGroupsBySize,
  getServiceUrl,
  pluralize,
} from '@/lib/dependencies-utils'

describe('dependencies-utils', () => {
  describe('groupServicesByLanguage', () => {
    it('groups services by language correctly', () => {
      const grouped = groupServicesByLanguage(mockServices)
      expect(Object.keys(grouped)).toContain('TypeScript')
      expect(Object.keys(grouped)).toContain('Python')
      expect(Object.keys(grouped)).toContain('Go')
      expect(grouped['TypeScript']).toHaveLength(3)
      expect(grouped['Python']).toHaveLength(1)
      expect(grouped['Go']).toHaveLength(1)
    })

    it('uses "Other" for services without language', () => {
      const services = [createService({ name: 'unknown', language: undefined })]
      const grouped = groupServicesByLanguage(services)
      expect(grouped['Other']).toHaveLength(1)
    })

    it('returns empty object for empty services', () => {
      const grouped = groupServicesByLanguage([])
      expect(Object.keys(grouped)).toHaveLength(0)
    })
  })

  describe('normalizeLanguage', () => {
    it('normalizes TypeScript variations', () => {
      expect(normalizeLanguage('ts')).toBe('TypeScript')
      expect(normalizeLanguage('typescript')).toBe('TypeScript')
      expect(normalizeLanguage('TypeScript')).toBe('TypeScript')
    })

    it('normalizes JavaScript variations', () => {
      expect(normalizeLanguage('js')).toBe('JavaScript')
      expect(normalizeLanguage('javascript')).toBe('JavaScript')
    })

    it('normalizes Python variations', () => {
      expect(normalizeLanguage('py')).toBe('Python')
      expect(normalizeLanguage('python')).toBe('Python')
    })

    it('normalizes Go variations', () => {
      expect(normalizeLanguage('go')).toBe('Go')
      expect(normalizeLanguage('golang')).toBe('Go')
    })

    it('normalizes Rust variations', () => {
      expect(normalizeLanguage('rs')).toBe('Rust')
      expect(normalizeLanguage('rust')).toBe('Rust')
    })

    it('normalizes C# variations', () => {
      expect(normalizeLanguage('c#')).toBe('C#')
      expect(normalizeLanguage('csharp')).toBe('C#')
    })

    it('normalizes .NET variations', () => {
      expect(normalizeLanguage('dotnet')).toBe('.NET')
      expect(normalizeLanguage('.net')).toBe('.NET')
    })

    it('returns original for unknown languages', () => {
      expect(normalizeLanguage('Kotlin')).toBe('Kotlin')
      expect(normalizeLanguage('Ruby')).toBe('Ruby')
    })
  })

  describe('getLanguageBadgeStyle', () => {
    it('returns correct style for TypeScript', () => {
      const style = getLanguageBadgeStyle('TypeScript')
      expect(style.abbr).toBe('TS')
      expect(style.bg).toContain('blue')
      expect(style.text).toContain('blue')
    })

    it('returns correct style for JavaScript', () => {
      const style = getLanguageBadgeStyle('JavaScript')
      expect(style.abbr).toBe('JS')
      expect(style.bg).toContain('yellow')
    })

    it('returns correct style for Python', () => {
      const style = getLanguageBadgeStyle('Python')
      expect(style.abbr).toBe('PY')
      expect(style.bg).toContain('green')
    })

    it('returns correct style for Go', () => {
      const style = getLanguageBadgeStyle('Go')
      expect(style.abbr).toBe('GO')
      expect(style.bg).toContain('cyan')
    })

    it('returns correct style for Rust', () => {
      const style = getLanguageBadgeStyle('Rust')
      expect(style.abbr).toBe('RS')
      expect(style.bg).toContain('orange')
    })

    it('returns correct style for Java', () => {
      const style = getLanguageBadgeStyle('Java')
      expect(style.abbr).toBe('JV')
      expect(style.bg).toContain('red')
    })

    it('returns correct style for C#', () => {
      const style = getLanguageBadgeStyle('C#')
      expect(style.abbr).toBe('C#')
      expect(style.bg).toContain('purple')
    })

    it('returns default style for unknown language', () => {
      const style = getLanguageBadgeStyle('Unknown')
      expect(style.abbr).toBe('??')
      expect(style.bg).toContain('gray')
    })
  })

  describe('getStatusIndicator', () => {
    it('returns running indicator', () => {
      const indicator = getStatusIndicator('running')
      expect(indicator.icon).toBe('●')
      expect(indicator.color).toContain('green')
      expect(indicator.animate).toContain('pulse')
    })

    it('returns ready indicator', () => {
      const indicator = getStatusIndicator('ready')
      expect(indicator.icon).toBe('●')
      expect(indicator.color).toContain('green')
      expect(indicator.animate).toBe('')
    })

    it('returns starting indicator', () => {
      const indicator = getStatusIndicator('starting')
      expect(indicator.icon).toBe('◐')
      expect(indicator.color).toContain('yellow')
      expect(indicator.animate).toContain('spin')
    })

    it('returns stopping indicator', () => {
      const indicator = getStatusIndicator('stopping')
      expect(indicator.icon).toBe('◑')
      expect(indicator.color).toContain('yellow')
    })

    it('returns stopped indicator', () => {
      const indicator = getStatusIndicator('stopped')
      expect(indicator.icon).toBe('○')
      expect(indicator.color).toContain('gray')
    })

    it('returns error indicator', () => {
      const indicator = getStatusIndicator('error')
      expect(indicator.icon).toBe('⚠')
      expect(indicator.color).toContain('red')
      expect(indicator.animate).toContain('pulse')
    })

    it('returns not-running indicator for undefined', () => {
      const indicator = getStatusIndicator(undefined)
      expect(indicator.icon).toBe('○')
      expect(indicator.color).toContain('gray')
    })

    it('returns not-running indicator for unknown status', () => {
      const indicator = getStatusIndicator('unknown-status')
      expect(indicator.icon).toBe('○')
      expect(indicator.color).toContain('gray')
    })
  })

  describe('countEnvVars', () => {
    it('counts environment variables correctly', () => {
      const service = createService({
        environmentVariables: { A: '1', B: '2', C: '3' },
      })
      expect(countEnvVars(service)).toBe(3)
    })

    it('returns 0 for empty environment variables', () => {
      const service = createService({ environmentVariables: {} })
      expect(countEnvVars(service)).toBe(0)
    })

    it('returns 0 for undefined environment variables', () => {
      const service = createService({ environmentVariables: undefined })
      expect(countEnvVars(service)).toBe(0)
    })
  })

  describe('sortGroupsBySize', () => {
    it('sorts groups by count descending', () => {
      const groups = {
        'Go': [createService({ name: 'go1' })],
        'TypeScript': [
          createService({ name: 'ts1' }),
          createService({ name: 'ts2' }),
          createService({ name: 'ts3' }),
        ],
        'Python': [
          createService({ name: 'py1' }),
          createService({ name: 'py2' }),
        ],
      }
      const sorted = sortGroupsBySize(groups)
      expect(sorted[0][0]).toBe('TypeScript')
      expect(sorted[1][0]).toBe('Python')
      expect(sorted[2][0]).toBe('Go')
    })

    it('sorts alphabetically when counts are equal', () => {
      const groups = {
        'Python': [createService({ name: 'py1' })],
        'Go': [createService({ name: 'go1' })],
        'Rust': [createService({ name: 'rs1' })],
      }
      const sorted = sortGroupsBySize(groups)
      expect(sorted[0][0]).toBe('Go')
      expect(sorted[1][0]).toBe('Python')
      expect(sorted[2][0]).toBe('Rust')
    })
  })

  describe('getServiceUrl', () => {
    it('returns URL from local info', () => {
      const service = createService({
        local: { status: 'running', health: 'healthy', url: 'http://localhost:3000' },
      })
      expect(getServiceUrl(service)).toBe('http://localhost:3000')
    })

    it('builds URL from port when no URL provided', () => {
      const service = createService({
        local: { status: 'running', health: 'healthy', port: 8080 },
      })
      expect(getServiceUrl(service)).toBe('http://localhost:8080')
    })

    it('returns null for services without URL or port', () => {
      const service = createService({
        local: { status: 'stopped', health: 'unknown' },
      })
      expect(getServiceUrl(service)).toBeNull()
    })

    it('returns null for services without local info', () => {
      const service = createService({ local: undefined })
      expect(getServiceUrl(service)).toBeNull()
    })
  })

  describe('pluralize', () => {
    it('returns singular for count 1', () => {
      expect(pluralize(1, 'service')).toBe('service')
    })

    it('returns plural for count 0', () => {
      expect(pluralize(0, 'service')).toBe('services')
    })

    it('returns plural for count > 1', () => {
      expect(pluralize(5, 'service')).toBe('services')
    })

    it('uses custom plural when provided', () => {
      expect(pluralize(0, 'entry', 'entries')).toBe('entries')
      expect(pluralize(1, 'entry', 'entries')).toBe('entry')
      expect(pluralize(5, 'entry', 'entries')).toBe('entries')
    })
  })
})

// =============================================================================
// ServiceDependencies Component Tests
// =============================================================================

describe('ServiceDependencies', () => {
  describe('rendering', () => {
    it('renders with services', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByTestId('service-dependencies')).toBeInTheDocument()
    })

    it('renders empty state when no services', () => {
      render(<ServiceDependencies services={[]} />)
      expect(screen.getByText('No services found')).toBeInTheDocument()
    })

    it('renders language groups', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByTestId('language-group-TypeScript')).toBeInTheDocument()
      expect(screen.getByTestId('language-group-Python')).toBeInTheDocument()
      expect(screen.getByTestId('language-group-Go')).toBeInTheDocument()
    })

    it('renders service cards', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByTestId('service-card-api')).toBeInTheDocument()
      expect(screen.getByTestId('service-card-web')).toBeInTheDocument()
      expect(screen.getByTestId('service-card-worker')).toBeInTheDocument()
      expect(screen.getByTestId('service-card-ml-service')).toBeInTheDocument()
      expect(screen.getByTestId('service-card-cache')).toBeInTheDocument()
    })

    it('sorts groups by size descending', () => {
      render(<ServiceDependencies services={mockServices} />)
      const groups = screen.getAllByTestId(/^language-group-/)
      // TypeScript has 3 services, should be first
      expect(groups[0]).toHaveAttribute('data-testid', 'language-group-TypeScript')
    })

    it('applies custom className', () => {
      render(<ServiceDependencies services={mockServices} className="custom-class" />)
      const container = screen.getByTestId('service-dependencies')
      expect(container).toHaveClass('custom-class')
    })

    it('uses custom data-testid', () => {
      render(<ServiceDependencies services={mockServices} data-testid="custom-id" />)
      expect(screen.getByTestId('custom-id')).toBeInTheDocument()
    })
  })

  describe('language groups', () => {
    it('displays language badge', () => {
      render(<ServiceDependencies services={mockServices} />)
      const tsGroup = screen.getByTestId('language-group-TypeScript')
      expect(within(tsGroup).getByText('TS')).toBeInTheDocument()
    })

    it('displays language name', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByRole('heading', { name: 'TypeScript' })).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: 'Python' })).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: 'Go' })).toBeInTheDocument()
    })

    it('displays service count', () => {
      render(<ServiceDependencies services={mockServices} />)
      const tsGroup = screen.getByTestId('language-group-TypeScript')
      expect(within(tsGroup).getByText('(3 services)')).toBeInTheDocument()

      const pyGroup = screen.getByTestId('language-group-Python')
      expect(within(pyGroup).getByText('(1 service)')).toBeInTheDocument()
    })
  })

  describe('service cards', () => {
    it('displays service name', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByText('api')).toBeInTheDocument()
      expect(screen.getByText('web')).toBeInTheDocument()
    })

    it('displays framework', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByText('Express')).toBeInTheDocument()
      expect(screen.getByText('Next.js')).toBeInTheDocument()
      expect(screen.getByText('FastAPI')).toBeInTheDocument()
    })

    it('displays port', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByText(':3100')).toBeInTheDocument()
      expect(screen.getByText(':3000')).toBeInTheDocument()
      expect(screen.getByText(':8000')).toBeInTheDocument()
    })

    it('displays environment variable count', () => {
      render(<ServiceDependencies services={mockServices} />)
      // api has 3 env vars
      const apiCard = screen.getByTestId('service-card-api')
      expect(within(apiCard).getByText('3 env vars')).toBeInTheDocument()

      // web has 1 env var
      const webCard = screen.getByTestId('service-card-web')
      expect(within(webCard).getByText('1 env var')).toBeInTheDocument()

      // worker has 0 env vars
      const workerCard = screen.getByTestId('service-card-worker')
      expect(within(workerCard).getByText('0 env vars')).toBeInTheDocument()
    })

    it('displays URL link when available', () => {
      render(<ServiceDependencies services={mockServices} />)
      const apiCard = screen.getByTestId('service-card-api')
      const link = within(apiCard).getByRole('link', { name: /Open api at/i })
      expect(link).toHaveAttribute('href', 'http://localhost:3100')
      expect(link).toHaveAttribute('target', '_blank')
    })

    it('does not display URL when not available', () => {
      render(<ServiceDependencies services={mockServices} />)
      const workerCard = screen.getByTestId('service-card-worker')
      expect(within(workerCard).queryByRole('link')).not.toBeInTheDocument()
    })

    it('displays status indicator for running services', () => {
      render(<ServiceDependencies services={mockServices} />)
      const apiCard = screen.getByTestId('service-card-api')
      const indicator = within(apiCard).getByText('●')
      expect(indicator).toHaveClass('text-green-500')
    })

    it('displays status indicator for stopped services', () => {
      render(<ServiceDependencies services={mockServices} />)
      const workerCard = screen.getByTestId('service-card-worker')
      const indicator = within(workerCard).getByText('○')
      expect(indicator).toHaveClass('text-gray-500')
    })

    it('displays status indicator for starting services', () => {
      render(<ServiceDependencies services={mockServices} />)
      const cacheCard = screen.getByTestId('service-card-cache')
      const indicator = within(cacheCard).getByText('◐')
      expect(indicator).toHaveClass('text-yellow-500')
    })
  })

  describe('interactions', () => {
    it('calls onServiceClick when card is clicked', () => {
      const handleClick = vi.fn()
      render(<ServiceDependencies services={mockServices} onServiceClick={handleClick} />)

      const apiCard = screen.getByTestId('service-card-api')
      fireEvent.click(apiCard)

      expect(handleClick).toHaveBeenCalledTimes(1)
      expect(handleClick).toHaveBeenCalledWith(mockServices[0])
    })

    it('URL click does not trigger card click', () => {
      const handleClick = vi.fn()
      render(<ServiceDependencies services={mockServices} onServiceClick={handleClick} />)

      const apiCard = screen.getByTestId('service-card-api')
      const link = within(apiCard).getByRole('link')
      fireEvent.click(link)

      expect(handleClick).not.toHaveBeenCalled()
    })

    it('cards are focusable', () => {
      render(<ServiceDependencies services={mockServices} />)
      const apiCard = screen.getByTestId('service-card-api')
      apiCard.focus()
      expect(document.activeElement).toBe(apiCard)
    })
  })

  describe('accessibility', () => {
    it('has screen reader title', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByText('Service Dependencies by Language')).toBeInTheDocument()
      expect(screen.getByText('Service Dependencies by Language')).toHaveClass('sr-only')
    })

    it('service cards have aria-label', () => {
      render(<ServiceDependencies services={mockServices} />)
      const apiCard = screen.getByTestId('service-card-api')
      expect(apiCard).toHaveAttribute('aria-label')
      expect(apiCard.getAttribute('aria-label')).toContain('api service')
      expect(apiCard.getAttribute('aria-label')).toContain('running')
      expect(apiCard.getAttribute('aria-label')).toContain('Express')
    })

    it('groups have heading labels', () => {
      render(<ServiceDependencies services={mockServices} />)
      expect(screen.getByRole('heading', { name: 'TypeScript' })).toBeInTheDocument()
    })

    it('uses role="list" for service cards container', () => {
      render(<ServiceDependencies services={mockServices} />)
      const lists = screen.getAllByRole('list')
      expect(lists.length).toBeGreaterThan(0)
    })
  })

  describe('grouping edge cases', () => {
    it('groups services without language as "Other"', () => {
      const servicesWithoutLang = [
        createService({ name: 'unknown1', language: undefined }),
        createService({ name: 'unknown2', language: '' }),
      ]
      render(<ServiceDependencies services={servicesWithoutLang} />)
      expect(screen.getByTestId('language-group-Other')).toBeInTheDocument()
    })

    it('normalizes language names when grouping', () => {
      const servicesWithVariants = [
        createService({ name: 'ts1', language: 'ts' }),
        createService({ name: 'ts2', language: 'TypeScript' }),
        createService({ name: 'ts3', language: 'typescript' }),
      ]
      render(<ServiceDependencies services={servicesWithVariants} />)
      // All should be in one TypeScript group
      const tsGroup = screen.getByTestId('language-group-TypeScript')
      expect(within(tsGroup).getByText('(3 services)')).toBeInTheDocument()
    })

    it('handles single service group', () => {
      const singleService = [createService({ name: 'solo', language: 'Kotlin' })]
      render(<ServiceDependencies services={singleService} />)
      expect(screen.getByTestId('language-group-Kotlin')).toBeInTheDocument()
      expect(screen.getByText('(1 service)')).toBeInTheDocument()
    })
  })
})
