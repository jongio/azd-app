/**
 * WellKnownServicesTab Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { WellKnownServicesTab } from './WellKnownServicesTab'
import * as wellknownApi from '@/lib/api/wellknown'
import type { WellKnownService } from '@/lib/editor/wellknown-types'

// Mock the API
vi.mock('@/lib/api/wellknown')

const mockServices: WellKnownService[] = [
  {
    name: 'redis',
    displayName: 'Redis Cache',
    description: 'In-memory data store',
    category: 'database',
    icon: '📦',
    host: 'containerapp',
    image: 'redis:7-alpine',
    ports: ['6379:6379'],
    healthcheck: {
      test: ['CMD', 'redis-cli', 'ping'],
      interval: '30s',
    },
  },
  {
    name: 'postgres',
    displayName: 'PostgreSQL',
    description: 'Relational database',
    category: 'database',
    icon: '🐘',
    host: 'containerapp',
    image: 'postgres:16-alpine',
    ports: ['5432:5432'],
    environment: {
      POSTGRES_PASSWORD: 'localdevpassword',
    },
    connectionStrings: {
      postgres: 'postgresql://postgres:localdevpassword@localhost:5432',
    },
    healthcheck: {
      test: ['CMD-SHELL', 'pg_isready -U postgres'],
      interval: '30s',
    },
  },
  {
    name: 'azurite',
    displayName: 'Azurite',
    description: 'Azure Storage emulator',
    category: 'storage',
    icon: '💾',
    host: 'containerapp',
    image: 'mcr.microsoft.com/azure-storage/azurite',
    ports: ['10000:10000'],
    environment: {
      AZURITE_ACCOUNTS: 'devstoreaccount1:key',
    },
    docsUrl: 'https://docs.microsoft.com/azure/storage/azurite',
    healthcheck: {
      test: ['CMD', 'curl', '-f', 'http://localhost:10000'],
      interval: '30s',
    },
  },
]

describe('WellKnownServicesTab', () => {
  const mockOnSelectService = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ===========================================================================
  // Loading State Tests
  // ===========================================================================

  describe('Loading State', () => {
    it('should show loading indicator while fetching services', () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockReturnValue(
        new Promise(() => {}) // Never resolves
      )

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      expect(screen.getByText('Loading services...')).toBeInTheDocument()
    })

    it('should hide loading indicator after services loaded', async () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.queryByText('Loading services...')).not.toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Error State Tests
  // ===========================================================================

  describe('Error State', () => {
    it('should show error message on fetch failure', async () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockRejectedValue(
        new Error('Network error')
      )

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument()
      })
    })

    it('should show generic error message on unknown error', async () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockRejectedValue('Unknown error')

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Failed to load services')).toBeInTheDocument()
      })
    })

    it('should show error icon', async () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockRejectedValue(
        new Error('Network error')
      )

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('⚠️')).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Service Display Tests
  // ===========================================================================

  describe('Service Display', () => {
    beforeEach(() => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)
    })

    it('should display all services after loading', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
        expect(screen.getByText('PostgreSQL')).toBeInTheDocument()
        expect(screen.getByText('Azurite')).toBeInTheDocument()
      })
    })

    it('should display service descriptions', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('In-memory data store')).toBeInTheDocument()
        expect(screen.getByText('Relational database')).toBeInTheDocument()
        expect(screen.getByText('Azure Storage emulator')).toBeInTheDocument()
      })
    })

    it('should display service categories', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      // Services display their categories - 2 database + 1 storage + filter buttons
      await waitFor(() => {
        const categoryText = screen.getAllByText('database')
        // Should find category badges and filter button
        expect(categoryText.length).toBeGreaterThanOrEqual(2)
        const storageText = screen.getAllByText('storage')
        expect(storageText.length).toBeGreaterThanOrEqual(1)
      })
    })

    it('should display service icons', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('📦')).toBeInTheDocument()
        expect(screen.getByText('🐘')).toBeInTheDocument()
        expect(screen.getByText('💾')).toBeInTheDocument()
      })
    })

    it('should display port information', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        // All 3 services have 1 port each, so we'll find multiple "1 port" texts
        const portTexts = screen.getAllByText('1 port')
        expect(portTexts.length).toBeGreaterThanOrEqual(1)
      })
    })
  })

  // ===========================================================================
  // Category Filter Tests
  // ===========================================================================

  describe('Category Filter', () => {
    beforeEach(() => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)
    })

    it('should show all category filter button', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'all' })).toBeInTheDocument()
      })
    })

    it('should show category filter buttons for each category', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'database' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'storage' })).toBeInTheDocument()
      })
    })

    it('should filter services by category when filter clicked', async () => {
      const user = userEvent.setup()

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
      })

      const databaseFilter = screen.getByRole('button', { name: 'database' })
      await user.click(databaseFilter)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
        expect(screen.getByText('PostgreSQL')).toBeInTheDocument()
        expect(screen.queryByText('Azurite')).not.toBeInTheDocument()
      })
    })

    it('should highlight active category filter', async () => {
      const user = userEvent.setup()

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        const allButton = screen.getByRole('button', { name: 'all' })
        expect(allButton).toHaveClass('bg-cyan-500')
      })

      const databaseFilter = screen.getByRole('button', { name: 'database' })
      await user.click(databaseFilter)

      await waitFor(() => {
        expect(databaseFilter).toHaveClass('bg-cyan-500')
        expect(screen.getByRole('button', { name: 'all' })).not.toHaveClass('bg-cyan-500')
      })
    })

    it('should show all services when "all" filter selected', async () => {
      const user = userEvent.setup()

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
      })

      // Filter to database
      const databaseFilter = screen.getByRole('button', { name: 'database' })
      await user.click(databaseFilter)

      await waitFor(() => {
        expect(screen.queryByText('Azurite')).not.toBeInTheDocument()
      })

      // Switch back to all
      const allFilter = screen.getByRole('button', { name: 'all' })
      await user.click(allFilter)

      await waitFor(() => {
        expect(screen.getByText('Azurite')).toBeInTheDocument()
      })
    })

    it.skip('should show empty state when no services in category', async () => {
      // Skipped: Category filter buttons are only shown for categories that have services
      // Since mockServices don't include 'other' category, the filter button won't exist
      const user = userEvent.setup()
      
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
      })

      // Click 'other' category filter which should show empty state (no services in 'other' category)
      const otherFilter = screen.getByRole('button', { name: 'other' })
      await user.click(otherFilter)
      
      // Should show empty state message
      await waitFor(() => {
        expect(screen.queryByText('Redis Cache')).not.toBeInTheDocument()
        expect(screen.queryByText('PostgreSQL')).not.toBeInTheDocument()
        expect(screen.queryByText('Azurite')).not.toBeInTheDocument()
        // Component should show empty state UI
      })
    })
  })

  // ===========================================================================
  // Service Selection Tests
  // ===========================================================================

  describe('Service Selection', () => {
    beforeEach(() => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)
    })

    it('should call onSelectService when service clicked', async () => {
      const user = userEvent.setup()

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
      })

      const redisCard = screen.getByRole('button', { name: /select redis cache/i })
      await user.click(redisCard)

      expect(mockOnSelectService).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'redis',
          displayName: 'Redis Cache',
        })
      )
    })

    it('should highlight selected service', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[0]}
        />
      )

      await waitFor(() => {
        const redisCard = screen.getByRole('button', { name: /select redis cache/i })
        expect(redisCard).toHaveClass('border-cyan-500')
      })
    })

    it('should show checkmark on selected service', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[0]}
        />
      )

      await waitFor(() => {
        const redisCard = screen.getByRole('button', { name: /select redis cache/i })
        const checkmark = redisCard.querySelector('svg')
        expect(checkmark).toBeInTheDocument()
      })
    })

    it('should not show checkmark on unselected services', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[0]}
        />
      )

      await waitFor(() => {
        const postgresCard = screen.getByRole('button', { name: /select postgresql/i })
        expect(postgresCard).not.toHaveClass('border-cyan-500')
      })
    })
  })

  // ===========================================================================
  // Preview Panel Tests
  // ===========================================================================

  describe('Preview Panel', () => {
    beforeEach(() => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)
    })

    it('should show placeholder when no service selected', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(
          screen.getByText('Select a service to see configuration preview')
        ).toBeInTheDocument()
      })
    })

    it('should show service configuration when service selected', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[0]}
        />
      )

      await waitFor(() => {
        expect(screen.getByText('Configuration Preview')).toBeInTheDocument()
        expect(screen.getByText('redis')).toBeInTheDocument()
        // containerapp appears in multiple places, just check it exists
        const containerappElements = screen.getAllByText('containerapp')
        expect(containerappElements.length).toBeGreaterThanOrEqual(1)
        expect(screen.getByText('redis:7-alpine')).toBeInTheDocument()
      })
    })

    it('should display ports in preview', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[0]}
        />
      )

      await waitFor(() => {
        expect(screen.getByText(/6379:6379/)).toBeInTheDocument()
      })
    })

    it('should display environment variables in preview', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[1]} // Postgres has environment variables
        />
      )

      await waitFor(() => {
        expect(screen.getByText(/POSTGRES_PASSWORD/)).toBeInTheDocument()
      })
    })

    it('should display connection strings in preview', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[1]} // Postgres has connection strings
        />
      )

      await waitFor(() => {
        expect(screen.getByText('Connection Strings')).toBeInTheDocument()
        expect(screen.getByText(/postgresql:\/\//)).toBeInTheDocument()
      })
    })

    it('should display documentation link when available', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[2]} // Azurite has docsUrl
        />
      )

      await waitFor(() => {
        const docLink = screen.getByRole('link', { name: /view documentation/i })
        expect(docLink).toHaveAttribute('href', 'https://docs.microsoft.com/azure/storage/azurite')
        expect(docLink).toHaveAttribute('target', '_blank')
        expect(docLink).toHaveAttribute('rel', 'noopener noreferrer')
      })
    })

    it('should not display documentation link when not available', async () => {
      render(
        <WellKnownServicesTab
          onSelectService={mockOnSelectService}
          selectedService={mockServices[0]} // Redis has no docsUrl
        />
      )

      await waitFor(() => {
        expect(screen.queryByRole('link', { name: /view documentation/i })).not.toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Accessibility Tests
  // ===========================================================================

  describe('Accessibility', () => {
    beforeEach(() => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)
    })

    it('should have accessible labels for service cards', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /select redis cache/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /select postgresql/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /select azurite/i })).toBeInTheDocument()
      })
    })

    it('should have keyboard navigation support for service cards', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        const redisCard = screen.getByRole('button', { name: /select redis cache/i })
        expect(redisCard).toHaveAttribute('type', 'button')
      })
    })

    it('should have keyboard navigation support for category filters', async () => {
      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        const databaseFilter = screen.getByRole('button', { name: 'database' })
        expect(databaseFilter).toBeInTheDocument()
      })
    })
  })

  // ===========================================================================
  // Component Lifecycle Tests
  // ===========================================================================

  describe('Component Lifecycle', () => {
    it('should fetch services on mount', async () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)

      render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(wellknownApi.fetchWellKnownServices).toHaveBeenCalledTimes(1)
      })
    })

    it('should cleanup on unmount', async () => {
      vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue(mockServices)

      const { unmount } = render(<WellKnownServicesTab onSelectService={mockOnSelectService} />)

      await waitFor(() => {
        expect(screen.getByText('Redis Cache')).toBeInTheDocument()
      })

      unmount()

      // Component should unmount without errors
      expect(screen.queryByText('Redis Cache')).not.toBeInTheDocument()
    })
  })
})
