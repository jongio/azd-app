# Spec: azd app init with Copilot SDK Integration

## Summary

A new `azd app init` command that uses GitHub Copilot SDK to help users generate their `azure.yaml` configuration file from natural language descriptions of their project. This command provides an intelligent, conversational interface for initializing Azure Developer CLI projects, reducing the learning curve and eliminating common configuration errors.

## Motivation

Currently, users must manually create `azure.yaml` files, which presents several challenges:

1. **Steep Learning Curve**: Understanding the azure.yaml schema structure, required vs optional fields, and proper YAML syntax requires extensive documentation reading
2. **Error-Prone**: Manual YAML creation is prone to indentation errors, typos in property names, and invalid value types
3. **Discovery Gap**: Users may not know about available azd app features like service types, health checks, or well-known services
4. **No Guidance**: No interactive help for determining appropriate ports, environment variables, or service configurations
5. **Context Loss**: Users must switch between documentation, their project structure, and the configuration file
6. **Repetitive Setup**: Similar projects require similar configurations, but users must recreate them each time

An AI-powered init command solves these problems by:
- Generating complete `azure.yaml` files from natural language descriptions
- Providing interactive, multi-turn conversations for complex projects
- Automatically detecting project structure and suggesting configurations
- Validating generated YAML against the JSON schema before writing
- Offering intelligent defaults based on detected languages and frameworks
- Reducing context switching through conversational refinement

## User Experience

### Entry Points

1. **New Project**: `azd app init` in an empty or new project directory
2. **Existing Project**: `azd app init` in a project with existing code but no `azure.yaml`
3. **Project Enhancement**: `azd app init` in a project with existing `azure.yaml` (adds services)

### Command Syntax

```bash
# Natural language description
azd app init "I have a Node.js API on port 3000 and a Python worker service"

# Interactive mode (guided conversation)
azd app init --interactive

# With automatic project detection
azd app init --detect

# Override output file
azd app init "description" --output custom.yaml

# Dry run (preview without writing)
azd app init "description" --dry-run

# Force overwrite existing azure.yaml
azd app init "description" --force
```

### Example Interaction Flow

#### Scenario 1: Simple Natural Language Description

```bash
$ azd app init "I have a Node.js API on port 3000 and a Python worker service"

🤖 Analyzing your project description...
🔍 Scanning project structure...

Detected:
  • Node.js project in ./api (package.json found)
  • Python project in ./worker (requirements.txt found)

Generated azure.yaml:
---
name: my-project
services:
  api:
    language: js
    project: ./api
    host: localhost
    ports:
      - "3000"
    environment:
      NODE_ENV: development
  worker:
    language: python
    project: ./worker
    host: localhost
    ports:
      - "8000"
    environment:
      PYTHON_ENV: development

Would you like to:
1. Save this configuration to azure.yaml
2. Make changes (interactive mode)
3. Add more services
4. Cancel

> 1

✓ Created azure.yaml
📝 Backup created: azure.yaml.backup.20260115T143022Z
```

#### Scenario 2: Interactive Mode

```bash
$ azd app init --interactive

🤖 Welcome to azd app init! I'll help you create your azure.yaml configuration.

Let's start by understanding your project. What services does your application have?

> I have a web frontend and a REST API backend

Great! I can help you set up both. Let me scan your project structure...

🔍 Detected:
  • React frontend in ./frontend
  • Node.js API in ./backend

For the frontend, what port should it run on?

> 3000

And for the backend API?

> 8080

Do you need any Azure services like a database or cache?

> Yes, I need a PostgreSQL database

Perfect! I'll add PostgreSQL. Would you like me to configure environment variables to connect the API to the database?

> Yes

[Generates configuration with database connection string...]

Here's your configuration:
---
name: my-project
services:
  frontend:
    language: js
    project: ./frontend
    host: localhost
    ports:
      - "3000"
  api:
    language: js
    project: ./backend
    host: localhost
    ports:
      - "8080"
    environment:
      DATABASE_URL: postgresql://localhost:5432/mydb
  postgres:
    host: containerapp
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: mydb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres

Does this look good? (yes/no/edit)

> edit

What would you like to change?

> Add a Redis cache

[Continues conversation to add Redis...]

✓ Created azure.yaml
```

#### Scenario 3: Project Detection Mode

