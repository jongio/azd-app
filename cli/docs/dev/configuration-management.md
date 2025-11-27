# Configuration Management - Developer Guide

## Overview

The logs dashboard uses azd's proven configuration patterns:
- **User-level settings**: `~/.azure/logs-dashboard/` (persistent across projects)
- **App-level settings**: `.azure/logs-dashboard/` (project-specific, can commit to repo)
- **Preference hierarchy**: App config > User config > Defaults

This guide helps developers implement proper file handling, loading, and merging.

## File Structure & Schema

### 1. Preferences File

**Location**: `~/.azure/logs-dashboard/preferences.json`

**Purpose**: User's personal dashboard settings (columns, theme, behavior)

**Schema**:
```json
{
  "version": "1.0",
  "ui": {
    "gridColumns": 2,          // 1-6, default 2
    "paneHeight": 500,         // 300-800px, default 500
    "viewMode": "grid",        // "grid" | "unified"
    "selectedServices": ["web", "api"]  // array of service names to show by default
  },
  "behavior": {
    "autoScroll": true,        // auto-scroll to bottom
    "pauseOnScroll": true,     // pause on manual scroll
    "timestampFormat": "hh:mm:ss.sss"  // 24h, 12h, ms, date, etc.
  },
  "copy": {
    "defaultFormat": "plaintext",  // plaintext | json | markdown | csv
    "includeTimestamp": true,
    "includeService": true
  }
}
```

**Defaults** (if file missing or incomplete):
```typescript
const DEFAULT_PREFERENCES = {
  version: "1.0",
  ui: {
    gridColumns: 2,
    paneHeight: 500,
    viewMode: "grid",
    selectedServices: []
  },
  behavior: {
    autoScroll: true,
    pauseOnScroll: true,
    timestampFormat: "hh:mm:ss.sss"
  },
  copy: {
    defaultFormat: "plaintext",
    includeTimestamp: true,
    includeService: true
  }
}
```

### 2. Patterns File (Two Locations)

**User Patterns**: `~/.azure/logs-dashboard/patterns.json`
**App Patterns**: `.azure/logs-dashboard/patterns.json`

**Purpose**: False positive pattern definitions (applied to error detection)

**Schema**:
```json
{
  "version": "1.0",
  "patterns": [
    {
      "id": "pattern-001",
      "name": "Zero Errors",
      "regex": "^.*\\b0\\s+errors?\\b.*$",
      "description": "Matches '0 errors' or '0 error'",
      "enabled": true,
      "createdAt": "2025-11-23T10:00:00Z",
      "source": "user|app",
      "tags": ["numeric", "error"]
    },
    {
      "id": "pattern-002",
      "name": "Error Rate Zero",
      "regex": "error\\s+rate\\s*:\\s*0%?",
      "description": "Matches 'error rate: 0'",
      "enabled": true,
      "createdAt": "2025-11-23T10:30:00Z",
      "source": "user"
    }
  ]
}
```

**Pattern Matching Logic**:
- Regex patterns are case-insensitive by default
- Pattern source: "user" or "app" (for UI organization)
- Enabled flag: can disable pattern without deleting

### 3. App Config File (Optional)

**Location**: `.azure/logs-dashboard/config.json`

**Purpose**: Team-level defaults and overrides (commit to repo)

**Schema**:
```json
{
  "version": "1.0",
  "name": "Dashboard Configuration",
  "description": "Team-shared logs dashboard settings",
  "ui": {
    "gridColumns": 3,          // team default: 3 columns
    "paneHeight": 600,         // team default: larger panes
    "viewMode": "grid"
  },
  "preloadedPatterns": [
    "pattern-nodejs-warnings",
    "pattern-python-deprecation"
  ],
  "defaultServices": ["web", "api", "db"],
  "tags": ["team", "shared"]
}
```

## Implementation Patterns

### File Management (TypeScript)

