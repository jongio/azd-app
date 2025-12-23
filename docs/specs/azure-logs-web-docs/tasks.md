<!-- NEXT:  -->
# Azure Logs Web Documentation & Screenshots Tasks

## TODO: Update Screenshot Infrastructure

### 1. Update screenshot script to use azure-logs-test project
**Assignee**: Developer
**Description**: Modify `capture-screenshots.ts` to use `azure-logs-test` instead of `demo` project. Add Azure CLI authentication checks and increased wait times for Log Analytics polling. Include pre-flight checks for Azure resources.
**Acceptance**:
- DEMO_DIR points to `cli/tests/projects/integration/azure-logs-test`
- Script checks `az account show` before starting
- Waits minimum 15s after dashboard loads for Azure logs to populate
- Clear error message if Azure resources not available

### 2. Add Azure logs screenshot configurations
**Assignee**: Developer
**Description**: Add three new screenshot configs to `screenshot-config.ts` for Azure logs views: main Azure mode logs, time range selector visible, and service filter active.
**Acceptance**:
- `dashboard-azure-logs` config added with Console tab navigation, mode switch to Azure, 15s wait for first poll
- `dashboard-azure-logs-time-range` config added showing time range dropdown (15m, 30m, 6h, 24h options)
- `dashboard-azure-logs-filters` config added showing service filter dropdown active
- All configs include sufficient delay (15s+) for Azure Log Analytics polling cycle

### 3. Capture and optimize new screenshots
**Assignee**: Developer
**Description**: Run updated screenshot script to capture all 6 screenshots (3 updated existing + 3 new Azure logs). Ensure azure-logs-test is deployed with active services generating logs. Note: Azure logs have 1-5 minute ingestion delay, so may need to wait or trigger activity to ensure logs are visible.
**Acceptance**:
- All 6 screenshots captured at 900x600 viewport, 2x scale
- Screenshots show real Azure logs from Container Apps, App Service, Functions
- Azure logs screenshots show actual log data (not empty state)
- Images optimized (compressed without quality loss)
- No sensitive data visible (subscription IDs redacted if present)
- Azure mode toggle visible in Azure screenshots

## TODO: Create Azure Logs Documentation

### 4. Create comprehensive Azure logs reference page
**Assignee**: Developer
**Description**: Create new page `/reference/azure-logs.astro` with full documentation of Azure Cloud Log Streaming feature. Include overview, supported services, configuration examples, table selection, authentication, and troubleshooting. Include custom KQL in advanced section as yaml-only feature.
**Acceptance**:
- Page structure follows existing reference page patterns
- Covers all supported Azure services (Container Apps, App Service, Functions)
- Shows azure.yaml logs.analytics configuration with code blocks
- Explains time range presets (15m, 30m, 6h, 24h)
- Explains table selection and service filtering (UI features)
- Documents authentication requirements (Azure CLI)
- Troubleshooting section for common issues (1-5min ingestion delay, etc.)
- Advanced section mentions custom KQL via azure.yaml (with example)
- Proper meta tags and SEO

### 5. Update azure-yaml reference with logs examples
**Assignee**: Developer
**Description**: Add logs.analytics configuration examples to `/reference/azure-yaml.astro`. Show both project-level and service-level configurations with real examples from azure-logs-test. Focus on common use cases: polling intervals, time spans, and table selection.
**Acceptance**:
- logs.analytics section added to page
- Example shows pollingInterval and defaultTimespan (project-level)
- Example shows table selection for service-level override
- Optional: Show custom KQL query example in "Advanced" callout
- Links to new azure-logs.astro reference page

## TODO: Update Marketing Content

### 6. Add Azure logs feature to homepage
**Assignee**: Marketer → Developer
**Description**: Add "Azure Cloud Monitoring" feature card to homepage features grid. Update "Unified Logs" feature description to mention Azure. Add Azure logs screenshot to DashboardCarousel. Keep messaging realistic about 1-5min latency and UI-supported features only.
**Acceptance**:
- New feature card with ☁️ icon, "Azure Cloud Monitoring" title
- Description: "Stream live logs from Azure Container Apps, App Service, and Functions directly into your local dashboard. Real-time insights with 1-5 minute latency."
- Links to `/reference/azure-logs/`
- "Unified Logs" feature updated to mention "including live Azure cloud logs"
- DashboardCarousel includes dashboard-azure-logs.png in rotation (4th position)
- No over-promising features (no mention of custom KQL, realtime, etc.)

