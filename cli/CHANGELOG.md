# Changelog

All notable changes to the App Extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-01-XX

### Added

#### Core Commands
- **`azd app reqs`** - Check prerequisites with version validation and auto-generation
  - Validates installed tools and their versions
  - Checks if services are running (e.g., Docker daemon)
  - Auto-generates requirements from detected project dependencies
  - Smart caching to skip unnecessary checks (1-hour cache)
  - Supports 20+ built-in tools (Node.js, Python, .NET, Docker, Git, Azure CLI, etc.)
  
- **`azd app deps`** - Install dependencies across all projects automatically
  - Multi-language detection (Node.js, Python, .NET)
  - Package manager auto-detection (npm/pnpm/yarn, pip/poetry/uv, dotnet)
  - Concurrent installation for faster setup
  - Python virtual environment auto-creation
  - Proper error handling and validation
  
- **`azd app run`** - Start development environment with live dashboard
  - Service orchestration from `azure.yaml` configuration
  - Launches embedded web dashboard with service monitoring
  - Real-time health checks for running services
  - Live log streaming from all services
  - Support for .NET Aspire, Docker Compose, and custom scripts
  - Azure environment integration with deployment info
  - Random port allocation for dashboard (40000-49999 range)
  
- **`azd app info`** - Display information about running services
  - Shows service names, ports, and URLs
  - Displays process IDs and health status
  - Shows Azure deployment information when available
  
- **`azd app logs`** - View and stream logs from running services
  - Stream logs from all services or filter by service name
  - Follow mode for real-time log streaming (`--follow`)
  - Tail mode to show last N lines (`--tail`)
  - Save logs to file (`--output`)
  - Structured JSON output option
  
- **`azd app version`** - Display version information

#### Project Detection & Support

- **Node.js Projects**
  - Automatic detection of npm, pnpm, and yarn via lock files
  - Support for `package.json` scripts (dev, start, serve)
  - Environment variable loading from `.env` files
  
- **Python Projects**
  - Support for pip, poetry, and uv package managers
  - Automatic virtual environment creation
  - Entry point auto-detection (main.py, app.py, manage.py, etc.)
  - Custom entry point support via `azure.yaml` configuration
  - Requirements.txt and pyproject.toml parsing
  
- **.NET Projects**
  - Support for .csproj and .sln files
  - `dotnet restore` for dependency installation
  - `dotnet run` for project execution
  
- **.NET Aspire**
  - Full Aspire application support
  - AppHost.cs detection and orchestration
  - Integration with Aspire dashboard

#### Architecture & Infrastructure

- **Command Orchestrator Pattern**
  - Automatic dependency resolution between commands
  - Memoization to prevent duplicate execution
  - Cycle detection for dependency chains
  - Clean error propagation
  
- **Security-First Design**
  - Path validation to prevent traversal attacks
  - Package manager whitelist validation
  - Script name sanitization (blocks shell metacharacters)
  - Input validation for all user-controlled data
  - Cryptographically secure random number generation
  
- **Execution Framework**
  - Safe command execution via executor package
  - Context-aware execution with 30-minute default timeout
  - Automatic environment inheritance from azd
  - Proper signal handling and cleanup
  
- **Azure Integration**
  - Automatic access to azd environment variables
  - Azure subscription and resource context propagation
  - Support for Azure deployment information display
  - JWT token authentication for gRPC communication
  
- **Live Dashboard** (React + TypeScript + Vite)
  - Real-time service status monitoring
  - URL management (local and Azure endpoints)
  - Log streaming with filtering
  - Service health indicators
  - One-click service access
  - Responsive design with Tailwind CSS
  - WebSocket-based live updates

#### Developer Experience

- **Extension Framework Integration**
  - `listen` command for bidirectional azd ↔ extension communication
  - Proper service provider registration
  - Clean integration with azd lifecycle
  
- **Build & Distribution**
  - Cross-platform support (Windows, Linux, macOS on AMD64 and ARM64)
  - Mage build system for consistent builds
  - Local installation scripts for development
  - Automated CI/CD with GitHub Actions
  
- **Testing & Quality**
  - 80%+ code coverage across core packages
  - Table-driven unit tests
  - Integration tests with real project fixtures
  - Security scanner integration (gosec)
  - 22 linters via golangci-lint

### Changed
- Improved error messages with actionable suggestions
- Enhanced logging with structured output
- Better path normalization across platforms

### Fixed
- **Security Issues** (12 fixes from gosec audit)
  - Replaced `math/rand` with `crypto/rand` for secure random generation
  - Added HTTP timeout configurations to prevent resource exhaustion
  - Added comprehensive error handling for Close() and Flush() operations
  - Fixed unhandled errors in critical paths
  
- Path traversal vulnerabilities via validation
- Race conditions in concurrent operations
- Proper cleanup of resources on exit

### Security
- Created comprehensive security policy (SECURITY.md)
- Implemented input validation across all user inputs
- Added security status documentation
- Regular security scanning with gosec
- 23 false positives properly documented and suppressed

[Unreleased]: https://github.com/jongio/azd-app-extension/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/jongio/azd-app-extension/releases/tag/v0.1.0
