# Reqs Generate Feature Specification

## Overview

Add a `--generate` flag to the `azd app reqs` command that automatically detects project dependencies and generates/updates the `reqs` section in `azure.yaml` based on the current project structure.

## User Story

**As a developer**, I want to automatically generate the requirements section in my `azure.yaml` file based on my project's actual dependencies, so I don't have to manually determine and configure all the tools and versions my project needs.

## Command Syntax

```bash
# Generate requirements and update azure.yaml
azd app reqs --generate

# Short aliases for convenience
azd app reqs --gen
azd app reqs -g

# Dry-run mode: show what would be generated without modifying files
azd app reqs --generate --dry-run
azd app reqs --gen --dry-run
```

## Behavior

### 1. Project Detection & Scanning

The command will leverage existing detection logic from the `detector` package to identify:

#### Node.js Projects
- **Detection**: Presence of `package.json`
- **Package Manager**: Detected via lock files (pnpm-lock.yaml → pnpm, yarn.lock → yarn, package-lock.json → npm)
- **Requirements Generated**:
  - `node` (runtime)
  - Package manager (`npm`, `pnpm`, or `yarn`)
  - Optional: `docker` if Docker Compose scripts detected

#### Python Projects
- **Detection**: Presence of `requirements.txt`, `pyproject.toml`, `poetry.lock`, or `uv.lock`
- **Package Manager**: Detected via lock files and pyproject.toml content
- **Requirements Generated**:
  - `python` (runtime)
  - Package manager (`pip`, `poetry`, or `uv`)

#### .NET Projects
- **Detection**: Presence of `.csproj` or `.sln` files
- **Requirements Generated**:
  - `dotnet` SDK
  
#### .NET Aspire Projects
- **Detection**: Presence of `AppHost.cs` with `.csproj` in same directory
- **Requirements Generated**:
  - `dotnet` SDK
  - `aspire` workload
- **Note**: Docker is only added if separately detected via Docker Compose scripts or other indicators

#### Docker Compose Projects
- **Detection**: Scripts in `package.json` containing `docker compose up` or `docker-compose up`
- **Requirements Generated**:
  - `docker` (with `checkRunning: true`)

#### Azure CLI Tools
- **Detection**: If any Azure deployment configurations or azd templates are detected
- **Requirements Generated**:
  - `azd` (Azure Developer CLI)
  - Optional: `az` (Azure CLI) if deployment scripts reference it

### 2. Version Detection

For each detected tool, the command will:

1. **Check if installed**: Run the tool's version command
2. **Extract current version**: Parse the version output
3. **Set as minimum version**: Use the installed version as `minVersion` in the generated config

Example:
- If Node.js 22.3.0 is installed → `minVersion: "22.0.0"` (major version match)
- If Python 3.12.5 is installed → `minVersion: "3.12.0"` (major.minor version match)
- If dotnet 9.0.100 is installed → `minVersion: "9.0.0"` (major version match)

**Version Normalization Strategy**:
- **Node.js, Python, .NET**: Major or Major.Minor (e.g., `20.0.0` or `3.12.0`)
- **Package managers**: Major version only (e.g., `9.0.0` for pnpm 9.x)
- **Azure tools**: Major.Minor version (e.g., `1.5.0` for azd 1.5.x)

### 3. Running Checks

Certain tools require runtime checks in addition to version checks:

| Tool | Check Running | Reason |
|------|---------------|--------|
| `docker` | Yes | Required for container-based apps and Aspire |
| `node` | No | Runtime only needed during execution |
| `python` | No | Runtime only needed during execution |
| `dotnet` | No | SDK available at build time |
| `aspire` | No | Workload installed via dotnet, not a running service |
| `npm/pnpm/yarn` | No | Package managers don't require background service |
| `pip/poetry/uv` | No | Package managers don't require background service |

**Default Running Check Configuration**:
```yaml
# Docker with running check
- id: docker
  minVersion: "20.0.0"
  checkRunning: true
  # Built-in check uses: docker ps
```

### 4. Azure.yaml File Management

#### Case A: azure.yaml exists with reqs section

**Behavior**: Merge strategy
- Preserve existing requirements
- Add new detected requirements that aren't already present
- Don't modify existing version constraints
- Add comment indicating auto-generated entries

Example:
```yaml
# Existing azure.yaml
name: my-project
reqs:
  - id: node
    minVersion: "18.0.0"  # User wants 18, keep it
```

After `azd app reqs --generate` with Node 22 installed:
```yaml
name: my-project
reqs:
  - id: node
    minVersion: "18.0.0"  # Preserved
  # Auto-generated requirements (added by azd app reqs --generate)
  - id: pnpm
    minVersion: "9.0.0"
  - id: docker
    minVersion: "20.0.0"
    checkRunning: true
```

