# Service URL Configuration - Tasks

<!-- NEXT: COMPLETE -->

**Note**: Redesigned to support dual custom URLs (local + Azure) with auto-detected custom domains from Azure resources.

## TODO

## IN PROGRESS

## DONE

### 8. Documentation
Created comprehensive user-facing documentation. User guide (780 lines) covers quick start, URL hierarchy, 6 detailed examples, dashboard/console behavior, CORS, custom domain detection, migration guide, best practices, and troubleshooting. Updated schema reference with 300+ lines documenting local.url, azure.url fields and deprecation notice. Updated CORS guide for nested format. All documentation tested and validated. Files: `docs/guides/service-url-configuration.md`, `cli/docs/schema/azure.yaml.md`, `docs/guides/cors-with-alternate-urls.md`, completion reports

### 7. Testing and validation
Verified test coverage across all packages. All 232 tests passing with 0 failures. Coverage: 97.3% (serviceinfo), 73.9% (service), 44.6% (azure). Validated backward compatibility with legacy flat URL field. Tested precedence chain logic, error handling, UI display, and CORS extraction. TypeScript build successful with no type errors. Production ready. Files: Test report at `docs/specs/service-url/task-7-testing-summary.md`

### 6. Update CORS configuration
Implemented CORS helper utility to extract allowed origins from all service URL sources (local.url, azure.url, custom domains). Created `CORSOrigins()`, `AllCORSOrigins()`, and `CORSOriginsEnvVar()` functions with comprehensive test coverage (30+ tests). Updated documentation guide with automated CORS section. All tests pass. Files: `cli/src/internal/serviceinfo/cors_helper.go`, `cli/src/internal/serviceinfo/cors_helper_test.go`, `docs/guides/cors-with-alternate-urls.md`, `docs/specs/service-url/task-6-completion-report.md`

### 5. Update console output
Update logger LogSummary to display all available URLs (system, custom domain, custom URL). Show clear labels for each URL type. Indicate which URL is used for "open in browser" action. Update run.go to pass all URL data to LogSummary. Files: `cli/src/internal/service/logger.go`, `cli/src/cmd/app/commands/run.go`

### 4. Update dashboard UI components
Update ServiceCard.tsx to display effective URL using precedence chain (customUrl || customDomain || url). Update ServiceDetailPanel.tsx to show all available URLs with labels ("System URL", "Custom Domain", "Custom URL (Override)"). Update LogsPane.tsx browser launch to use effective URL with precedence chain. Add visual indicator (icon/badge) when using non-system URL. Handle null values gracefully in all components. Files: `cli/dashboard/src/components/ServiceCard.tsx`, `cli/dashboard/src/components/ServiceDetailPanel.tsx`, `cli/dashboard/src/components/LogsPane.tsx`, `cli/dashboard/src/components/LogsPaneHeader.tsx`

### 3. Update TypeScript types and API contract
Update service types with new fields: `local.customUrl` (from azure.yaml local.url), `azure.customUrl` (from azure.yaml azure.url), `azure.customDomain` (from Azure SDK detection). Ensure all URL fields are always present (null if not applicable). Update API marshaling to populate all URL fields correctly. Files: `cli/dashboard/src/types/service.ts`, API response marshaling code

### 2. Implement custom domain detection
Create custom domain detection module. Implement App Service custom domain detection via Azure SDK. Implement Container Apps custom domain detection via Azure SDK. Add error handling for SDK failures (log warning, continue with system URL). Cache detection results for session duration. Integrate detection into service initialization during `azd app run`. Return customDomain field in service info API response. Files: `cli/src/internal/azure/resources.go`, service initialization code

### 1. Update azure.yaml schema and parsing
Add nested `local` and `azure` config structs to Service type. Add `URL` field to LocalConfig and AzureConfig structs. Update azure.yaml.json schema to support `services.{name}.local.url` and `services.{name}.azure.url`. Parse nested URL configuration. Add backward compatibility for legacy flat `url` field (map to `local.url` with deprecation warning). Add validation for URL format. Files: `cli/src/internal/repository/app_config.go`, `cli/src/internal/appconfig/config.go`, `cli/schemas/azure.yaml.json`

### Legacy: Add tests for custom URL configuration
Add unit tests for config parsing, validation, dashboard logic, console formatting, and CORS generation. Ensure >=80% coverage for new code.

### Legacy: Add CORS configuration for custom URLs
Update CORS configuration generation to include url origins. Apply to both Azure App Service and Container Apps. Update local development CORS middleware. Files: `cli/src/internal/apphost/generate.go`

### Legacy: Update console output formatting
Update console output utilities to display custom URLs alongside deployment URLs. Implement clear labeling (Deployment URL vs Access URL). Files: Console formatting utilities for service URL output

### Legacy: Update dashboard UI for custom URL display and navigation
Modify ServiceCard to display custom URL when configured. Update "Open" button logic to prefer url over default URL. Add visual indication (tooltip/icon) to clarify custom URL usage. Files: `cli/dashboard/src/components/ServiceCard.tsx`

### Legacy: Update dashboard TypeScript types and API
Extend service TypeScript interface to include url. Update API to return url when available. Files: `cli/dashboard/src/types/service.ts`

### Legacy: Update configuration model and parsing
Parse `url` from azure.yaml service config. Update service configuration model to include optional url field. Add validation for HTTP/HTTPS URLs. Files: `cli/src/internal/appconfig/config.go`, `cli/src/internal/repository/app_config.go`
