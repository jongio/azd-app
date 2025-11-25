# Azure CLI & Azure Developer CLI Configuration Storage Research

## Research Overview

This document outlines the findings on how Azure CLI and Azure Developer CLI (azd) store user configuration and settings, including per-user storage locations, project-level storage, file formats, and best practices from the azd ecosystem.

---

## 1. Per-User Storage Locations

### Windows
**Primary Location**: `%APPDATA%\.azure` (typically `C:\Users\<username>\AppData\Roaming\.azure`)

**Alternative Search Paths** (from azd codebase):
- `%LOCALAPPDATA%\Programs\Python` (for Python installations)
- `%APPDATA%\npm` (for npm installations)
- `%USERPROFILE%\AppData\Roaming\npm` (alternate npm location)

**Registry Paths** (for system PATH):
- `HKCU:\Environment\Path` (User PATH in Windows Registry)
- `HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment\Path` (Machine PATH)

### Linux/macOS
**Primary Location**: `~/.azure` (user home directory)

**Additional Cache/Package Locations**:
- `~/.local/bin` (user local executables)
- `~/.cargo/bin` (Rust/Cargo binaries)
- `~/.pnpm-store` (pnpm package cache)
- `~/.npm` (npm cache)
- `~/.cache/pip` (Python pip cache)
- `~/.cache/uv` (uv package cache)
- `~/.cache/pypoetry` (Poetry cache)
- `~/.nuget/packages` (.NET NuGet cache)

---

## 2. Per-App/Project Storage Location

### `.azure` Directory (Project-Level)

azd uses a **project-relative `.azure` directory** that takes precedence over user-level settings:

```
project-root/
├── azure.yaml
├── .azure/                    ← Project-level storage
│   ├── cache/
│   │   └── reqs_cache.json   (requirement checks cache)
│   ├── registry/
│   │   └── services.json     (running services registry)
│   └── ports.json            (port assignments for services)
└── services/
```

**Key Implementation Detail** (from `reqs_cache.go`):
- The cache manager searches for `.azure` directory starting from current directory up to parent directories
- It **stops at the home directory to avoid finding the user's global `.azure`**
- If no `.azure` is found in the hierarchy, it creates one in the current working directory
- Directory permissions: `0750` (read/write/execute for owner, read/execute for group)
- File permissions: `0600` (read/write for owner only) for sensitive data

### Directory Hierarchy Search Algorithm

```go
findAzureDir(startDir string) → string
1. Start at startDir
2. Check for .azure/ in current directory
3. If found, return it
4. Move to parent directory
5. Stop at home directory boundary (to avoid cross-contamination)
6. Stop at filesystem root
7. Return empty string if not found
```

---

## 3. File Format Conventions

### JSON Format (Primary)

#### a. **reqs_cache.json** (Requirements Cache)
```json
{
  "version": "1.0",
  "timestamp": "2025-01-15T10:30:45Z",
  "azureYamlHash": "sha256hash...",
  "results": [
    {
      "name": "node",
      "installed": true,
      "version": "18.0.0",
      "required": ">=16.0.0",
      "satisfied": true,
      "running": true,
      "checkedRunning": true,
      "message": ""
    }
  ],
  "allPassed": true
}
```

**Purpose**: Caches prerequisite tool checks with SHA256 hash of `azure.yaml` to invalidate cache when configuration changes

**Cache Strategy**:
- Default TTL: 1 hour
- Hash-based invalidation: Cache is only valid if `azure.yaml` hasn't changed
- Version-based invalidation: Schema version mismatch invalidates cache
- Atomic write: Uses temporary file + rename pattern to prevent corruption

#### b. **ports.json** (Port Assignments)
```json
{
  "serviceA": 3000,
  "serviceB": 3001,
  "database": 5432
}
```

**Purpose**: Persistently stores port assignments to maintain consistency across runs

#### c. **services.json** (Service Registry)
```json
{
  "services": [
    {
      "name": "api",
      "status": "running",
      "port": 3000,
      "url": "http://localhost:3000"
    }
  ]
}
```

**Purpose**: Tracks running services discovered in the environment

### YAML Format (Project Configuration)

#### **azure.yaml** (Root Configuration)
```yaml
name: my-app
metadata:
  description: My application
  version: 1.0.0

# Service definitions
services:
  api:
    language: python
    project: ./api
    host: localhost
    ports:
      - "3000"
    environment:
      NODE_ENV: development
      DATABASE_URL: postgresql://localhost:5432/db

# Lifecycle hooks
hooks:
  prerun:
    run: ./scripts/setup.sh
    shell: bash
    continueOnError: false
  postrun:
    run: echo "Ready"
    shell: sh

# Prerequisites
reqs:
  - name: node
    minVersion: "16.0.0"
  - name: docker
    minVersion: "20.0.0"
```

**File Permissions**: `0644` (readable by team)

### Environment Files

