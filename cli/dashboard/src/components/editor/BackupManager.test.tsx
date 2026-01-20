/**
 * Backup Manager Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BackupManager } from './BackupManager'
import * as configApi from '@/lib/editor/config-api'

// Mock the config API
vi.mock('@/lib/editor/config-api', () => ({
  listBackups: vi.fn(),
  getBackup: vi.fn(),
  restoreBackup: vi.fn(),
  deleteBackup: vi.fn(),
}))

describe('BackupManager', () => {
  const mockBackups = [
    {
      timestamp: '2026-01-11T14:30:00Z',
      path: 'azure.yaml.backup.2026-01-11T143000Z',
      size: 1024,
    },
    {
      timestamp: '2026-01-11T12:00:00Z',
      path: 'azure.yaml.backup.2026-01-11T120000Z',
      size: 2048,
    },
  ]

  const mockBackupContent = 'name: my-app\nservices:\n  api:\n    host: containerapp'

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(configApi.listBackups).mockResolvedValue({ backups: mockBackups })
    vi.mocked(configApi.getBackup).mockResolvedValue({
      content: mockBackupContent,
      timestamp: mockBackups[0].timestamp,
    })
    vi.mocked(configApi.restoreBackup).mockResolvedValue({
      success: true,
      restoredFrom: mockBackups[0].timestamp,
      backupCreated: '2026-01-11T150000Z',
    })
    vi.mocked(configApi.deleteBackup).mockResolvedValue(undefined)
  })

  it('renders backups button', () => {
    render(<BackupManager />)

    expect(screen.getByRole('button', { name: 'Manage backups' })).toBeInTheDocument()
  })

  it('opens backup list modal when button clicked', async () => {
    const user = userEvent.setup()
    render(<BackupManager />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByText('Backups')).toHaveLength(2)
      expect(configApi.listBackups).toHaveBeenCalled()
    })
  })

  it('loads backups when modal opened', async () => {
    const user = userEvent.setup()
    render(<BackupManager />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(configApi.listBackups).toHaveBeenCalled()
    })
  })

  it('handles load backups error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    vi.mocked(configApi.listBackups).mockRejectedValueOnce(new Error('Load failed'))

    const user = userEvent.setup()
    render(<BackupManager />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to load backups. Please try again.')
    })

    alertSpy.mockRestore()
  })

  it('opens view modal when view button clicked', async () => {
    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    // Wait for backups to load
    await waitFor(() => {
      expect(screen.getAllByLabelText('View backup')).toHaveLength(2)
    })

    // Click view button
    const viewButtons = screen.getAllByLabelText('View backup')
    await user.click(viewButtons[0])

    await waitFor(() => {
      expect(screen.getByText('View Backup')).toBeInTheDocument()
      expect(configApi.getBackup).toHaveBeenCalledWith(mockBackups[0].timestamp)
    })
  })

  it('handles view backup error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    vi.mocked(configApi.getBackup).mockRejectedValueOnce(new Error('Get failed'))

    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('View backup')).toHaveLength(2)
    })

    // Click view button
    const viewButtons = screen.getAllByLabelText('View backup')
    await user.click(viewButtons[0])

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to load backup content. Please try again.')
    })

    alertSpy.mockRestore()
  })

  it('opens restore confirmation when restore button clicked', async () => {
    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('Restore backup')).toHaveLength(2)
    })

    // Click restore button
    const restoreButtons = screen.getAllByLabelText('Restore backup')
    await user.click(restoreButtons[0])

    await waitFor(() => {
      expect(screen.getAllByText('Restore Backup')).toHaveLength(2)
      expect(screen.getByText(/Confirm backup restoration/)).toBeInTheDocument()
    })
  })

  it('restores backup when confirmed', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const onRestoreSuccess = vi.fn()
    const user = userEvent.setup()

    render(<BackupManager onRestoreSuccess={onRestoreSuccess} />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('Restore backup')).toHaveLength(2)
    })

    // Click restore button
    const restoreButtons = screen.getAllByLabelText('Restore backup')
    await user.click(restoreButtons[0])

    await waitFor(() => {
      expect(screen.getAllByText('Restore Backup')).toHaveLength(2)
    })

    // Confirm restore
    const confirmButton = screen.getByRole('button', { name: 'Restore Backup' })
    await user.click(confirmButton)

    await waitFor(() => {
      expect(configApi.restoreBackup).toHaveBeenCalledWith(mockBackups[0].timestamp)
      expect(alertSpy).toHaveBeenCalledWith(expect.stringContaining('Restored backup from'))
      expect(onRestoreSuccess).toHaveBeenCalled()
    })

    alertSpy.mockRestore()
  })

  it('handles restore error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const _consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(configApi.restoreBackup).mockRejectedValueOnce(new Error('Restore failed'))

    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('Restore backup')).toHaveLength(2)
    })

    // Click restore button
    const restoreButtons = screen.getAllByLabelText('Restore backup')
    await user.click(restoreButtons[0])

    await waitFor(() => {
      expect(screen.getAllByText('Restore Backup')).toHaveLength(2)
    })

    // Confirm restore
    const confirmButton = screen.getByRole('button', { name: 'Restore Backup' })
    await user.click(confirmButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to restore backup. Please try again.')
    })

    alertSpy.mockRestore()
  })

  it('opens delete confirmation when delete button clicked', async () => {
    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('Delete backup')).toHaveLength(2)
    })

    // Click delete button
    const deleteButtons = screen.getAllByLabelText('Delete backup')
    await user.click(deleteButtons[0])

    await waitFor(() => {
      expect(screen.getByText('Delete Backup')).toBeInTheDocument()
      expect(screen.getByText('This action cannot be undone')).toBeInTheDocument()
    })
  })

  it('deletes backup when confirmed', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const user = userEvent.setup()

    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('Delete backup')).toHaveLength(2)
    })

    // Click delete button
    const deleteButtons = screen.getAllByLabelText('Delete backup')
    await user.click(deleteButtons[0])

    await waitFor(() => {
      const deleteTexts = screen.getAllByText('Delete Backup')
      expect(deleteTexts.length).toBeGreaterThan(0)
    })

    // Confirm delete
    const confirmButtons = screen.getAllByRole('button', { name: 'Confirm Delete' })
    const confirmButton = confirmButtons[confirmButtons.length - 1] // Get the actual button, not the title
    await user.click(confirmButton)

    await waitFor(() => {
      expect(configApi.deleteBackup).toHaveBeenCalledWith(mockBackups[0].timestamp)
      expect(configApi.listBackups).toHaveBeenCalledTimes(2) // Initial + reload
      expect(alertSpy).toHaveBeenCalledWith('✓ Backup deleted successfully')
    })

    alertSpy.mockRestore()
  })

  it('handles delete error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const _consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(configApi.deleteBackup).mockRejectedValueOnce(new Error('Delete failed'))

    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    await waitFor(() => {
      expect(screen.getAllByLabelText('Delete backup')).toHaveLength(2)
    })

    // Click delete button
    const deleteButtons = screen.getAllByLabelText('Delete backup')
    await user.click(deleteButtons[0])

    await waitFor(() => {
      expect(screen.getByText('Delete Backup')).toBeInTheDocument()
    })

    // Confirm delete
    const confirmButton = screen.getByRole('button', { name: 'Confirm Delete' })
    await user.click(confirmButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to delete backup. Please try again.')
    })

    alertSpy.mockRestore()
  })

  it('applies custom className to button', () => {
    render(<BackupManager className="custom-class" />)

    const button = screen.getByRole('button', { name: 'Manage backups' })
    expect(button).toHaveClass('custom-class')
  })

  it('gets preview content for backups', async () => {
    const user = userEvent.setup()
    render(<BackupManager />)

    // Open backup list
    const button = screen.getByRole('button', { name: 'Manage backups' })
    await user.click(button)

    // Wait for previews to load
    await waitFor(() => {
      expect(configApi.getBackup).toHaveBeenCalled()
    })
  })
})