#### Case B: azure.yaml exists without reqs section

**Behavior**: Add new section
- Insert `reqs:` section after metadata (if present) or after name
- Generate all detected requirements
- Add comment header

Example:
```yaml
# Existing azure.yaml
name: my-project
metadata:
  template: my-template@1.0.0
```

After generation:
```yaml
name: my-project
metadata:
  template: my-template@1.0.0

# Requirements auto-generated by azd app reqs --generate
reqs:
  - id: node
    minVersion: "22.0.0"
  - id: pnpm
    minVersion: "9.0.0"
```

#### Case C: azure.yaml does not exist

**Behavior**: Create new file
- Create minimal `azure.yaml` with project name and requirements
- Project name derived from directory name
- Add helpful comments

Example generated file:
```yaml
# This file was auto-generated by azd app reqs --generate
# Customize as needed for your project

name: my-project

# Requirements auto-generated based on detected project dependencies
reqs:
  - id: node
    minVersion: "22.0.0"
  - id: pnpm
    minVersion: "9.0.0"
  - id: docker
    minVersion: "20.0.0"
    checkRunning: true
```

### 5. Search Logic for azure.yaml

The command uses the existing `FindAzureYaml()` function from the detector package:

1. Start in current working directory
2. Search upward through parent directories
3. Stop at:
   - First `azure.yaml` found
   - `.git` directory (repository root)
   - Filesystem root

If not found, create `azure.yaml` in the current working directory.

## Output & User Feedback

### Success Output

```
🔍 Scanning project for dependencies...

Found:
  ✓ Node.js project (pnpm)
  ✓ .NET Aspire project
  ✓ Docker Compose configuration

📝 Detected requirements:
  • node (22.3.0 installed) → minVersion: "22.0.0"
  • pnpm (9.1.4 installed) → minVersion: "9.0.0"
  • dotnet (9.0.100 installed) → minVersion: "9.0.0"
  • aspire (9.1.0 installed) → minVersion: "9.0.0"
  • docker (27.0.3 installed, must be running) → minVersion: "27.0.0"

✅ Updated azure.yaml with 5 requirements
   Path: c:\code\my-project\azure.yaml

Run 'azd app reqs' to verify all requirements are met.
```

### Dry-Run Output

```
🔍 Scanning project for dependencies... (dry-run mode)

Found:
  ✓ Node.js project (pnpm)

📝 Would generate the following requirements:
  • node → minVersion: "22.0.0"
  • pnpm → minVersion: "9.0.0"

Would update: c:\code\my-project\azure.yaml

Run without --dry-run to apply changes.
```

### Warning Cases

**No dependencies detected**:
```
⚠️  No project dependencies detected in current directory
   Searched: c:\code\empty-project

   Supported project types:
   • Node.js (package.json)
   • Python (requirements.txt, pyproject.toml)
   • .NET (.csproj, .sln)
   • .NET Aspire (AppHost.cs)
   • Docker Compose (docker-compose.yml or package.json scripts)

   Make sure you're in a valid project directory.
```

**Tool not installed**:
```
⚠️  Some detected dependencies are not installed:

  ❌ pnpm: NOT INSTALLED
     Install: npm install -g pnpm

  ✓ node: 22.3.0

Generating requirements anyway. Run 'azd app reqs' to check status.
```

## Tool Detection Logic

Each tool requires specific file-based indicators to trigger detection. This section maps tools to their detection criteria:

### Detection Strategies by Tool Category

#### JavaScript/Node.js Ecosystem

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `node` | `package.json` exists | Always | Base requirement for any Node project |
| `pnpm` | `pnpm-lock.yaml` OR `pnpm-workspace.yaml` exists | High | Preferred over npm/yarn if lock file present |
| `yarn` | `yarn.lock` exists (no pnpm lock) | Medium | Only if pnpm not detected |
| `npm` | `package-lock.json` exists OR `package.json` exists (no other locks) | Low | Default fallback |
| `bun` | `bun.lockb` exists OR `bunfig.toml` exists | High | Bun-specific project |
| `deno` | `deno.json` OR `deno.jsonc` exists | High | Deno-specific project |

#### Python Ecosystem

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `python` | Any Python indicator file exists | Always | Base requirement |
| `uv` | `uv.lock` exists OR `pyproject.toml` contains `[tool.uv]` | High | Modern fast installer |
| `poetry` | `poetry.lock` exists OR `pyproject.toml` contains `[tool.poetry]` | High | Popular dep manager |
| `pipenv` | `Pipfile` OR `Pipfile.lock` exists | Medium | Environment manager |
| `pdm` | `pdm.lock` exists OR `pyproject.toml` contains `[tool.pdm]` | Medium | Modern package manager |
| `conda` | `environment.yml` OR `environment.yaml` OR `conda.yml` exists | Medium | Scientific computing |
| `pip` | `requirements.txt` exists OR any Python file (fallback) | Low | Default fallback |

