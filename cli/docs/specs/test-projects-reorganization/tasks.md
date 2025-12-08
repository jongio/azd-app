# Test Projects Reorganization Tasks

## Status Legend
- **TODO**: Not started
- **IN PROGRESS**: Currently working
- **DONE**: Completed

---

## Tasks

### 1. Create Directory Structure
- [x] **DONE** - Create `test-command/` directory
- [x] **DONE** - Create subdirectories for each language

### 2. Node.js Frameworks Project
- [x] **DONE** - Create node-frameworks/azure.yaml with 4 services
- [x] **DONE** - Create jest/ service with calculator module + tests (19 tests)
- [x] **DONE** - Create vitest/ service with string utils + tests (35 tests)
- [x] **DONE** - Create mocha/ service with array utils + tests (29 tests)
- [x] **DONE** - Create jasmine/ service with validator utils + tests (24 tests)
- [x] **DONE** - Verified all 4 frameworks pass tests

### 3. Python Frameworks Project
- [x] **DONE** - Create python-frameworks/azure.yaml with 4 services
- [x] **DONE** - Create pytest-svc/ service with math utils + tests (43 tests)
- [x] **DONE** - Create unittest-svc/ service with string utils + tests (33 tests)
- [x] **DONE** - Create nose2-svc/ service with data utils + tests (26 tests)
- [x] **DONE** - Create doctest-svc/ service with format utils + tests (47 tests)
- [x] **DONE** - Verified all 4 frameworks pass tests

### 4. .NET Frameworks Project
- [x] **DONE** - Create dotnet-frameworks/azure.yaml with 3 services
- [x] **DONE** - Create xunit/ service with Calculator + tests
- [x] **DONE** - Create nunit/ service with StringOperations + tests (25 tests)
- [x] **DONE** - Create mstest/ service with Validator + tests
- [x] **DONE** - Verified all 3 frameworks pass tests (98 total)

### 5. Go Frameworks Project
- [x] **DONE** - Create go-frameworks/azure.yaml with 3 services
- [x] **DONE** - Create testing-svc/ service with standard Go tests
- [x] **DONE** - Create testify-svc/ service with assertion-style tests
- [x] **DONE** - Create ginkgo-svc/ service with BDD-style tests
- [x] **DONE** - Verified all 3 frameworks pass tests

### 6. Cleanup Old Projects
- [x] **DONE** - Remove cli/tests/projects/node-jest-test/
- [x] **DONE** - Remove cli/tests/projects/node-mocha-test/
- [x] **DONE** - Remove cli/tests/projects/python-unittest-test/
- [x] **DONE** - Remove cli/tests/projects/dotnet-nunit-test/
- [x] **DONE** - Remove cli/tests/projects/dotnet-mstest-test/

### 7. Documentation
- [x] **DONE** - Create test-command/README.md
- [ ] **TODO** - Update cli/tests/projects/README.md to reference new structure

### 8. Final Validation
- [ ] **TODO** - Run `azd app test` on each multi-framework project
- [ ] **TODO** - Ensure polyglot-test still works (regression)

---

## Summary

**Total Frameworks**: 14 frameworks across 4 languages
- Node.js: Jest, Vitest, Mocha, Jasmine (107 tests)
- Python: pytest, unittest, nose2, doctest (149 tests)
- .NET: xUnit, NUnit, MSTest (98 tests)
- Go: testing, testify, ginkgo

## Assignments
- **All tasks**: Developer agent

## Notes
- Keep polyglot-test as integration reference (one framework per language)
- Each framework service has 15-50 meaningful tests
- Services are minimal but realistic