```bash
$ azd app init --detect

🔍 Scanning project structure...

Detected:
  • Next.js application in ./web (next.config.js found)
  • FastAPI backend in ./api (main.py found)
  • PostgreSQL database dependency (requirements.txt includes psycopg2)

🤖 Based on your project structure, I suggest:

---
name: my-project
services:
  web:
    language: js
    project: ./web
    host: localhost
    ports:
      - "3000"
  api:
    language: python
    project: ./api
    host: localhost
    ports:
      - "8000"
    environment:
      DATABASE_URL: postgresql://localhost:5432/appdb
  postgres:
    host: containerapp
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: appdb

Would you like to:
1. Save this configuration
2. Customize it (interactive mode)
3. Add more services
4. Cancel

> 2

[Switches to interactive mode for customization...]
```

## Goals

1. **Reduce Setup Time**: Generate valid `azure.yaml` files in under 2 minutes
2. **Zero Configuration Errors**: All generated YAML validates against schema
3. **Intelligent Defaults**: Auto-detect languages, frameworks, and suggest appropriate configurations
4. **Conversational Refinement**: Multi-turn conversations for complex projects
5. **Project Awareness**: Integrate with existing project structure detection
6. **Safe Operations**: Always backup existing files before writing
7. **Schema Compliance**: Generated YAML matches azure.yaml v1.1 schema
8. **User Control**: Preview, edit, and refine before saving

## Non-Goals

1. **Full Editor Replacement**: Not replacing the visual YAML editor (complementary feature)
2. **Infrastructure Generation**: Not generating Bicep/Terraform files (only azure.yaml)
3. **Git Integration**: Not handling commits/pushes (separate concern)
4. **Multi-File Projects**: Only generating azure.yaml (not package.json, etc.)
5. **Production Configurations**: Focus on local development setup (Azure resources optional)
6. **Template Gallery**: Not browsing/importing from template gallery (future enhancement)

## Functional Requirements

### FR-1: Natural Language Processing

**Description**: Parse user descriptions and extract project requirements

**Requirements**:
- Accept natural language input describing project structure
- Extract key information:
  - Service names and types
  - Languages and frameworks
  - Port numbers
  - Dependencies (databases, caches, etc.)
  - Environment variables
- Handle ambiguous descriptions with clarifying questions
- Support multiple description formats:
  - "I have a Node.js API and Python worker"
  - "My app consists of a React frontend on port 3000 and Express backend on 8080"
  - "Web service, API service, and PostgreSQL database"

**Acceptance Criteria**:
- Parses service descriptions correctly
- Extracts port numbers when mentioned
- Identifies languages and frameworks from context
- Asks clarifying questions for ambiguous input
- Handles common variations in description format

---

### FR-2: Project Structure Detection

**Description**: Automatically detect existing project structure to inform configuration

**Requirements**:
- Scan current directory for:
  - Package manager files (package.json, requirements.txt, go.mod, etc.)
  - Framework indicators (next.config.js, vite.config.ts, etc.)
  - Service directories (./api, ./frontend, ./backend, etc.)
  - Docker files (Dockerfile, docker-compose.yml)
- Detect languages and frameworks:
  - Node.js (package.json, node_modules)
  - Python (requirements.txt, pyproject.toml, virtualenv)
  - .NET (csproj, sln files)
  - Go (go.mod, go.sum)
  - Docker (Dockerfile, docker-compose.yml)
- Identify port usage:
  - Scan for port configurations in code
  - Check for common port patterns (3000, 8080, 5000, etc.)
- Detect dependencies:
  - Database drivers (psycopg2, pg, mongodb, etc.)
  - Cache clients (redis, ioredis)
  - Message queues (amqp, kafka)

**Acceptance Criteria**:
- Correctly identifies project languages
- Detects framework types (Next.js, Express, FastAPI, etc.)
- Finds service directories
- Identifies likely port numbers
- Detects dependency requirements

---

### FR-3: Copilot SDK Integration

**Description**: Integrate GitHub Copilot SDK for AI-powered YAML generation

**Requirements**:
- Initialize Copilot SDK Go client
- Configure agent with azure.yaml-specific tools:
  - `validate_service_name`: Check service name format
  - `suggest_ports`: Recommend ports based on service type
  - `detect_language`: Identify language from project files
  - `suggest_environment_vars`: Recommend environment variables
  - `validate_yaml`: Validate generated YAML against schema
- Support multi-turn conversations:
  - Maintain conversation context
  - Handle follow-up questions
  - Refine configurations iteratively