#### .NET Ecosystem

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `dotnet` | `.csproj` OR `.sln` OR `.fsproj` OR `.vbproj` exists | Always | Any .NET project |
| `aspire` | `AppHost.cs` OR `Program.cs` in project with Aspire NuGet refs | High | Check for Aspire.Hosting package |
| `nuget` | `.csproj` OR `.sln` exists | Auto | Implied by dotnet projects |
| `paket` | `paket.dependencies` OR `paket.lock` exists | Medium | Alternative to NuGet |

#### Container & Orchestration

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `docker` | `Dockerfile` OR `docker-compose.yml` OR `docker-compose.yaml` OR `compose.yml` OR `compose.yaml` OR package.json scripts contain "docker" | High | Requires running check |
| `podman` | `Containerfile` OR explicit podman usage in scripts | Medium | Docker alternative |
| `docker-compose` | `docker-compose.yml` OR `docker-compose.yaml` exists (legacy) | Medium | V1 compose |
| `compose` | `compose.yml` OR `compose.yaml` exists | Medium | V2 compose |
| `kubectl` | `.kube/config` OR `k8s/` dir OR `kubernetes/` dir OR `*.yaml` with `kind: Deployment` | Medium | Kubernetes manifests |
| `helm` | `Chart.yaml` OR `helmfile.yaml` exists | Medium | Helm charts |
| `minikube` | `.minikube/` directory OR config references | Low | Local K8s |
| `kind` | `kind-config.yaml` OR config references | Low | K8s in Docker |

#### Cloud & Infrastructure

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `azd` | `azure.yaml` exists | High | Azure Developer CLI |
| `az` | `azure.yaml` OR `.azure/` dir OR scripts reference `az` command | Medium | Azure CLI |
| `terraform` | `.tf` files OR `.terraform/` dir OR `terraform.tfstate` exists | High | IaC tool |
| `pulumi` | `Pulumi.yaml` OR `Pulumi.*.yaml` exists | High | Modern IaC |
| `bicep` | `.bicep` files exist | Medium | Azure Bicep |
| `aws` | `.aws/` dir OR `aws-*.yml` OR scripts reference `aws` command | Medium | AWS CLI |
| `gcloud` | `.gcloud/` OR `gcloud-*.yml` OR scripts reference `gcloud` | Medium | Google Cloud CLI |
| `ansible` | `ansible.cfg` OR `playbook.yml` OR `inventory` file exists | Medium | Automation |
| `vagrant` | `Vagrantfile` exists | Medium | VM management |

#### Build Tools

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `make` | `Makefile` OR `makefile` OR `GNUmakefile` exists | Medium | Build automation |
| `cmake` | `CMakeLists.txt` exists | High | C/C++ build system |
| `gradle` | `build.gradle` OR `build.gradle.kts` OR `gradlew` exists | High | Java/Kotlin build |
| `maven` | `pom.xml` exists | High | Java build tool |
| `msbuild` | `.csproj` OR `.sln` exists | Auto | Implied by .NET |
| `bazel` | `BUILD` OR `WORKSPACE` OR `BUILD.bazel` exists | High | Google build tool |
| `webpack` | `webpack.config.js` OR `webpack.config.ts` exists | Medium | JS bundler |
| `vite` | `vite.config.js` OR `vite.config.ts` exists | Medium | Frontend tooling |
| `rollup` | `rollup.config.js` OR `rollup.config.ts` exists | Medium | Module bundler |
| `esbuild` | `esbuild.config.js` OR package.json scripts use esbuild | Low | Fast bundler |
| `turbo` | `turbo.json` exists | Medium | Monorepo tool |

#### Version Control

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `git` | `.git/` directory exists | Always | Standard VCS |
| `gh` | `.github/` directory exists OR GitHub Actions workflows | Medium | GitHub CLI |
| `svn` | `.svn/` directory exists | Low | Subversion |

#### Database Tools (Optional Detection)

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `psql` | `postgresql` in docker-compose OR connection strings in config | Low | PostgreSQL client |
| `mysql` | `mysql` in docker-compose OR connection strings in config | Low | MySQL client |
| `mongodb` | `mongodb` in docker-compose OR connection strings in config | Low | MongoDB |
| `redis-cli` | `redis` in docker-compose OR connection strings in config | Low | Redis client |
| `sqlite3` | `*.db` OR `*.sqlite` OR `*.sqlite3` files exist | Low | SQLite |

