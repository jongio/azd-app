# Task 5: CORS Configuration for Custom URLs - Analysis

## Task Assignment
Add CORS configuration for custom URLs (GitHub issue #95, task 5)

## Investigation Results

### Key Findings

1. **File mentioned in spec does not exist**
   - Spec mentions: `cli/src/internal/apphost/generate.go`
   - Reality: This file does not exist in the codebase
   - The `apphost` directory does not exist under `cli/src/internal/`

2. **Tool Purpose Mismatch**
   - `azd app` is a **local development orchestration tool** (runs services locally via Docker/processes)
   - The spec describes **Azure deployment CORS configuration** (App Service, Container Apps)
   - These are separate concerns handled by different tools

3. **No CORS Configuration in Current Codebase**
   - No CORS middleware found in local development server code
   - No infrastructure/bicep generation for CORS settings
   - No deployment-related CORS configuration

4. **What EXISTS vs What SPEC Describes**
   
   **EXISTS:**
   - `url` field in service configuration (`cli/src/internal/service/types.go`)
   - Validation for `url` (`cli/src/internal/service/config.go`)
   - Tests for `url` parsing and validation
   - Dashboard displays service URLs (local development)
   
   **SPEC DESCRIBES (doesn't exist):**
   - CORS configuration generation for Azure deployments
   - Bicep/ARM template generation with CORS settings
   - Local development CORS middleware
   - `cli/src/internal/apphost/generate.go` file

### Architecture Context

**`azd app` (this repository):**
- Runs services locally in Docker containers or as processes
- Provides dashboard UI for local development
- Does NOT deploy to Azure
- Does NOT generate infrastructure as code

**`azd` (different tool - azure-dev CLI):**
- Deploys to Azure (App Service, Container Apps, Functions, etc.)
- Generates/uses Bicep/Terraform for infrastructure
- Would be responsible for CORS configuration in Azure resources
- Not part of this repository

### Possible Interpretations

#### Option A: Spec is Incorrect/Outdated
The spec may have been written assuming `azd app` handles Azure deployments, when it actually only handles local development.

#### Option B: Future Feature Planning
The spec may be describing a future enhancement where `azd app` generates deployment configurations.

#### Option C: Local Development CORS Only
The task might only be about adding CORS headers to local development servers to support custom URLs during development.

## Recommendations

### 1. Clarify Task Scope with Stakeholders

**Questions to ask:**
- Should `azd app` generate Azure deployment configurations?
- Is this task only about local development CORS?
- Should this integrate with `azd` (azure-dev) for deployment?
- Is the file path `cli/src/internal/apphost/generate.go` correct?

### 2. If Task is Local Development CORS Only

**Implementation would involve:**
- No changes needed - local services define their own CORS
- Each service (Express, Flask, etc.) manages its own CORS
- `url` is already available in config for services to use
- Update documentation to show how services can use `url` for CORS

**Example for Express service:**
```javascript
const cors = require('cors');
const config = require('./azure-config.json'); // Contains url

const allowedOrigins = [
  'http://localhost:3000',
  config.services.api.url
].filter(Boolean);

app.use(cors({ origin: allowedOrigins }));
```

### 3. If Task is Azure Deployment CORS

**This would require:**
- Creating infrastructure generation module
- Creating `cli/src/internal/apphost/generate.go` or similar
- Generating Bicep/ARM templates with CORS settings
- Extracting origin from `url` (protocol + host)
- Adding to allowed origins for App Service/Container Apps
- Integration with deployment workflow

**This is a MAJOR feature** that doesn't exist and would require:
- Design document for infrastructure generation
- Integration with deployment tools
- Testing with actual Azure resources
- 40+ hours of development effort

## Current Status

**BLOCKED** - Cannot proceed without clarification:

1. The file specified in the spec doesn't exist
2. The tool's purpose (local dev) doesn't match the spec's description (Azure deployment)
3. No existing CORS configuration infrastructure to extend

## Next Steps

1. **Immediate:** Document findings (this file)
2. **Required:** Get clarification from stakeholders/product owner
3. **Then:** Implement based on clarified requirements

## Related Files

- Service configuration: `cli/src/internal/service/types.go`
- Config validation: `cli/src/internal/service/config.go`
- Tests: `cli/src/internal/service/config_test.go`
- Spec: `docs/specs/service-url/spec.md`
- Tasks: `docs/specs/service-url/tasks.md`

## Dashboard HTTP Server Analysis

**What exists:**
- Dashboard HTTP server at `cli/src/internal/dashboard/server_core.go`
- Serves dashboard UI and API endpoints (websockets, service status, logs)
- Routes defined in `server_routes.go`

**What it does NOT do:**
- Does NOT proxy user application HTTP requests
- Does NOT add CORS headers to user services
- Does NOT generate infrastructure/deployment configurations

**User services manage their own CORS:**
```javascript
// Example: Express service in user's code
const express = require('express');
const cors = require('cors');
const app = express();

// Service configures its own CORS
app.use(cors({
  origin: ['http://localhost:3000', 'https://myapp.example.com']
}));
```

## Implementation Options

### Option 1: Documentation Only (Recommended)

Since `url` already exists in configuration, document how services can use it:

**File:** `docs/guides/cors-with-alternate-urls.md`
```markdown
# Configuring CORS with Custom URLs

If you've configured a `url` for your service, you'll need to add
it to your service's CORS configuration.

## Express (Node.js)
\```javascript
const cors = require('cors');
const config = require('./config');  // Your config loader

const allowedOrigins = [
  'http://localhost:3000',
  config.url  // Use custom URL from azure.yaml
].filter(Boolean);

app.use(cors({ origin: allowedOrigins }));
\```

## Flask (Python)
\```python
from flask_cors import CORS

allowed_origins = [
    "http://localhost:5000",
    config.get("url")  # From azure.yaml
]
app = Flask(__name__)
CORS(app, origins=list(filter(None, allowed_origins)))
\```
```

**Effort:** 2 hours (documentation only)

### Option 2: Service Configuration Helper Utility

Create a Go utility to help services extract CORS origins from config:

**File:** `cli/src/internal/service/cors_helper.go`
```go
package service

import "net/url"

// ExtractCORSOrigins returns allowed origins including url
func ExtractCORSOrigins(svc *Service, localURL string) []string {
    origins := []string{localURL}
    
    if svc.Config != nil && svc.Config.URL != "" {
        if origin := extractOrigin(svc.Config.URL); origin != "" {
            origins = append(origins, origin)
        }
    }
    
    return origins
}

// extractOrigin extracts protocol + host from URL
func extractOrigin(rawURL string) string {
    u, err := url.Parse(rawURL)
    if err != nil {
        return ""
    }
    return u.Scheme + "://" + u.Host
}
```

**Use case:** Services could call this utility if written in Go
**Problem:** User services are typically Node.js, Python, etc., not Go
**Effort:** 4 hours + tests

### Option 3: Create Infrastructure Generation Feature (NOT RECOMMENDED)

This would require:
1. New package `cli/src/internal/infragen`
2. Bicep/ARM template generation for App Service & Container Apps
3. CORS configuration in templates
4. Integration with deployment workflow
5. Testing with actual Azure resources

**Effort:** 40+ hours
**Problem:** Doesn't fit tool's purpose (local dev, not deployment)

## Conclusion

**RECOMMENDATION: Option 1 (Documentation Only)**

**Rationale:**
1. `url` configuration already exists and is validated
2. Services manage their own CORS (as they should)
3. No code changes needed - just documentation
4. Minimal effort, maximum clarity

**Cannot implement task as specified because:**
1. Required file (`apphost/generate.go`) doesn't exist
2. Tool doesn't handle Azure deployments/CORS
3. Spec describes functionality that doesn't fit the tool's architecture
4. User services independently manage CORS (correct design)

**Final Recommendation:** 
- Create documentation guide (Option 1) to help users configure CORS
- Mark task as "Blocked - Spec Mismatch" or redefine scope
- If Azure deployment CORS is truly needed, reassign to `azd` CLI tool
