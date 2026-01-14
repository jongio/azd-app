/**
 * useAutoSave Hook Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAutoSave } from './useAutoSave'
import { saveDraft, loadDraft } from '../../../lib/errors'

describe('useAutoSave', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
    localStorage.clear()
  })

  const mockGetConfig = vi.fn(() => ({ name: 'test-app' }))
  const mockIsDirty = vi.fn(() => true)

  it('should initialize with auto-save disabled', () => {
    const { result } = renderHook(() =>
      useAutoSave(mockGetConfig, mockIsDirty, { enabled: false })
    )

    expect(result.current.isAutoSaving).toBe(false)
  })

  it('should start auto-save when enabled', () => {
    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    // Auto-save should start automatically
    expect(result.current.isAutoSaving).toBe(true)
  })

  it('should auto-save every 30 seconds', () => {
    renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    // Clear any initial saves
    localStorage.clear()

    // Advance 30 seconds
    act(() => {
      vi.advanceTimersByTime(30000)
    })

    // Check that draft was saved
    const draft = loadDraft()
    expect(draft).toBeTruthy()
    expect(draft?.config).toEqual({ name: 'test-app' })
  })

  it('should not save when not dirty', () => {
    const isDirty = vi.fn(() => false)
    renderHook(() => useAutoSave(mockGetConfig, isDirty))

    localStorage.clear()

    act(() => {
      vi.advanceTimersByTime(30000)
    })

    // No draft should be saved
    expect(loadDraft()).toBeNull()
  })

  it('should save immediately with saveNow', () => {
    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    localStorage.clear()

    act(() => {
      result.current.saveNow({ name: 'immediate-save' }, true)
      vi.advanceTimersByTime(1000) // Debounce delay
    })

    const draft = loadDraft()
    expect(draft?.config).toEqual({ name: 'immediate-save' })
  })

  it('should clear draft', () => {
    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    // Save a draft first
    act(() => {
      result.current.saveNow({ name: 'test' }, true)
      vi.advanceTimersByTime(1000)
    })

    expect(loadDraft()).toBeTruthy()

    // Clear it
    act(() => {
      result.current.clearSavedDraft()
    })

    expect(loadDraft()).toBeNull()
  })

  it('should get draft', () => {
    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    // Save a draft
    act(() => {
      result.current.saveNow({ name: 'test' }, true)
      vi.advanceTimersByTime(1000)
    })

    // Get draft
    const draft = result.current.getDraft()
    expect(draft?.config).toEqual({ name: 'test' })
  })

  it('should show recovery modal if draft exists on mount', () => {
    // Save a draft before mounting
    saveDraft({ name: 'existing-draft' }, true)

    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    expect(result.current.showRecovery).toBe(true)
    expect(result.current.recoveryDraft).toBeTruthy()
    expect(result.current.recoveryDraft?.config).toEqual({ name: 'existing-draft' })
  })

  it('should not show recovery modal if draft is not dirty', () => {
    // Save a clean draft
    saveDraft({ name: 'clean-draft' }, false)

    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    expect(result.current.showRecovery).toBe(false)
  })

  it('should hide recovery modal', () => {
    saveDraft({ name: 'existing-draft' }, true)

    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    expect(result.current.showRecovery).toBe(true)

    act(() => {
      result.current.hideRecovery()
    })

    expect(result.current.showRecovery).toBe(false)
  })

  it('should call onDraftLoaded when draft exists', () => {
    const onDraftLoaded = vi.fn()
    saveDraft({ name: 'existing-draft' }, true)

    renderHook(() => useAutoSave(mockGetConfig, mockIsDirty, { onDraftLoaded }))

    expect(onDraftLoaded).toHaveBeenCalledWith(
      expect.objectContaining({
        config: { name: 'existing-draft' },
        dirty: true,
      })
    )
  })

  it('should stop auto-save on unmount', () => {
    const { unmount } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    expect(mockGetConfig).toHaveBeenCalled()

    const callCount = mockGetConfig.mock.calls.length
    unmount()

    // Advance time after unmount
    act(() => {
      vi.advanceTimersByTime(60000)
    })

    // Should not have made additional calls
    expect(mockGetConfig.mock.calls.length).toBe(callCount)
  })

  it('should manually start and stop auto-save', () => {
    const { result } = renderHook(() =>
      useAutoSave(mockGetConfig, mockIsDirty, { enabled: false })
    )

    expect(result.current.isAutoSaving).toBe(false)

    act(() => {
      result.current.startAutoSave()
    })

    expect(result.current.isAutoSaving).toBe(true)

    act(() => {
      result.current.stopAutoSave()
    })

    expect(result.current.isAutoSaving).toBe(false)
  })

  it('should cleanup stale drafts on mount', () => {
    // Create a stale draft (8 days old)
    const staleDraft = {
      config: { name: 'stale' },
      timestamp: Date.now() - 8 * 24 * 60 * 60 * 1000,
      dirty: true,
    }
    localStorage.setItem('azure-yaml-editor-draft', JSON.stringify(staleDraft))

    const { result } = renderHook(() => useAutoSave(mockGetConfig, mockIsDirty))

    // Stale draft should be cleaned up, so no recovery modal
    expect(result.current.showRecovery).toBe(false)
    expect(loadDraft()).toBeNull()
  })
})
