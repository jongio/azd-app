# Deps Command Test Coverage Improvement

## Overview

Improve test coverage for the `deps` command and related functionality from 37.5% to 80%.

## Scope

The deps command installs dependencies for detected projects (Node.js, Python, .NET). Testing coverage needs improvement across command initialization, project filtering, dry-run mode, error handling, and the various installation paths.

## Requirements

### Command Initialization
- Command creation with all flags (verbose, clean, no-cache, force, dry-run, service)
- Flag short forms work correctly
- Force flag combines clean and no-cache behavior
- PreRunE sets output format correctly

### Project Filtering
- Filter projects by service name when azure.yaml exists
- Return original projects when azure.yaml is missing
- Handle multiple service filters
- Match Node.js, Python, and .NET projects to service paths

### Dry-Run Mode
- Display summary of projects that would be installed
- Support both text and JSON output formats
- Show correct project counts and details

### Error Handling
- Handle search root determination errors
- Handle project detection errors
- Handle clean dependencies errors
- Return appropriate JSON output on errors

### Clean Dependencies
- Remove node_modules directories for Node.js projects
- Remove .venv directories for Python projects
- Remove obj/bin directories for .NET projects
- Handle partial failures gracefully
- Report detailed error information

### No Projects Case
- Handle empty workspace gracefully
- Show appropriate message for service filter with no matches
- Detect Logic Apps-only workspaces

### Search Root Determination
- Use azure.yaml directory when present
- Fall back to current working directory
- Handle errors appropriately

### Azure YAML Parsing
- Parse valid azure.yaml files
- Handle invalid YAML gracefully

## Success Criteria

- Test coverage reaches 80% for deps.go and related deps functions in core.go
- All tests pass with no regressions
- Tests are isolated and do not require real file system operations where possible
