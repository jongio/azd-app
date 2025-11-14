# Deferred Improvements

This document tracks improvements that were identified during code review but deferred due to low priority or high complexity relative to their benefit.

## Global Orchestrator Dependency Injection

**Status:** Deferred  
**Priority:** Low  
**Effort:** High

### Description
Refactor the global orchestrator to use dependency injection instead of package-level state.

### Rationale for Deferral
- Requires major refactoring across multiple packages
- Low security impact
- Current implementation is functional
- Would be breaking change for internal code

### Future Considerations
- Consider during next major architectural refactor
- Could improve testability
- Would reduce package-level coupling

---

## Commands Test Coverage

**Status:** N/A  
**Priority:** N/A  
**Effort:** N/A

### Description
Increase test coverage for commands package.

### Rationale for Deferral
- Commands package doesn't exist in internal/ directory
- Command tests exist at different level (cmd/app/commands)
- Not applicable to current codebase structure

---

## Runner Test Coverage

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium

### Description
Increase unit test coverage for runner package beyond current 37.5%.

### Rationale for Deferral
- Existing integration tests provide adequate coverage
- Runner has good integration test coverage
- Functions primarily orchestrate external processes
- Unit testing would require extensive mocking
- Current coverage adequate for reliability

### Future Considerations
- Add integration tests for new runner functions
- Consider table-driven tests for edge cases
- Focus on error handling paths

---

## Functional Options Pattern

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium

### Description
Implement functional options pattern for internal packages (e.g., installer, runner, executor).

### Rationale for Deferral
- Breaking API change for internal code
- Low ROI given internal-only usage
- Current explicit parameter approach is clear
- Would add complexity without significant benefit
- Not a common pattern in Go CLI tools

### Future Considerations
- Consider if APIs become public
- Evaluate if configuration complexity grows significantly
- Review if option combinations become problematic

---

## Review Criteria

Before moving any of these items from deferred to active:

1. **Security Impact**: Does it address a security vulnerability?
2. **User Impact**: Does it directly improve user experience?
3. **Maintenance Burden**: Does it reduce ongoing maintenance costs?
4. **API Stability**: Is it worth the breaking change?
5. **Test Value**: Does it meaningfully improve test reliability?

Items should only be reconsidered if they meet at least 2 of the above criteria.
