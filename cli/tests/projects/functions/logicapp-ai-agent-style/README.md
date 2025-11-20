# Logic Apps AI Agent Style Test Project

## Purpose

This test project validates Azure Functions support for Logic Apps Standard integrated with Azure AI Foundry. It mirrors real-world AI agent patterns and tests:

1. **Logic Apps Detection** - Validates `azd app` correctly detects Logic Apps via extension bundle
2. **Complex Configuration** - Tests support for AI service connections and managed identities
3. **Environment Variables** - Ensures AI Foundry settings are passed correctly
4. **Modular Bicep** - Validates infrastructure provisioning patterns
5. **Prerequisites Detection** - Verifies `azd app reqs` detects Azure Functions Core Tools

## Architecture

```
┌─────────────────────┐
│   Logic App Workflow│
│   (Stateful)        │
└──────────┬──────────┘
           │
           │ Managed Identity Auth
           │
┌──────────▼──────────┐
│   Azure AI Foundry  │
│   (GPT-4o model)    │
└─────────────────────┘
```

### Components

- **Logic App Standard**: Stateful workflow with HTTP trigger
- **Azure AI Foundry**: GPT-4o deployment for AI completions
- **Key Vault**: Secrets management (optional for testing)
- **Storage Account**: Logic Apps state persistence
- **Application Insights**: Monitoring and logging
- **Managed Identities**: Both user-assigned and system-assigned

## Project Structure

```
logicapp-ai-agent-style/
├── azure.yaml                          # azd service configuration
├── src/                                # Logic Apps source code
│   ├── host.json                       # Functions runtime configuration
│   ├── connections.json                # AI Foundry API connection
│   └── workflows/
│       └── AIChatWorkflow/
│           └── workflow.json           # Stateful workflow definition
└── infra/                              # Bicep infrastructure
    ├── main.bicep                      # Main orchestration (subscription scope)
    ├── resources.bicep                 # Resource collection module
    ├── monitoring.bicep                # App Insights + Log Analytics
    ├── abbreviations.json              # Azure naming conventions
    ├── aifoundry/
    │   ├── aifoundry.bicep             # AI Foundry + GPT-4o deployment
    │   └── role-assignment.bicep       # RBAC for Logic App
    ├── storage/
    │   └── storage.bicep               # Storage with role assignments
    ├── keyvault/
    │   └── keyvault.bicep              # Key Vault with RBAC
    └── logicapp/
        ├── plan.bicep                  # App Service Plan (WS1)
        └── workflows.bicep             # Logic App site + identities
```

## Workflow Definition

The `AIChatWorkflow` is a stateful Logic Apps workflow that:

1. **Trigger**: HTTP request with JSON body `{"message": "user prompt"}`
2. **Action**: Call Azure OpenAI GPT-4o model via AI Foundry connection
3. **Response**: Return AI-generated completion

### Request Example

```bash
POST http://localhost:7071/runtime/webhooks/workflow/api/management/workflows/AIChatWorkflow/triggers/manual/paths/invoke
Content-Type: application/json

{
  "message": "What is Azure Logic Apps?"
}
```

### Response Example

```json
{
  "content": "Azure Logic Apps is a cloud-based platform that allows you to automate workflows..."
}
```

## Environment-Based Configuration

The project supports multiple environments (dev/tst/acc/prd):

### App Settings

| Setting | Description | Example |
|---------|-------------|---------|
| `AI_FOUNDRY_NAME` | AI Foundry instance name | `ai-foundry-dev` |
| `AI_FOUNDRY_ENDPOINT` | AI Foundry endpoint URL | `https://ai-foundry-dev.openai.azure.com/` |
| `AI_PROJECT_NAME` | AI project name | `chat-agent` |
| `AI_PROJECT_ENDPOINT` | AI project endpoint | `https://ai-foundry-dev.openai.azure.com/...` |

### Managed Identity Configuration

- **User-Assigned Identity**: For AI Foundry access
- **System-Assigned Identity**: For Key Vault and Storage access

### RBAC Roles

| Resource | Role | Purpose |
|----------|------|---------|
| AI Foundry | Cognitive Services Contributor | Model deployment management |
| AI Foundry | Azure AI Administrator | Full AI service admin |
| AI Foundry | Azure AI User | Runtime AI API access |
| Key Vault | Key Vault Secrets User | Read secrets |
| Storage | Storage Blob Data Contributor | Read/write blob data |

## azd Workflow

### 1. Prerequisites Detection

```bash
azd app reqs
```

**Expected Output**:
```
Checking prerequisites for service 'workflows' (Azure Functions - Logic Apps)...

✓ Azure Functions Core Tools: 4.0.5907
  Minimum required: 4.0.0

All prerequisites met.
```

### 2. Provision Infrastructure

```bash
azd provision
```

Creates:
- Resource Group
- AI Foundry instance with GPT-4o deployment
- Storage Account
- Key Vault
- App Service Plan (Workflow Standard WS1)
- Logic App with managed identities
- RBAC role assignments
- Application Insights + Log Analytics

