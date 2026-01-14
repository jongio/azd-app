/**
 * useAutoSave Hook - Auto-save and recovery integration
 */

import { useEffect, useState, useCallback, useRef } from 'react'
import {
  AutoSaveManager,
  saveDraft,
  loadDraft,
  clearDraft,
  cleanupStaleDrafts,
} from '../../../lib/errors'
import type { DraftData } from './RecoveryModal'

interface UseAutoSaveOptions {
  /** Enable auto-save (default: true) */
  enabled?: boolean
  /** Auto-save interval in ms (default: 30000) */
  interval?: number
  /** Callback when draft is loaded */
  onDraftLoaded?: (draft: DraftData) => void
}

interface UseAutoSaveReturn {
  /** Whether auto-save is active */
  isAutoSaving: boolean
  /** Start auto-save */
  startAutoSave: () => void
  /** Stop auto-save */
  stopAutoSave: () => void
  /** Trigger immediate save */
  saveNow: (config: unknown, dirty: boolean) => void
  /** Clear draft */
  clearSavedDraft: () => void
  /** Load draft (returns null if no draft) */
  getDraft: () => DraftData | null
  /** Show recovery modal */
  showRecovery: boolean
  /** Draft data for recovery */
  recoveryDraft: DraftData | null
  /** Hide recovery modal */
  hideRecovery: () => void
}

export function useAutoSave(
  getConfig: () => unknown,
  isDirty: () => boolean,
  options: UseAutoSaveOptions = {}
): UseAutoSaveReturn {
  const { enabled = true, onDraftLoaded } = options

  const [isAutoSaving, setIsAutoSaving] = useState(false)
  const [showRecovery, setShowRecovery] = useState(false)
  const [recoveryDraft, setRecoveryDraft] = useState<DraftData | null>(null)
  const managerRef = useRef<AutoSaveManager | null>(null)

  // Initialize manager
  useEffect(() => {
    if (!managerRef.current) {
      managerRef.current = new AutoSaveManager()
    }

    // Cleanup stale drafts on mount
    cleanupStaleDrafts()

    // Check for existing draft
    const draft = loadDraft()
    if (draft && draft.dirty) {
      // Use setTimeout to avoid setState during render
      setTimeout(() => {
        setRecoveryDraft(draft)
        setShowRecovery(true)
        if (onDraftLoaded) {
          onDraftLoaded(draft)
        }
      }, 0)
    }

    return () => {
      if (managerRef.current) {
        managerRef.current.stop()
      }
    }
  }, [onDraftLoaded])

  const startAutoSave = useCallback(() => {
    if (!enabled || !managerRef.current) {
      return
    }

    managerRef.current.start(getConfig, isDirty)
    setIsAutoSaving(true)
  }, [enabled, getConfig, isDirty])

  const stopAutoSave = useCallback(() => {
    if (managerRef.current) {
      managerRef.current.stop()
      setIsAutoSaving(false)
    }
  }, [])

  const saveNow = useCallback((config: unknown, dirty: boolean) => {
    if (managerRef.current) {
      managerRef.current.triggerSave(config, dirty)
    } else {
      saveDraft(config, dirty)
    }
  }, [])

  const clearSavedDraft = useCallback(() => {
    clearDraft()
    setRecoveryDraft(null)
    setShowRecovery(false)
  }, [])

  const getDraft = useCallback(() => {
    return loadDraft()
  }, [])

  const hideRecovery = useCallback(() => {
    setShowRecovery(false)
  }, [])

  // Auto-start if enabled
  useEffect(() => {
    if (enabled && !isAutoSaving) {
      // Use setTimeout to avoid setState during effect setup
      setTimeout(() => startAutoSave(), 0)
    }

    return () => {
      if (isAutoSaving) {
        stopAutoSave()
      }
    }
  }, [enabled, isAutoSaving, startAutoSave, stopAutoSave])

  return {
    isAutoSaving,
    startAutoSave,
    stopAutoSave,
    saveNow,
    clearSavedDraft: clearSavedDraft,
    getDraft,
    showRecovery,
    recoveryDraft,
    hideRecovery,
  }
}
