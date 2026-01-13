# Service URL Fields Implementation Tasks

<!-- NEXT: 1 -->

## TODO

### 1. Update Schema and Data Models (P0)
Update `azure.yaml` schema and TypeScript/Go data models to support five URL fields: `local.url`, `local.customUrl`, `azure.url`, `azure.customUrl`, and `azure.customDomain` (user-configurable with SDK fallback). Implement URL precedence logic.

**Files:**
- `schemas/v1.1/azure.yaml.json` - Add `local.customUrl`, `azure.customUrl`, and `azure.customDomain` schema definitions
- `cli/dashboard/src/types.ts` - Update `LocalServiceInfo` and `AzureServiceInfo` interfaces (add `customDomainSource`)
- `cli/src/internal/repository/app_config.go` - Update Go structs (add `CustomDomainSource`)

**Acceptance:**
- Schema validates `local.customUrl`, `azure.customUrl`, and `azure.customDomain` as optional strings
- TypeScript types match spec (5 URL fields + `customDomainSource`)
- Go types match spec (5 URL fields + `CustomDomainSource`)
- URL validation rules enforced (HTTP/HTTPS, max 2048 chars)

### 2. Configuration Parsing and Validation (P0)
Parse `local.customUrl`, `azure.customUrl`, and `azure.customDomain` from `azure.yaml` with proper validation. Implement backward compatibility for deprecated root-level `url` field.

**Files:**
- `cli/src/internal/appconfig/config.go` - Parse new fields, validate URLs, track customDomainSource
- Add URL validation utility (or use existing from azd-core)

**Acceptance:**
- Parses `local.customUrl`, `azure.customUrl`, and `azure.customDomain` correctly
- Sets `customDomainSource` to "user" when `azure.customDomain` is in config
- Validates URLs (protocol, host, length)
- Backward compat: root-level `url` → `azure.customUrl` with deprecation warning
- Clear error messages for invalid URLs

### 3. Dashboard UI - URL Precedence Display (P0)
Update dashboard UI to display URLs based on precedence rules with visual indicators (badges) and tooltips. Distinguish between user-configured and SDK-discovered `azure.customDomain`.

**Files:**
- `cli/dashboard/src/components/ServiceCard.tsx` - Implement precedence logic, badges (purple for user, gold for SDK)
- `cli/dashboard/src/components/ServiceDetailPanel.tsx` - Show all URLs with source indicators
- `cli/dashboard/src/components/ServiceTable.tsx` - Update table display

**Acceptance:**
- Local: Shows `local.customUrl` > `local.url`
- Azure: Shows `azure.customDomain` (user) > `azure.customDomain` (SDK) > `azure.customUrl` > `azure.url`
- Purple badge for user-configured fields (`local.customUrl`, `azure.customUrl`, `azure.customDomain` when user-set)
- Gold badge for `azure.customDomain` when SDK-discovered
- Cyan badge for auto-discovered URLs
- Tooltips show all URLs and source of `azure.customDomain` when overrides active

### 4. Azure SDK Custom Domain Discovery (P1)
Implement Azure SDK integration to retrieve custom domains from App Service and Container Apps AS FALLBACK (only when not user-configured). Add caching layer (5-minute TTL).

**Files:**
- `cli/src/internal/azure/customdomain.go` - NEW: Custom domain discovery
- `cli/src/internal/serviceinfo/serviceinfo.go` - Integrate SDK calls, check if user-configured first

**Acceptance:**
- Queries App Service custom domains (armappservice SDK) only when `azure.customDomain` not in config
- Queries Container Apps custom domains (armappcontainers SDK) only when `azure.customDomain` not in config
- Skips SDK query entirely if user has configured `azure.customDomain` (performance optimization)
- Sets `customDomainSource` to "azure-sdk" when discovered from Azure
- Async loading, doesn't block startup
- Caches SDK results for 5 minutes
- Graceful error handling (fallback to other URLs)
- Uses Azure CLI credentials

### 5. Console Output Formatting (P1)
Update `azd info` and other console commands to display all five URL fields clearly, indicating source of `azure.customDomain`.

**Files:**
- `cli/src/internal/cmd/info.go` - Format output to show all URLs with source indicators

**Acceptance:**
- Shows all available URLs (local.url, local.customUrl, azure.url, azure.customUrl, azure.customDomain)
- Indicates which URL is the effective access URL
- Shows source of azure.customDomain (user-configured vs Azure SDK)
- Simplified output when no overrides configured

### 6. Testing (P0)
Comprehensive unit, integration, and visual tests for all URL scenarios, including user-configured vs SDK-discovered `azure.customDomain`.

**Files:**
- Unit tests for URL precedence logic (including user vs SDK customDomain)
- Unit tests for URL validation
- Unit tests for `customDomainSource` tracking
- Integration tests for configuration parsing
- Dashboard visual tests for badges/tooltips (purple vs gold)
- Azure SDK mock tests

**Acceptance:**
- >=80% code coverage
- All precedence combinations tested (including user-configured customDomain overriding SDK)
- Validation edge cases covered
- SDK error handling tested
- SDK skip logic tested when user configures customDomain
- Dashboard visual regression tests pass (badge colors correct for user vs SDK)

### 7. Documentation (P1)
User and developer documentation for multi-field URL system, emphasizing dual nature of `azure.customDomain`.

**Files:**
- `docs/azure-yaml-reference.md` - Document all 5 URL fields, explain `azure.customDomain` can be user-configured OR SDK-discovered
- `docs/guides/cors-multi-url.md` - NEW: CORS guide for multiple URLs
- `docs/guides/custom-domains.md` - NEW: Azure custom domain (user override vs SDK discovery)
- `docs/guides/migration-service-url.md` - Migration from single-field

**Acceptance:**
- All 5 fields documented with examples
- `azure.customDomain` dual behavior clearly explained (user takes precedence over SDK)
- CORS configuration patterns explained
- Custom domain workflow documented (when to configure vs let SDK discover)
- Migration guide from old format
- Clear precedence rules documented

## IN PROGRESS

## DONE
