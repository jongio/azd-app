# Service URL Configuration - Final Specification

## Overview

Allow users to configure custom URLs for accessing services when they differ from auto-discovered endpoints. This enables scenarios like custom domains, reverse proxies, CDNs, tunneling services (ngrok), and API gateways.

**Key Design Principle**: Original auto-discovered URLs are ALWAYS preserved. Custom URLs are stored separately and used for display/navigation purposes.

## Final Design

### URL Property Structure

Services have **two separate readonly URL properties** that preserve the original endpoints:

```yaml
services:
  api:
    project: ./backend
    ports: ["8080"]
    local:
      customUrl: https://api.ngrok.io  # Optional user override
    azure:
      customUrl: https://api.contoso.com  # Optional user override
      customDomain: api.contoso.com  # Alternative: domain-only format
```

**At Runtime**:
- `local.url` = `"http://localhost:8080"` (auto-discovered, readonly, NEVER overwritten)
- `local.customUrl` = `"https://api.ngrok.io"` (user-configured, used for display)
- `azure.url` = `"https://api-xyz.azurecontainerapps.io"` (auto-discovered, readonly, NEVER overwritten)
- `azure.customUrl` or computed from `azure.customDomain` = `"https://api.contoso.com"` (user-configured, used for display)

**Both URLs are always preserved** - the system shows:
- Original endpoint (for debugging, direct access)
- Custom access URL (for user-facing navigation)

### Schema v1.1 Changes

#### Removed
- **Deprecated `url` property** at service root level (was confusing, implied single computed value)

#### Added
- **`local` object** with:
  - `customUrl` (string, optional): Full HTTP/HTTPS URL for local development (e.g., ngrok tunnel)
  
- **`azure` object** with:
  - `customUrl` (string, optional): Full HTTP/HTTPS URL for Azure deployment (e.g., CDN endpoint)
  - `customDomain` (string, optional): Domain-only format (e.g., `www.contoso.com`), auto-converted to `https://domain`
    - Can be auto-discovered from Azure resource settings (App Service custom domain)
    - OR set locally in azure.yaml (overrides auto-discovery)

### Validation Rules

#### `local.customUrl` & `azure.customUrl`
- **Must** start with `http://` or `https://`
- **Must** have valid hostname
- **Maximum** length: 2048 characters
- Validated by: `urlutil.Validate()` in azd-core

#### `azure.customDomain`
- **Must NOT** include protocol (`http://` or `https://`)
- **Must** be valid domain name format
- **Maximum** length: 253 characters
- Each label (between dots) max 63 characters
- Validated by: `urlutil.ValidateDomain()` in azd-core

### Implementation Files

#### Core Changes
1. **c:\code\azd-core\urlutil\validate.go**
   - Added `ValidateDomain()` for domain-only validation
   - Existing `Validate()` for full HTTP/HTTPS URLs

2. **c:\code\azd-core\urlutil\validate_test.go**
   - 23 test cases for `ValidateDomain()`
   - Existing tests for `Validate()`

3. **c:\code\azd-app\cli\src\internal\service\config.go**
   - Updated `ValidateServiceConfig()` to use `ValidateDomain()` for `customDomain`
   - Validates `customUrl` fields with `Validate()`

4. **c:\code\azd-app\cli\src\internal\service\config_test.go**
   - Updated test cases for domain-only format
   - Added tests for all validation scenarios

5. **c:\code\azd-app\schemas\v1.1\azure.yaml.json**
   - Removed deprecated root `url` property
   - Added `local` and `azure` objects with proper patterns
   - Domain pattern excludes protocols: `^(?!https?://)[a-zA-Z0-9]...`

6. **c:\code\azd-app\web\src\pages\reference\azure-yaml.astro**
   - Complete documentation rewrite
   - Clear tables showing readonly vs writable properties
   - Examples emphasizing URL separation
   - Removed confusing "computed/priority" language

7. **c:\code\azd-app\cli\tests\projects\url-demo\azure.yaml**
   - Comprehensive test cases for all URL property combinations
   - 5 services demonstrating different scenarios

## Use Cases

### 1. Development Tunnels (ngrok, localhost.run)
```yaml
services:
  api:
    project: ./backend
    ports: ["8080"]
    local:
      customUrl: https://myapi.ngrok.io
```
- `local.url`: `http://localhost:8080` (original)
- Display: `https://myapi.ngrok.io` (for access)

### 2. Custom Domain with CDN
```yaml
services:
  web:
    host: containerapp
    project: ./frontend
    azure:
      customDomain: www.contoso.com
```
- `azure.url`: `https://web-abc.azurecontainerapps.io` (original)
- Display: `https://www.contoso.com` (user-facing)

