# Service URL Configuration - Tasks

<!-- NEXT: COMPLETE -->

**Note**: Refactored from `config.altUrl` to direct `url` property for simplicity and consistency with other service fields.

## TODO

## IN PROGRESS

## DONE

### 8. Add ports to url-demo azure.yaml
Update `cli/tests/projects/url-demo/azure.yaml` to specify service `ports` per schema (v1.1) so fixtures align with runtime detection.

### 7. Fix url-demo test app entrypoint
Add minimal app files for `cli/tests/projects/url-demo` so runtime detection succeeds. Provide Python entrypoint at `src/api/main.py` (and any other required stubs) or set explicit entrypoints in `azure.yaml`. Keep fixtures lightweight and aligned with service-url scenarios.

### 6. Add tests for custom URL configuration
Add unit tests for config parsing, validation, dashboard logic, console formatting, and CORS generation. Ensure >=80% coverage for new code.

### 5. Add CORS configuration for custom URLs
Update CORS configuration generation to include url origins. Apply to both Azure App Service and Container Apps. Update local development CORS middleware. Files: `cli/src/internal/apphost/generate.go`

### 4. Update console output formatting
Update console output utilities to display custom URLs alongside deployment URLs. Implement clear labeling (Deployment URL vs Access URL). Files: Console formatting utilities for service URL output

### 3. Update dashboard UI for custom URL display and navigation
Modify ServiceCard to display custom URL when configured. Update "Open" button logic to prefer url over default URL. Add visual indication (tooltip/icon) to clarify custom URL usage. Files: `cli/dashboard/src/components/ServiceCard.tsx`

### 2. Update dashboard TypeScript types and API
Extend service TypeScript interface to include url. Update API to return url when available. Files: `cli/dashboard/src/types/service.ts`

### 1. Update configuration model and parsing
Parse `url` from azure.yaml service config. Update service configuration model to include optional url field. Add validation for HTTP/HTTPS URLs. Files: `cli/src/internal/appconfig/config.go`, `cli/src/internal/repository/app_config.go`
