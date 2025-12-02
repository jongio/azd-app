# Process Services - Archive 001

**Completed**: 2025  
**Agents**: Developer  
**Spec**: [docs/specs/process-services/spec.md](../specs/process-services/spec.md)

---

## Summary

Implemented comprehensive support for "process" services - services that don't expose network endpoints but need to be built, watched, or run as background processes. This includes CLI tools, MCP servers, and other non-HTTP/TCP services.

## Key Features

### Service Type System
- **http**: Services with HTTP endpoints (default when ports specified)
- **tcp**: Raw TCP services
- **process**: Services without network endpoints (default when no ports)

### Service Mode System
- **watch**: Continuous file watching (nodemon, air, etc.)
- **build**: One-time build that exits on completion
- **daemon**: Long-running background process
- **task**: On-demand one-time execution

### Auto-Detection
Language-specific detection for watch mode:
- TypeScript/JavaScript: nodemon, ts-node-dev, --watch flag
- Go: air.toml, .air.toml
- Python: --reload flag
- .NET: dotnet watch
- Rust: cargo-watch
- Java: spring-boot-devtools

### New Statuses
- `watching`: Service is in watch mode, monitoring for changes
- `building`: Build in progress
- `built`: Build completed successfully
- `failed`: Build or process failed

---

## Files Modified

### Backend (Go)
| File | Changes |
|------|---------|
| cli/src/internal/service/types.go | Added Type/Mode fields, constants, helper methods |
| cli/src/internal/service/detector.go | Mode detection functions, language-specific detection |
| cli/src/internal/registry/registry.go | Added Type/Mode to ServiceRegistryEntry |
| cli/src/internal/healthcheck/monitor.go | Process health checks, Type/Mode in results |

### Dashboard (TypeScript)
| File | Changes |
|------|---------|
| cli/dashboard/src/types.ts | ServiceType, ServiceMode unions, extended ServiceStatus |
| cli/dashboard/src/lib/service-utils.ts | Type/mode badge configs, display utilities |
| cli/dashboard/src/components/ServiceCard.tsx | Process service UI, mode badges |
| cli/dashboard/src/components/StatusCell.tsx | Process status handling |
| cli/dashboard/src/hooks/useServiceOperations.ts | New status action handling |

### Tests
| File | Changes |
|------|---------|
| cli/src/internal/service/detector_test.go | Mode detection tests, type constant tests, helper method tests |
| cli/dashboard/src/lib/service-utils.test.ts | Status display tests, type/mode utility tests |

### Documentation
| File | Changes |
|------|---------|
| cli/docs/commands/health.md | Service Types and Modes section |

---

## Test Results

**Go Tests**: All passing
- TestServiceModeDetection (4 cases)
- TestServiceTypeConstants
- TestServiceHelperMethods (5 cases)

**Dashboard Tests**: 106/106 passing
- Process Service Status Display tests
- Service Type and Mode Utilities tests

---

## Completed Tasks

1. ✅ Backend - Add Service Type and Mode to Types
2. ✅ Backend - Service Type Detection
3. ✅ Backend - Language-Specific Watch Detection
4. ✅ Backend - Health Monitor Updates
5. ✅ Backend - Registry Updates
6. ✅ Dashboard - Add Types
7. ✅ Dashboard - Update Status Display
8. ✅ Dashboard - Update UI Components
9. ✅ Tests
10. ✅ Documentation
