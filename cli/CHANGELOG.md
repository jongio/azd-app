# Changelog

All notable changes to the App Extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- `azd app reqs --generate` now automatically creates the `reqs` section in `azure.yaml` if it doesn't exist, instead of failing with "section 'reqs' not found in YAML" error

## [0.1.0] - 2025-11-04

Initial release of azd-app CLI extension.

### Added

#### Core Commands
- `azd app reqs` - Check prerequisites and validate installed tools
- `azd app deps` - Install dependencies across all projects automatically  
- `azd app run` - Start development environment with live dashboard
- `azd app info` - Display information about running services
- `azd app logs` - View and stream logs from running services

#### Language & Framework Support
- Node.js (npm, pnpm, yarn)
- Python (pip, poetry, uv)
- .NET (.csproj, .sln)
- .NET Aspire (AppHost orchestration)

#### Live Dashboard
- Real-time service monitoring with React/TypeScript/Vite
- Log streaming with filtering
- Local and Azure endpoint management
- Service health indicators
- WebSocket-based updates

#### Features
- Automatic package manager detection
- Azure environment integration
- Secure command execution with validation
- Cross-platform support (Windows, Linux, macOS on AMD64 and ARM64)
- 80%+ code coverage with comprehensive testing

### Security
- Input validation and path traversal protection
- Secure random number generation (crypto/rand)
- HTTP timeout configurations
- Regular security scanning with gosec

[Unreleased]: https://github.com/jongio/azd-app/compare/azd-app-cli-v0.1.0...HEAD
[0.1.0]: https://github.com/jongio/azd-app/releases/tag/azd-app-cli-v0.1.0
