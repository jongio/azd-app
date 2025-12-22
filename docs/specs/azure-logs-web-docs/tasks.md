<!-- NEXT: 1 -->
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

## TODO: Enhance Tour Pages with Screenshots

### 11. Capture tour enhancement screenshots
**Assignee**: Developer
**Description**: Capture additional screenshots for tour pages showing dashboard health view, console with logs, and search features. Use same screenshot script but add configs for these views.
**Acceptance**:
- `dashboard-services-health.png` captured (Services view showing health status indicators)
- `console-local-logs.png` captured (Console view with local logs, filters visible)
- `console-log-search.png` captured (Console with search term highlighted)
- `health-view.png` captured (Health tab or status view showing service health details)
- All at 900x600, 2x scale, optimized

### 12. Add screenshots to tour step 5 (Dashboard)
**Assignee**: Developer
**Description**: Use Screenshot component in `/tour/5-dashboard.astro` to show dashboard views. Add screenshots showing Services view and health indicators.
**Acceptance**:
- Import Screenshot component at top of file
- Screenshot of dashboard-resources-table.png added to "Services Overview" section
- Screenshot of dashboard-services-health.png added to "Real-time Updates" section
- Screenshots use `<Screenshot>` component with lightSrc/darkSrc (same file for now)
- Alt text descriptive: "Dashboard services view showing health indicators and port mappings"
- Screenshots properly positioned with margin above/below

### 13. Add screenshots to tour step 6 (Logs)
**Assignee**: Developer
**Description**: Add screenshots to `/tour/6-logs.astro` showing Console view with local logs and Azure logs. Show filtering and search capabilities.
**Acceptance**:
- Import Screenshot component
- Screenshot of console-local-logs.png added after "View All Logs" section
- Screenshot of console-log-search.png added to "Filter and Search Logs" section
- Screenshot of dashboard-azure-logs.png added to "Local vs Azure Logs" section in LearnMoreSection
- Captions explain what user is seeing: "Console view with real-time log streaming and service filters"
- Screenshots clickable with lightbox enabled

### 14. Add screenshot to tour step 7 (Health)
**Assignee**: Developer
**Description**: Add screenshot to `/tour/7-health.astro` showing health status view in dashboard.
**Acceptance**:
- Import Screenshot component
- Screenshot of health-view.png added after "Understanding Status vs Health" section
- Alt text describes health indicators visible in screenshot
- Caption: "Dashboard health view showing service status, uptime, and last check time"

### 15. Add hero screenshot to Quick Start page
**Assignee**: Developer
**Description**: Add screenshot to `/quick-start.astro` after step 3 showing successful `azd app run` with dashboard.
**Acceptance**:
- Screenshot added in Step 3 section after code block
- Uses dashboard-console.png showing services running
- Not too large (maxWidth: "700px")
- Screenshot has priority loading (above fold)
- Caption: "Dashboard opens automatically showing all your services running locally"

## TODO: Polish and Review

### 16. Review all marketing copy for consistency
**Assignee**: Marketer
**Description**: Review all new/updated marketing copy across homepage, tour, and reference pages. Ensure tone is developer-focused, messaging is clear, and value propositions align.
**Acceptance**:
- Consistent tone across all pages
- Key messages emphasized: local+cloud unified, no context switching, real-time insights with 1-5min latency
- No typos or grammatical errors
- Technical accuracy verified (no over-promising features)

### 17. Validate screenshots and documentation accuracy
**Assignee**: Tester
**Description**: Test all screenshots display correctly across all pages, all links work, and all code examples from documentation are valid. Verify screenshots show real data and proper resolution.
**Acceptance**:
- All 10+ screenshots visible on website at proper resolution
- Screenshots show in lightbox correctly when clicked (tour pages)
- All links to /reference/azure-logs/ resolve
- azure.yaml examples tested and work
- Alt text present for all screenshots
- Screenshots pass visual review (no artifacts, proper dark mode)
- Tour pages load quickly with lazy-loaded screenshots
- Quick Start hero screenshot loads with priority (not lazy)
- Technical accuracy verified

## Done

(Completed tasks will be moved here)