### 3. API Gateway
```yaml
services:
  api:
    host: containerapp
    project: ./api
    azure:
      customUrl: https://api.contoso.com/v1
```
- `azure.url`: `https://api-xyz.azurecontainerapps.io` (original)
- Display: `https://api.contoso.com/v1` (through gateway)

### 4. Both Local and Azure Overrides
```yaml
services:
  fullstack:
    host: appservice
    project: ./app
    local:
      customUrl: https://app.ngrok.io
    azure:
      customDomain: app.contoso.com
```
- Local: `http://localhost:3000` → `https://app.ngrok.io`
- Azure: `https://app.azurewebsites.net` → `https://app.contoso.com`

## Display Behavior

### CLI Output (azd app info)
```
Services:
  api
    Local URL:    http://localhost:8080
    Custom URL:   https://api.ngrok.io  <-- Click to open
    
    Azure URL:    https://api-xyz.azurecontainerapps.io
    Custom URL:   https://api.contoso.com  <-- Click to open
```

### Dashboard
- **Primary Link**: Uses custom URL when configured
- **Tooltip/Secondary**: Shows original endpoint
- **Visual Indicator**: Icon or badge when custom URL active

### Logs
- Both URLs included in startup messages
- Clear labeling for debugging

## Backward Compatibility

### Deprecated Property
- Root-level `url` property removed from schema v1.1
- Old azure.yaml files with `url` may trigger warnings but won't break
- Migration path: Move to `local.customUrl` or `azure.customUrl`

### Default Behavior
- Services without custom URLs work exactly as before
- Auto-discovery unchanged
- No impact on existing configurations

## Testing Coverage

### Unit Tests
- ✅ `ValidateDomain()` - 23 test cases
- ✅ `ValidateServiceConfig()` - All validation scenarios
- ✅ Config parsing - v1.1 structure
- ✅ Backward compatibility

### Integration Tests
- ✅ url-demo project with 5 service variations
- ✅ Schema validation via JSON Schema
- ✅ Documentation examples

### Test Files
- `c:\code\azd-core\urlutil\validate_test.go`
- `c:\code\azd-app\cli\src\internal\service\config_test.go`
- `c:\code\azd-app\cli\tests\projects\url-demo\` (5 minimal test services)

## Documentation

### Updated Files
1. **azure.yaml Reference** (`web/src/pages/reference/azure-yaml.astro`)
   - Complete URL section rewrite
   - Clear readonly vs writable distinction
   - Multiple examples
   - Use case explanations

2. **Schema** (`schemas/v1.1/azure.yaml.json`)
   - IntelliSense support
   - Proper patterns and descriptions
   - Examples

3. **Test Project** (`cli/tests/projects/url-demo/`)
   - Demonstrates all scenarios
   - Minimal implementation for testing

## Open Questions (Resolved)

### ❓ Should `local.url` and `azure.url` be separate readonly fields?
**✅ Resolution**: Yes, absolutely. This preserves the original auto-discovered URLs for debugging and direct access.

### ❓ Should customDomain require protocol?
**✅ Resolution**: No. Domain-only format is cleaner and less error-prone. System auto-converts to `https://domain`.

### ❓ Can customDomain be auto-discovered from Azure?
**✅ Resolution**: Yes. System attempts Azure SDK discovery first, then falls back to user configuration in azure.yaml.

### ❓ Priority/override logic for URL display?
**✅ Resolution**: No priority/override. Both URLs preserved separately. Display logic shows custom URL for navigation, original URL for reference.

## Success Criteria

✅ Users can configure custom URLs via `local` and `azure` objects  
✅ Original URLs always preserved (never overwritten)  
✅ Both URLs displayed when they differ  
✅ Domain-only validation working (`ValidateDomain()`)  
✅ Full URL validation working (`Validate()`)  
✅ Schema v1.1 updated with correct patterns  
✅ Documentation comprehensive and accurate  
✅ Test coverage >=80% for new code  
✅ No breaking changes to existing configs  
✅ Clear migration path from deprecated `url` property  

## Timeline

- **Initial Implementation**: Deprecated `url` property, basic validation
- **Iteration 1**: Realized `url` as computed/priority was confusing
- **Iteration 2**: Redesigned to separate readonly URLs from custom overrides
- **Final**: Implemented v1.1 schema with `local`/`azure` objects, dual URL preservation

## Related Work

- **azd-core**: `urlutil` package for validation
- **Schema v1.1**: JSON Schema for azure.yaml
- **Documentation**: Complete reference guide
- **Testing**: url-demo project

## Future Enhancements (Out of Scope)

- Environment-specific URLs (dev/staging/prod)
- Multiple custom URLs per service
- Automatic reachability validation
- URL templates with variable substitution
- Integration with service mesh/ingress controllers
