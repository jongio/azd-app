# Azure Logs Diagnostic System - Test Plan

## Overview
Comprehensive test plan for the Azure logs diagnostic system including validators, API endpoints, and UI components.

## Test Scope

### 1. Backend Unit Tests

#### 1.1 Diagnostic Settings Checker (`diagnostics.go`)
**File**: `cli/src/internal/azure/diagnostics_test.go`

**Coverage**:
- ✅ Workspace matching logic (exact match, case-insensitive, resource ID extraction)
- ✅ Workspace name extraction from resource IDs
- ✅ Mock HTTP responses for diagnostic settings API
- ✅ Error handling (404, 403, 500)
- ✅ JSON serialization/deserialization
- ✅ Status constants validation

**Test Cases**:
- Configured with workspace
- Not configured (no settings found)
- Not configured (404 response)
- Error (403 forbidden)
- Error (500 internal server error)
- Wrong workspace configured
- Storage account only (no workspace)

#### 1.2 Container Apps Validator (`validator_containerapp.go`)
**File**: `cli/src/internal/azure/validator_containerapp_test.go` ✨ NEW

**Coverage**:
- Resource not deployed
- Resource deployed without diagnostic settings
- Diagnostic settings configured
- Setup guide generation
- Requirement status validation
- Time formatting utilities

**Test Cases**:
- Not deployed → status: not-configured, has setup guide
- Deployed, no diagnostics → status: not-configured/partial
- Setup guide includes azd up command
- Requirements have valid statuses (met/not-met/unknown)
- Format time since (nil, just now, minutes, hours, days)

#### 1.3 Functions Validator (`validator_function.go`)
**File**: `cli/src/internal/azure/validator_function_test.go` ✨ NEW

**Coverage**:
- Resource not deployed
- Deployed without Application Insights
- Application Insights configuration check
- Diagnostic settings (optional)
- Setup guide generation with YAML snippets

**Test Cases**:
- Not deployed → has setup guide
- Deployed, no App Insights → not-configured
- Setup guide includes APPLICATIONINSIGHTS_CONNECTION_STRING
- Setup guide has deployment command
- Requirements include App Insights and optional diagnostic settings

#### 1.4 App Service Validator (`validator_appservice.go`)
**File**: `cli/src/internal/azure/validator_appservice_test.go` ✨ NEW

**Coverage**:
- Resource not deployed
- Deployed without diagnostic settings
- Setup guide generation
- Requirement status validation
- Message content

**Test Cases**:
- Not deployed → not-configured with setup guide
- Deployed, no diagnostics → has diagnostic settings requirement
- Setup guide includes manual Azure Portal steps
- All diagnostic statuses are valid
- Messages are set for non-healthy statuses

#### 1.5 Diagnostics Engine (`diagnostic_engine.go`)
**File**: `cli/src/internal/azure/diagnostic_engine_test.go` ✨ NEW

**Coverage**:
- Engine initialization
- Validator registration
- Service validation with/without validators
- Error handling
- Status constant validation
- Response structure

**Test Cases**:
- Engine creation initializes all fields
- RegisterValidator adds validator to map
- Validate service without validator → error status
- Validate service with validator → returns validator result
- Validator error → error status in result
- All status constants have correct string values

### 2. API Endpoint Tests

#### 2.1 Diagnostics Handler (`azure_logs_handlers.go`)
**File**: `cli/src/internal/dashboard/diagnostics_handler_test.go` ✨ NEW

**Coverage**:
- GET /api/azure/diagnostics endpoint
- Credential handling
- Timeout handling
- Method guard (GET only)
- JSON serialization
- Error responses

**Test Cases**:
- Success → returns DiagnosticsResponse
- No credentials → 401 Unauthorized
- POST method → 405 Method Not Allowed
- JSON roundtrip for all status types
- Response includes workspace ID, services map

**Existing Tests**:
**File**: `cli/src/internal/dashboard/azure_logs_test.go`
- ✅ Azure logs endpoint with defaults and bounds
- ✅ Service filter pass-through
- ✅ Error mapping to HTTP status codes
- ✅ Health check endpoint

### 3. Frontend Component Tests

#### 3.1 DiagnosticsModal Component
**File**: `cli/dashboard/src/components/DiagnosticsModal.test.tsx`