#### Testing Frameworks (Optional Detection)

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `jest` | `jest.config.js` OR package.json has jest dependency | Low | JS testing |
| `vitest` | `vitest.config.js` OR package.json has vitest dependency | Low | Vite testing |
| `pytest` | `pytest.ini` OR `setup.cfg` with pytest OR `pyproject.toml` with pytest | Low | Python testing |
| `playwright` | `playwright.config.js` OR package.json has playwright dependency | Low | Browser testing |
| `cypress` | `cypress.json` OR `cypress.config.js` exists | Low | E2E testing |

#### Go Ecosystem

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `go` | `go.mod` OR `*.go` files exist | High | Go language |

#### Rust Ecosystem

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `rust` / `rustc` | `Cargo.toml` OR `*.rs` files exist | High | Rust language |
| `cargo` | `Cargo.toml` exists | Auto | Implied by Rust |

#### Java Ecosystem

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `java` | `.java` files OR `pom.xml` OR `build.gradle` exists | High | Java runtime |

#### Other Languages

| Tool | Detection Criteria | Priority | Notes |
|------|-------------------|----------|-------|
| `ruby` | `Gemfile` OR `*.rb` files exist | High | Ruby runtime |
| `php` | `composer.json` OR `*.php` files exist | High | PHP runtime |
| `julia` | `Project.toml` OR `*.jl` files exist | Medium | Julia language |
| `r` | `DESCRIPTION` file OR `*.R` files exist | Medium | R language |

### Detection Implementation Strategy

The detection logic will be implemented in phases:

#### Phase 1: Essential Tools (MVP)
- Node.js ecosystem (node, npm, pnpm, yarn)
- Python ecosystem (python, pip, poetry, uv)
- .NET ecosystem (dotnet, aspire)
- Docker/Compose
- Azure tools (azd, az)
- Git

#### Phase 2: Common Build & Cloud Tools
- Terraform, Pulumi, Bicep
- Kubernetes tools (kubectl, helm)
- Build tools (make, cmake, gradle, maven)
- Go, Rust, Java runtimes

#### Phase 3: Extended Ecosystem
- Additional package managers
- Testing frameworks
- Database clients
- CI/CD tools

### Detection Algorithm

