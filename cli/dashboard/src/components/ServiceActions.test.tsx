import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ServiceActions } from './ServiceActions'
import type { Service } from '@/types'

const startService = vi.fn<() => Promise<boolean>>()
const stopService = vi.fn<() => Promise<boolean>>()
const restartService = vi.fn<() => Promise<boolean>>()
const isOperationInProgress = vi.fn<(serviceName: string) => boolean>()
const getOperationState = vi.fn<(serviceName: string) => string>()
const canPerformAction = vi.fn<(service: Service, action: 'start' | 'stop' | 'restart') => boolean>()
let errorValue: string | null = null

vi.mock('@/hooks/useServiceOperations', () => ({
  useServiceOperations: () => ({
    startService,
    stopService,
    restartService,
    isOperationInProgress,
    getOperationState,
    canPerformAction,
    error: errorValue,
  }),
}))

describe('ServiceActions', () => {
  const service: Service = {
    name: 'api',
    host: 'local',
    local: { status: 'ready', health: 'healthy', port: 3000, url: 'http://localhost:3000' },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    errorValue = null
    startService.mockResolvedValue(true)
    stopService.mockResolvedValue(true)
    restartService.mockResolvedValue(true)
    isOperationInProgress.mockReturnValue(false)
    getOperationState.mockReturnValue('idle')
    canPerformAction.mockReturnValue(true)
  })

  it('renders default action buttons and invokes operations', async () => {
    const user = userEvent.setup()

    render(<ServiceActions service={service} />)

    await user.click(screen.getByRole('button', { name: /^start$/i }))
    await waitFor(() => expect(startService).toHaveBeenCalledWith('api'))

    await user.click(screen.getByRole('button', { name: /^restart$/i }))
    await waitFor(() => expect(restartService).toHaveBeenCalledWith('api'))

    await user.click(screen.getByRole('button', { name: /^stop$/i }))
    await waitFor(() => expect(stopService).toHaveBeenCalledWith('api'))
  })

  it('shows compact loading label when operation in progress', () => {
    isOperationInProgress.mockReturnValue(true)
    getOperationState.mockReturnValue('restarting')

    render(<ServiceActions service={service} variant="compact" />)

    expect(screen.getByText('restarting...')).toBeInTheDocument()
  })

  it('shows compact error message when showError=true', () => {
    errorValue = 'Boom'

    render(<ServiceActions service={service} variant="compact" showError={true} />)

    expect(screen.getByText('Boom')).toBeInTheDocument()
  })
})