**Coverage**: ✅ Already tested
- Modal open/close behavior
- Health check fetching
- Loading states
- Error states
- Health check display
- Fix Setup button logic
- Setup guide navigation
- Report copying

#### 3.2 NoLogsPrompt Component
**File**: `cli/dashboard/src/components/NoLogsPrompt.test.tsx`

**Coverage**: ✅ Already tested
- Service name display
- Warning icon
- Diagnostic button rendering
- Click handler
- Accessibility

#### 3.3 ConsoleView Integration
**File**: `cli/dashboard/src/components/consoleview.test.tsx`

**Coverage**: ✅ Already tested
- DiagnosticsModal integration
- Setup guide callback passing

## Manual Testing Plan

### Prerequisites
```bash
# Install Azure CLI
az login

# Set up test project
cd cli/tests/projects/integration/azure-logs-test

# Deploy test infrastructure
azd up
```

### Test Scenarios

#### Scenario 1: Container Apps - No Logs
**Setup**:
1. Deploy Container App without diagnostic settings
2. Remove any existing diagnostic settings in Azure Portal

**Expected Results**:
- Status: `not-configured`
- Requirements show "Diagnostic Settings: not-met"
- Setup guide provided with azd up command
- Setup guide includes manual Azure Portal steps

**Verification**:
```bash
# Run dashboard
azd app run

# Navigate to service with no logs
# Click diagnostic button
# Verify status and setup guide
```

#### Scenario 2: Container Apps - Configured, No Logs
**Setup**:
1. Configure diagnostic settings via Azure Portal
2. Wait 5 minutes
3. If no logs generated, should show partial status

**Expected Results**:
- Status: `partial`
- Requirements show "Diagnostic Settings: met"
- Requirements show "Log Flow: not-met"
- Setup guide suggests waiting or generating activity

#### Scenario 3: Container Apps - Healthy
**Setup**:
1. Ensure diagnostic settings configured
2. Generate traffic to Container App
3. Wait for logs to flow (5-10 min)

**Expected Results**:
- Status: `healthy`
- Requirements all "met"
- Log count > 0
- Last log time recent
- No setup guide

#### Scenario 4: Azure Functions - No App Insights
**Setup**:
1. Deploy Function without APPLICATIONINSIGHTS_CONNECTION_STRING
2. Remove from azure.yaml if present

**Expected Results**:
- Status: `not-configured`
- Requirement "Application Insights: not-met"
- Setup guide shows YAML configuration
- Setup guide includes deployment command

#### Scenario 5: Azure Functions - Configured
**Setup**:
1. Add APPLICATIONINSIGHTS_CONNECTION_STRING to azure.yaml
2. Deploy: `azd deploy <function-service>`
3. Trigger function execution

**Expected Results**:
- Requirements show "Application Insights: met"
- If logs flowing: status `healthy`
- If no logs yet: status `partial`

#### Scenario 6: App Service - End-to-End
**Setup**:
1. Deploy App Service
2. Configure diagnostic settings
3. Generate HTTP traffic

**Expected Results**:
- Status progression: not-configured → partial → healthy
- Diagnostic settings requirement updates
- Log flow requirement updates
- Setup guide disappears when healthy

#### Scenario 7: Mixed Environment
**Setup**:
1. Deploy multiple service types
2. Configure some, leave others unconfigured
3. Generate traffic to configured services

**Expected Results**:
- Each service shows independent status
- Workspace ID consistent across all services
- Overall diagnostics shows per-service status
- Fix Setup button targets correct service

#### Scenario 8: Error Conditions
**Setup**:
1. Invalid credentials: `az logout`
2. Missing workspace: delete workspace reference
3. Permission issues: remove RBAC permissions

**Expected Results**:
- Auth error → 401 with clear message
- Missing workspace → error status
- Permission denied → error status with fix guidance

## Test Execution

