# Azure Configuration Storage Research & Best Practices

## Overview

This document captures research on how Azure CLI and Azure Developer CLI store configuration, used to inform the logs dashboard specification.

## Key Findings

### 1. Two-Level Storage Architecture

**Why Two Levels?**
- **User Level** (`~/.azure/`): Personal preferences, credentials, settings
- **Project Level** (`.azure/`): Team-shared configuration, can commit to repo

This pattern allows:
- Teams to share common settings (patterns, defaults)
- Individuals to override with personal preferences
- Clear precedence: Project config > User config > Hardcoded defaults

### 2. Directory Structure

**User Home Directory** (`~/.azure/`)
```
~/.azure/
├── config                         # Azure CLI account settings
├── clouds.config                  # Cloud configuration
├── logs-dashboard/               # (NEW) Logs dashboard configs
│   ├── preferences.json          # User's UI preferences
│   └── patterns.json             # User's pattern definitions
└── <other tool dirs>/            # Other tools follow this pattern
```

**Project Directory** (`.azure/`)
```
.azure/
├── .gitignore                    # Exclude sensitive local files
├── reqs-cache.json               # (in cache/) Deps check results
├── services-running.json         # (in cache/) Running services registry
├── ports.json                    # Port assignments
├── logs-dashboard/               # (NEW) Team-shared dashboard config
│   ├── config.json               # Team defaults
│   ├── patterns.json             # Shared false positive patterns
│   └── .gitignore                # Protect any sensitive files
└── <other files>/                # Other project configs
```

### 3. File Formats & Conventions