- Error handling:
  - Graceful degradation if Copilot SDK unavailable
  - Clear error messages
  - Fallback to template-based generation

**Acceptance Criteria**:
- Copilot SDK client initializes successfully
- Tools are registered and callable
- Multi-turn conversations maintain context
- Graceful fallback when SDK unavailable
- Error messages are clear and actionable

---

### FR-4: YAML Generation

**Description**: Generate valid azure.yaml configuration from user input and project context

**Requirements**:
- Generate YAML structure matching azure.yaml v1.1 schema
- Include required fields:
  - `name`: Project name (validated against schema pattern)
- Generate service definitions with:
  - `language`: Detected or specified
  - `project`: Path to service directory
  - `host`: Appropriate host type (localhost, containerapp, etc.)
  - `ports`: Detected or specified ports
  - `environment`: Suggested environment variables
- Add well-known services when requested:
  - Azurite, Cosmos DB, Redis, PostgreSQL, etc.
  - Pre-configured with correct defaults
- Generate requirements (`reqs`) section:
  - Based on detected package managers
  - Include minimum versions when detected
- Format YAML with:
  - Consistent 2-space indentation
  - Proper key ordering
  - Comments for clarity (optional)

**Acceptance Criteria**:
- Generated YAML validates against JSON schema
- All required fields present
- Service configurations are complete and valid
- Well-known services include full configuration
- YAML formatting is consistent and readable

---

### FR-5: Schema Validation

**Description**: Validate generated YAML against azure.yaml JSON schema before writing

**Requirements**:
- Load azure.yaml v1.1 JSON schema
- Validate generated configuration:
  - Required fields present
  - Field types correct (string, number, boolean, etc.)
  - Enum values valid
  - Pattern matching (service names, etc.)
  - Min/max constraints
- Provide clear error messages:
  - Field path (e.g., `services.api.ports[0]`)
  - Error type (required, type mismatch, pattern violation)
  - Suggested fix
- Fix common errors automatically:
  - Invalid service names (sanitize to valid format)
  - Missing required fields (add with defaults)
  - Type mismatches (coerce when safe)

**Acceptance Criteria**:
- All generated YAML passes schema validation
- Error messages are clear and actionable
- Common errors are auto-fixed
- Invalid configurations are rejected with helpful messages

---

### FR-6: Interactive Mode

**Description**: Multi-turn conversational interface for guided setup

**Requirements**:
- Start interactive session with Copilot SDK
- Ask clarifying questions:
  - Service details (ports, languages, paths)
  - Dependencies (databases, caches, message queues)
  - Environment variables
  - Health checks
- Support conversation commands:
  - `yes`/`no` for confirmations
  - `edit` to modify previous answer
  - `skip` to use defaults
  - `cancel` to exit
- Show progress:
  - Current step in process
  - Estimated remaining questions
- Preview generated YAML:
  - Show after each major section
  - Allow refinement before final save

**Acceptance Criteria**:
- Interactive mode guides users through setup
- Questions are clear and contextual
- Users can edit previous answers
- Progress is visible
- Preview shown before final save

---

### FR-7: File Operations

**Description**: Safely read, backup, and write azure.yaml files

**Requirements**:
- Check for existing `azure.yaml`:
  - Warn if file exists (unless `--force`)
  - Offer to backup before overwriting
- Create backup before writing:
  - Format: `azure.yaml.backup.{ISO8601-timestamp}`
  - Location: Same directory as azure.yaml
  - Preserve original file permissions
- Write azure.yaml atomically:
  - Write to temp file first
  - Validate written content can be parsed
  - Atomic rename to azure.yaml
- Handle file system errors:
  - Permission denied → show error with remediation
  - Disk full → prevent write, show error
  - Invalid path → validate and suggest correction

**Acceptance Criteria**:
- Existing files are backed up before overwriting
- Writes are atomic (no corruption on failure)
- File system errors are handled gracefully
- Backup files are properly timestamped

---

### FR-8: Well-Known Services Integration

**Description**: Add common Azure services with intelligent defaults

**Requirements**:
- Support adding well-known services:
  - Azurite (Azure Storage emulator)
  - Cosmos DB emulator
  - Redis cache
  - PostgreSQL database
  - MongoDB (future)
- Auto-configure services with:
  - Correct Docker images
  - Appropriate ports
  - Environment variables
  - Health checks
  - Connection strings (displayed to user)
- Link services to application services:
  - Auto-populate environment variables
  - Suggest connection string formats