```go
// Pseudo-code for detection logic
func detectRequirements(projectDir string) []DetectedRequirement {
    var requirements []DetectedRequirement
    
    // 1. Detect by file presence (high confidence)
    if exists("package.json") {
        requirements = append(requirements, detectNode(projectDir))
        requirements = append(requirements, detectNodePackageManager(projectDir))
    }
    
    if exists("*.py") || exists("requirements.txt") || exists("pyproject.toml") {
        requirements = append(requirements, detectPython(projectDir))
        requirements = append(requirements, detectPythonPackageManager(projectDir))
    }
    
    if exists("*.csproj") || exists("*.sln") {
        requirements = append(requirements, detectDotnet(projectDir))
        if detectAspire(projectDir) {
            requirements = append(requirements, detectAspireWorkload(projectDir))
        }
    }
    
    // 2. Detect containers
    if exists("Dockerfile") || exists("docker-compose.yml") || hasDockerScripts(projectDir) {
        req := detectDocker(projectDir)
        req.CheckRunning = true // Docker must be running
        requirements = append(requirements, req)
    }
    
    // 3. Detect cloud/IaC tools
    if exists("azure.yaml") {
        requirements = append(requirements, detectAzd(projectDir))
    }
    
    if exists("*.tf") {
        requirements = append(requirements, detectTerraform(projectDir))
    }
    
    // 4. Detect VCS
    if exists(".git") {
        requirements = append(requirements, detectGit(projectDir))
    }
    
    return requirements
}

// detectNode detects Node.js and queries installed version
func detectNode(projectDir string) DetectedRequirement {
    req := DetectedRequirement{
        ID:     "node",
        Source: "package.json",
    }
    
    // Get installed version from the system
    installedVersion, err := getInstalledVersion("node")
    if err != nil {
        // Not installed - leave version empty, will warn user
        return req
    }
    
    req.InstalledVersion = installedVersion // e.g., "22.3.0"
    req.MinVersion = normalizeVersion(installedVersion, "node") // e.g., "22.0.0"
    
    return req
}

// detectPnpm detects pnpm and queries installed version
func detectPnpm(projectDir string) DetectedRequirement {
    req := DetectedRequirement{
        ID:     "pnpm",
        Source: "pnpm-lock.yaml",
    }
    
    // Get installed version from the system
    installedVersion, err := getInstalledVersion("pnpm")
    if err != nil {
        // Not installed - leave version empty, will warn user
        return req
    }
    
    req.InstalledVersion = installedVersion // e.g., "9.1.4"
    req.MinVersion = normalizeVersion(installedVersion, "pnpm") // e.g., "9.0.0"
    
    return req
}

// getInstalledVersion queries the system for the installed version of a tool
func getInstalledVersion(toolID string) (string, error) {
    // Look up tool configuration from registry
    toolConfig, exists := toolRegistry[toolID]
    if !exists {
        return "", fmt.Errorf("unknown tool: %s", toolID)
    }
    
    // Execute version command using executor package
    output, err := executor.RunCommand(toolConfig.Command, toolConfig.Args, ".")
    if err != nil {
        return "", fmt.Errorf("tool not installed: %s", toolID)
    }
    
    // Parse version from output
    version := parseVersion(output, toolConfig.VersionPrefix, toolConfig.VersionField)
    return version, nil
}

// normalizeVersion converts installed version to minimum version constraint
func normalizeVersion(installedVersion string, toolID string) string {
    parts := strings.Split(installedVersion, ".")
    
    switch toolID {
    case "node", "dotnet", "go", "rust":
        // Major version only: "22.3.0" -> "22.0.0"
        if len(parts) >= 1 {
            return parts[0] + ".0.0"
        }
    case "python":
        // Major.Minor version: "3.12.5" -> "3.12.0"
        if len(parts) >= 2 {
            return parts[0] + "." + parts[1] + ".0"
        }
    case "pnpm", "npm", "yarn", "poetry", "uv":
        // Major version for package managers: "9.1.4" -> "9.0.0"
        if len(parts) >= 1 {
            return parts[0] + ".0.0"
        }
    case "azd", "az":
        // Major.Minor for Azure tools: "1.5.3" -> "1.5.0"
        if len(parts) >= 2 {
            return parts[0] + "." + parts[1] + ".0"
        }
    default:
        // Default: use as-is
        return installedVersion
    }
    
    return installedVersion
}

func detectNodePackageManager(projectDir string) DetectedRequirement {
    // Priority: pnpm > yarn > npm
    if exists("pnpm-lock.yaml") || exists("pnpm-workspace.yaml") {
        return detectPnpm(projectDir)
    }
    if exists("yarn.lock") {
        return detectYarn(projectDir)
    }
    if exists("package-lock.json") {
        return detectNpm(projectDir)
    }
    // Default to npm if only package.json exists
    return detectNpm(projectDir)
}

func detectAspire(projectDir string) bool {
    // Look for AppHost.cs
    if exists("AppHost.cs") {
        return true
    }
    
    // Or check .csproj for Aspire packages
    csprojFiles := findFiles("*.csproj", projectDir)
    for _, csproj := range csprojFiles {
        content := readFile(csproj)
        if contains(content, "Aspire.Hosting") {
            return true
        }
    }
    
    return false
}
```

### Version Detection Flow

For every detected tool, the generation process follows this flow:

1. **Detect presence** via file indicators (e.g., `pnpm-lock.yaml` → pnpm required)
2. **Query installed version** by executing the tool's version command on the user's machine
3. **Normalize version** to an appropriate minimum version constraint
4. **Add to requirements** with the normalized version as `minVersion`

**Example Flow:**
```
1. Detect: pnpm-lock.yaml exists → pnpm is required
2. Execute: pnpm --version
3. Output: "9.1.4"
4. Normalize: "9.1.4" → "9.0.0" (major version)
5. Generate YAML:
   - id: pnpm
     minVersion: "9.0.0"
```

**Handling Missing Tools:**
If a tool is detected but not installed:
- Still add to requirements list
- Leave version as `minVersion: "0.0.0"` or omit it
- Show warning to user with installation instructions
- User can run `azd app reqs` after installing to verify

## Extended Tool Registry

To support a comprehensive set of development tools, the tool registry (in `reqs.go`) should be expanded to include:

### Runtime & Language Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `node` | node | --version | v | 0 | JavaScript runtime |
| `nodejs` | node | --version | v | 0 | Alias for node |
| `python` | python | --version | | 1 | Python runtime |
| `python3` | python3 | --version | | 1 | Python 3.x |
| `dotnet` | dotnet | --version | | 0 | .NET SDK |
| `go` | go | version | go | 2 | Go language |
| `rust` | rustc | --version | | 1 | Rust compiler |
| `rustc` | rustc | --version | | 1 | Rust compiler |
| `java` | java | --version | | 1 | Java runtime |
| `ruby` | ruby | --version | | 1 | Ruby runtime |
| `php` | php | --version | | 1 | PHP runtime |
| `perl` | perl | --version | | 1 | Perl runtime |
| `r` | R | --version | | 2 | R language |
| `julia` | julia | --version | | 2 | Julia language |