#### **.env** (Plain Text Key=Value)
```bash
# Database configuration
DATABASE_URL=postgresql://localhost:5432/mydb
DATABASE_POOL_SIZE=20

# API keys (sensitive - should be .gitignored)
OPENAI_API_KEY=sk-xyz123
STRIPE_API_KEY=sk_test_abc456

# Application settings
LOG_LEVEL=debug
ENABLE_METRICS=true
```

**Best Practice**: Keep in `.gitignore`, use for local secrets only

#### **local.settings.json** (Azure Functions/Logic Apps)
```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "node",
    "Workflows.AIChatWorkflow.FlowState": "Enabled",
    "AI_FOUNDRY_NAME": "your-ai-foundry-instance"
  }
}
```

**Purpose**: Azure Functions and Logic Apps local configuration

---

## 4. Existing Configuration Examples in azd

### Dashboard Settings (Browser-Based)

**Storage Method**: `localStorage`

```typescript
// From App.tsx
const saved = localStorage.getItem('dashboard-view-preference')
localStorage.setItem('dashboard-view-preference', viewMode)
```

**Keys Stored**:
- `dashboard-view-preference`: 'cards' | 'table' (view mode)

**Scope**: Per-browser, session-based

**Configuration Patterns for Logs Dashboard** (planned feature):
```javascript
// Pattern storage in localStorage
localStorage.setItem('false-positive-patterns', JSON.stringify([
  { name: "zero-errors", regex: "0 errors" },
  { name: "error-rate-zero", regex: "error rate: 0%" }
]))

// Per-pane state
localStorage.setItem('pane-state-{serviceName}', JSON.stringify({
  searchTerm: "",
  logLevel: "all",
  markedFalsePositives: [1, 5, 10],
  markedFalseNegatives: [3, 7]
}))
```

### Environment Variable Management

**Priority Order** (highest to lowest):
1. Service-specific `environment` (from `azure.yaml`)
2. `.env` file variables
3. Azure environment variables (from `azd env get-values`)
4. OS environment variables

**Implementation**:
```go
// From environment.go
func ResolveEnvironment(
  service *Service,
  azureEnv map[string]string,
  dotEnvPath string,
  serviceURLs map[string]string,
) map[string]string {
  // Merge in priority order
}
```

### Service Registry

**Location**: `.azure/registry/services.json`

**Content**:
```json
{
  "services": [
    {
      "name": "web",
      "port": 3000,
      "status": "running",
      "language": "javascript",
      "entrypoint": "npm start"
    },
    {
      "name": "api",
      "port": 5000,
      "status": "running",
      "language": "python",
      "entrypoint": "python app.py"
    }
  ]
}
```

### Cache Management

**Location**: `.azure/cache/`

**Files**:
- `reqs_cache.json`: Cached requirement checks (1 hour TTL)
- Additional caches can be added per feature

**Statistics Tracking**:
```json
{
  "hits": 42,
  "misses": 8
}
```

---

## 5. Best Practices for Tool-Specific Settings Persistence

### Directory Structure Best Practices

✅ **DO**:
1. Use `.azure/` as project-level storage root
2. Organize by feature/function: `.azure/cache/`, `.azure/registry/`, `.azure/settings/`
3. Create directories with `0750` permissions (owner: rwx, group: rx)
4. Write files with `0600` permissions (owner: rw only)
5. Use atomic writes (temp file → rename) for critical files
6. Include schema versioning for breaking changes

❌ **DON'T**:
1. Store sensitive data without file permission protections
2. Write directly to config files without atomic operations
3. Search into user's home `~/.azure` from project context
4. Mix per-user and per-project settings without clear boundary

### Configuration Versioning

```go
// Schema version for cache invalidation
const CacheVersion = "1.0"

// Include version in cache files
type ReqsCache struct {
  Version       string    `json:"version"`
  Timestamp     time.Time `json:"timestamp"`
  // ...
}

// Check version and invalidate if mismatch
if cache.Version != CacheVersion {
  // Cache miss - file is stale
}
```

### Atomic Write Pattern

```go
// Write to temporary file first
tempFile := cacheFile + ".tmp"
if err := os.WriteFile(tempFile, data, 0600); err != nil {
  return err
}

// Atomic rename prevents corruption on concurrent writes
if err := os.Rename(tempFile, cacheFile); err != nil {
  os.Remove(tempFile) // Clean up on error
  return err
}
```

### Hash-Based Cache Invalidation

```go
// Calculate config file hash
func calculateFileHash(filePath string) (string, error) {
  file, err := os.Open(filePath)
  if err != nil {
    return "", err
  }
  defer file.Close()
  
  hasher := sha256.New()
  io.Copy(hasher, file)
  return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// Check if config changed
if cache.AzureYamlHash != currentHash {
  // Config changed - cache invalid
}
```

### File Permission Management

**Directory Permissions**: `0750`
- Owner: read, write, execute
- Group: read, execute
- Others: none
- Rationale: Allow tool processes to access, prevent others from viewing

**Sensitive File Permissions**: `0600`
- Owner: read, write
- Group: none
- Others: none
- Rationale: Prevent exposure of API keys, tokens, etc.

**General File Permissions**: `0644`
- Owner: read, write
- Group: read
- Others: read
- Rationale: Allow team collaboration on project configs

