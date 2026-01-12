# Spec: Azure YAML Editor for Dashboard

## Summary

A modern, schema-driven visual editor for `azure.yaml` files integrated into the azd app dashboard. This editor provides an intuitive interface for creating and modifying Azure YAML configurations with full support for azd and azd app features, including intelligent service addition (azurite, cosmos, postgres, redis, etc.), schema validation, autocomplete, and automatic backup management.

## Motivation

Currently, users must manually edit `azure.yaml` files in text editors, which presents several challenges:

1. **Steep Learning Curve**: Understanding the azure.yaml schema structure, required vs optional fields, and proper YAML syntax requires extensive documentation reading
2. **Error-Prone**: Manual YAML editing is prone to indentation errors, typos in property names, and invalid value types
3. **Discovery Gap**: Users may not know about available azd app features like service types, modes, health checks, or the `azd app add` command
4. **No Validation**: Errors are only discovered when running azd commands, creating a slow feedback loop
5. **Missing Context**: No inline help or examples showing how to configure specific Azure services
6. **Limited Lookups**: Users must reference documentation to find valid values for enums (host types, languages, service types, etc.)

A visual editor solves these problems by:
- Providing guided form-based editing with inline validation
- Offering intelligent autocomplete based on JSON Schema
- Surfacing all available azd app features through UI affordances
- Enabling one-click service addition with proper defaults
- Automatically backing up files before changes
- Reducing context switching between documentation and implementation

## User Experience

### Entry Points

1. **Dashboard Home**: Prominent "Edit Configuration" button in header or service panel
2. **Service Card**: "Edit Service" action on individual service cards
3. **Settings**: "Configuration Editor" option in settings menu
4. **Command Palette**: "Edit azure.yaml" command

### Editor Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  Azure YAML Editor                               [Save] [Cancel] │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐  ┌──────────────────────────────────────────┐ │
│  │  Navigation  │  │             Editor Pane                   │ │
│  │              │  │                                           │ │
│  │ • Overview   │  │  ┌────────────────────────────────────┐  │ │
│  │ • Services   │  │  │  Property Name        Value        │  │ │
│  │   - api      │  │  │  ────────────────────────────────  │  │ │
│  │   - web      │  │  │  Application Name:  [my-app___]    │  │ │
│  │   - cosmos   │  │  │                                    │  │ │
│  │ • Resources  │  │  │  Resource Group:    [rg-dev____]   │  │ │
│  │ • Hooks      │  │  │                                    │  │ │
│  │ • Pipeline   │  │  │  [+] Add Metadata                  │  │ │
│  │ • Metadata   │  │  └────────────────────────────────────┘  │ │
│  │              │  │                                           │ │
│  │ [+ Add       │  │  ┌─────── Service: api ────────────┐    │ │
│  │  Service]    │  │  │  Host: [containerapp ▼]         │    │ │
│  │              │  │  │  Language: [node ▼]             │    │ │
│  │              │  │  │  Project: [./src/api_____]      │    │ │
│  │              │  │  │                                  │    │ │
│  │              │  │  │  Ports: [8080________] [+ Add]  │    │ │
│  │              │  │  │                                  │    │ │
│  │              │  │  │  Environment Variables:          │    │ │
│  │              │  │  │  [+ Add Variable]                │    │ │
│  │              │  │  │                                  │    │ │
│  │              │  │  │  Health Check:                   │    │ │
│  │              │  │  │  [Configure Health Check...]     │    │ │
│  │              │  │  └──────────────────────────────────┘    │ │
│  │              │  │                                           │ │
│  └──────────────┘  └──────────────────────────────────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ 📋 Quick Actions                                             │ │
│  │ [+ Add Azurite] [+ Add Cosmos] [+ Add Redis] [+ Add Postgres]│ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ ⚠️ Validation: 2 warnings                                    │ │
│  │ • Service 'api' missing health check (recommended)           │ │
│  │ • Consider adding resource definitions for production        │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Key User Flows

#### 1. Adding a New Service (Quick Path)

```
User clicks "+ Add Service" button
  ↓
Modal appears with service type selection:
  - "Application Service" (custom code)
  - "Container Service" (pre-built containers)
  - "Well-Known Service" (azurite, cosmos, redis, postgres)
  ↓
User selects "Well-Known Service"
  ↓
Visual grid shows available services with icons:
  [Azurite Storage]  [Cosmos DB]  [Redis Cache]  [PostgreSQL]
  ↓
User clicks "Cosmos DB"
  ↓
Editor automatically populates:
  - Service name: "cosmos"
  - Host: "containerapp"
  - Image: "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest"
  - Ports: 8081, 10250-10254
  - Environment variables: PARTITION_COUNT, etc.
  - Health check configuration
  - Connection string template shown in info panel
  ↓
User clicks "Add Service"
  ↓
Service appears in navigation + editor updates
  ↓
Automatic backup created: azure.yaml.backup.{timestamp}
  ↓
Success notification: "✓ Added Cosmos DB emulator"
```

#### 2. Editing an Existing Service

```
User clicks service name in navigation (e.g., "api")
  ↓
Editor pane loads service configuration form
  ↓
User modifies port from "8080" to "3000"
  ↓
Real-time validation runs:
  - Schema validation passes
  - Port conflict check (warns if port in use)
  - Visual indicator: green checkmark
  ↓
User clicks "Save" button
  ↓
System creates backup: azure.yaml.backup.{timestamp}
  ↓
Writes new azure.yaml file
  ↓
Dashboard auto-reloads with new configuration
  ↓
Success notification: "✓ Configuration saved"
```

#### 3. Advanced Configuration (Environment Variables)

```
User navigates to service "api"
  ↓
Scrolls to "Environment Variables" section
  ↓
Clicks "+ Add Variable" button
  ↓
Inline form appears:
  - Name: [DATABASE_URL______]
  - Value: [________________]
  - Type: [Text ▼] (options: Text, Secret Reference, Expression)
  ↓
User enters:
  - Name: "DATABASE_URL"
  - Value: "postgresql://localhost:5432/app"
  - Type: "Text"
  ↓
Clicks "Add"
  ↓
Variable appears in list with inline edit/delete actions
  ↓
User clicks "Add Variable" again for secret:
  - Name: "API_KEY"
  - Type: "Secret Reference" → shows additional field
  - Secret Name: "MY_SECRET"
  ↓
Editor renders YAML appropriately:
  environment:
    - DATABASE_URL=postgresql://localhost:5432/app
    - name: API_KEY
      secret: MY_SECRET
```

## Goals

