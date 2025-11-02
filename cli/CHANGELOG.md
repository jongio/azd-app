# Changelog

All notable changes to the App Extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-11-01

### Added
- `reqs` command - Check for required prerequisites with version validation
- `deps` command - Automatically install dependencies for all detected projects
- `run` command - Run development environments (Aspire, pnpm, docker compose)
- Multi-language support: Node.js (npm/pnpm/yarn), Python (uv/poetry/pip), .NET
- Smart package manager detection based on lock files
- Python virtual environment auto-creation
- Runtime checks for dependencies - verify services are running (e.g., Docker daemon)
- AZD context access - all commands automatically receive azd environment variables
- Built-in support for .NET Aspire CLI in prerequisite checks
- Command dependency chain with orchestrator pattern
- Security validation for paths and inputs

### Features
- Cross-platform support (Windows, Linux, macOS AMD64 and ARM64)
- Custom command capability
- Clean and simple CLI interface

[Unreleased]: https://github.com/jongio/azd-app-extension/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/jongio/azd-app-extension/releases/tag/v0.1.0
