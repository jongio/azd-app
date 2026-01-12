/**
 * API client for azure.yaml editor configuration operations
 */

export interface ConfigResponse {
  path: string
  content: string
  lastModified: string
}

export interface SaveConfigRequest {
  content: string
}

export interface SaveConfigResponse {
  success: boolean
  backup: string
  written: boolean
  errors?: string[]
}

export interface BackupInfo {
  timestamp: string
  path: string
  size: number
}

export interface BackupsListResponse {
  backups: BackupInfo[]
}

export interface BackupContentResponse {
  content: string
  timestamp: string
}

export interface RestoreBackupResponse {
  success: boolean
  restoredFrom: string
  backupCreated: string
}

/**
 * Load current azure.yaml configuration
 */
export async function loadConfig(): Promise<ConfigResponse> {
  const response = await fetch('/api/editor/config', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to load configuration' }))
    throw new Error(error.error || 'Failed to load configuration')
  }

  return response.json()
}

/**
 * Save azure.yaml configuration (creates backup automatically)
 */
export async function saveConfig(content: string): Promise<SaveConfigResponse> {
  const request: SaveConfigRequest = { content }

  const response = await fetch('/api/editor/config', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to save configuration' }))
    throw new Error(error.error || 'Failed to save configuration')
  }

  return response.json()
}

/**
 * List all backup files
 */
export async function listBackups(): Promise<BackupsListResponse> {
  const response = await fetch('/api/editor/backups', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to list backups' }))
    throw new Error(error.error || 'Failed to list backups')
  }

  return response.json()
}

/**
 * Get specific backup content by timestamp
 */
export async function getBackup(timestamp: string): Promise<BackupContentResponse> {
  const response = await fetch(`/api/editor/backups/${timestamp}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to get backup' }))
    throw new Error(error.error || 'Failed to get backup')
  }

  return response.json()
}

/**
 * Restore a backup by timestamp
 */
export async function restoreBackup(timestamp: string): Promise<RestoreBackupResponse> {
  const response = await fetch(`/api/editor/backups/${timestamp}/restore`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to restore backup' }))
    throw new Error(error.error || 'Failed to restore backup')
  }

  return response.json()
}

/**
 * Delete a backup by timestamp
 */
export async function deleteBackup(timestamp: string): Promise<void> {
  const response = await fetch(`/api/editor/backups/${timestamp}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to delete backup' }))
    throw new Error(error.error || 'Failed to delete backup')
  }
}