- Validate service names:
  - Unique names
  - Valid format (lowercase, hyphens, numbers)

**Acceptance Criteria**:
- Well-known services can be added via conversation
- Services include complete default configuration
- Connection strings are displayed
- Environment variables are auto-linked
- Service names are validated

---

### FR-9: Requirements Detection

**Description**: Auto-generate `reqs` section from project dependencies

**Requirements**:
- Detect package managers:
  - npm/yarn/pnpm (Node.js)
  - pip/poetry/uv (Python)
  - dotnet (.NET)
  - go (Go)
- Extract version requirements:
  - Minimum versions from package files
  - Runtime versions (Node.js, Python, etc.)
- Generate `reqs` section:
  ```yaml
  reqs:
    - name: node
      minVersion: 18.0.0
    - name: npm
      minVersion: 9.0.0
  ```
- Validate detected versions:
  - Check against known version formats
  - Warn if versions seem incorrect

**Acceptance Criteria**:
- Requirements detected from project files
- Version numbers extracted correctly
- `reqs` section generated with valid format
- Invalid versions are flagged

---

### FR-10: Error Handling and Recovery

**Description**: Graceful error handling with recovery options

**Requirements**:
- Handle Copilot SDK errors:
  - Network timeouts → retry with exponential backoff
  - Authentication failures → clear error message
  - Rate limiting → wait and retry
  - Service unavailable → fallback to templates
- Handle YAML generation errors:
  - Invalid schema → show validation errors
  - Parsing failures → suggest fixes
  - Missing required fields → prompt for input
- Recovery options:
  - Retry generation with modified input
  - Fall back to template-based generation
  - Save partial configuration (with warnings)
- User feedback:
  - Clear error messages
  - Suggested actions
  - Links to documentation

**Acceptance Criteria**:
- Errors are caught and handled gracefully
- Recovery options are provided
- Error messages are clear and actionable
- Fallback mechanisms work when primary fails

## Technical Requirements

### TR-1: Technology Stack

**Dependencies**:
- `github.com/github/copilot-sdk/go` - Copilot SDK (when available)
- `gopkg.in/yaml.v3` - YAML parsing and generation
- `github.com/xeipuuv/gojsonschema` - JSON Schema validation
- Existing azd-app packages:
  - `internal/detector` - Project structure detection
  - `internal/service` - Service configuration types
  - `internal/yamlutil` - YAML utilities

**Architecture**:
```
commands/
  init.go                    # Main init command
  init_interactive.go        # Interactive mode handler
  init_generate.go           # YAML generation logic
  
internal/ai/
  copilot.go                 # Copilot SDK client wrapper
  yaml_generator.go          # YAML generation from AI responses
  prompts.go                 # Prompt templates
  tools.go                   # Copilot SDK tool definitions
```

### TR-2: Copilot SDK Integration

**Client Initialization**:
```go
type CopilotClient struct {
    client *copilotsdk.Client
    ctx    context.Context
}

func NewCopilotClient() (*CopilotClient, error) {
    client, err := copilotsdk.NewClient(copilotsdk.Options{
        // Configuration
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize Copilot SDK: %w", err)
    }
    return &CopilotClient{client: client}, nil
}
```

**Tool Definitions**:
```go
var azureYAMLTools = []copilotsdk.Tool{
    {
        Name: "validate_service_name",
        Description: "Validates service name against azure.yaml schema pattern",
        Handler: validateServiceName,
    },
    {
        Name: "suggest_ports",
        Description: "Suggests port numbers based on service type and language",
        Handler: suggestPorts,
    },
    {
        Name: "detect_language",
        Description: "Detects programming language from project files",
        Handler: detectLanguage,
    },
    {
        Name: "validate_yaml",
        Description: "Validates YAML against azure.yaml JSON schema",
        Handler: validateYAML,
    },
}
```

**YAML Generation**:
```go
func (c *CopilotClient) GenerateYAML(description string, context ProjectContext) ([]byte, error) {
    prompt := buildYAMLPrompt(description, context)
    
    response, err := c.client.Prompt(prompt, copilotsdk.Options{
        Tools: azureYAMLTools,
        MaxTokens: 4000,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to generate YAML: %w", err)
    }
    
    // Extract YAML from response
    yamlContent := extractYAML(response.Text)
    
    // Validate against schema
    if err := validateAgainstSchema(yamlContent); err != nil {
        return nil, fmt.Errorf("generated YAML invalid: %w", err)
    }
    
    return yamlContent, nil
}
```