**JSON Format Standards**:
- All structured data uses JSON (not YAML for configs)
- Schema versioning: always include `"version": "1.0"`
- Comments: Use description fields instead of comments (JSON doesn't support them)
- Newlines: Unix-style (`\n`), not Windows (`\r\n`)

**Example Structure** (from azd reqs cache):
```json
{
  "version": "1.0",
  "data": [
    {
      "id": "requirement-001",
      "name": "node",
      "status": "ok|pending|failed",
      "timestamp": "2025-11-23T10:00:00Z"
    }
  ],
  "metadata": {
    "lastChecked": "2025-11-23T10:30:00Z",
    "checksum": "sha256:abc123..."
  }
}
```

### 4. File Permissions

**Unix Permissions**:
- Directories: `0o750` (user rwx, group rx, others none)
  - Allows multiple users in same group to access
  - Prevents world access
- Config files: `0o644` (user rw, group r, others r)
  - Non-sensitive team configs
- Sensitive files: `0o600` (user rw only)
  - Tokens, credentials, private keys

**Windows**: Inherits from directory ACLs (equivalent to above)

### 5. Cache Invalidation & Versioning

**Approaches Observed in azd**:

**Approach 1: Hash-Based (Reqs Cache)**
```json
{
  "version": "1.0",
  "configChecksum": "sha256:abc123def456",
  "cachedResults": [ ... ]
}
```
- When azure.yaml changes, hash changes
- Cache invalidates automatically
- No stale data issues

**Approach 2: Timestamp-Based (Services Registry)**
```json
{
  "lastUpdated": "2025-11-23T10:30:00Z",
  "services": [ ... ]
}
```
- More human-readable
- Manual invalidation if needed

**For Logs Dashboard**: Use timestamp for patterns (patterns can be updated independently)

### 6. How azd Actually Uses `.azure/`

**Real Examples from Codebase**:

**Cache Manager** (`cli/src/internal/cache/reqs_cache.go`):
```go
// findAzureDir searches for .azure directory in current and parent directories.
// It stops at filesystem boundaries and does not search in user home directory.
func findAzureDir(startDir string) string {
    dir := startDir
    homeDir, _ := os.UserHomeDir()
    
    for {
        azureDir := filepath.Join(dir, ".azure")
        if info, err := os.Stat(azureDir); err == nil && info.IsDir() {
            return azureDir
        }
        
        parent := filepath.Dir(dir)
        if parent == dir {
            break  // Reached root
        }
        
        // Stop at home directory - don't go into user's global .azure
        if homeDir != "" && parent == homeDir {
            break
        }
        
        dir = parent
    }
    return ""
}
```

**Key Insight**: 
- Searches from current dir UP to project root
- Stops at home dir (doesn't cross project boundaries)
- This is the **preferred behavior for team settings**

### 7. User Settings Location (Non-Project)

**Azure CLI** uses `~/.azure/` for all user-level settings:
- Account info (`config`, `clouds.config`)
- Cloud configurations
- Any tool-specific user data

**Pattern for Logs Dashboard**:
- Individual preferences: `~/.azure/logs-dashboard/preferences.json`
- Individual patterns (defaults): `~/.azure/logs-dashboard/patterns.json`

**Why this works**:
- User can have different preferences on different machines
- Separate from project (no commits needed)
- Portable between projects on same machine

---

## Storage Decision Matrix

| Setting | Location | Commit to Repo? | Example |
|---------|----------|-----------------|---------|
| User preferences (grid columns) | `~/.azure/logs-dashboard/preferences.json` | NO | gridColumns: 4 |
| Team patterns (false positives) | `.azure/logs-dashboard/patterns.json` | YES | "0 errors" pattern |
| App defaults | `.azure/logs-dashboard/config.json` | YES | defaultServices: ["web"] |
| User patterns (personal) | `~/.azure/logs-dashboard/patterns.json` | NO | User's custom patterns |
| Sensitive data | N/A | NEVER | API keys, credentials |

---

## File Access Patterns

### Backend Requirements

**Express/Node.js Backend**:
```typescript
import { promises as fs } from 'fs'
import os from 'os'
import path from 'path'

// User level
const userConfigDir = path.join(os.homedir(), '.azure', 'logs-dashboard')

// Project level (needs to know project root)
const projectConfigDir = path.join(projectRoot, '.azure', 'logs-dashboard')
```

**Key Consideration**: 
- Backend runs with same user as CLI
- Can read/write to `~/.azure/` and project `.azure/`
- Need to handle permission errors gracefully

### Cross-Platform Compatibility

| Platform | Home Dir | Separator | Paths Work? |
|----------|----------|-----------|------------|
| Linux | `~/.azure/` | `/` | ✅ Yes |
| macOS | `~/.azure/` | `/` | ✅ Yes |
| Windows | `%APPDATA%\.azure\` | `\` | ✅ Yes (use path module) |

**Implementation**: Always use `path.join()` or `os.path.join()` - never hardcode slashes.

---

## Configuration Loading Strategy

### Precedence Order (Highest to Lowest)

1. **Environment Variables** (if supported)
   ```bash
   LOGS_VIEW_MODE=grid LOGS_GRID_COLUMNS=4
   ```

2. **Project Config** (`.azure/logs-dashboard/config.json`)
   - Team has decided on defaults
   - Dev commits this to repo

3. **User Config** (`~/.azure/logs-dashboard/preferences.json`)
   - Individual developer preference
   - Portable across projects

4. **Hardcoded Defaults**
   - gridColumns: 2
   - viewMode: "grid"
   - paneHeight: 500

### Implementation Pattern

```typescript
function loadConfiguration(): Config {
  // 1. Start with defaults
  const config = { ...DEFAULTS }
  
  // 2. Overlay project config
  try {
    const projectConfig = loadJSON('.azure/logs-dashboard/config.json')
    Object.assign(config, projectConfig)
  } catch (e) {
    // Project config optional
  }
  
  // 3. Overlay user config (always wins over project)
  try {
    const userConfig = loadJSON('~/.azure/logs-dashboard/preferences.json')
    Object.assign(config, userConfig)
  } catch (e) {
    // User config optional
  }
  
  // 4. Overlay environment variables (always win)
  if (process.env.LOGS_GRID_COLUMNS) {
    config.gridColumns = parseInt(process.env.LOGS_GRID_COLUMNS)
  }
  
  return config
}
```

---

## Patterns & False Positive Management

### Pattern File Format (Standardized)

**Reasoning**: Each pattern needs metadata for UI management
- `id`: Unique identifier (for editing/deleting)
- `name`: Human-readable name
- `regex`: The actual pattern
- `description`: What it matches
- `enabled`: Can disable without deleting
- `source`: Track origin ("user" vs "app")
- `createdAt`: Timestamp for sorting

### Pattern Merging (for Combined View)

**Scenario**: User has personal patterns + team shares patterns

**Solution**:
```typescript
async function getMergedPatterns(projectRoot: string): Promise<Pattern[]> {
  const userPatterns = await loadUserPatterns()
  const projectPatterns = await loadProjectPatterns(projectRoot)
  
  // Use Map to detect duplicates by ID
  const merged = new Map<string, Pattern>()
  
  // Add user patterns first
  userPatterns.forEach(p => merged.set(p.id, { ...p, source: 'user' }))
  
  // Add project patterns (override if same ID)
  projectPatterns.forEach(p => merged.set(p.id, { ...p, source: 'app' }))
  
  return Array.from(merged.values())
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
}
```

**UI Display**: Show both "User Patterns" tab and "Team Patterns" tab separately

---

## Error Handling Best Practices

### Graceful Degradation

```typescript
async function safeLoadConfig(filePath: string): Promise<Config> {
  try {
    return JSON.parse(await fs.readFile(filePath, 'utf-8'))
  } catch (error) {
    if (error.code === 'ENOENT') {
      // File doesn't exist - return defaults
      logger.debug(`Config file not found: ${filePath}, using defaults`)
      return DEFAULTS
    } else if (error instanceof SyntaxError) {
      // JSON parse error - log warning
      logger.warn(`Config file malformed: ${filePath}, using defaults`, error)
      return DEFAULTS
    } else {
      // Permission or other filesystem error
      logger.error(`Failed to read config: ${filePath}`, error)
      return DEFAULTS
    }
  }
}
```

### UI Feedback

- Show warning badge if config fails to load: "⚠ Using defaults"
- Button to "Reset to Defaults"
- Toast notification on successful save: "✓ Preferences saved"
- Toast on save error: "✗ Failed to save (check permissions)"

---

## Recommendations for Logs Dashboard

### Phase 1 Implementation

✅ **Use**: Two-level storage (user + project)
✅ **Use**: File-based JSON configs (not localStorage)
✅ **Use**: azd's precedence pattern (project > user > defaults)
✅ **Use**: Atomic writes (temp file + rename)
✅ **Use**: Schema versioning (version: "1.0")

### Phase 2+ Enhancements

- Consider environment variable overrides
- Add configuration UI to edit patterns in dashboard
- Export/import configs for team sharing
- Consider git-friendly YAML for team configs (v2.0)

### Avoid

❌ **Don't use**: Cookies (can be lost, browser-specific)
❌ **Don't use**: Local Storage (not team-shareable)
❌ **Don't use**: IndexedDB (not portable, not human-readable)
❌ **Don't store**: Credentials or secrets anywhere in config

---

## Testing Configuration Management

### Test Scenarios

```typescript
describe('Configuration Management', () => {
  
  it('loads defaults when files missing', async () => {
    // Setup: ensure no config files exist
    // Execute: loadConfiguration()
    // Assert: returns DEFAULTS
  })
  
  it('project config overrides user config', async () => {
    // Setup: create both files with different gridColumns
    // Execute: loadConfiguration(projectRoot)
    // Assert: project value wins
  })
  
  it('merges patterns without duplication', async () => {
    // Setup: user has pattern id=p1, project has id=p1
    // Execute: getMergedPatterns()
    // Assert: length=1, source=app (project wins)
  })
  
  it('saves preferences atomically', async () => {
    // Setup: mock fs.rename to throw
    // Execute: savePreferences(prefs)
    // Assert: temp file cleanup happened
  })
})
```

---

## References

### azd Codebase References
- **Cache Manager**: `cli/src/internal/cache/reqs_cache.go`
- **Service Info**: `cli/src/internal/service/environment.go`
- **Port Management**: `cli/src/internal/service/port_integration_test.go`

### Azure CLI Documentation
- [Azure CLI Configuration](https://learn.microsoft.com/en-us/cli/azure/azure-cli-configuration)
- [Default Values](https://learn.microsoft.com/en-us/cli/azure/azure-cli-configuration#defaults)

### Best Practices
- File permissions: [Unix File Permissions](https://en.wikipedia.org/wiki/File_system_permissions)
- Atomic writes: [atomic file writes pattern](https://en.wikipedia.org/wiki/Atomic_operation)
- JSON schema: [JSON Schema standard](https://json-schema.org/)