### JavaScript/Node Package Managers

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `npm` | npm | --version | | 0 | Node package manager |
| `pnpm` | pnpm | --version | | 0 | Fast npm alternative |
| `yarn` | yarn | --version | | 0 | Facebook package manager |
| `bun` | bun | --version | | 1 | All-in-one JS runtime |
| `deno` | deno | --version | deno | 1 | Secure JS/TS runtime |

### Python Package Managers

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `pip` | pip | --version | | 1 | Python package installer |
| `pip3` | pip3 | --version | | 1 | Python 3 pip |
| `poetry` | poetry | --version | | 2 | Python dependency manager |
| `uv` | uv | --version | | 1 | Fast Python package installer |
| `pipenv` | pipenv | --version | | 1 | Python env manager |
| `conda` | conda | --version | | 1 | Anaconda package manager |
| `pdm` | pdm | --version | | 1 | Modern Python package manager |

### Build Tools & Task Runners

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `make` | make | --version | | 3 | GNU Make |
| `cmake` | cmake | --version | | 2 | Cross-platform build |
| `gradle` | gradle | --version | | 1 | Java build tool |
| `maven` | mvn | --version | | 2 | Java build tool |
| `ant` | ant | -version | | 3 | Java build tool |
| `msbuild` | msbuild | /version | | 0 | Microsoft Build Engine |
| `ninja` | ninja | --version | | 0 | Small build system |
| `bazel` | bazel | version | | 1 | Google build tool |
| `webpack` | webpack | --version | | 0 | JS bundler |
| `vite` | vite | --version | | 0 | Fast frontend build tool |
| `rollup` | rollup | --version | | 0 | JS module bundler |
| `esbuild` | esbuild | --version | | 0 | Extremely fast bundler |
| `turbo` | turbo | --version | | 0 | Incremental bundler |
| `gulp` | gulp | --version | | 1 | JS task runner |
| `grunt` | grunt | --version | | 1 | JS task runner |

### Version Control

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `git` | git | --version | git version | 2 | Version control |
| `gh` | gh | --version | gh version | 2 | GitHub CLI |
| `svn` | svn | --version | version | 2 | Subversion |
| `hg` | hg | --version | | 4 | Mercurial |

### Container & Orchestration Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `docker` | docker | --version | Docker version | 2 | Container platform |
| `podman` | podman | --version | podman version | 2 | Daemonless container engine |
| `kubectl` | kubectl | version --client --short | | 2 | Kubernetes CLI |
| `k3s` | k3s | --version | | 2 | Lightweight Kubernetes |
| `helm` | helm | version --short | v | 0 | Kubernetes package manager |
| `k9s` | k9s | version --short | v | 0 | Kubernetes TUI |
| `minikube` | minikube | version --short | v | 0 | Local Kubernetes |
| `kind` | kind | version | kind v | 1 | Kubernetes in Docker |
| `docker-compose` | docker-compose | --version | | 3 | Multi-container Docker |
| `compose` | docker | compose version --short | v | 0 | Docker Compose V2 |

### Cloud & Infrastructure Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `azd` | azd | version | | 0 | Azure Developer CLI |
| `az` | az | version --output tsv --query "azure-cli" | | 0 | Azure CLI |
| `azure-cli` | az | version --output tsv --query "azure-cli" | | 0 | Alias for az |
| `aws` | aws | --version | | 1 | AWS CLI |
| `gcloud` | gcloud | version --format="value(version)" | | 0 | Google Cloud CLI |
| `terraform` | terraform | --version | Terraform v | 1 | Infrastructure as code |
| `tf` | terraform | --version | Terraform v | 1 | Terraform alias |
| `pulumi` | pulumi | version | v | 0 | Modern IaC |
| `bicep` | az | bicep version | | 3 | Azure Bicep |
| `ansible` | ansible | --version | | 1 | Automation tool |
| `vagrant` | vagrant | --version | Vagrant | 1 | VM manager |

### Database Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `psql` | psql | --version | | 2 | PostgreSQL client |
| `postgres` | postgres | --version | | 2 | PostgreSQL server |
| `mysql` | mysql | --version | | 2 | MySQL client |
| `mongodb` | mongod | --version | db version v | 3 | MongoDB |
| `redis-cli` | redis-cli | --version | | 1 | Redis client |
| `sqlite3` | sqlite3 | --version | | 0 | SQLite |