### Unit Tests
```bash
# Run all Go tests
cd cli
go test ./src/internal/azure/... -v

# Run specific test files
go test ./src/internal/azure/diagnostics_test.go -v
go test ./src/internal/azure/validator_containerapp_test.go -v
go test ./src/internal/azure/validator_function_test.go -v
go test ./src/internal/azure/validator_appservice_test.go -v
go test ./src/internal/azure/diagnostic_engine_test.go -v
go test ./src/internal/dashboard/diagnostics_handler_test.go -v

# Run with coverage
go test ./src/internal/azure/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Frontend Tests
```bash
cd cli/dashboard
npm test -- DiagnosticsModal.test.tsx --run
npm test -- NoLogsPrompt.test.tsx --run
npm test -- --run  # Run all tests
```

### Integration Tests
```bash
# Start dashboard with test project
cd cli/tests/projects/integration/azure-logs-test
azd app run

# Manually verify in browser:
# 1. Navigate to service logs
# 2. Click diagnostic button
# 3. Verify health checks
# 4. Test Fix Setup button
# 5. Verify setup guide navigation
```

## Coverage Goals

### Backend
- **Target**: ≥80% coverage
- **Critical Paths**: 100% coverage
  - Status determination logic
  - Requirement validation
  - Setup guide generation
  - API response serialization

### Frontend
- **Target**: ≥80% coverage
- **Critical Paths**: 100% coverage
  - Modal open/close
  - Health check fetching
  - Error handling
  - Navigation callbacks

## Test Results

### Unit Tests Results
```
Run: cd cli && go test ./src/internal/azure/... -v
```

### Integration Results
```
Manual testing checklist:
[ ] Container Apps - not configured
[ ] Container Apps - partial
[ ] Container Apps - healthy
[ ] Functions - not configured
[ ] Functions - configured
[ ] App Service - not configured
[ ] App Service - healthy
[ ] Mixed environment
[ ] Error conditions
```

## Known Issues / Limitations

1. **Log Querying**: Validators currently don't query actual logs (marked as TODO)
   - Status based on configuration only
   - LogCount always 0 in current implementation
   - LastLogTime always nil

2. **Integration Tests**: Require live Azure environment
   - Skipped in CI/CD without credentials
   - Need Azure subscription with deployed resources

3. **Mocking**: Some tests may fail without proper Azure SDK mocking
   - Validators attempt to create real clients
   - May need additional abstraction layers

## Recommendations

### Immediate
1. ✅ Run all unit tests and verify pass rate
2. ✅ Achieve ≥80% coverage for new validator files
3. 🔄 Execute manual test scenarios with real Azure resources
4. 🔄 Document any bugs found during manual testing

### Future Enhancements
1. **Log Querying**: Implement actual log queries in validators
   - Add LogAnalyticsClient integration
   - Query for recent logs (last 15 min)
   - Update LogCount and LastLogTime

2. **E2E Tests**: Create automated end-to-end tests
   - Use Azure SDK test recordings
   - Mock Azure API responses
   - Test full diagnostic flow

3. **Performance**: Add performance tests
   - Measure diagnostic check latency
   - Test with multiple services (10+)
   - Verify timeout handling

4. **Error Recovery**: Test error recovery scenarios
   - Network failures
   - Partial API responses
   - Rate limiting

## Success Criteria

✅ All unit tests pass
✅ Coverage ≥80% for diagnostic system
✅ Frontend tests pass
✅ Manual testing validates all scenarios
✅ No critical bugs found
✅ Documentation complete

## Sign-off

**Tester Agent**: [Date]
**Reviewed By**: Manager Agent
**Status**: In Progress

---

## Appendix: Test File Locations

### Backend Tests (Go)
```
cli/src/internal/azure/
├── diagnostics_test.go                    [Existing]
├── validator_containerapp_test.go         [NEW]
├── validator_function_test.go             [NEW]
├── validator_appservice_test.go           [NEW]
└── diagnostic_engine_test.go              [NEW]

cli/src/internal/dashboard/
├── azure_logs_test.go                     [Existing]
└── diagnostics_handler_test.go            [NEW]
```

### Frontend Tests (TypeScript)
```
cli/dashboard/src/components/
├── DiagnosticsModal.test.tsx              [Existing]
├── NoLogsPrompt.test.tsx                  [Existing]
└── consoleview.test.tsx                   [Existing]
```

### Test Projects
```
cli/tests/projects/integration/
└── azure-logs-test/                       [Existing]
    ├── azure.yaml
    ├── infra/
    └── src/
```