### TR-3: Prompt Engineering

**System Prompt Template**:
```
You are an expert Azure Developer CLI (azd) configuration assistant. 
Your task is to generate valid azure.yaml configuration files based on 
user descriptions of their projects.

Key requirements:
- Generate YAML that validates against azure.yaml v1.1 JSON schema
- Use appropriate service configurations based on detected languages
- Include well-known services (azurite, cosmos, redis, postgres) when requested
- Suggest sensible defaults for ports, environment variables, and health checks
- Format YAML with 2-space indentation

Project context:
{project_context}

Available tools:
- validate_service_name: Validate service names
- suggest_ports: Get port recommendations
- detect_language: Identify languages from files
- validate_yaml: Check YAML against schema

User description: {user_description}
```

**Context Building**:
```go
func buildProjectContext(workingDir string) (ProjectContext, error) {
    ctx := ProjectContext{
        WorkingDir: workingDir,
        Services:   []DetectedService{},
        Languages:  []string{},
        Ports:      []int{},
    }
    
    // Detect services
    services, err := detector.DetectServices(workingDir)
    if err == nil {
        ctx.Services = services
    }
    
    // Detect languages
    languages, err := detector.DetectLanguages(workingDir)
    if err == nil {
        ctx.Languages = languages
    }
    
    // Detect ports
    ports, err := detector.DetectPorts(workingDir)
    if err == nil {
        ctx.Ports = ports
    }
    
    return ctx, nil
}
```

### TR-4: Schema Validation

**Validation Pipeline**:
```go
func validateYAML(yamlContent []byte) error {
    // 1. Parse YAML
    var config map[string]interface{}
    if err := yaml.Unmarshal(yamlContent, &config); err != nil {
        return fmt.Errorf("invalid YAML syntax: %w", err)
    }
    
    // 2. Load schema
    schema, err := loadAzureYAMLSchema()
    if err != nil {
        return fmt.Errorf("failed to load schema: %w", err)
    }
    
    // 3. Validate against schema
    result, err := schema.Validate(gojsonschema.NewBytesLoader(yamlContent))
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }
    
    if !result.Valid() {
        errors := result.Errors()
        return fmt.Errorf("schema validation failed: %v", errors)
    }
    
    return nil
}
```

### TR-5: Error Handling

**Error Categories**:
1. **Copilot SDK Errors**: Network, authentication, rate limiting
2. **YAML Generation Errors**: Invalid structure, missing fields
3. **Schema Validation Errors**: Type mismatches, pattern violations
4. **File System Errors**: Permission denied, disk full

**Recovery Strategies**:
- Retry with exponential backoff for transient errors
- Fallback to template-based generation if AI fails
- Partial saves with warnings for non-critical errors
- Clear error messages with suggested actions

## User Stories

### US-1: Quick Project Setup
**As a** developer starting a new project  
**I want to** generate azure.yaml from a simple description  
**So that** I can get started quickly without reading documentation

**Acceptance Criteria**:
- Command accepts natural language description
- Generates valid azure.yaml in <30 seconds
- Configuration matches project structure
- File is saved with backup

---

### US-2: Interactive Guided Setup
**As a** developer new to azd  
**I want to** use interactive mode to be guided through setup  
**So that** I learn the configuration options and make informed choices

**Acceptance Criteria**:
- Interactive mode asks clarifying questions
- Questions are contextual and helpful
- Preview shown before saving
- Can edit previous answers

---

### US-3: Project Detection
**As a** developer with existing code  
**I want to** use --detect to auto-configure based on my project  
**So that** I don't have to manually describe what I already have

**Acceptance Criteria**:
- Detects languages and frameworks
- Identifies service directories
- Suggests appropriate ports
- Generates configuration matching project

---

### US-4: Well-Known Services
**As a** developer  
**I want to** add common services like PostgreSQL through conversation  
**So that** I get properly configured services without manual setup

**Acceptance Criteria**:
- Can request well-known services in description
- Services added with complete configuration
- Connection strings displayed
- Environment variables auto-linked

---

### US-5: Safe File Operations
**As a** developer  
**I want to** have my existing azure.yaml backed up before overwriting  
**So that** I can recover if something goes wrong

**Acceptance Criteria**:
- Backup created before writing
- Backup files are timestamped
- Can restore from backup if needed
- Warns before overwriting existing file

## Success Metrics

