import { useState, useEffect, useCallback, useMemo } from 'react'
import { type Transport } from '@connectrpc/connect'

import { create } from '@bufbuild/protobuf'

import { createLogsClient } from '@/lib/connectClient'
import {
  PreferencesSchema,
  UIPreferencesSchema,
  BehaviorPreferencesSchema,
  CopyPreferencesSchema,
  type Preferences as PbPreferences,
} from '@/gen/proto/azdapp/v1/logs_pb.js'

export type Theme = 'light' | 'dark'

export interface UserPreferences {
  version: string
  theme: Theme
  ui: {
    gridColumns: number
    viewMode: 'grid' | 'unified'
    gridAutoFit: boolean
    selectedServices: string[]
  }
  behavior: {
    autoScroll: boolean
    pauseOnScroll: boolean
    timestampFormat: string
  }
  copy: {
    defaultFormat: 'plaintext' | 'json' | 'markdown' | 'csv'
    includeTimestamp: boolean
    includeService: boolean
  }
}

const DEFAULT_PREFERENCES: UserPreferences = {
  version: '1.0',
  theme: 'light',
  ui: {
    gridColumns: 2,
    viewMode: 'grid',
    gridAutoFit: true,
    selectedServices: [],
  },
  behavior: {
    autoScroll: true,
    pauseOnScroll: true,
    timestampFormat: 'hh:mm:ss.sss',
  },
  copy: {
    defaultFormat: 'plaintext',
    includeTimestamp: true,
    includeService: true,
  },
}

function isValidViewMode(v: string): v is 'grid' | 'unified' {
  return v === 'grid' || v === 'unified'
}

function isValidCopyFormat(
  v: string
): v is 'plaintext' | 'json' | 'markdown' | 'csv' {
  return v === 'plaintext' || v === 'json' || v === 'markdown' || v === 'csv'
}

function isValidTheme(v: string): v is Theme {
  return v === 'light' || v === 'dark'
}

/**
 * Convert the proto Preferences message into the dashboard's
 * UserPreferences shape, falling back to defaults for any field that's
 * missing or invalid. This is the moral equivalent of the legacy
 * `validatePreferences` function: the wire is now strongly typed but
 * the SERVER may emit a future schema we can't fully decode (theme is
 * a string, not an enum, precisely so the dashboard can drift ahead),
 * and a corrupt blob round-trips as a default Preferences message.
 */
function pbToUserPreferences(pb: PbPreferences | undefined): UserPreferences {
  if (!pb) {
    return DEFAULT_PREFERENCES
  }
  const ui = pb.ui ?? create(UIPreferencesSchema)
  const behavior = pb.behavior ?? create(BehaviorPreferencesSchema)
  const copy = pb.copy ?? create(CopyPreferencesSchema)

  return {
    version: pb.version || DEFAULT_PREFERENCES.version,
    theme: isValidTheme(pb.theme) ? pb.theme : DEFAULT_PREFERENCES.theme,
    ui: {
      gridColumns:
        ui.gridColumns >= 1 && ui.gridColumns <= 6
          ? ui.gridColumns
          : DEFAULT_PREFERENCES.ui.gridColumns,
      viewMode: isValidViewMode(ui.viewMode)
        ? ui.viewMode
        : DEFAULT_PREFERENCES.ui.viewMode,
      // gridAutoFit is a proto bool with default false, but the
      // dashboard default is true. We can't tell "user set false" from
      // "field absent" on the wire, so we honour what the server sends.
      // The server-side handler returns the dashboard default when the
      // stored blob is empty, which preserves the legacy behaviour end
      // to end.
      gridAutoFit: ui.gridAutoFit,
      selectedServices: ui.selectedServices ?? [],
    },
    behavior: {
      autoScroll: behavior.autoScroll,
      pauseOnScroll: behavior.pauseOnScroll,
      timestampFormat:
        behavior.timestampFormat || DEFAULT_PREFERENCES.behavior.timestampFormat,
    },
    copy: {
      defaultFormat: isValidCopyFormat(copy.defaultFormat)
        ? copy.defaultFormat
        : DEFAULT_PREFERENCES.copy.defaultFormat,
      includeTimestamp: copy.includeTimestamp,
      includeService: copy.includeService,
    },
  }
}

