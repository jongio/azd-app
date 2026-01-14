/**
 * Auto-Save to localStorage - Draft recovery system
 */

const STORAGE_KEY_PREFIX = 'azure-yaml-editor-draft'

interface DraftData {
  /** Configuration data */
  config: any
  /** Last save timestamp */
  timestamp: number
  /** Has unsaved changes */
  dirty: boolean
}

/**
 * Get storage keys
 */
function getStorageKeys() {
  return {
    draft: `${STORAGE_KEY_PREFIX}`,
    timestamp: `${STORAGE_KEY_PREFIX}-timestamp`,
    dirty: `${STORAGE_KEY_PREFIX}-dirty`,
  }
}

/**
 * Save draft to localStorage
 */
export function saveDraft(
  config: any,
  dirty = true,
  onError?: (error: Error) => void
): boolean {
  try {
    const keys = getStorageKeys()
    const timestamp = Date.now()

    // Save as single object for easier management
    const draftData: DraftData = {
      config,
      timestamp,
      dirty,
    }

    const serialized = JSON.stringify(draftData)

    // Check size (limit to 5MB)
    if (serialized.length > 5 * 1024 * 1024) {
      const error = new Error('Draft too large to save (>5MB)')
      console.warn(error.message)
      if (onError) {
        onError(error)
      }
      return false
    }

    localStorage.setItem(keys.draft, serialized)
    return true
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error))
    console.error('Failed to save draft:', err)
    if (onError) {
      onError(err)
    }
    return false
  }
}

/**
 * Type guard to validate draft data structure
 */
function isDraftData(obj: unknown): obj is DraftData {
  return (
    typeof obj === 'object' &&
    obj !== null &&
    'config' in obj &&
    'timestamp' in obj &&
    typeof (obj as any).timestamp === 'number' &&
    'dirty' in obj &&
    typeof (obj as any).dirty === 'boolean'
  )
}

/**
 * Load draft from localStorage
 */
export function loadDraft(): DraftData | null {
  try {
    const keys = getStorageKeys()
    const stored = localStorage.getItem(keys.draft)

    if (!stored) {
      return null
    }

    const parsed = JSON.parse(stored)
    
    // Validate structure before trusting localStorage data
    if (!isDraftData(parsed)) {
      console.warn('Invalid draft data structure in localStorage')
      return null
    }
    
    return parsed
  } catch (error) {
    console.error('Failed to load draft:', error)
    return null
  }
}

/**
 * Clear draft from localStorage
 */
export function clearDraft(): void {
  try {
    const keys = getStorageKeys()
    localStorage.removeItem(keys.draft)
  } catch (error) {
    console.error('Failed to clear draft:', error)
  }
}

/**
 * Get draft age in milliseconds
 */
export function getDraftAge(draft: DraftData): number {
  return Date.now() - draft.timestamp
}

/**
 * Format draft age for display
 */
export function formatDraftAge(ageMs: number): string {
  const seconds = Math.floor(ageMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) {
    return `${days} day${days > 1 ? 's' : ''} ago`
  }
  if (hours > 0) {
    return `${hours} hour${hours > 1 ? 's' : ''} ago`
  }
  if (minutes > 0) {
    return `${minutes} minute${minutes > 1 ? 's' : ''} ago`
  }
  return `${seconds} second${seconds > 1 ? 's' : ''} ago`
}

/**
 * Check if draft is stale (>7 days old)
 */
export function isDraftStale(draft: DraftData): boolean {
  const ageMs = getDraftAge(draft)
  const sevenDaysMs = 7 * 24 * 60 * 60 * 1000
  return ageMs > sevenDaysMs
}

/**
 * Clean up stale drafts (>7 days old)
 */
export function cleanupStaleDrafts(): void {
  const draft = loadDraft()
  if (draft && isDraftStale(draft)) {
    clearDraft()
    console.log('Cleaned up stale draft')
  }
}

/**
 * Auto-save manager with debouncing
 */
export class AutoSaveManager {
  private saveInterval = 30000 // 30 seconds
  private debounceTimeout: number | null = null
  private intervalId: number | null = null
  private lastSaveTime = 0 // Track last save to prevent race conditions

  /**
   * Start auto-save with debouncing
   */
  start(getConfig: () => any, isDirty: () => boolean): void {
    // Clear existing interval
    this.stop()

    // Set up periodic save
    this.intervalId = window.setInterval(() => {
      const now = Date.now()
      // Skip if recently saved (within 5 seconds)
      if (now - this.lastSaveTime < 5000) {
        return
      }
      
      if (isDirty()) {
        const config = getConfig()
        saveDraft(config, true)
        this.lastSaveTime = now
      }
    }, this.saveInterval)

    // Save on window beforeunload
    window.addEventListener('beforeunload', this.handleBeforeUnload)
  }

  /**
   * Stop auto-save
   */
  stop(): void {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId)
      this.intervalId = null
    }

    if (this.debounceTimeout !== null) {
      clearTimeout(this.debounceTimeout)
      this.debounceTimeout = null
    }

    window.removeEventListener('beforeunload', this.handleBeforeUnload)
  }

  /**
   * Trigger immediate save (debounced)
   */
  triggerSave(config: any, dirty: boolean): void {
    if (this.debounceTimeout !== null) {
      clearTimeout(this.debounceTimeout)
    }

    this.debounceTimeout = window.setTimeout(() => {
      const now = Date.now()
      // Prevent duplicate saves within 5 seconds
      if (now - this.lastSaveTime < 5000) {
        this.debounceTimeout = null
        return
      }
      
      saveDraft(config, dirty)
      this.lastSaveTime = now
      this.debounceTimeout = null
    }, 1000) // 1 second debounce
  }

  private handleBeforeUnload = (): void => {
    // This will be called when window is about to unload
    // The actual save logic is handled by the component
  }
}
