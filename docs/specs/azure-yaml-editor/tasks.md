# Azure YAML Editor - Tasks

<!-- NEXT: 1 -->

## TODO

### 1. Schema Infrastructure
**Priority**: P0  
**Size**: M  
**Description**: Implement schema loading, parsing, and caching infrastructure. Load azure.yaml JSON Schema from remote URL with local fallback. Parse schema definitions to build internal model of properties, types, validation rules, and enums. Cache parsed schema in memory.

### 2. File System Operations
**Priority**: P0  
**Size**: M  
**Description**: Implement safe file read/write operations with automatic backup management. Read azure.yaml, parse YAML to object, create timestamped backups before write, implement atomic write with temp file, manage backup retention (max 10), handle file system errors gracefully.

### 3. Navigation Component
**Priority**: P0  
**Size**: S  
**Description**: Build hierarchical navigation sidebar reflecting azure.yaml structure. Implement tree navigation with sections (Overview, Services, Resources, Hooks, Pipeline, Metadata), highlight active section, show validation badges (error/warning dots), keyboard navigation support.

### 4. Schema-Driven Form Generator
**Priority**: P0  
**Size**: L  
**Description**: Create dynamic form generation engine from JSON Schema. Build SchemaForm component that renders appropriate field types (string, number, boolean, enum, array, object) based on schema. Implement FieldRenderer with type detection and field component routing.

### 5. Basic Field Components
**Priority**: P0  
**Size**: M  
**Description**: Implement core form field components: StringField, NumberField, BooleanField (toggle), EnumField (dropdown), with validation, help tooltips, required indicators, and error display.

### 6. Service Management UI
**Priority**: P0  
**Size**: L  
**Description**: Build service add/edit/delete interface. Create AddServiceModal with three tabs (Well-Known, Application, Container). Implement service list in navigation, service editor form, delete confirmation, and service name uniqueness validation.

### 7. Well-Known Services Integration
**Priority**: P0  
**Size**: M  
**Description**: Integrate wellknown services registry from backend API. Build visual service grid with icons, categories, and descriptions. Implement quick action buttons (Add Azurite, Add Cosmos, Add Redis, Add Postgres) for one-click service addition. Show connection strings after addition.

### 8. Environment Variables Editor
**Priority**: P1  
**Size**: M  
**Description**: Build flexible environment variable editor supporting map format, array of strings, and array of objects (with secrets). Implement add/edit/delete inline forms, secret reference validation, bulk import from .env, and automatic format selection.

### 9. Validation Engine
**Priority**: P0  
**Size**: L  
**Description**: Implement multi-stage validation: JSON Schema validation with Ajv, custom business rules (unique names, port conflicts, circular dependencies), validation levels (error/warning/info), inline field errors, and validation summary panel.

### 10. Health Check Configuration
**Priority**: P1  
**Size**: M  
**Description**: Create health check editor modal with support for http, tcp, process, output, and none types. Implement type-specific fields (URL for http, port for tcp, command for process), duration inputs (interval, timeout), default suggestions based on service type.

### 11. Resource Configuration UI
**Priority**: P1  
**Size**: M  
**Description**: Build resource add/edit interface with resource type selection, dependency management (uses field multi-select), resource templates (Storage, Cosmos, Event Hubs, Service Bus), and circular dependency detection.

### 12. Hooks Configuration UI
**Priority**: P1  
**Size**: S  
**Description**: Create hooks editor for lifecycle events (preprovision, postdeploy, etc.). Implement enable/disable toggle, script command input, shell selection, platform-specific overrides (Windows/POSIX), and hook execution timeline visualization.

### 13. Backup Management UI
**Priority**: P0  
**Size**: S  
**Description**: Build backup list modal with timestamp display, preview, restore/view/delete actions, backup notes support, and diff view comparing backup to current configuration. Implement restore flow with confirmation and automatic backup before restore.