### .NET Specific Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `aspire` | dotnet | workload list | | - | Check for Aspire workload |
| `nuget` | nuget | help | Version: | 1 | NuGet package manager |
| `paket` | paket | --version | | 0 | .NET dependency manager |

### Testing & Quality Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `jest` | jest | --version | | 0 | JavaScript testing |
| `vitest` | vitest | --version | | 0 | Vite-native testing |
| `pytest` | pytest | --version | | 1 | Python testing |
| `playwright` | playwright | --version | Version | 1 | Browser automation |
| `cypress` | cypress | --version | Cypress version | 2 | E2E testing |
| `selenium` | selenium-server | --version | | 2 | Browser automation |

### Linters & Formatters

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `eslint` | eslint | --version | v | 0 | JavaScript linter |
| `prettier` | prettier | --version | | 0 | Code formatter |
| `black` | black | --version | black, | 1 | Python formatter |
| `ruff` | ruff | --version | | 1 | Fast Python linter |
| `pylint` | pylint | --version | pylint | 1 | Python linter |
| `flake8` | flake8 | --version | | 0 | Python linter |

### AI & ML Tools

| Tool ID | Command | Version Args | Version Prefix | Version Field | Notes |
|---------|---------|--------------|----------------|---------------|-------|
| `jupyter` | jupyter | --version | | 1 | Interactive notebooks |
| `tensorboard` | tensorboard | --version | | 0 | TensorFlow viz |

## Implementation Plan

### Phase 1: Core Detection Logic

**File**: `src/cmd/app/commands/generate.go`

Create new module with:
- `DetectedRequirement` struct
- `GenerateRequirements()` function - main orchestrator
- `detectProjectRequirements()` - scans for all project types
- `getInstalledVersion()` - reuse from reqs.go
- `normalizeVersion()` - version string normalization
- Expanded `toolRegistry` with 60+ popular tools
- Expanded `toolAliases` for alternative names

### Phase 2: Tool Registry Enhancement

**File**: `src/cmd/app/commands/reqs.go`

Expand the existing `toolRegistry` and `toolAliases`:
- Add all tools from the Extended Tool Registry section above
- Support custom version parsing logic for complex tools
- Add helper function `detectToolVersion()` with fallback logic
- Handle tools that embed version in different formats

Example expanded registry:
```go
var toolRegistry = map[string]ToolConfig{
    // Existing tools...
    "bun": {
        Command: "bun",
        Args: []string{"--version"},
        VersionField: 1, // "bun 1.0.0" -> take field 1
    },
    "terraform": {
        Command: "terraform",
        Args: []string{"--version"},
        VersionPrefix: "Terraform v",
        VersionField: 1,
    },
    "kubectl": {
        Command: "kubectl",
        Args: []string{"version", "--client", "--short"},
        VersionField: 2,
    },
    // ... 50+ more tools
}

var toolAliases = map[string]string{
    "nodejs": "node",
    "azure-cli": "az",
    "tf": "terraform",
    "k8s": "kubectl",
    "python3": "python",
    "pip3": "pip",
    // ... more aliases
}
```

### Phase 3: YAML Manipulation

**File**: `src/cmd/app/commands/generate.go`

Implement:
- `findOrCreateAzureYaml()` - locate or create azure.yaml
- `mergeRequirements()` - merge detected with existing
- `writeAzureYaml()` - preserve formatting, add comments
- Use `gopkg.in/yaml.v3` for structure-preserving YAML editing

### Phase 4: Command Integration

**File**: `src/cmd/app/commands/reqs.go`

Update:
- Add `--generate`, `--gen`, `-g` flags to `NewReqsCommand()`
- Add `--dry-run` flag
- Route to generation logic when flag present
- Update help text with extended tool support

### Phase 5: Testing

**Files**: 
- `src/cmd/app/commands/generate_test.go`
- `src/cmd/app/commands/generate_integration_test.go`
- `src/cmd/app/commands/reqs_test.go` (expand for new tools)

Test scenarios:
- Each project type detection
- Version normalization for all major tool categories
- YAML file creation
- YAML file merging
- Dry-run mode
- Edge cases (no deps, missing tools, nested projects)
- Tool alias resolution
- Complex version parsing (terraform, kubectl, etc.)

## Data Structures