/**
 * Convert a dashboard UserPreferences object into the proto Preferences
 * message used by SavePreferences. The handler immediately re-serialises
 * to JSON via protojson, so we don't need to worry about field defaults
 * being elided here - everything we set lands in the persisted blob.
 */
function userPreferencesToPb(prefs: UserPreferences): PbPreferences {
  return create(PreferencesSchema, {
    version: prefs.version,
    theme: prefs.theme,
    ui: create(UIPreferencesSchema, {
      gridColumns: prefs.ui.gridColumns,
      gridAutoFit: prefs.ui.gridAutoFit,
      viewMode: prefs.ui.viewMode,
      selectedServices: prefs.ui.selectedServices,
    }),
    behavior: create(BehaviorPreferencesSchema, {
      autoScroll: prefs.behavior.autoScroll,
      pauseOnScroll: prefs.behavior.pauseOnScroll,
      timestampFormat: prefs.behavior.timestampFormat,
    }),
    copy: create(CopyPreferencesSchema, {
      defaultFormat: prefs.copy.defaultFormat,
      includeTimestamp: prefs.copy.includeTimestamp,
      includeService: prefs.copy.includeService,
    }),
  })
}

/** Return type of usePreferences hook */
export interface UsePreferencesReturn {
  preferences: UserPreferences
  isLoading: boolean
  savePreferences: (updates: Partial<UserPreferences>) => Promise<void>
  updateUI: (updates: Partial<UserPreferences['ui']>) => void
  setTheme: (theme: Theme) => void
  reload: () => Promise<void>
}

/**
 * Persist user preferences (theme, grid layout, copy format, etc.) via
 * the LogsService Connect handler. The optional `transport` argument
 * exists for tests that wire a `createRouterTransport`; production
 * callers omit it and use the singleton transport.
 */
export function usePreferences(transport?: Transport): UsePreferencesReturn {
  const [preferences, setPreferences] = useState<UserPreferences>(
    DEFAULT_PREFERENCES
  )
  const [isLoading, setIsLoading] = useState(true)

  const client = useMemo(() => createLogsClient(transport), [transport])

  const loadPreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      const resp = await client.getPreferences({})
      setPreferences(pbToUserPreferences(resp.preferences))
    } catch (err) {
      // Preferences are best-effort: a network blip should not break
      // the dashboard. Fall back to defaults and log so devtools shows
      // the underlying error without throwing inside React's render.
      console.error('Failed to load preferences:', err)
      setPreferences(DEFAULT_PREFERENCES)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  /* eslint-disable react-hooks/set-state-in-effect -- async fetch; setState happens asynchronously */
  useEffect(() => {
    void loadPreferences()
  }, [loadPreferences])
  /* eslint-enable react-hooks/set-state-in-effect */

  const savePreferences = useCallback(
    async (updates: Partial<UserPreferences>) => {
      const merged: UserPreferences = { ...preferences, ...updates }
      // Optimistic local apply so the UI doesn't lag the network. If
      // the save fails we keep the optimistic state - matches legacy
      // behaviour and avoids snapping a slider back mid-drag.
      setPreferences(merged)
      try {
        const resp = await client.savePreferences({
          preferences: userPreferencesToPb(merged),
        })
        // Server normalises (e.g. clamps invalid grid columns); honour
        // its echo so client + server agree.
        setPreferences(pbToUserPreferences(resp.preferences))
      } catch (err) {
        console.error('Failed to save preferences:', err)
      }
    },
    [client, preferences]
  )

  const updateUI = useCallback(
    (updates: Partial<UserPreferences['ui']>) => {
      void savePreferences({ ui: { ...preferences.ui, ...updates } })
    },
    [preferences.ui, savePreferences]
  )

  const setTheme = useCallback(
    (theme: Theme) => {
      void savePreferences({ theme })
    },
    [savePreferences]
  )

  return {
    preferences,
    isLoading,
    savePreferences,
    updateUI,
    setTheme,
    reload: loadPreferences,
  }
}