### 14. Quick Actions Bar
**Priority**: P1  
**Size**: S  
**Description**: Implement fixed bottom quick actions bar with context-sensitive actions: global actions (Add services, Import, Validate), service context actions (Add env var, Configure health check), resource context actions (Add container, Link to service).

### 15. Command Palette
**Priority**: P1  
**Size**: M  
**Description**: Build global command palette (Cmd/Ctrl+K) with fuzzy search over navigation items, actions, field names, and help topics. Implement keyboard navigation, recent command history, and grouped results (Navigation, Actions, Fields, Help).

### 16. Import/Export Features
**Priority**: P2  
**Size**: M  
**Description**: Implement import from templates/file/paste YAML with merge strategies (Replace, Merge, Cherry-pick) and preview. Add export to YAML/JSON/Template formats with options (include comments, minify, include secrets with warning).

### 17. Array and Object Field Components
**Priority**: P0  
**Size**: M  
**Description**: Implement ArrayField with add/remove/reorder (drag-and-drop), repeatable item groups, and virtual scrolling for large arrays. Build ObjectField with nested fieldsets, expandable sections, and conditional field visibility.

### 18. Accessibility Implementation
**Priority**: P0  
**Size**: M  
**Description**: Ensure WCAG AA compliance: keyboard navigation (tab order, shortcuts, escape/enter), screen reader support (ARIA labels, live regions, announcements), visual accessibility (4.5:1 contrast, focus indicators, no color-only errors), tested with NVDA/VoiceOver/JAWS.

### 19. Performance Optimization
**Priority**: P1  
**Size**: M  
**Description**: Optimize for large configurations (100+ services): implement virtual scrolling for lists, lazy load editor sections, debounce validation (500ms), memoize schema parsing, code-split components, cache API responses, ensure <500ms load time and <50ms field updates.

### 20. Visual Design and Styling
**Priority**: P1  
**Size**: L  
**Description**: Apply dashboard design system: integrate color palette/typography/spacing, implement light/dark modes, style all components (inputs, buttons, dropdowns, toggles, cards, modals), add micro-interactions (200ms transitions, hover states, animations), ensure responsive design (1024px+).

### 21. API Endpoints Implementation
**Priority**: P0  
**Size**: M  
**Description**: Build Go backend API endpoints: GET/POST /api/editor/config (load/save with backup), GET /api/editor/backups (list/get/restore/delete), GET /api/editor/schema, GET /api/editor/wellknown, POST /api/editor/validate.

### 22. Error Handling and Recovery
**Priority**: P0  
**Size**: S  
**Description**: Implement comprehensive error handling: categorize errors (schema, file system, validation, network), display appropriately (inline, summary, modal, toast), auto-save draft to localStorage every 30s, recover from localStorage on reopen, retry failed API calls with exponential backoff.

### 23. Testing Suite
**Priority**: P0  
**Size**: L  
**Description**: Write comprehensive test suite: unit tests (schema parsing, YAML serialization, form fields, validation) with Vitest, integration tests (add service, save/restore backup, import) with Playwright, accessibility tests (axe-core scans, screen reader testing), performance tests (large configs, field latency). Target ≥80% coverage.

### 24. Security Hardening
**Priority**: P0  
**Size**: S  
**Description**: Implement security measures: sanitize all user input (XSS prevention), validate file paths (prevent directory traversal), limit file sizes (azure.yaml <10MB, backups <50MB total), rate limit API requests, mask secrets in UI, warn when exporting with secrets, atomic file writes.

### 25. Documentation and Help
**Priority**: P2  
**Size**: M  
**Description**: Create user documentation: inline help tooltips from schema descriptions, context-sensitive help panels, keyboard shortcut reference, video walkthrough, migration guide from manual editing, troubleshooting guide.

## IN PROGRESS

## DONE

---

**Legend**:
- **Priority**: P0 (Critical) > P1 (High) > P2 (Medium) > P3 (Low)
- **Size**: S (Small, <1 day) | M (Medium, 1-3 days) | L (Large, 3-5 days)