```go
// DetectedRequirement represents a requirement found during project scanning.
type DetectedRequirement struct {
    ID               string   // Tool identifier (e.g., "node", "docker")
    InstalledVersion string   // Currently installed version (e.g., "22.3.0")
    MinVersion       string   // Normalized minimum version (e.g., "22.0.0")
    CheckRunning     bool     // Whether tool must be running
    Source           string   // What triggered detection (e.g., "package.json", "AppHost.cs")
}

// GenerateConfig holds configuration for requirement generation.
type GenerateConfig struct {
    DryRun      bool   // Don't write files, just show what would happen
    WorkingDir  string // Directory to start search from
    ForceCreate bool   // Create azure.yaml even if no deps detected
}

// GenerateResult contains the outcome of requirement generation.
type GenerateResult struct {
    Requirements []DetectedRequirement
    AzureYamlPath string
    Created      bool // True if azure.yaml was created vs updated
    Added        int  // Number of requirements added
    Skipped      int  // Number of existing requirements preserved
}
```

## Security Considerations

1. **Path Validation**: All file paths validated with `security.ValidatePath()`
2. **Command Execution**: Use `executor.RunCommand()` for version checks
3. **YAML Parsing**: Validate YAML structure before writing
4. **User Input**: No direct user input in generated YAML (all values from detection)

## User Experience Enhancements

### Interactive Mode (Future Enhancement)

```bash
azd app reqs --generate --interactive

🔍 Scanning project...

Found: Node.js project with pnpm

? Detected node 22.3.0. Set minimum version to: (Use arrow keys)
  ▸ 22.0.0 (major version match)
    22.3.0 (exact match)
    20.0.0 (LTS version)
    Custom...

? Detected pnpm 9.1.4. Set minimum version to:
  ▸ 9.0.0 (major version match)
    9.1.4 (exact match)
    Custom...
```

### Template Support (Future Enhancement)

Recognize common project templates and suggest additional requirements:

```
Detected: Next.js project
Suggested additional requirements:
  • TypeScript support
  • Vercel CLI (optional)

Add these? (y/N)
```

## Alternatives Considered

### Alternative 1: Separate `azd app generate-reqs` Command
**Rejected**: Adds command clutter; `--generate`/`--gen` flag is cleaner UX

### Alternative 2: Always Update on `azd app reqs`
**Rejected**: Surprising behavior; explicit flag better for control

### Alternative 3: AI-Based Detection
**Rejected**: Overkill; file-based detection is reliable and fast

## Success Metrics

- **Accuracy**: 95%+ correct requirement detection for supported project types
- **Coverage**: 80%+ test coverage for generation logic
- **Performance**: Generation completes in <3 seconds for typical projects
- **User Adoption**: Track usage of `--generate` flag via telemetry

## Documentation Updates Required

1. **README.md**: Add generate flag example
2. **docs/reqs-command.md**: New section on auto-generation
3. **docs/quickstart.md**: Update with generate workflow
4. **CHANGELOG.md**: Document new feature

## Example Workflows

### New Project Setup
```bash
# Clone a project
git clone https://github.com/org/project
cd project

# Generate requirements based on project (short form)
azd app reqs --gen

# Verify everything is installed and ready
azd app reqs

# Install dependencies
azd app deps

# Run the project
azd app run
```

### Adding New Dependency
```bash
# Install new tool (e.g., Docker for containerization)
# Add Docker Compose to package.json scripts

# Check what would be added (dry-run first)
azd app reqs --gen --dry-run

# Regenerate requirements
azd app reqs --gen
```

### Quick Check Before Commit
```bash
# Ensure azure.yaml reflects current project state
azd app reqs -g --dry-run

# Update if needed
azd app reqs -g
```

## Open Questions

1. **Q**: Should we detect dev vs production dependencies differently?
   **A**: No, for now all detected dependencies are required. Future enhancement could add `dev` flag.

2. **Q**: How to handle multiple versions of same tool (e.g., Node 18 and 22)?
   **A**: Use the version in the current PATH. User can manually adjust if needed.

3. **Q**: Should we detect optional dependencies (e.g., linters, formatters)?
   **A**: No, focus on runtime requirements only. Optional tools can be added manually.

4. **Q**: What if a tool is installed but not in PATH?
   **A**: It won't be detected. User must ensure tools are in PATH for detection.

## Timeline Estimate

- **Phase 1** (Detection Logic): 2 days
- **Phase 2** (Tool Registry Enhancement): 2 days  
- **Phase 3** (YAML Manipulation): 2 days  
- **Phase 4** (Command Integration): 1 day
- **Phase 5** (Testing): 3 days (expanded for 60+ tools)
- **Documentation**: 1 day

**Total**: ~11 development days

## Future Enhancements

1. **Multi-language Projects**: Better handling of polyglot projects (e.g., Node + Python)
2. **Custom Detectors**: Plugin system for user-defined project type detection
3. **CI/CD Integration**: GitHub Actions workflow to auto-update requirements
4. **Dependency Graphs**: Visualize requirement relationships
5. **Version Suggestions**: Recommend upgrading to newer LTS versions