### 3. Deploy Logic App

```bash
azd deploy
```

Deploys workflow to Logic App using `azd-service-name` tag.

### 4. Test Workflow

```bash
azd app run
```

Starts Logic App locally for development:
- Runs `func start --port 7071`
- Loads workflows from `src/workflows/`
- Connects to local storage emulator or Azure Storage
- Uses managed identity for AI Foundry (when deployed)

## Testing Validation Criteria

### Detection Tests

- ✅ Detected as Logic Apps via extension bundle in `host.json`
- ✅ Language detected as "Logic Apps"
- ✅ Framework detected as "Logic Apps Functions"
- ✅ Port defaults to 7071

### Runtime Tests

- ✅ Starts with `func start --port 7071`
- ✅ Health check succeeds at `/runtime/webhooks/workflow/api/management/workflows`
- ✅ Workflow appears in Functions runtime
- ✅ HTTP trigger is accessible

### Prerequisites Tests

- ✅ `azd app reqs` detects Azure Functions Core Tools
- ✅ Minimum version 4.0.0 validated
- ✅ Helpful installation instructions if missing
- ✅ Optional .NET SDK detected for local debugging

### Configuration Tests

- ✅ Supports complex `connections.json` with AI Foundry settings
- ✅ Environment variables passed to `func` process
- ✅ Managed identity configuration preserved
- ✅ App settings loaded correctly

### Infrastructure Tests

- ✅ `azd provision` creates all resources
- ✅ AI Foundry instance created with GPT-4o deployment
- ✅ Managed identities configured
- ✅ RBAC roles assigned correctly
- ✅ `azd deploy` succeeds using service tag

## Prerequisites

### Required Tools

| Tool | Version | Install Command |
|------|---------|-----------------|
| Azure Functions Core Tools | 4.0+ | `winget install Microsoft.Azure.FunctionsCoreTools` |
| .NET SDK (optional) | 6.0 or 8.0 | For local debugging of Logic Apps runtime |

### Verify Installation

```bash
# Check Functions Core Tools
func --version
# Expected: 4.0.5907 or later

# Check .NET SDK (optional)
dotnet --version
# Expected: 6.0.x or 8.0.x
```

## Local Development

### 1. Install Dependencies

```bash
# No package installation needed for Logic Apps
# Workflows are JSON-defined
```

### 2. Configure Local Settings

Create `src/local.settings.json`:

```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "node",
    "Workflows.AIChatWorkflow.FlowState": "Enabled",
    "AI_FOUNDRY_NAME": "your-ai-foundry-instance",
    "AI_FOUNDRY_ENDPOINT": "https://your-ai-foundry.openai.azure.com/"
  }
}
```

### 3. Start Logic App

```bash
cd src
func start --port 7071
```

### 4. Test Workflow

```bash
# Get workflow callback URL
curl http://localhost:7071/runtime/webhooks/workflow/api/management/workflows/AIChatWorkflow/triggers/manual/listCallbackUrl -X POST

# Invoke workflow
curl -X POST http://localhost:7071/runtime/webhooks/workflow/api/management/workflows/AIChatWorkflow/triggers/manual/paths/invoke \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from Logic Apps!"}'
```

## Troubleshooting

### Issue: `func: command not found`

**Solution**: Install Azure Functions Core Tools v4+
```bash
winget install Microsoft.Azure.FunctionsCoreTools
```

### Issue: Workflow not found

**Solution**: Ensure workflow.json is in correct location
```
src/workflows/AIChatWorkflow/workflow.json
```

### Issue: Connection not found

**Solution**: Verify connections.json exists and has valid AI Foundry config
```json
{
  "serviceProviderConnections": {
    "AzureOpenAI": { ... }
  }
}
```

### Issue: Health check fails

**Solution**: 
1. Ensure Logic Apps runtime started successfully
2. Check logs for workflow loading errors
3. Verify `func` version is 4.0+

## Integration with azd app

This test project validates:

1. **Detection**: `azd app` correctly identifies Logic Apps via extension bundle
2. **Configuration**: Complex AI Foundry connections are supported
3. **Environment**: AI settings passed as environment variables
4. **Runtime**: `func start` runs Logic Apps with workflows
5. **Health**: Health check endpoint responds correctly
6. **Prerequisites**: `azd app reqs` detects required tools

## References

- [Azure Logic Apps Standard](https://learn.microsoft.com/azure/logic-apps/single-tenant-overview-compare)
- [Azure AI Foundry](https://learn.microsoft.com/azure/ai-services/openai/)
- [Azure Functions Core Tools](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Managed Identities](https://learn.microsoft.com/entra/identity/managed-identities-azure-resources/overview)
- [Original AI Agent Pattern](https://github.com/marnixcox/logicapp-ai-agent) (inspiration)

## Contributing

When adding features or fixing bugs related to Logic Apps AI Agent scenarios:

1. Update workflow.json if trigger/action changes
2. Update connections.json if AI service config changes
3. Update infra/ bicep if resources change
4. Update this README with new validation criteria
5. Run full test suite to verify no regressions
