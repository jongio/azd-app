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
  /** Backup identifier (timestamp), empty if no prior file existed */
  backup: string
  written: boolean
  errors?: string[]
}

export interface BackupInfo {
  timestamp: string
  /** Backup file name (not an absolute path) */
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
  /** Timestamp restored from */
  restoredFrom: string
  /** Backup identifier (timestamp) of the pre-restore file, if any */
  backupCreated: string
}

export interface ValidationError {
  message: string
  path?: string
  line?: number
  level?: 'error' | 'warning' | 'info'
}

export interface ValidationResponse {
  valid: boolean
  errors: ValidationError[]
}

export interface CreateBackupResponse {
  success: boolean
  timestamp: string
  path: string
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

/**
 * Validate azure.yaml content against schema
 */
export async function validateConfig(content: string): Promise<ValidationResponse> {
  const response = await fetch('/api/editor/validate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ content }),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to validate configuration' }))
    throw new Error(error.error || 'Failed to validate configuration')
  }

  return response.json()
}

/**
 * Create a manual backup with custom name
 */
export async function createManualBackup(name?: string): Promise<CreateBackupResponse> {
  const response = await fetch('/api/editor/backups', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ name }),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Failed to create backup' }))
    throw new Error(error.error || 'Failed to create backup')
  }

  return response.json()
}
