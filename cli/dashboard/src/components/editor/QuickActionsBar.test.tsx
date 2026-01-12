/**
 * Quick Actions Bar Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QuickActionsBar } from './QuickActionsBar'
import type { WellKnownService } from '@/lib/editor/wellknown-types'

const mockServices: WellKnownService[] = [
  {
    name: 'azurite',
    displayName: 'Azurite (Azure Storage Emulator)',
    description: 'Local Azure Storage emulator',
    category: 'storage',
    icon: '📦',
    host: 'containerapp',
    image: 'mcr.microsoft.com/azure-storage/azurite:latest',
    ports: ['10000:10000'],
    connectionStrings: {
      default: 'UseDevelopmentStorage=true',
    },
  },
  {
    name: 'cosmos',
    displayName: 'Azure Cosmos DB Emulator',
    description: 'Local Azure Cosmos DB emulator',
    category: 'database',
    icon: '🌍',
    host: 'containerapp',
    image: 'mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest',
    ports: ['8081:8081'],
    connectionStrings: {
      default: 'AccountEndpoint=https://localhost:8081/',
    },
  },
  {
    name: 'redis',
    displayName: 'Redis Cache',
    description: 'In-memory data structure store',
    category: 'cache',
    icon: '🔴',
    host: 'containerapp',
    image: 'redis:7-alpine',
    ports: ['6379:6379'],
    connectionStrings: {
      default: 'redis://localhost:6379',
    },
  },
  {
    name: 'postgres',
    displayName: 'PostgreSQL Database',
    description: 'Open-source relational database',
    category: 'database',
    icon: '🐘',
    host: 'containerapp',
    image: 'postgres:16-alpine',
    ports: ['5432:5432'],
    connectionStrings: {
      default: 'postgresql://postgres:postgres@localhost:5432/app',
    },
  },
]

describe('QuickActionsBar', () => {
  it('renders quick action buttons for default services', () => {
    const onAddService = vi.fn()
    render(<QuickActionsBar onAddService={onAddService} services={mockServices} />)

    // Should have Quick Actions label
    expect(screen.getByText('Quick Actions')).toBeInTheDocument()

    // Should have buttons for all 4 default services (azurite, cosmos, redis, postgres)
    // Using getAllByRole since we have multiple buttons
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBeGreaterThanOrEqual(4)
  })

  it('calls onAddService when button is clicked', async () => {
    const user = userEvent.setup()
    const onAddService = vi.fn()
    render(<QuickActionsBar onAddService={onAddService} services={mockServices} />)

    // Click the first button (Azurite)
    const azuriteButton = screen.getByLabelText(/Add Azurite/i)
    await user.click(azuriteButton)

    expect(onAddService).toHaveBeenCalledTimes(1)
    expect(onAddService).toHaveBeenCalledWith(mockServices[0])
  })

  it('supports custom quick service list', () => {
    const onAddService = vi.fn()
    render(
      <QuickActionsBar
        onAddService={onAddService}
        services={mockServices}
        quickServices={['redis', 'postgres']}
      />
    )

    // Should only have buttons for redis and postgres
    const redisButton = screen.getByLabelText(/Add Redis/i)
    const postgresButton = screen.getByLabelText(/Add PostgreSQL/i)

    expect(redisButton).toBeInTheDocument()
    expect(postgresButton).toBeInTheDocument()

    // Should not have azurite or cosmos
    expect(screen.queryByLabelText(/Add Azurite/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Add.*Cosmos/i)).not.toBeInTheDocument()
  })

  it('filters out services not in the available list', () => {
    const onAddService = vi.fn()
    render(
      <QuickActionsBar
        onAddService={onAddService}
        services={mockServices}
        quickServices={['azurite', 'nonexistent']}
      />
    )

    // Should only show azurite button
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBe(1)
    expect(screen.getByLabelText(/Add Azurite/i)).toBeInTheDocument()
  })

  it('returns null when no services match quick actions', () => {
    const onAddService = vi.fn()
    const { container } = render(
      <QuickActionsBar
        onAddService={onAddService}
        services={mockServices}
        quickServices={['nonexistent1', 'nonexistent2']}
      />
    )

    expect(container.firstChild).toBeNull()
  })

  it('applies custom className', () => {
    const onAddService = vi.fn()
    const { container } = render(
      <QuickActionsBar
        onAddService={onAddService}
        services={mockServices}
        className="custom-class"
      />
    )

    const bar = container.firstChild as HTMLElement
    expect(bar).toHaveClass('custom-class')
  })

  it('has accessible labels for all buttons', () => {
    const onAddService = vi.fn()
    render(<QuickActionsBar onAddService={onAddService} services={mockServices} />)

    mockServices.forEach((service) => {
      // Labels use the full displayName
      const button = screen.getByLabelText(`Add ${service.displayName}`)
      expect(button).toBeInTheDocument()
    })
  })

  it('shows responsive button text', () => {
    const onAddService = vi.fn()
    render(<QuickActionsBar onAddService={onAddService} services={mockServices} />)

    // Desktop text (hidden on mobile)
    const desktopText = screen.getAllByText(/Add Azurite/i)
    expect(desktopText.length).toBeGreaterThan(0)

    // Mobile text with icon (hidden on desktop)
    // Note: This is rendered but hidden via CSS classes
    const mobileElements = screen.getAllByText(/azurite/i)
    expect(mobileElements.length).toBeGreaterThan(0)
  })

  it('maintains fixed position at bottom', () => {
    const onAddService = vi.fn()
    const { container } = render(
      <QuickActionsBar onAddService={onAddService} services={mockServices} />
    )

    const bar = container.firstChild as HTMLElement
    expect(bar).toHaveClass('fixed')
    expect(bar).toHaveClass('bottom-0')
  })
})
