# Code Review Fixes Specification

## Overview

This specification addresses all issues identified during the comprehensive code review of the `logdash` branch. The branch implements multi-pane logs dashboard, OS notifications, browser auto-launch, port management, and service state monitoring.

## Functional Requirements

### FR-1: Security Hardening

#### FR-1.1: PowerShell Injection Prevention
- Enhance sanitization for PowerShell script generation in Windows notifications
- Add comprehensive input validation for all user-controlled content
- Implement proper escaping for XML content embedded in PowerShell

#### FR-1.2: Service Name Validation  
- Validate service names from query parameters against known services
- Sanitize service names for log output to prevent log injection
- Return appropriate error responses for invalid service names

### FR-2: Concurrency Safety

#### FR-2.1: Mutex Race Documentation
- Document the intentional mutex release during user input in port manager
- Add clear warnings about concurrent usage limitations
- Ensure callers understand the TOCTOU implications

#### FR-2.2: Context Propagation
- Fix notification pipeline to properly propagate context cancellation
- Ensure handlers respect parent context cancellation signals

### FR-3: Memory Management

#### FR-3.1: Efficient History Trimming
- Optimize state history slice trimming to avoid unnecessary allocations
- Use proper slice reslicing instead of copy operations

#### FR-3.2: Toast Notification Cleanup
- Add auto-dismiss timeout for toast notifications
- Prevent unbounded growth of notification arrays

### FR-4: Error Handling & Resilience

#### FR-4.1: WebSocket Reconnection
- Implement exponential backoff reconnection for WebSocket connections
- Track connection state and retry attempts
- Provide visual feedback for connection status

#### FR-4.2: Sentinel Error Types
- Convert `fmt.Errorf` sentinel errors to `errors.New`
- Follow Go 1.13+ error handling best practices

### FR-5: Input Validation

#### FR-5.1: Grid Columns Bounds
- Add bounds checking for grid columns preference (1-6 range)
- Handle zero, negative, and unreasonably large values

#### FR-5.2: TypeScript Runtime Validation
- Add runtime validation for localStorage parsed JSON
- Handle malformed data gracefully

### FR-6: Code Quality

#### FR-6.1: Log Pattern Consolidation
- Consolidate duplicate log classification logic into single location
- Export unified functions for error/warning detection

#### FR-6.2: Magic Number Elimination
- Move hardcoded timing values to constants package
- Ensure all magic numbers have named constants

#### FR-6.3: Documentation
- Add Go doc comments to exported functions
- Ensure all public APIs are properly documented

## Non-Functional Requirements

### NFR-1: Performance
- History trimming must be O(1) amortized
- WebSocket reconnection must use exponential backoff (max 30s)

### NFR-2: Security
- All user input must be sanitized before use in scripts
- Service names must be validated against whitelist when available

### NFR-3: Maintainability
- No duplicate logic across files
- All constants centralized in constants package

## Acceptance Criteria

1. All PowerShell script generation uses comprehensive sanitization
2. WebSocket connections auto-recover with exponential backoff
3. No memory leaks from unbounded array growth
4. All sentinel errors use `errors.New`
5. Grid columns bounded to 1-6 range
6. All magic numbers replaced with named constants
7. All exported Go functions have doc comments
8. Log pattern detection consolidated in single location
