/**
 * useAutoSave Hook - Auto-save and recovery integration
 */

import { useEffect, useState, useCallback, useRef, useLayoutEffect } from 'react'
import {
  AutoSaveManager,
  saveDraft,
  loadDraft,
  clearDraft,
  cleanupStaleDrafts,
  isDraftStale,
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
  const cleanupRef = useRef(false)

  if (!managerRef.current) {
    managerRef.current = new AutoSaveManager()
  }

  if (!cleanupRef.current) {
    cleanupStaleDrafts()
    const existingDraft = loadDraft()
    if (existingDraft && isDraftStale(existingDraft)) {
      clearDraft()
    }

    // Defensive cleanup for any stale serialized draft data
    const rawDraft = localStorage.getItem('azure-yaml-editor-draft')
    if (rawDraft) {
      try {
        const parsed = JSON.parse(rawDraft)
        if (parsed?.timestamp && Date.now() - parsed.timestamp > 7 * 24 * 60 * 60 * 1000) {
          localStorage.removeItem('azure-yaml-editor-draft')
        }
      } catch {
        // Ignore malformed payloads
      }
    }
    cleanupRef.current = true
  }

  // Initialize drafts and cleanup
  useLayoutEffect(() => {
    cleanupStaleDrafts()

    const draft = loadDraft()
    if (draft && isDraftStale(draft)) {
      clearDraft()
      return
    }
    if (draft && draft.dirty) {
      setRecoveryDraft(draft)
      setShowRecovery(true)
      onDraftLoaded?.(draft)
    } else if (draft && !draft.dirty) {
      clearDraft()
    }

    return () => {
      if (managerRef.current) {
        managerRef.current.stop()
      }
    }
  }, [onDraftLoaded])

  const startAutoSave = useCallback(() => {
    if (!managerRef.current) {
      return
    }

    managerRef.current.start(getConfig, isDirty)

    // Prime any consumers that rely on an initial read without persisting a draft
    getConfig()

    setIsAutoSaving(true)
  }, [getConfig, isDirty])

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
      startAutoSave()
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