### Adoption
- **Target**: 70% of new azd app users try `azd app init` within first week
- **Measurement**: Track command usage in telemetry

### Time Savings
- **Target**: Average setup time reduced from 10 minutes to 2 minutes
- **Measurement**: User research study (before/after comparison)

### Error Reduction
- **Target**: 50% reduction in azure.yaml syntax errors for init-generated files
- **Measurement**: Compare error rates between manual and init-generated files

### Satisfaction
- **Target**: ≥4.5/5 average rating in user feedback
- **Measurement**: In-app feedback prompt after init use

### Accuracy
- **Target**: 95% of generated YAML files pass schema validation
- **Measurement**: Automated validation of generated files

## Risks and Mitigations

### Risk 1: Copilot SDK Availability
**Description**: Copilot SDK may not be available or may have breaking changes during technical preview.

**Likelihood**: Medium  
**Impact**: High

**Mitigation**:
- Implement fallback to template-based generation
- Pin SDK version in go.mod
- Graceful degradation with clear error messages
- Feature flag to disable AI features if needed

---

### Risk 2: Generated YAML Quality
**Description**: AI-generated YAML may not match user intent or may have subtle errors.

**Likelihood**: Medium  
**Impact**: Medium

**Mitigation**:
- Always validate against JSON schema
- Show preview before saving
- Allow editing and refinement
- Provide clear error messages for invalid configurations

---

### Risk 3: Project Detection Accuracy
**Description**: Automatic project detection may misidentify languages or services.

**Likelihood**: Low  
**Impact**: Medium

**Mitigation**:
- Use multiple detection heuristics
- Show detected information for user confirmation
- Allow manual override in interactive mode
- Fallback to user description if detection fails

---

### Risk 4: Performance
**Description**: Copilot SDK calls may be slow, impacting user experience.

**Likelihood**: Medium  
**Impact**: Low

**Mitigation**:
- Show progress indicators
- Cache common responses
- Set reasonable timeouts (30s)
- Provide async mode option

---

### Risk 5: Cost
**Description**: Copilot SDK API calls may incur costs.

**Likelihood**: Low  
**Impact**: Low

**Mitigation**:
- Rate limiting per user
- Cache responses when possible
- Monitor usage
- Consider local model fallback (future)

## Open Questions

1. **Q**: Should we support generating partial configurations (just services, not full file)?  
   **Status**: Open  
   **Decision Needed**: By implementation  
   **Options**: (a) Always generate full file, (b) Allow partial generation, (c) Merge with existing

2. **Q**: How should we handle existing azure.yaml files?  
   **Status**: Open  
   **Decision Needed**: By implementation  
   **Options**: (a) Always backup and overwrite, (b) Merge with existing, (c) Ask user

3. **Q**: Should we support generating configurations for multiple environments?  
   **Status**: Open  
   **Decision Needed**: By v2  
   **Options**: (a) Not in v1, (b) Support dev/staging/prod, (c) Environment-specific flags

4. **Q**: How to handle secrets in generated configurations?  
   **Status**: Open  
   **Decision Needed**: By implementation  
   **Options**: (a) Use placeholders, (b) Reference Key Vault, (c) Warn user to add manually

5. **Q**: Should we integrate with azd templates?  
   **Status**: Open  
   **Decision Needed**: By v2  
   **Options**: (a) Not in v1, (b) Suggest templates, (c) Import from templates

## Future Enhancements

### v2.0
- **Template Integration**: Import from azd template gallery
- **Multi-Environment**: Generate dev/staging/prod configurations
- **Visual Preview**: Show generated configuration in visual editor
- **Configuration Diff**: Compare generated vs existing configuration
- **Batch Generation**: Generate multiple service configurations at once

### v2.1
- **Learning Mode**: Remember user preferences for future generations
- **Configuration Templates**: Save and reuse common patterns
- **Validation Hints**: Suggest improvements to existing configurations
- **Migration Assistant**: Help migrate from older azure.yaml versions

### v3.0
- **Infrastructure Generation**: Generate Bicep/Terraform from azure.yaml
- **Cost Estimation**: Show estimated Azure costs
- **Security Scanning**: Detect security issues in configurations
- **Performance Optimization**: Suggest optimizations based on services

## References

- [GitHub Copilot SDK](https://github.com/github/copilot-sdk)
- [azure.yaml Schema v1.1](https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json)
- [azd Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Project Detection Logic](c:\code\azd-app\cli\src\internal\detector) - Existing detection implementation
