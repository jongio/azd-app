# Azure Logs Web Documentation & Screenshots

## Overview

Update the azd-app marketing website (#file:web) to showcase the new Azure Cloud Log Streaming feature with enhanced screenshots, comprehensive documentation, and compelling marketing materials. **EXPANDED**: Also enhance tour pages and quick start with screenshots throughout to provide visual learning aids.

## Problem

The current website lacks:
- Screenshots showing Azure logs integration with real Azure services (Container Apps, App Service, Functions)
- Dedicated documentation page for Azure logs feature
- Marketing copy highlighting the Azure logs capability
- Tour step demonstrating Azure cloud log streaming
- **Visual aids in tour pages** - text-heavy with no screenshots to show features
- **Quick start visual confirmation** - users can't see what success looks like

The existing screenshot capture system uses the `demo` project which contains only local services, not Azure-deployed services with real Log Analytics integration.

## Goals

1. Capture high-quality screenshots showing live Azure logs from real Azure services
2. Add Azure logs to the website's feature showcase (homepage, tour, features)
3. Create comprehensive Azure logs documentation page
4. Update marketing copy to highlight Azure cloud monitoring capabilities
5. Ensure screenshot automation uses `azure-logs-test` project for Azure-specific captures

## Solution

### 1. Screenshot Updates

**Switch Target Project**: Change screenshot script from `demo` → `azure-logs-test`

**Prerequisites**: 
- Azure resources must be deployed via `azd provision` in azure-logs-test
- Services must be running and generating logs (wait 5-10min for initial logs)
- Azure CLI authenticated (`az login`)
- Log Analytics workspace operational with 1-5min ingestion delay

**New Screenshots to Capture**:
- `dashboard-console.png` - Console view (update with azure-logs-test)
- `dashboard-resources-table.png` - Resources table view (update)
- `dashboard-resources-grid.png` - Resources grid view (update)
- **`dashboard-azure-logs.png`** - NEW: Logs view in Azure mode showing real logs from Container Apps, App Service, Functions
- **`dashboard-azure-logs-time-range.png`** - NEW: Logs view with time range preset selector visible (15m, 30m, 6h, 24h)
- **`dashboard-azure-logs-filters.png`** - NEW: Logs view with service, state, and health filters active (showing filtered results)

**Screenshot Configuration** (add to `screenshot-config.ts`):
```typescript
// Azure logs view - main view showing logs in Azure mode
{
  name: 'dashboard-azure-logs',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Console")' },
    { type: 'wait', delay: 2000 },
    { type: 'click', selector: 'button:has-text("Azure")' }, // Mode toggle to Azure
    { type: 'wait', delay: 15000 }, // Wait for Azure Log Analytics polling (1st cycle)
  ],
  requireServices: true,
},
// Azure logs with time range selector visible
{
  name: 'dashboard-azure-logs-time-range',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Console")' },
    { type: 'wait', delay: 2000 },
    { type: 'click', selector: 'button:has-text("Azure")' },
    { type: 'wait', delay: 15000 },
    { type: 'click', selector: 'select[aria-label="Time range"]' }, // Open dropdown
    { type: 'wait', delay: 500 },
  ],
  requireServices: true,
},
```

### 2. Documentation Pages

**New Page**: `/azd-app/reference/azure-logs.astro`
- Overview of Azure Cloud Log Streaming
- Supported Azure services (Container Apps, App Service, Functions)
- Configuration in azure.yaml (logs.analytics section)
- Table selection and filtering options
- Polling intervals and performance
- Troubleshooting and authentication requirements
- Advanced: Custom KQL queries via azure.yaml (for power users)

**Update Existing Pages**:
- `/azd-app/index.astro` - Add Azure logs to features grid
- `/azd-app/tour/6-logs.astro` - Add section on Azure cloud logs
- `/azd-app/quick-start.astro` - Mention Azure logs capability

### 3. Marketing Content

**Homepage Updates** (`index.astro`):
- Add "Azure Cloud Monitoring" feature card
- Update "Unified Logs" feature description to mention Azure
- Add Azure logs screenshot to carousel
- Keep messaging focused on supported features (avoid over-promising)

**Feature Highlights**:
```
Icon: ☁️
Title: "Azure Cloud Monitoring"
Description: "Stream live logs from Azure Container Apps, App Service, and Functions directly into your local dashboard. Real-time insights with 1-5 minute latency."
Href: "/azd-app/reference/azure-logs/"
```

**Key Messages to Emphasize**:
- Unified view of local AND Azure logs in one dashboard
- No need to switch to Azure Portal for log diagnostics
- Automatic service detection and configuration
- Time range presets for quick diagnostics (15m, 30m, 6h, 24h)
- Supports Container Apps, App Service, Azure Functions

**Avoid Over-Marketing**:
- Don't emphasize custom KQL (yaml-only, not a primary feature)
- Be clear about 1-5 minute ingestion latency (not true realtime)
- Don't promise features not in the UI (historical custom ranges, etc.)

### 4. Tour Integration

**New Tour Step**: `/azd-app/tour/6b-azure-logs.astro` (or update existing 6-logs.astro)
- Introduction to Azure cloud log streaming
- Screenshot showing Azure logs in dashboard
- Code example: azure.yaml logs.analytics configuration
- Try It Yourself: Deploy to Azure and view live logs locally

### 5. Component Updates

**DashboardCarousel.astro**:
- Add Azure logs screenshot to rotation
- Update alt text to be descriptive: "Dashboard showing Azure cloud logs from Container Apps, App Service, and Functions with time range presets"
- Ensure screenshots cycle shows the progression: Console → Services Table → Services Grid → Azure Logs

### 6. Tour Page Enhancements (NEW)

**Problem**: Tour pages are text-heavy with no visual aids. Users can't see what features look like before trying them.

**Solution**: Add Screenshot component usage throughout tour pages to show actual dashboard/console views.

**Tour Step 5 (Dashboard)**:
- Screenshot of Services view after "Services Overview" heading
- Screenshot showing health indicators after "Real-time Updates" heading
- Shows users what to expect when opening dashboard

**Tour Step 6 (Logs)**:
- Screenshot of Console with local logs after "View All Logs" heading
- Screenshot showing log search/filtering after "Filter and Search Logs" heading
- Screenshot of Azure logs in "Local vs Azure Logs" LearnMore section
- Demonstrates log viewing capabilities visually

**Tour Step 7 (Health)**:
- Screenshot of Health view after "Understanding Status vs Health" heading
- Shows health status table and indicators in dashboard
- Complements CLI output examples with visual UI reference

**Quick Start Page**:
- Add hero screenshot after Step 3 showing successful dashboard launch
- Helps users know what success looks like
- Increases confidence that setup worked correctly

**Screenshot Component Benefits**:
- Theme-aware (light/dark variants)
- Click-to-expand lightbox for detailed viewing
- Lazy loading for performance
- Accessible with proper alt text and captions

## Technical Details

### Feature Scope Clarification

**Supported in UI**:
- View Azure logs from Container Apps, App Service, Functions
- Time range presets (15m, 30m, 6h, 24h)
- Service filtering (show/hide specific services)
- State filtering (show only running, stopped, or starting services)
- Health filtering (show only healthy, degraded, unhealthy, or unknown services)
- Log level filtering (info, warning, error)
- Search and highlighting
- Table selection per service (via settings)
- Polling interval configuration

**Filter Behavior**:
- Filters work when users actively click them to show/hide services
- Services never disappear automatically when their state/health changes
- All filters persist across sessions

**Supported via azure.yaml Configuration Only** (Not in UI):
- Custom KQL queries (advanced users)
- Custom table combinations beyond defaults
- Historical time ranges beyond presets

**Documentation Strategy**:
- Primary docs focus on UI-supported features
- Custom KQL mentioned in "Advanced Configuration" section
- Clear distinction between dashboard features and yaml-only features

### Screenshot Script Changes

**File**: `web/scripts/capture-screenshots.ts`

**Changes Required**:
1. Change `DEMO_DIR` path from `cli/demo` to `cli/tests/projects/integration/azure-logs-test`
2. Add pre-deployment step to ensure Azure resources exist:
   ```typescript
   // Before starting azd app run, check if Azure resources are deployed
   // If not, run: azd provision (or document manual setup requirement)
   ```
3. Increase wait times for Azure Log Analytics (polling takes longer than local logs: 15s minimum)
4. Add authentication check (Azure CLI login required: `az account show`)

**New Screenshot Configs** (add to `screenshot-config.ts`):

**Azure Logs Screenshots**:
```typescript
// Main Azure logs view
{
  name: 'dashboard-azure-logs',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Console")' },
    { type: 'wait', delay: 2000 },
    { type: 'click', selector: 'button[aria-label*="Azure"]' }, // Mode toggle
    { type: 'wait', delay: 15000 }, // Wait for Azure Log Analytics polling
  ],
  requireServices: true,
},
// Time range selector
{
  name: 'dashboard-azure-logs-time-range',
  viewport: { width: 900, height: 600 },
  actions: [
    // ... same as above ...
    { type: 'click', selector: 'select[aria-label="Time range"]' },
    { type: 'wait', delay: 500 },
  ],
  requireServices: true,
},
```

**Tour Enhancement Screenshots**:
```typescript
// Services with health indicators
{
  name: 'dashboard-services-health',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Services")' },
    { type: 'wait', delay: 1500 },
  ],
  requireServices: true,
},
// Console with local logs
{
  name: 'console-local-logs',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Console")' },
    { type: 'wait', delay: 2000 },
  ],
  requireServices: true,
},
// Console with search active
{
  name: 'console-log-search',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Console")' },
    { type: 'wait', delay: 2000 },
    { type: 'click', selector: 'input[placeholder*="Search"]' },
    { type: 'evaluate', script: 'document.querySelector("input[placeholder*=\"Search\"]").value = "error"' },
    { type: 'wait', delay: 1000 },
  ],
  requireServices: true,
},
// Health view (if separate tab exists)
{
  name: 'health-view',
  viewport: { width: 900, height: 600 },
  actions: [
    { type: 'click', selector: '[role="tab"]:has-text("Services")' },
    { type: 'wait', delay: 1500 },
    // Capture services view which includes health - may need adjustment based on actual UI
  ],
  requireServices: true,
},
```

**File**: `web/scripts/screenshot-config.ts`

Add configurations for Azure logs views (see Solution #1 above)

### Azure Logs Test Project

**Project**: `cli/tests/projects/integration/azure-logs-test`

**Requirements**:
- Azure subscription with deployed resources
- Azure CLI authentication
- Log Analytics workspace configured
- Services running and generating logs

**Pre-requisites** (document in script or README):
1. `az login` - authenticate to Azure
2. `azd provision` - deploy resources if not already deployed
3. Ensure services are generating logs (may need to trigger some activity)

### Documentation Structure

```
web/src/pages/reference/
├── azure-logs.astro          # NEW - comprehensive Azure logs docs
├── azure-yaml.astro          # Update with logs.analytics examples
├── containers.astro
└── cli/
    └── listen.astro          # Update with Azure logs info
```

## Marketing Copy Guidelines

**Tone**: Developer-focused, practical, emphasizes time-saving and convenience

**Key Messages**:
- "Monitor production without leaving your terminal"
- "Bridge local development and Azure cloud with a single command"
- "Real-time insights from Container Apps, App Service, and Functions"
- "No context switching between Azure Portal and local environment"

**Competitive Differentiators**:
- Unified local + cloud log streaming in one dashboard
- Automatic table selection optimized per Azure service type
- Time range presets (15m, 30m, 6h, 24h) for quick diagnostics
- Automatic service detection and log configuration
- Works with existing Azure deployments (no code changes)
- Advanced: Custom KQL queries via configuration for power users

## Success Criteria

- [ ] New Azure logs screenshots captured showing real Azure logs from azure-logs-test project
- [ ] Azure logs feature prominently displayed on homepage
- [ ] Comprehensive Azure logs documentation page created
- [ ] Tour updated with Azure cloud monitoring section
- [ ] **Tour pages enhanced with screenshots (5 new screenshots added across 4 pages)**
- [ ] **Quick Start page includes hero screenshot showing dashboard**
- [ ] Screenshot automation updated to use azure-logs-test
- [ ] All screenshots (10+ total) optimized and properly sized
- [ ] Marketing copy reviewed and approved
- [ ] Screenshots show real data from live Azure services (not mock/demo data)
- [ ] All pages use Screenshot component for consistency and accessibility
- [ ] No over-promising of features (realistic about latency and capabilities)

## Dependencies

- Azure subscription with deployed resources from azure-logs-test
- Azure CLI authentication configured
- Log Analytics workspace operational
- Services in azure-logs-test actively generating logs

## Out of Scope

- Creating new Azure resources (assumes azure-logs-test is already deployed)
- Modifying Azure logs feature functionality (feature is complete)
- Video recordings or animated GIFs (screenshots only)
- Documentation for other Azure services not yet supported

## Future Considerations

- Interactive demo allowing users to filter/search screenshots
- Video walkthrough of Azure logs feature
- Screenshots showing Azure logs in different color themes
- Mobile-responsive screenshots (smaller viewports)

## Implementation Notes

**Screenshot Timing**: Azure logs have higher latency than local logs due to Log Analytics polling. Screenshot script should:
1. Wait for initial poll cycle (configurable, ~10-30s)
2. Verify logs are visible before capture
3. Show meaningful log data (not empty state)

**Authentication**: Script should check for:
- Azure CLI logged in (`az account show`)
- Proper subscription selected
- Access to Log Analytics workspace

**Fallback Strategy**: If Azure resources aren't available:
- Document manual screenshot process
- Provide pre-captured screenshots as backup
- Skip Azure-specific screenshots in CI/CD (document deployment requirement)

## Files Affected

### Scripts
- `web/scripts/capture-screenshots.ts` - Update demo project path to azure-logs-test, add Azure CLI checks, increase wait times for Log Analytics polling (15s+)
- `web/scripts/screenshot-config.ts` - Add 3 Azure logs screenshot configs
- `web/scripts/screenshot-io.ts` - May need Azure CLI helpers

### Pages
- `web/src/pages/index.astro` - Add Azure logs feature card (realistic messaging)
- `web/src/pages/quick-start.astro` - Brief mention of Azure logs capability + add hero screenshot after step 3
- `web/src/pages/reference/azure-logs.astro` - NEW PAGE (comprehensive Azure logs docs)
- `web/src/pages/reference/azure-yaml.astro` - Add logs.analytics examples (focus on presets, table selection; custom KQL in "Advanced" section)
- `web/src/pages/tour/5-dashboard.astro` - Add 2 screenshots (services view, health indicators)
- `web/src/pages/tour/6-logs.astro` - Add 3 screenshots (local logs, search, Azure logs)
- `web/src/pages/tour/7-health.astro` - Add 1 screenshot (health view)

### Components
- `web/src/components/DashboardCarousel.astro` - Add dashboard-azure-logs.png to rotation

### Assets (NEW - Azure Logs)
- `web/public/screenshots/dashboard-azure-logs.png` - Main Azure logs view
- `web/public/screenshots/dashboard-azure-logs-time-range.png` - Time range selector visible (15m, 30m, 6h, 24h)
- `web/public/screenshots/dashboard-azure-logs-filters.png` - Service filter active

### Assets (NEW - Tour Enhancement)
- `web/public/screenshots/dashboard-services-health.png` - Services view with health indicators
- `web/public/screenshots/console-local-logs.png` - Console view showing local logs with filtering
- `web/public/screenshots/console-log-search.png` - Console with search/highlight active
- `web/public/screenshots/health-view.png` - Health status view from dashboard

### Assets (UPDATE)
- `web/public/screenshots/dashboard-console.png` - Re-capture with azure-logs-test
- `web/public/screenshots/dashboard-resources-table.png` - Re-capture
- `web/public/screenshots/dashboard-resources-grid.png` - Re-capture

## Testing Plan

1. **Screenshot Quality**
   - Verify all screenshots show real Azure logs
   - Check for proper resolution (2x for retina)
   - Validate no sensitive data visible (subscription IDs, etc.)

2. **Documentation Accuracy**
   - Test all code examples from docs
   - Verify Azure logs configuration examples work
   - Check all links resolve correctly

3. **Visual Consistency**
   - Screenshots match current dashboard design
   - Consistent viewport sizes across all screenshots
   - Dark mode theme applied consistently

4. **Accessibility**
   - Alt text provided for all screenshots
   - Screenshots have proper captions
   - Color contrast sufficient in screenshots

## Timeline Estimate

### Azure Logs Implementation
- Screenshot script updates: 2-3 hours
- Azure logs screenshot capture + optimization: 1-2 hours
- Azure logs documentation page: 3-4 hours
- Homepage and feature updates: 2-3 hours
- Tour Azure logs integration: 2 hours

### Tour Enhancement (NEW)
- Capture tour enhancement screenshots: 1-2 hours
- Add screenshots to tour step 5: 1 hour
- Add screenshots to tour step 6: 1 hour
- Add screenshot to tour step 7: 0.5 hours
- Add screenshot to quick start: 0.5 hours

### Polish
- Review and polish: 1-2 hours
- Testing and validation: 1 hour

**Total**: ~16-23 hours (up from original 11-16 hours due to expanded screenshot coverage)

## Risks

1. **Azure Resource Availability**: azure-logs-test may not be deployed or generating logs
   - Mitigation: Document pre-requisites, provide setup script

2. **Log Analytics Latency**: Logs may take time to appear
   - Mitigation: Build in sufficient wait times, retry logic

3. **Authentication Issues**: Azure CLI not logged in or expired token
   - Mitigation: Pre-flight checks, clear error messages

4. **Screenshot Consistency**: Timing issues may cause partial renders
   - Mitigation: Multiple validation checks, retry failed captures