### Configuration Layering

```yaml
# Layer 1: Project defaults (tracked in git)
azure.yaml: Explicit configuration

# Layer 2: User overrides (not tracked)
.env: Local environment variables

# Layer 3: Azure environment
azd env get-values: Dynamic values from provisioned resources

# Layer 4: OS environment
$env:VAR or $VAR: System-wide configuration
```

**Merge Strategy**: Lower layer overrides higher (reverse priority)

---

## 6. Azure CLI Integration Patterns

### Azure CLI Configuration Storage

**Standard Locations**:
- Windows: `%APPDATA%\.azure` or `%USERPROFILE%\.azure`
- Linux/macOS: `~/.azure`

**Azure CLI Config Files** (not used by azd directly, but reference):
- `config`: INI format configuration
- `clouds.config`: Cloud environment definitions
- `commands_modules`: Module cache

### Azure Developer CLI (azd) Specific

azd builds on Azure CLI patterns but adds:

1. **Extension Manifest** (from `extension.yaml`):
   ```yaml
   name: app
   description: "Orchestrate local development services"
   version: 1.0.0
   ```

2. **Configuration via `azd config set`**:
   ```bash
   azd config set alpha.extension.enabled on
   azd config set extension.registry https://registry.json
   ```

3. **Hierarchical Project Discovery**:
   - Searches for `azure.yaml` starting from current directory up to root
   - Stops at `.git` directory or filesystem boundary
   - Avoids crossing into user home directory accidentally

---

## 7. Recommended Patterns for Dashboard Settings Persistence

### Option 1: Browser localStorage (Current - Session Only)
```typescript
// Pros: Simple, no backend needed, per-browser
// Cons: Lost on browser clear, no cross-device sync

localStorage.setItem('dashboard-view-preference', 'cards')
```

### Option 2: .azure/dashboard.json (Project-Level)
```json
{
  "version": "1.0",
  "lastView": "cards",
  "columnWidths": [200, 150, 300],
  "sortBy": "name",
  "paneLayout": "2-column"
}
```

**Recommended for team sharing**

### Option 3: ~/.azure/dashboard.json (User-Level)
```json
{
  "version": "1.0",
  "defaultView": "cards",
  "theme": "dark",
  "autoRefresh": true,
  "refreshInterval": 5000
}
```

**Recommended for personal preferences**

### Option 4: Hybrid Approach (Recommended)
```javascript
// 1. Load user defaults from ~/.azure/dashboard.json
// 2. Override with project settings from .azure/dashboard.json
// 3. Override with browser localStorage preferences
// 4. Save changes to appropriate level (localStorage by default for session)

const userDefaults = loadUserConfig('~/.azure/dashboard.json')
const projectDefaults = loadProjectConfig('.azure/dashboard.json')
const browserPrefs = loadBrowserPreferences()

const finalConfig = merge(userDefaults, projectDefaults, browserPrefs)
```

---

## 8. Summary Table: Configuration Storage Locations

| **Type** | **Location** | **Format** | **Scope** | **Permissions** | **TTL** |
|----------|------------|-----------|-----------|-----------------|---------|
| Cache (reqs) | `.azure/cache/reqs_cache.json` | JSON | Project | `0600` | 1 hour |
| Ports | `.azure/ports.json` | JSON | Project | `0600` | Persistent |
| Registry | `.azure/registry/services.json` | JSON | Project | `0600` | Session |
| Project Config | `azure.yaml` | YAML | Project | `0644` | N/A |
| Local Env | `.env` | Text | Project | `0600` | Session |
| Azure Env | (from `azd env get-values`) | N/A | Azure | N/A | Session |
| Dashboard UI | `localStorage` | JSON | Browser | N/A | Session |
| Dashboard Pref | `~/.azure/dashboard.json` | JSON | User | `0600` | Persistent |
| Dashboard Proj | `.azure/dashboard.json` | JSON | Project | `0600` | Persistent |

---

## 9. File Permission Consistency Notes

**Current Issue in azd** (from `todo.md`):
- `ServiceRegistry` creates `.azure/` with `0750` but writes files with `0600`
- `PortManager` creates `.azure/` with `0700`
- **Inconsistency**: Different directory permissions across features

**Recommended Standard**:
- **Directories**: `0750` (owner: rwx, group: rx, others: none)
- **Sensitive Files**: `0600` (owner: rw only)
- **Shared Config**: `0644` (readable by all)

---

## References

**Codebase Locations**:
- Cache implementation: `cli/src/internal/cache/reqs_cache.go`
- Path utilities: `cli/src/internal/pathutil/pathutil.go`
- Environment handling: `cli/src/internal/service/environment.go`
- Dashboard: `cli/dashboard/src/`
- Schema: `cli/docs/schema/azure.yaml.md`

**Key Functions**:
- `findAzureDir()`: Locates project-relative `.azure` directory
- `NewCacheManagerWithOptions()`: Initializes cache with custom directory
- `SearchToolInSystemPath()`: Searches common installation directories
- `ResolveEnvironment()`: Merges environment variables by priority