1. **Intuitive Editing**: Users can configure azure.yaml without YAML knowledge
2. **Schema-Driven**: All fields, validation, and help text derived from JSON Schema
3. **Zero Data Loss**: Automatic backups before any write operation
4. **Real-Time Validation**: Immediate feedback on configuration errors
5. **Service Discovery**: Surfacing all well-known services with one-click addition
6. **Intelligent Defaults**: Pre-populate configurations with sensible defaults
7. **Visual Appeal**: Modern, accessible interface following dashboard design system
8. **Performance**: Fast loading and responsive interactions (<100ms field updates)
9. **Accessibility**: WCAG AA compliant, keyboard navigable, screen reader friendly

## Non-Goals

1. **Direct YAML Editing**: Not a text editor (users can still use VS Code/other editors)
2. **Infrastructure Editing**: Not editing Bicep/Terraform files (only azure.yaml)
3. **Git Integration**: Not handling commits/pushes (separate concern)
4. **Multi-File Projects**: Only editing azure.yaml (not package.json, etc.)
5. **Advanced YAML Features**: Not supporting comments, anchors, aliases (preserve but don't edit)

## Functional Requirements

### FR-1: Schema Loading and Parsing

**Description**: Load and parse the azure.yaml JSON Schema to drive the editor UI

**Requirements**:
- Load schema from: `https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json`
- Support local schema bundling for offline use
- Parse schema definitions, properties, enums, patterns, descriptions
- Build internal model of:
  - Required vs optional fields
  - Field types (string, number, boolean, object, array)
  - Validation rules (min/max, pattern, enum)
  - Default values
  - Help text from descriptions
- Cache parsed schema in memory
- Handle schema fetch failures gracefully (use bundled fallback)

**Acceptance Criteria**:
- Schema loads successfully on editor open
- All properties from schema.properties are mapped to editor fields
- Enum values populate dropdown menus
- Validation rules enforce schema constraints
- Help tooltips display schema descriptions

---

### FR-2: File System Operations

**Description**: Read, backup, and write azure.yaml files safely

**Requirements**:
- Locate azure.yaml in current workspace root
- Read current azure.yaml content on editor open
- Parse YAML to JavaScript object
- Create backup before ANY write:
  - Format: `azure.yaml.backup.{ISO8601-timestamp}` (e.g., `azure.yaml.backup.2026-01-11T143055Z`)
  - Location: Same directory as azure.yaml
  - Preserve original file permissions
  - Maximum 10 most recent backups (auto-delete older)
- Write azure.yaml atomically:
  - Write to temp file first
  - Validate written content can be parsed
  - Atomic rename to azure.yaml
  - Preserve original line endings (LF vs CRLF)
- Handle file system errors:
  - Permission denied → show error modal with remediation
  - File not found → offer to create new azure.yaml
  - Disk full → prevent write, show error
- Preserve YAML formatting where possible:
  - Maintain key ordering
  - Preserve existing comments (best effort)
  - Use consistent indentation (2 spaces)

**Acceptance Criteria**:
- Editor loads existing azure.yaml content correctly
- Backup file created before first save
- Multiple backups accumulate with timestamps
- Old backups auto-deleted when >10 exist
- Write failures don't corrupt original file
- Saved YAML is valid and parseable

---

### FR-3: Navigation Structure

**Description**: Hierarchical navigation reflecting azure.yaml structure

**Requirements**:
- Left sidebar navigation panel (collapsible)
- Navigation tree structure:
  ```
  Overview (name, resourceGroup, metadata)
  Services
    ├─ service-name-1
    ├─ service-name-2
    └─ [+ Add Service]
  Resources
    ├─ resource-name-1
    ├─ resource-name-2
    └─ [+ Add Resource]
  Hooks
  Pipeline
  Required Versions
  State
  ```
- Highlight active section
- Show validation badge on sections with errors (red dot)
- Show warning badge on sections with warnings (yellow dot)
- Keyboard navigation (arrow keys + enter)
- Search/filter in navigation (Cmd/Ctrl+F in sidebar)
- Expand/collapse service list when >5 services
- Sticky header showing current section

**Acceptance Criteria**:
- All top-level azure.yaml properties appear in navigation
- Clicking navigation item loads that section in editor pane
- Active section highlighted in navigation
- Error/warning badges visible when validation issues exist
- Keyboard navigation works smoothly
- Search filters navigation items correctly

---

### FR-4: Form-Based Editing

**Description**: Dynamic form generation from schema for each configuration section

**Requirements**:
- Generate form fields based on schema property types:
  - **String**: Text input with pattern validation
  - **Number**: Numeric input with min/max
  - **Boolean**: Toggle switch or checkbox
  - **Enum**: Dropdown select with all valid values
  - **Object**: Nested fieldset with sub-properties
  - **Array**: Repeatable field group with add/remove
- Field components:
  - **Text Input**: Single-line for strings, pattern validation, placeholder
  - **Textarea**: Multi-line for long strings (>100 chars)
  - **Numeric Input**: Spinner controls, min/max enforcement
  - **Dropdown**: Searchable select for enums
  - **Toggle**: Modern switch UI for booleans
  - **Array Field**: List with reorderable items, add/remove buttons
  - **Object Field**: Expandable section with nested fields
- Field behaviors:
  - Auto-save on blur (debounced 500ms)
  - Real-time validation with inline error messages
  - Help tooltip icon (ⓘ) showing schema description
  - Required field indicator (*)
  - Default value suggestion (light gray placeholder)
  - Clear/reset button for non-required fields
- Conditional fields:
  - Show/hide fields based on other field values (e.g., docker.path only if docker exists)
  - Schema-driven conditional logic from `if/then/else` in JSON Schema
- Special field types:
  - **File Path**: Text input + browse button
  - **Port**: Numeric input with conflict detection
  - **Environment Variable**: Key-value pair editor with type selection

**Acceptance Criteria**:
- All schema properties render as appropriate form fields
- Required fields visually distinguished
- Field validation runs on blur
- Invalid fields show clear error messages
- Tooltips provide helpful context
- Conditional fields show/hide correctly
- Array fields support add/remove/reorder

---

### FR-5: Service Management

**Description**: Add, edit, and remove services with intelligent templates

**Requirements**:
- **Add Service Flow**:
  - Button: "+ Add Service" in navigation
  - Modal with three tabs:
    1. **Well-Known Services**: Grid of common services (azurite, cosmos, redis, postgres, etc.)
    2. **Application Service**: Form for custom code services
    3. **Container Service**: Form for Docker image services
  - Well-Known Services tab:
    - Load from `wellknown.ServiceDefinition` registry (via API)
    - Visual grid with icons, names, descriptions
    - Click to select → show preview of configuration
    - "Add Service" button applies full configuration
    - Auto-populated: name, host, image, ports, environment, healthcheck
  - Application Service tab:
    - Required: name, host, project path
    - Optional: language (auto-detected if omitted), ports, environment
  - Container Service tab:
    - Required: name, host, image
    - Optional: ports, environment, healthcheck
- **Edit Service**:
  - Click service in navigation → load form
  - All service properties editable
  - Delete service button (with confirmation)
- **Service Templates**:
  - Pre-fill based on host type:
    - `containerapp` → suggest ports, environment
    - `function` → suggest language, triggers
    - `appservice` → suggest runtime stack
  - Language detection hints (show detected language with override option)
- **Service Validation**:
  - Unique service names
  - Valid host types from schema enum
  - Port conflict detection (warn if port in use by another service)
  - Required fields (name, host) enforced

**Acceptance Criteria**:
- "+ Add Service" opens modal with three tabs
- Well-known services grid displays all services from registry
- Clicking well-known service pre-fills all configuration
- Application/container tabs collect required fields
- Added service appears in navigation and editor
- Service can be edited via form
- Service can be deleted with confirmation
- Validation prevents invalid configurations

---

### FR-6: Well-Known Services Integration

**Description**: One-click addition of common Azure services using wellknown registry

**Requirements**:
- Fetch well-known services from backend API endpoint (e.g., `/api/wellknown/services`)
- Display as visual grid with:
  - Service icon (use Azure service icons)
  - Service display name
  - Short description
  - Category badge (storage, database, cache, etc.)
- Quick action buttons in editor footer:
  - `[+ Add Azurite]` `[+ Add Cosmos]` `[+ Add Redis]` `[+ Add Postgres]`
  - Click → immediate service addition (no modal)
  - Show toast notification: "✓ Added {service name}"
- Service configuration includes:
  - All default values from `wellknown.ServiceDefinition`
  - Connection strings displayed in info panel after addition
  - Health check pre-configured
- Support custom service addition (user-provided Docker images)
- Category filtering in add service modal

**Acceptance Criteria**:
- Well-known services load from API
- Visual grid shows all available services with icons
- Quick action buttons add services instantly
- Added services include full default configuration
- Connection strings shown in info panel
- Custom Docker images can be added via form

---

### FR-7: Environment Variable Editor

**Description**: Flexible environment variable editor supporting multiple formats

**Requirements**:
- Three environment variable formats (Docker Compose compatible):
  1. **Map Format** (simple):
     ```yaml
     environment:
       KEY: value
       DATABASE_URL: postgresql://...
     ```
  2. **Array of Strings**:
     ```yaml
     environment:
       - KEY=value
       - DATABASE_URL=postgresql://...
     ```
  3. **Array of Objects** (with secrets):
     ```yaml
     environment:
       - name: KEY
         value: value
       - name: API_KEY
         secret: MY_SECRET
     ```
- Editor UI:
  - List of environment variables with inline add/edit/delete
  - "+ Add Variable" button opens inline form:
    - Name field (required, validated: uppercase, underscores, numbers)
    - Value type dropdown: `Text`, `Secret Reference`, `Expression`
    - Value field (changes based on type)
      - Text: Plain text input
      - Secret Reference: Dropdown of available secrets + secret name input
      - Expression: Text input with syntax hint (e.g., `${VAR_NAME}`)
  - Reorderable list (drag handles)
  - Bulk import from .env file (paste or upload)
  - Export to .env file
- Automatic format conversion:
  - If all variables are simple: render as map format (most concise)
  - If any variable uses secret reference: render as array of objects
  - Preserve user's chosen format if explicitly set
- Secret validation:
  - Warn if secret name doesn't match any defined secrets
  - Link to secrets management (future integration)

**Acceptance Criteria**:
- Environment variables display in list format
- "+ Add Variable" allows adding new variables
- All three format types supported
- Secret references validate against known secrets
- Variables reorderable via drag-and-drop
- Bulk import from .env works
- YAML output uses most appropriate format

---

### FR-8: Validation and Error Handling

**Description**: Real-time validation with clear error messages and remediation

**Requirements**:
- **Schema Validation**:
  - Validate all fields against JSON Schema rules
  - Check required fields present
  - Validate types (string, number, boolean, etc.)
  - Enforce patterns (regex)
  - Validate enums (only allowed values)
  - Check min/max values for numbers
  - Validate array min/max items
- **Custom Business Rules**:
  - Unique service names
  - Unique resource names
  - Port conflict detection (same port in multiple services)
  - Circular dependency detection (service A uses B, B uses A)
  - Path existence checks (project paths, file paths)
  - Health check URL reachability (optional async check)
- **Validation Levels**:
  - **Error**: Blocks save (red, ❌ icon)
  - **Warning**: Allows save but shows caution (yellow, ⚠️ icon)
  - **Info**: Helpful suggestions (blue, ℹ️ icon)
- **Error Display**:
  - Inline field errors (below field)
  - Validation summary panel (bottom of editor)
  - Badge on navigation items with errors
  - Prevent save when errors exist
  - Allow save with warnings (show confirmation)
- **Error Messages**:
  - Clear, actionable messages (not raw schema errors)
  - Examples:
    - ❌ "Service name must contain only lowercase letters, numbers, and hyphens"
    - ⚠️ "Port 8080 is already used by service 'api'. This may cause conflicts."
    - ℹ️ "Consider adding a health check for service 'web' to monitor availability"
  - Link to documentation for complex errors

**Acceptance Criteria**:
- All schema validation rules enforced
- Custom business rules validate correctly
- Errors block save, warnings allow save
- Error messages are clear and actionable
- Validation summary shows all issues
- Navigation badges indicate error/warning count
- Users can't save invalid configurations

---

### FR-9: Health Check Configuration

**Description**: Visual editor for Docker Compose-compatible health checks

**Requirements**:
- Health check modal/panel with form fields:
  - **Type**: Dropdown (`http`, `tcp`, `process`, `output`, `none`)
  - **Test Command**: Based on type:
    - `http`: URL input (e.g., `http://localhost:8080/health`)
    - `tcp`: Port input (auto-filled from service ports)
    - `process`: Process name input
    - `output`: Regex pattern input
    - `none`: Disabled (no fields)
  - **Interval**: Duration input (e.g., `30s`, `1m`)
  - **Timeout**: Duration input
  - **Retries**: Numeric input (1-10)
  - **Start Period**: Duration input (optional)
- Default suggestions based on service type:
  - HTTP services → http health check with common patterns (`/health`, `/healthz`, `/api/health`)
  - Database services → tcp health check on primary port
  - Container services → process check
- Visual preview of generated YAML
- Test button to run health check against current service (if running)
- Health check templates for common frameworks:
  - Express.js: `http://localhost:${PORT}/health`
  - ASP.NET: `http://localhost:${PORT}/healthz`
  - Spring Boot: `http://localhost:${PORT}/actuator/health`

**Acceptance Criteria**:
- Health check modal accessible from service editor
- All health check types supported
- Form validates duration formats (e.g., `30s`, `1m`)
- Defaults suggested based on service type
- Test button validates health check (if service running)
- Generated YAML matches Docker Compose spec
- Health check templates available for selection

---

### FR-10: Resource Configuration

**Description**: Editor for Azure resource definitions with dependency management

**Requirements**:
- Resource list in navigation (under "Resources" section)
- "+ Add Resource" button with resource type selection:
  - Common types: Storage, Cosmos DB, Event Hubs, Service Bus, App Service
  - Full list from schema (Microsoft.Storage/storageAccounts, etc.)
- Resource form:
  - Name (required, unique)
  - Type (required, searchable dropdown)
  - Uses (dependencies): Multi-select of other resources/services
  - Existing (toggle): Whether resource is pre-existing
  - Additional properties based on type (from schema)
- Resource templates:
  - Storage Account: Include containers to create
  - Cosmos DB: Include databases/containers with partition keys
  - Event Hubs: Include hub names
  - Service Bus: Include queues/topics
- Dependency visualization:
  - Show dependency graph (D3.js or similar)
  - Highlight circular dependencies (error)
  - Topological sort for deployment order
- Link resources to services:
  - Service uses resource → show in service editor
  - Auto-populate environment variables for resource connections

**Acceptance Criteria**:
- Resources can be added via "+ Add Resource"
- Resource types searchable and filterable
- Dependencies manageable via multi-select
- Circular dependencies detected and prevented
- Resource templates available for common types
- Dependency graph visualizes relationships
- Service-resource links visible in UI

---

### FR-11: Hooks and Lifecycle Management

**Description**: Editor for azd lifecycle hooks (preprovision, postdeploy, etc.)

**Requirements**:
- Hooks section in navigation
- List of available hooks from schema:
  - Provision: preprovision, postprovision
  - Infrastructure: preinfracreate, postinfracreate, preinfradelete, postinfradelete
  - Deployment: predeploy, postdeploy
  - Package: prepackage, postpackage
  - Publish: prepublish, postpublish
  - Restore: prerestore, postrestore
  - Up/Down: preup, postup, predown, postdown
  - Run (azd app): prerun, postrun
- Hook editor:
  - Enable/disable hook (toggle)
  - Script command: Text input or file reference
  - Working directory: Path input
  - Shell: Dropdown (sh, bash, pwsh, cmd)
  - Continue on error: Toggle
  - Platform-specific overrides:
    - Windows: Override command/shell for Windows
    - POSIX: Override command/shell for Linux/macOS
- Hook templates:
  - Common patterns (npm run build, dotnet publish, etc.)
  - Platform-specific examples
- Visual timeline showing hook execution order

**Acceptance Criteria**:
- All azd hooks available in editor
- Hooks can be enabled/disabled
- Script commands editable via text input
- Platform overrides configurable
- Hook templates available for selection
- Execution order visualized in timeline

---

### FR-12: Quick Actions Bar

**Description**: Context-sensitive quick actions for common tasks

**Requirements**:
- Fixed quick actions bar at bottom of editor (above validation summary)
- Dynamic actions based on current context:
  - **Global Actions** (always visible):
    - `[+ Add Azurite]` `[+ Add Cosmos]` `[+ Add Redis]` `[+ Add Postgres]`
    - `[Import from Template]` (load from azure.yaml template)
    - `[Validate Configuration]` (run full validation)
  - **Service Context Actions** (when service selected):
    - `[Add Environment Variable]`
    - `[Configure Health Check]`
    - `[Add Dependency]`
    - `[View Connection Strings]` (for container services)
  - **Resource Context Actions** (when resource selected):
    - `[Add Container/Queue/Topic]` (based on resource type)
    - `[Link to Service]`
- Click action → immediate execution or modal
- Keyboard shortcuts for common actions (Cmd/Ctrl+K for command palette)

**Acceptance Criteria**:
- Quick actions bar visible at bottom
- Actions change based on selected context
- Well-known service buttons add services instantly
- Service/resource actions context-appropriate
- Keyboard shortcuts work correctly

---

### FR-13: Backup Management

**Description**: Automatic backups with restore capability

**Requirements**:
- **Automatic Backup**:
  - Create backup before EVERY save
  - Format: `azure.yaml.backup.{ISO8601-timestamp}`
  - Preserve up to 10 most recent backups
  - Delete oldest when >10 exist
- **Backup UI**:
  - "Backups" button in editor header
  - Backup list modal showing:
    - Timestamp (formatted: "Jan 11, 2026 2:30 PM")
    - File size
    - Preview (first 10 lines)
    - Actions: Restore, View, Delete
  - Search/filter backups by date
  - Compare backup to current (diff view)
- **Restore Flow**:
  - Click "Restore" on backup
  - Show confirmation: "Restore backup from {timestamp}? Current configuration will be backed up first."
  - Backup current azure.yaml
  - Restore selected backup to azure.yaml
  - Reload editor with restored content
  - Success notification: "✓ Restored backup from {timestamp}"
- **Backup Metadata**:
  - Store metadata file: `azure.yaml.backups.json`
  - Track: timestamp, file size, user notes (optional)
  - Allow users to add notes to backups ("before major refactor", etc.)

**Acceptance Criteria**:
- Backup created before every save
- Backups list accessible via header button
- Backups sorted by timestamp (newest first)
- Restore flow works correctly
- Current config backed up before restore
- Old backups auto-deleted when >10 exist
- Users can add notes to backups

---

### FR-14: Import/Export

**Description**: Import configuration from templates and export for sharing

**Requirements**:
- **Import**:
  - "Import" button in header
  - Import sources:
    - **Template**: Load from azd template gallery (API endpoint)
    - **File Upload**: Upload azure.yaml from disk
    - **Paste YAML**: Paste YAML content directly
  - Import flow:
    - Select source
    - Preview imported configuration (side-by-side with current)
    - Choose merge strategy:
      - Replace: Overwrite entire configuration
      - Merge: Combine with current (services, resources added)
      - Cherry-pick: Select specific services/resources to import
    - Confirm import
    - Backup current configuration first
- **Export**:
  - "Export" button in header
  - Export formats:
    - **YAML**: Download azure.yaml
    - **JSON**: Download as JSON (for API consumption)
    - **Template**: Save as reusable template (with metadata)
  - Export options:
    - Include comments (if preserved)
    - Minify (remove whitespace)
    - Include secrets (warning: security risk)

**Acceptance Criteria**:
- Import from template gallery works
- File upload accepts .yaml/.yml files
- Paste YAML parses correctly
- Merge strategies work as expected
- Current config backed up before import
- Export generates valid YAML
- Export formats (YAML, JSON, Template) all work
- Security warning shown when exporting secrets

---

### FR-15: Search and Command Palette

**Description**: Global search and command palette for quick navigation

**Requirements**:
- **Command Palette**:
  - Keyboard shortcut: `Cmd/Ctrl+K`
  - Fuzzy search over:
    - Navigation items (sections, services, resources)
    - Actions ("Add Service", "Configure Health Check", etc.)
    - Field names (jump to specific field)
    - Help topics
  - Results grouped by category:
    - Navigation (jump to section)
    - Actions (execute action)
    - Fields (jump to field)
    - Help (open docs)
  - Keyboard navigation (arrow keys, enter)
  - Recent commands (history)
- **Search in Navigation**:
  - Search input in navigation sidebar
  - Filter navigation items by name
  - Highlight matches
  - Clear button to reset
- **Global Search**:
  - Search across all configuration values
  - Find services using specific ports, images, etc.
  - Results show context (service name, property path)
  - Click result → jump to that field

**Acceptance Criteria**:
- Command palette opens on Cmd/Ctrl+K
- Fuzzy search works across all items
- Results grouped and navigable
- Recent commands tracked
- Navigation search filters items
- Global search finds values in config

---

### FR-16: Accessibility

**Description**: WCAG AA compliant interface with full keyboard support

**Requirements**:
- **Keyboard Navigation**:
  - Tab order follows visual flow
  - All actions accessible via keyboard
  - Shortcut hints visible on hover
  - Escape to close modals/dropdowns
  - Enter to submit forms
  - Arrow keys for navigation tree
- **Screen Reader Support**:
  - ARIA labels on all interactive elements
  - ARIA live regions for notifications
  - Form field labels properly associated
  - Error messages announced
  - Loading states announced
- **Visual Accessibility**:
  - Color contrast ≥4.5:1 (WCAG AA)
  - Focus indicators visible (2px outline)
  - Error states not color-only (icon + text)
  - Resizable text (up to 200%)
  - No flashing/strobing animations
- **Assistive Technologies**:
  - Tested with NVDA (Windows)
  - Tested with VoiceOver (macOS)
  - Tested with JAWS (Windows)
- **Accessibility Audit**:
  - Run axe DevTools audit (0 violations)
  - Run Lighthouse accessibility audit (score ≥95)

**Acceptance Criteria**:
- All interactive elements keyboard accessible
- Screen reader announces all content correctly
- Color contrast meets WCAG AA (4.5:1)
- Focus indicators visible on all elements
- No axe violations
- Lighthouse accessibility score ≥95

---

### FR-17: Performance

**Description**: Fast, responsive editor with optimized rendering

**Requirements**:
- **Load Performance**:
  - Initial editor load: <500ms
  - Schema fetch/parse: <200ms
  - Azure.yaml parse: <100ms
  - Navigation render: <100ms
- **Interaction Performance**:
  - Field update: <50ms (debounced)
  - Validation: <100ms
  - Navigation click: <50ms
  - Modal open: <100ms
- **Optimization Techniques**:
  - Lazy load service/resource editors (only render active section)
  - Virtual scrolling for long lists (>50 items)
  - Debounce validation (500ms after last keystroke)
  - Memoize schema parsing
  - Code-split editor components (dynamic imports)
  - Cache API responses (well-known services, templates)
- **Large Configuration Handling**:
  - Support configurations with >100 services
  - Support services with >100 environment variables
  - Pagination for large arrays
  - Progress indicators for slow operations (>200ms)

**Acceptance Criteria**:
- Editor loads in <500ms
- Field updates feel instant (<50ms)
- No UI lag with 100+ services
- Virtual scrolling works for large lists
- Validation completes in <100ms
- No memory leaks (tested with 1hr continuous use)

---

### FR-18: Visual Design

**Description**: Modern, beautiful interface consistent with dashboard design system

**Requirements**:
- **Design System Integration**:
  - Use existing dashboard color palette
  - Use existing typography scale
  - Use existing spacing system (4px grid)
  - Use existing component library (buttons, inputs, etc.)
  - Support light and dark modes
- **Visual Hierarchy**:
  - Clear section headers (typography scale)
  - Grouped related fields (fieldsets)
  - Whitespace for breathing room
  - Subtle dividers between sections
  - Depth through shadows (not excessive)
- **Component Styling**:
  - **Inputs**: Rounded corners (4px), subtle border, focus state
  - **Buttons**: Primary (filled), secondary (outlined), text (no border)
  - **Dropdowns**: Search-enabled, keyboard navigable, max height with scroll
  - **Toggles**: Modern switch design (not checkbox)
  - **Cards**: Subtle shadow, hover effect
  - **Modals**: Centered, overlay dimmed (40% opacity), slide-in animation
- **Micro-interactions**:
  - Smooth transitions (200ms ease-in-out)
  - Hover states on interactive elements
  - Loading spinners (not intrusive)
  - Success/error animations (subtle)
  - Drag-and-drop visual feedback
- **Responsive Design**:
  - Support 1024px+ width (editor not optimized for mobile)
  - Navigation collapsible on smaller screens
  - Editor pane scrollable
  - Modals responsive (max-width)
- **Icons**:
  - Consistent icon set (Lucide or similar)
  - Appropriate sizes (16px, 20px, 24px)
  - Color-coded by meaning (error=red, warning=yellow, success=green, info=blue)

**Acceptance Criteria**:
- Matches dashboard design system
- Light and dark modes both look polished
- Visual hierarchy clear and intuitive
- Micro-interactions feel smooth (200ms transitions)
- Responsive on 1024px+ screens
- Icons consistent and appropriately sized

## Technical Requirements

### TR-1: Technology Stack

**Frontend**:
- **Framework**: React 18+ with TypeScript
- **UI Components**: Radix UI (headless, accessible primitives)
- **Styling**: Tailwind CSS (existing dashboard system)
- **Forms**: React Hook Form (performance, validation)
- **Schema**: Ajv (JSON Schema validation)
- **YAML**: js-yaml (parse/stringify)
- **State**: Zustand (lightweight state management)
- **API**: Fetch API with error handling

**Backend**:
- **Language**: Go (existing azd app codebase)
- **API**: HTTP endpoints for file operations, wellknown services
- **Validation**: Go YAML parser (gopkg.in/yaml.v3)

### TR-2: Architecture

**Component Structure**:
```
components/
  editor/
    YamlEditor.tsx              # Main editor container
    Navigation.tsx              # Left sidebar navigation
    EditorPane.tsx              # Right pane form editor
    ValidationSummary.tsx       # Bottom validation panel
    QuickActions.tsx            # Bottom quick actions bar
    BackupManager.tsx           # Backup list modal
    
  forms/
    SchemaForm.tsx              # Dynamic form generator from schema
    FieldRenderer.tsx           # Field type router
    fields/
      StringField.tsx
      NumberField.tsx
      BooleanField.tsx
      EnumField.tsx
      ArrayField.tsx
      ObjectField.tsx
      PortField.tsx
      PathField.tsx
      EnvironmentVariablesField.tsx
      
  modals/
    AddServiceModal.tsx         # Add service modal with tabs
    WellKnownServicesGrid.tsx   # Grid of well-known services
    HealthCheckModal.tsx        # Health check configuration
    ImportModal.tsx             # Import configuration modal
    BackupModal.tsx             # Backup list and restore
    
  common/
    ValidationMessage.tsx       # Inline validation display
    HelpTooltip.tsx             # Help icon with description
    CommandPalette.tsx          # Cmd+K search palette
```

**State Management**:
```typescript
interface EditorState {
  // Configuration
  config: AzureYamlConfig | null
  schema: JsonSchema | null
  
  // UI State
  activeSection: string
  validationErrors: ValidationError[]
  validationWarnings: ValidationError[]
  isDirty: boolean
  
  // Actions
  loadConfig: (path: string) => Promise<void>
  saveConfig: () => Promise<void>
  updateField: (path: string, value: any) => void
  addService: (service: Service) => void
  removeService: (name: string) => void
  validate: () => ValidationError[]
  createBackup: () => Promise<void>
  restoreBackup: (timestamp: string) => Promise<void>
}
```

### TR-3: API Endpoints

**File Operations**:
- `GET /api/editor/config` - Load current azure.yaml
- `POST /api/editor/config` - Save azure.yaml (creates backup first)
- `GET /api/editor/backups` - List all backups
- `GET /api/editor/backups/:timestamp` - Get specific backup content
- `POST /api/editor/backups/:timestamp/restore` - Restore backup
- `DELETE /api/editor/backups/:timestamp` - Delete backup

**Schema**:
- `GET /api/editor/schema` - Get azure.yaml JSON Schema

**Well-Known Services**:
- `GET /api/editor/wellknown` - List all well-known services
- `GET /api/editor/wellknown/:name` - Get specific service definition

**Validation**:
- `POST /api/editor/validate` - Validate configuration (full check including file paths, port conflicts)

### TR-4: Error Handling

**Error Categories**:
1. **Schema Errors**: Invalid types, missing required fields, pattern mismatches
2. **File System Errors**: Permission denied, disk full, file not found
3. **Validation Errors**: Business rule violations (duplicate names, circular deps)
4. **Network Errors**: API timeouts, schema fetch failures

**Error Display**:
- Inline field errors (below field)
- Validation summary panel (bottom)
- Modal dialogs for critical errors
- Toast notifications for background operations

**Recovery Strategies**:
- Auto-save draft to localStorage (every 30s)
- Recover from localStorage on editor reopen
- Fallback to bundled schema if fetch fails
- Retry failed API calls (3 attempts with exponential backoff)

### TR-5: Testing

**Unit Tests** (Vitest):
- Schema parsing and validation logic
- YAML parsing and serialization
- Form field components
- State management actions
- Validation rules

**Integration Tests** (Playwright):
- End-to-end add service flow
- Save and restore backup flow
- Import from template flow
- Validation error display
- Keyboard navigation

**Accessibility Tests**:
- axe-core automated scans
- Manual screen reader testing
- Keyboard navigation testing

**Performance Tests**:
- Load time with large configurations (100+ services)
- Field update latency
- Validation performance with complex rules

**Coverage Target**: ≥80% line coverage

### TR-6: Security

**Input Validation**:
- Sanitize all user input (XSS prevention)
- Validate file paths (prevent directory traversal)
- Limit file sizes (azure.yaml <10MB, backups <50MB total)
- Rate limit API requests (prevent DoS)

**Secret Handling**:
- Never log secret values
- Mask secrets in UI (show `***` unless explicitly revealed)
- Warn when exporting configuration with secrets
- Support secret references (not plain text)

**File System**:
- Verify file permissions before write
- Atomic writes (temp file + rename)
- No arbitrary file access (only azure.yaml and backups)

## Design Details

### Schema-Driven Form Generation

The editor dynamically generates forms from the JSON Schema:

```typescript
interface SchemaFormProps {
  schema: JsonSchema
  value: any
  onChange: (value: any) => void
  path: string
}

function SchemaForm({ schema, value, onChange, path }: SchemaFormProps) {
  // Determine field type from schema
  const fieldType = determineFieldType(schema)
  
  // Render appropriate field component
  switch (fieldType) {
    case 'string':
      if (schema.enum) return <EnumField ... />
      if (schema.pattern) return <PatternField ... />
      return <StringField ... />
    
    case 'number':
      return <NumberField min={schema.minimum} max={schema.maximum} ... />
    
    case 'boolean':
      return <BooleanField ... />
    
    case 'object':
      return <ObjectField schema={schema.properties} ... />
    
    case 'array':
      return <ArrayField itemSchema={schema.items} ... />
  }
}
```

### Well-Known Services Integration

Fetch from backend API:

```typescript
interface WellKnownService {
  name: string
  displayName: string
  description: string
  category: string
  host: string
  image: string
  ports: string[]
  environment: Record<string, string>
  healthcheck: HealthCheckConfig
  connectionStrings: Record<string, string>
}

// API call
const services = await fetch('/api/editor/wellknown').then(r => r.json())

// Render grid
<WellKnownServicesGrid
  services={services}
  onSelect={(service) => {
    addService({
      [service.name]: {
        host: service.host,
        image: service.image,
        ports: service.ports,
        environment: service.environment,
        healthcheck: service.healthcheck,
      }
    })
  }}
/>
```

### Backup Management

Automatic backup on save:

```typescript
async function saveConfig(config: AzureYamlConfig) {
  // 1. Create backup
  const timestamp = new Date().toISOString().replace(/[:.]/g, '')
  const backupPath = `azure.yaml.backup.${timestamp}`
  await fs.copyFile('azure.yaml', backupPath)
  
  // 2. Write new config
  const yamlContent = yaml.dump(config, { indent: 2 })
  await fs.writeFile('azure.yaml', yamlContent)
  
  // 3. Cleanup old backups
  const backups = await listBackups()
  if (backups.length > 10) {
    const toDelete = backups.slice(10)
    await Promise.all(toDelete.map(b => fs.unlink(b.path)))
  }
  
  return { success: true, backup: backupPath }
}
```

### Validation Pipeline

Multi-stage validation:

```typescript
function validateConfig(config: AzureYamlConfig, schema: JsonSchema) {
  const errors: ValidationError[] = []
  const warnings: ValidationError[] = []
  
  // 1. Schema validation
  const ajv = new Ajv({ allErrors: true })
  const validate = ajv.compile(schema)
  if (!validate(config)) {
    errors.push(...formatAjvErrors(validate.errors))
  }
  
  // 2. Business rule validation
  const serviceNames = Object.keys(config.services || {})
  const duplicates = findDuplicates(serviceNames)
  if (duplicates.length > 0) {
    errors.push({ path: 'services', message: `Duplicate service names: ${duplicates.join(', ')}` })
  }
  
  // 3. Port conflict detection
  const portMap = buildPortMap(config.services)
  for (const [port, services] of portMap.entries()) {
    if (services.length > 1) {
      warnings.push({ 
        path: 'services', 
        message: `Port ${port} used by multiple services: ${services.join(', ')}` 
      })
    }
  }
  
  // 4. Circular dependency detection
  const cycles = detectCycles(config.services, config.resources)
  if (cycles.length > 0) {
    errors.push({ 
      path: 'services', 
      message: `Circular dependencies detected: ${cycles.join(' → ')}` 
    })
  }
  
  return { errors, warnings }
}
```

## User Stories

### US-1: Quick Service Addition
**As a** developer  
**I want to** add common Azure services (like Cosmos DB) with one click  
**So that** I can quickly configure local development dependencies without reading documentation

**Acceptance Criteria**:
- Click "+ Add Cosmos" button
- Cosmos DB service added with correct defaults (image, ports, environment, health check)
- Connection string displayed for easy copy
- Service starts successfully with `azd app run`

---

### US-2: Visual Health Check Configuration
**As a** developer  
**I want to** configure health checks through a visual form  
**So that** I don't have to remember Docker Compose health check syntax

**Acceptance Criteria**:
- Click "Configure Health Check" on service
- Select health check type (http, tcp, process)
- Fill in appropriate fields (URL, port, command)
- Health check YAML generated correctly
- Health check works when service runs

---

### US-3: Environment Variable Management
**As a** developer  
**I want to** manage environment variables with support for secrets  
**So that** I can configure my services without exposing sensitive data

**Acceptance Criteria**:
- Add environment variable with "+ Add Variable"
- Choose variable type (text, secret reference, expression)
- Secret references validated against available secrets
- YAML output uses appropriate format (map vs array of objects)

---

### US-4: Safe Configuration Editing
**As a** developer  
**I want to** have automatic backups before saving  
**So that** I can safely experiment with configuration changes

**Acceptance Criteria**:
- Every save creates timestamped backup
- Can view list of backups with timestamps
- Can restore any backup with confirmation
- Old backups auto-deleted when >10 exist

---

### US-5: Validation Feedback
**As a** developer  
**I want to** see validation errors as I edit  
**So that** I can fix issues before running azd commands

**Acceptance Criteria**:
- Inline errors show below invalid fields
- Validation summary lists all errors/warnings
- Errors prevent save, warnings allow save with confirmation
- Error messages are clear and actionable

## Success Metrics

### Adoption
- **Target**: 60% of dashboard users try the editor within 30 days of release
- **Measurement**: Track editor open events in telemetry

### Engagement
- **Target**: 40% of users who try editor use it for >50% of their azure.yaml edits
- **Measurement**: Compare editor saves vs manual file saves

### Error Reduction
- **Target**: 30% reduction in azure.yaml syntax errors reported in issues/support
- **Measurement**: Track validation errors caught pre-save vs errors encountered at runtime

### Time Savings
- **Target**: Average time to add a service reduced from 5 minutes to 30 seconds
- **Measurement**: User research study (before/after comparison)

### Satisfaction
- **Target**: ≥4.5/5 average rating in user feedback survey
- **Measurement**: In-app feedback prompt after editor use

## Risks and Mitigations

### Risk 1: Schema Complexity
**Description**: Azure.yaml schema is complex with many conditional fields, which may be difficult to represent in a form UI.

**Likelihood**: High  
**Impact**: Medium

**Mitigation**:
- Start with most common properties (services, resources)
- Progressively enhance with advanced features
- Always allow "raw YAML" editing for edge cases
- Provide "advanced mode" toggle for complex scenarios

---

### Risk 2: Performance with Large Configurations
**Description**: Configurations with 100+ services may render slowly or cause UI lag.

**Likelihood**: Medium  
**Impact**: High

**Mitigation**:
- Implement virtual scrolling for service/resource lists
- Lazy load service editors (only render active section)
- Optimize validation with debouncing and memoization
- Performance test with large configurations (100+ services)

---

### Risk 3: YAML Formatting Preservation
**Description**: Round-tripping through YAML parser may lose comments, formatting, or key ordering.

**Likelihood**: High  
**Impact**: Medium

**Mitigation**:
- Use YAML parser with comment preservation (yaml-js with preserveComments)
- Document that comments may be lost (backups preserve original)
- Maintain consistent key ordering (alphabetical or schema-defined)
- Allow users to export/import to preserve formatting if needed

---

### Risk 4: Schema Evolution
**Description**: Azure.yaml schema may change frequently, requiring editor updates.

**Likelihood**: Medium  
**Impact**: Medium

**Mitigation**:
- Fetch schema dynamically from URL (not hardcoded)
- Support multiple schema versions (user can select)
- Graceful degradation for unknown properties (show as raw fields)
- Version schema changes with semantic versioning

---

### Risk 5: User Adoption
**Description**: Users may prefer text editing and not adopt the visual editor.

**Likelihood**: Medium  
**Impact**: High

**Mitigation**:
- Provide clear value proposition (faster, fewer errors, discovery)
- Smooth onboarding (tutorial, examples)
- Allow hybrid workflow (visual + text editing)
- Gather user feedback early and iterate

## Open Questions

1. **Q**: Should the editor support editing YAML comments?  
   **Status**: Open  
   **Decision Needed**: By design phase  
   **Options**: (a) Best-effort preservation, (b) Strip comments with warning, (c) Read-only comment display

2. **Q**: How should we handle custom/extension properties not in schema?  
   **Status**: Open  
   **Decision Needed**: By implementation  
   **Options**: (a) Show as raw fields, (b) Ignore, (c) Warn and preserve

3. **Q**: Should we support multi-file azure.yaml (imports/references)?  
   **Status**: Open  
   **Decision Needed**: By v2  
   **Options**: (a) Not in v1, (b) Read-only support, (c) Full editing support

4. **Q**: How to handle concurrent edits (editor open + manual file edit)?  
   **Status**: Open  
   **Decision Needed**: By implementation  
   **Options**: (a) File watcher with reload prompt, (b) Lock file during editing, (c) Merge changes

5. **Q**: Should we integrate with azd templates (scaffold from template)?  
   **Status**: Open  
   **Decision Needed**: By v1  
   **Options**: (a) Yes, import from template gallery, (b) No, manual import only

## Future Enhancements

### v2.0
- **Multi-file Support**: Edit imported/referenced YAML files
- **Template Gallery**: Browse and scaffold from azd templates
- **Visual Dependency Graph**: Interactive D3.js graph of service/resource dependencies
- **Bicep Integration**: Generate Bicep files from resource definitions
- **AI-Assisted Configuration**: Suggest services based on project structure
- **Collaborative Editing**: Real-time multi-user editing with conflict resolution

### v2.1
- **Configuration Diff**: Compare configurations side-by-side
- **Configuration Versioning**: Track configuration changes over time (Git integration)
- **Export to Terraform**: Generate Terraform HCL from azure.yaml
- **Secrets Management UI**: Integrated secrets editor with encryption

### v3.0
- **Visual Pipeline Builder**: Drag-and-drop CI/CD pipeline configuration
- **Cost Estimation**: Show estimated Azure costs based on resources
- **Security Scanning**: Detect security issues (exposed secrets, insecure configs)
- **Performance Optimization**: Suggest optimizations (scaling, caching, etc.)

## Appendix

### A: Schema Property Coverage

**Phase 1 (MVP)** - Most Common Properties:
- name, services, resources
- service: host, language, project, image, ports, environment
- resource: type, uses, existing

**Phase 2** - Advanced Features:
- hooks, pipeline, metadata
- service: healthcheck, test, type, mode, entrypoint, command
- resource: containers, queues, topics

**Phase 3** - Complete Coverage:
- requiredVersions, state, infra
- All conditional properties

### B: Well-Known Services Registry

Current services (from `wellknown.services.go`):
- **azurite**: Azure Storage emulator (blob, queue, table)
- **cosmos**: Azure Cosmos DB emulator
- **redis**: Redis in-memory cache
- **postgres**: PostgreSQL database

Future additions:
- **mongodb**: MongoDB database
- **mysql**: MySQL database
- **rabbitmq**: RabbitMQ message broker
- **sqlserver**: SQL Server database
- **elasticsearch**: Elasticsearch search engine
- **minio**: MinIO S3-compatible storage

### C: Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl+K` | Open command palette |
| `Cmd/Ctrl+S` | Save configuration |
| `Cmd/Ctrl+Z` | Undo last change |
| `Cmd/Ctrl+Shift+Z` | Redo last change |
| `Cmd/Ctrl+F` | Search in navigation |
| `Cmd/Ctrl+N` | Add new service |
| `Cmd/Ctrl+B` | Toggle navigation sidebar |
| `Escape` | Close modal/dropdown |
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Enter` | Submit form |
| `Arrow Keys` | Navigate navigation tree |

### D: Validation Error Messages

**Schema Errors**:
- `name` required: "Application name is required"
- `name` pattern: "Application name must contain only lowercase letters, numbers, and hyphens"
- `service.host` enum: "Host must be one of: containerapp, appservice, function, springapp, staticwebapp, aks"
- `port` type: "Port must be a number between 1 and 65535"

**Business Rule Errors**:
- Duplicate service: "Service name '{name}' already exists"
- Circular dependency: "Circular dependency detected: {service1} → {service2} → {service1}"
- Port conflict: "Port {port} is already used by service '{service}'"
- Invalid path: "Project path '{path}' does not exist"

**Warnings**:
- Missing health check: "Consider adding a health check for service '{name}' to monitor availability"
- No resources: "No resources defined. Consider adding resource definitions for Azure provisioning."
- Unused service: "Service '{name}' is not used by any other service or resource"

### E: API Response Formats

**GET /api/editor/config**:
```json
{
  "path": "/workspace/azure.yaml",
  "content": "name: my-app\nservices:\n  api:\n    host: containerapp",
  "lastModified": "2026-01-11T14:30:55Z"
}
```

**POST /api/editor/config**:
```json
{
  "success": true,
  "backup": "/workspace/azure.yaml.backup.20260111T143055Z",
  "written": true,
  "errors": []
}
```

**GET /api/editor/wellknown**:
```json
{
  "services": [
    {
      "name": "azurite",
      "displayName": "Azurite (Azure Storage Emulator)",
      "description": "Local Azure Storage emulator for Blob, Queue, and Table services",
      "category": "storage",
      "host": "containerapp",
      "image": "mcr.microsoft.com/azure-storage/azurite:latest",
      "ports": ["10000:10000", "10001:10001", "10002:10002"],
      "environment": {},
      "healthcheck": {
        "test": ["CMD", "curl", "-f", "http://127.0.0.1:10000/"],
        "interval": "10s",
        "timeout": "5s",
        "retries": 3
      },
      "connectionStrings": {
        "blob": "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;...",
        "queue": "...",
        "table": "...",
        "default": "UseDevelopmentStorage=true"
      }
    }
  ]
}
```

### F: File Structure

```
azure.yaml                          # Main configuration file
azure.yaml.backup.20260111T143055Z  # Backup file (timestamped)
azure.yaml.backup.20260111T120000Z  # Older backup
azure.yaml.backups.json             # Backup metadata (notes, sizes)
```

### G: Component Hierarchy

```
YamlEditor
├─ Header
│  ├─ Title
│  ├─ BackupButton
│  ├─ ImportButton
│  ├─ ExportButton
│  ├─ SaveButton
│  └─ CancelButton
├─ EditorBody
│  ├─ Navigation
│  │  ├─ SearchInput
│  │  ├─ NavigationTree
│  │  │  ├─ OverviewItem
│  │  │  ├─ ServicesSection
│  │  │  │  ├─ ServiceItem[]
│  │  │  │  └─ AddServiceButton
│  │  │  ├─ ResourcesSection
│  │  │  ├─ HooksSection
│  │  │  ├─ PipelineSection
│  │  │  └─ MetadataSection
│  │  └─ CollapseButton
│  └─ EditorPane
│     └─ SchemaForm
│        └─ FieldRenderer[]
├─ QuickActionsBar
│  ├─ AddAzuriteButton
│  ├─ AddCosmosButton
│  ├─ AddRedisButton
│  └─ AddPostgresButton
└─ ValidationSummary
   ├─ ErrorList
   └─ WarningList
```

## References

- [Azure Developer CLI Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [azure.yaml Schema v1.1](https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json)
- [Docker Compose Specification](https://docs.docker.com/compose/compose-file/)
- [JSON Schema Specification](https://json-schema.org/specification.html)
- [WCAG 2.1 Level AA Guidelines](https://www.w3.org/WAI/WCAG21/quickref/?currentsidebar=%23col_customize&levels=aa)
- [React Hook Form Documentation](https://react-hook-form.com/)
- [Radix UI Documentation](https://www.radix-ui.com/)