```typescript
import { promises as fs } from 'fs'
import { join, dirname } from 'path'
import os from 'os'

// Paths
function getUserPreferencesPath(): string {
  const azureDir = join(os.homedir(), '.azure', 'logs-dashboard')
  return join(azureDir, 'preferences.json')
}

function getUserPatternsPath(): string {
  const azureDir = join(os.homedir(), '.azure', 'logs-dashboard')
  return join(azureDir, 'patterns.json')
}

function getAppPatternsPath(projectRoot: string): string {
  return join(projectRoot, '.azure', 'logs-dashboard', 'patterns.json')
}

// Loading with Error Handling
async function loadPreferences(): Promise<UserPreferences> {
  try {
    const path = getUserPreferencesPath()
    const content = await fs.readFile(path, 'utf-8')
    const loaded = JSON.parse(content)
    
    // Validate schema and merge with defaults
    return mergeWithDefaults(loaded, DEFAULT_PREFERENCES)
  } catch (err) {
    if (err instanceof Error && err.message.includes('ENOENT')) {
      // File doesn't exist - return defaults
      return DEFAULT_PREFERENCES
    }
    console.warn('Failed to load preferences:', err)
    return DEFAULT_PREFERENCES
  }
}

// Saving with Atomic Write
async function savePreferences(prefs: UserPreferences): Promise<void> {
  try {
    const path = getUserPreferencesPath()
    const dir = dirname(path)
    
    // Ensure directory exists
    await fs.mkdir(dir, { recursive: true, mode: 0o750 })
    
    // Atomic write: temp file -> rename
    const tempPath = path + '.tmp'
    const content = JSON.stringify(prefs, null, 2)
    await fs.writeFile(tempPath, content, { mode: 0o644 })
    await fs.rename(tempPath, path)
  } catch (err) {
    console.error('Failed to save preferences:', err)
    throw err
  }
}

// Pattern Loading with Merge
async function loadPatterns(projectRoot: string): Promise<Pattern[]> {
  const userPatterns: Pattern[] = []
  const appPatterns: Pattern[] = []
  
  // Load user patterns
  try {
    const path = getUserPatternsPath()
    const content = await fs.readFile(path, 'utf-8')
    const { patterns } = JSON.parse(content)
    userPatterns.push(...patterns.map(p => ({ ...p, source: 'user' })))
  } catch (err) {
    // Ignore if not found
  }
  
  // Load app patterns
  try {
    const path = getAppPatternsPath(projectRoot)
    const content = await fs.readFile(path, 'utf-8')
    const { patterns } = JSON.parse(content)
    appPatterns.push(...patterns.map(p => ({ ...p, source: 'app' })))
  } catch (err) {
    // Ignore if not found
  }
  
  // Merge: app patterns override user patterns by ID
  const merged = new Map<string, Pattern>()
  userPatterns.forEach(p => merged.set(p.id, p))
  appPatterns.forEach(p => merged.set(p.id, p))  // Overrides
  
  return Array.from(merged.values()).sort((a, b) => 
    b.createdAt.localeCompare(a.createdAt)  // Newest first
  )
}

// Deep merge with defaults
function mergeWithDefaults<T>(partial: Partial<T>, defaults: T): T {
  return {
    ...defaults,
    ...Object.fromEntries(
      Object.entries(partial || {}).filter(([, v]) => v !== undefined)
    )
  } as T
}
```

### React Hook for Configuration

```typescript
// useLogsConfiguration.ts
import { useState, useEffect } from 'react'

export function useLogsConfiguration() {
  const [preferences, setPreferences] = useState<UserPreferences | null>(null)
  const [patterns, setPatterns] = useState<Pattern[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  // Load configuration on mount
  useEffect(() => {
    async function load() {
      try {
        setIsLoading(true)
        const [prefs, pats] = await Promise.all([
          loadPreferences(),
          loadPatterns(window.location.pathname)  // or get from context
        ])
        setPreferences(prefs)
        setPatterns(pats)
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Unknown error'))
      } finally {
        setIsLoading(false)
      }
    }
    void load()
  }, [])

  // Save preferences with debounce
  const savePreferencesDebounced = useMemo(
    () => debounce(async (prefs: UserPreferences) => {
      try {
        await savePreferences(prefs)
        setPreferences(prefs)
      } catch (err) {
        console.error('Failed to save preferences:', err)
      }
    }, 500),
    []
  )

  return {
    preferences: preferences || DEFAULT_PREFERENCES,
    patterns,
    isLoading,
    error,
    savePreferences: savePreferencesDebounced,
    addPattern: async (pattern: Pattern) => {
      const updated = [...patterns, pattern]
      setPatterns(updated)
      // Save to file via API
    },
    deletePattern: async (id: string) => {
      const updated = patterns.filter(p => p.id !== id)
      setPatterns(updated)
      // Save to file via API
    }
  }
}
```

### API Endpoints (Backend)

The dashboard needs backend support to read/write config files:

```typescript
// GET /api/logs/preferences
// Returns current user preferences
// 200: UserPreferences

// POST /api/logs/preferences
// Update user preferences
// Body: Partial<UserPreferences>
// 200: UserPreferences

// GET /api/logs/patterns
// Get merged user + app patterns
// Query: ?projectRoot=/path/to/project
// 200: Pattern[]

// POST /api/logs/patterns
// Add pattern to user or app level
// Body: { pattern: Pattern, level: 'user' | 'app' }
// 201: Pattern

// DELETE /api/logs/patterns/:id
// Delete pattern (from which level?)
// 200: Pattern

// PATCH /api/logs/patterns/:id
// Update pattern
// Body: Partial<Pattern>
// 200: Pattern
```

## Best Practices

### 1. Schema Versioning
- Include `"version": "1.0"` in all config files
- When schema changes, bump version and provide migration
- Example: v1.0 → v1.1 adds new field with default value

### 2. Error Handling
- Always provide sensible defaults if config missing
- Log warnings but don't crash if files corrupt
- Show user warning UI: "Could not load preferences, using defaults"
- Provide UI to "Reset to Defaults" button

### 3. Atomic Writes
- Always write to temp file first, then rename
- Prevents corruption if process crashes mid-write
- Critical for preferences.json (frequently updated)

### 4. File Permissions
- `0o750` for directories (user rwx, group rx, others none)
- `0o644` for config files (user rw, group r, others r)
- `0o600` for sensitive files (user rw only)

### 5. Merging Logic
- App config > User config > Defaults
- ID-based merging for patterns (same ID in both = app wins)
- Deep merge for nested objects (don't lose sibling keys)

### 6. Change Detection
- Use file watching in development mode
- Reload config when files change on disk
- Show "Config reloaded" toast in UI

### 7. User Privacy
- Never store sensitive data (tokens, passwords) in config
- Config files are plain text and version-controlled
- Anonymize if exporting logs that contain config-related info

## Testing Configuration

```typescript
// Example test
describe('Configuration Management', () => {
  it('merges app patterns over user patterns', async () => {
    // Setup: create user and app pattern files
    const userPatterns = [{ id: 'p1', name: 'User Pattern' }]
    const appPatterns = [{ id: 'p1', name: 'App Pattern' }]  // Same ID
    
    // Load
    const result = await loadPatterns(projectRoot)
    
    // Assert: app pattern wins
    expect(result.find(p => p.id === 'p1')).toHaveProperty('name', 'App Pattern')
    expect(result).toHaveLength(1)  // Not duplicated
  })

  it('saves preferences atomically', async () => {
    const prefs = { ...DEFAULT_PREFERENCES, ui: { gridColumns: 4 } }
    await savePreferences(prefs)
    
    // Verify file exists and contains correct data
    const loaded = await loadPreferences()
    expect(loaded.ui.gridColumns).toBe(4)
  })
})
```

## Environment Variable Overrides (Optional)

For advanced users, allow environment variables to override config:

```typescript
function getEffectivePreferences(): UserPreferences {
  const filePrefs = loadPreferencesSync()
  
  // Environment variable overrides
  const envOverrides = {
    gridColumns: process.env.LOGS_GRID_COLUMNS ? 
      parseInt(process.env.LOGS_GRID_COLUMNS) : undefined,
    viewMode: process.env.LOGS_VIEW_MODE as 'grid' | 'unified' | undefined,
  }
  
  return mergeWithDefaults(envOverrides, filePrefs)
}
```

Environment variables:
- `LOGS_GRID_COLUMNS`: 1-6
- `LOGS_VIEW_MODE`: grid|unified
- `LOGS_PATTERN_SOURCE`: user|app|both

---

## Migration Path

If migrating from localStorage to files:

```typescript
async function migrateFromLocalStorage() {
  const stored = localStorage.getItem('logs-dashboard-preferences')
  if (stored) {
    const prefs = JSON.parse(stored)
    await savePreferences(prefs)
    localStorage.removeItem('logs-dashboard-preferences')
    console.log('Migrated preferences from localStorage to ~/.azure/')
  }
}
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Permission denied" on save | Check `~/.azure/` permissions (should be 0o750) |
| Config not persisting | Check if file permissions are 0o644 |
| Old settings still showing | Clear browser cache + reload |
| Pattern not matching | Test regex in settings UI, enable debug logging |
| App patterns not loading | Check `.azure/logs-dashboard/` exists and path is correct |
