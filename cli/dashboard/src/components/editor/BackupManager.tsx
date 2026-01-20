/**
 * Backup Manager
 * Main component orchestrating backup management UI
 */

import * as React from 'react'
import { BackupListModal } from './modals/BackupListModal'
import { ViewBackupModal } from './modals/ViewBackupModal'
import { RestoreConfirmationDialog } from './modals/RestoreConfirmationDialog'
import { DeleteBackupDialog } from './modals/DeleteBackupDialog'
import { BackupsButton } from './BackupsButton'
import {
  listBackups,
  getBackup,
  restoreBackup,
  deleteBackup,
  type BackupInfo,
} from '@/lib/editor/config-api'

export interface BackupManagerProps {
  /** Callback when backup is restored successfully */
  onRestoreSuccess?: () => void
  /** Optional className for the button */
  className?: string
}

/**
 * Backup Manager Component
 * 
 * Provides complete backup management UI including:
 * - Backups button for header
 * - Backup list modal
 * - View backup modal
 * - Restore confirmation dialog
 * - Delete confirmation dialog
 */
export function BackupManager({ onRestoreSuccess, className }: BackupManagerProps) {
  const [isListOpen, setIsListOpen] = React.useState(false)
  const [isViewOpen, setIsViewOpen] = React.useState(false)
  const [isRestoreDialogOpen, setIsRestoreDialogOpen] = React.useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = React.useState(false)

  const [backups, setBackups] = React.useState<BackupInfo[]>([])
  const [selectedTimestamp, setSelectedTimestamp] = React.useState<string>('')
  const [viewContent, setViewContent] = React.useState<string>('')

  const [isLoadingList, setIsLoadingList] = React.useState(false)
  const [isLoadingView, setIsLoadingView] = React.useState(false)
  const [isRestoring, setIsRestoring] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)

  // Load backups list
  const loadBackups = React.useCallback(async () => {
    setIsLoadingList(true)
    try {
      const response = await listBackups()
      setBackups(response.backups)
    } catch {
      alert('Failed to load backups. Please try again.')
    } finally {
      setIsLoadingList(false)
    }
  }, [])

  // Open backup list modal
  const handleOpenList = React.useCallback(() => {
    setIsListOpen(true)
    void loadBackups()
  }, [loadBackups])

  // Get backup preview (first 10 lines)
  const handleGetPreview = React.useCallback(async (timestamp: string): Promise<string> => {
    try {
      const response = await getBackup(timestamp)
      const lines = response.content.split('\n').slice(0, 10)
      return lines.join('\n')
    } catch {
      alert('Failed to load backup content. Please try again.')
      return 'Failed to load preview'
    }
  }, [])

  // Handle view backup
  const handleView = React.useCallback((timestamp: string) => {
    void (async () => {
      setSelectedTimestamp(timestamp)
      setIsViewOpen(true)
      setIsLoadingView(true)

      try {
        const response = await getBackup(timestamp)
        setViewContent(response.content)
      } catch {
        alert('Failed to load backup content. Please try again.')
        setIsViewOpen(false)
      } finally {
        setIsLoadingView(false)
      }
    })()
  }, [])

  // Handle restore backup (show confirmation)
  const handleRestoreClick = React.useCallback((timestamp: string) => {
    setSelectedTimestamp(timestamp)
    setIsRestoreDialogOpen(true)
  }, [])

  // Confirm restore
  const handleRestoreConfirm = React.useCallback(() => {
    void (async () => {
      setIsRestoring(true)
      try {
        await restoreBackup(selectedTimestamp)
        
        // Close all dialogs
        setIsRestoreDialogOpen(false)
        setIsListOpen(false)
        
        // Show success notification
        const formattedDate = new Date(selectedTimestamp).toLocaleString('en-US', {
          year: 'numeric',
          month: 'short',
          day: 'numeric',
          hour: 'numeric',
          minute: '2-digit',
          hour12: true,
        })
        alert(`✓ Restored backup from ${formattedDate}`)
        
        // Callback to parent
        onRestoreSuccess?.()
      } catch {
        alert('Failed to restore backup. Please try again.')
      } finally {
        setIsRestoring(false)
      }
    })()
  }, [selectedTimestamp, onRestoreSuccess])

  // Handle delete backup (show confirmation)
  const handleDeleteClick = React.useCallback((timestamp: string) => {
    setSelectedTimestamp(timestamp)
    setIsDeleteDialogOpen(true)
  }, [])

  // Confirm delete
  const handleDeleteConfirm = React.useCallback(() => {
    void (async () => {
      setIsDeleting(true)
      try {
        await deleteBackup(selectedTimestamp)
        
        // Reload backups list
        await loadBackups()
        
        // Close dialog
        setIsDeleteDialogOpen(false)
        
        // Show success notification
        alert('✓ Backup deleted successfully')
      } catch {
        alert('Failed to delete backup. Please try again.')
      } finally {
        setIsDeleting(false)
      }
    })()
  }, [selectedTimestamp, loadBackups])

  return (
    <>
      {/* Backups Button */}
      <BackupsButton onClick={handleOpenList} className={className} />

      {/* Backup List Modal */}
      <BackupListModal
        isOpen={isListOpen}
        onClose={() => setIsListOpen(false)}
        backups={backups}
        onRestore={handleRestoreClick}
        onView={handleView}
        onDelete={handleDeleteClick}
        onGetPreview={handleGetPreview}
        isLoading={isLoadingList}
      />

      {/* View Backup Modal */}
      <ViewBackupModal
        isOpen={isViewOpen}
        onClose={() => setIsViewOpen(false)}
        timestamp={selectedTimestamp}
        content={viewContent}
        isLoading={isLoadingView}
      />

      {/* Restore Confirmation Dialog */}
      <RestoreConfirmationDialog
        isOpen={isRestoreDialogOpen}
        onClose={() => setIsRestoreDialogOpen(false)}
        onConfirm={handleRestoreConfirm}
        timestamp={selectedTimestamp}
        isRestoring={isRestoring}
      />

      {/* Delete Confirmation Dialog */}
      <DeleteBackupDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={handleDeleteConfirm}
        timestamp={selectedTimestamp}
        isDeleting={isDeleting}
      />
    </>
  )
}