### 7. Update quick-start with Azure logs mention
**Assignee**: Developer
**Description**: Add section or callout in quick-start.astro mentioning Azure logs capability. Brief mention only, link to full docs.
**Acceptance**:
- Section added after main tutorial (e.g., "What's Next" or "Advanced Features")
- 1-2 sentences about Azure cloud log streaming
- Link to `/reference/azure-logs/`
- Does not distract from core quick-start flow

### 8. Create or update tour step for Azure logs
**Assignee**: Developer
**Description**: Update `/tour/6-logs.astro` to include section on Azure cloud logs, or create new `/tour/6b-azure-logs.astro` tour step. Show screenshot and configuration example.
**Acceptance**:
- Tour step includes Azure logs screenshot
- Shows azure.yaml configuration example
- Explains when to use (deployed services vs local-only)
- "Try It Yourself" section with azd provision mention
- Tour navigation updated if new step created

## TODO: Polish and Review

## Done

### 1. Update screenshot script to use azure-logs-test project ✅
Updated capture-screenshots.ts to use azure-logs-test, added Azure CLI auth checks, and increased wait times for Log Analytics polling.

### 2. Add Azure logs screenshot configurations ✅
Added three new screenshot configs: dashboard-azure-logs, dashboard-azure-logs-time-range, and dashboard-azure-logs-filters.

### 3. Capture and optimize new screenshots ✅
All 6 screenshots captured successfully (3 updated existing + 3 new Azure logs).

### 4. Create comprehensive Azure logs reference page ✅
Created /reference/azure-logs.astro with full documentation of Azure Cloud Log Streaming feature.

### 5. Update azure-yaml reference with logs examples ✅
Added logs.analytics configuration examples and link to new azure-logs reference page.

### 6. Add Azure logs feature to homepage ✅
Added "Azure Cloud Monitoring" feature card, updated "Unified Logs" description, and added Azure logs screenshot to carousel.

### 7. Update quick-start with Azure logs mention ✅
Added Azure Cloud Logs to "What's Next?" section in quick-start.astro.

### 8. Create or update tour step for Azure logs ✅
Updated /tour/6-logs.astro with comprehensive Azure cloud logs section, screenshot, and configuration example.

### 11. Capture tour enhancement screenshots ✅
All 4 screenshots captured successfully: dashboard-services-health.png, console-local-logs.png, console-log-search.png, and health-view.png.

### 12. Add screenshots to tour step 5 (Dashboard) ✅
Added Screenshot component import and integrated dashboard-resources-table.png and dashboard-services-health.png into tour step 5.

### 13. Add screenshots to tour step 6 (Logs) ✅
Added console-local-logs.png and console-log-search.png screenshots to tour step 6, dashboard-azure-logs.png already present.

### 14. Add screenshot to tour step 7 (Health) ✅
Added Screenshot component import and integrated health-view.png into tour step 7 after "Understanding Status vs Health" section.

### 15. Add hero screenshot to Quick Start page ✅
Added dashboard-console.png screenshot to Step 3 in quick-start.astro with priority loading and 700px max width.

### 16. Review all marketing copy for consistency ✅
Completed comprehensive review of all pages. Made multiple corrections to ensure realistic Azure log latency messaging (1-5min), removed over-promising language ("live", "realtime" for Azure logs), and verified tone consistency.

### 17. Validate screenshots and documentation accuracy ✅
All 9 acceptance criteria validated and passing. All screenshots exist at proper resolution, all links work, all YAML examples valid, alt text present, loading strategies correct, technical accuracy verified. Test report created at docs/specs/azure-logs-web-docs/test-report.md.

---

**All Azure Logs Web Documentation Tasks Complete (Tasks 1-17)** ✅

See [docs/archive/azure-logs-web-docs-archive-001.md](../../archive/azure-logs-web-docs-archive-001.md) for full task history.
