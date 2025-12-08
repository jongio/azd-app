# Jest Test Framework Test Project

## Purpose

This test project validates that `azd app test` correctly **detects and runs Jest test frameworks** for Node.js applications, ensuring comprehensive test discovery and execution.

## What Is Being Tested

### Test Framework Detection
When running `azd app test`:
1. Jest is correctly identified as the test runner
2. Test files are properly discovered (*.test.js, *.spec.js patterns)
3. Test configuration (jest.config.js or package.json) is parsed
4. `azd app test` executes Jest with proper reporting
5. Test results are aggregated across services

### Validation Points
- ✅ Jest is the most installed testing framework (~95M downloads/month)
- ✅ jest.config.js configuration is recognized
- ✅ Test scripts in package.json are properly parsed
- ✅ All-in-one test framework (no extra assertion library needed)
- ✅ Coverage reports are generated
- ✅ HTML and JSON output formats work
- ✅ Proper exit codes on test pass/fail

## Project Structure

```
jest/
├── jest.config.js          # Jest configuration
├── package.json           # Jest dependency + test script
├── src/
│   ├── math.js           # Source code to test
│   └── string.js         # Source code to test
├── __tests__/
│   ├── math.test.js      # Jest test file
│   └── string.test.js    # Jest test file
└── README.md             # This file
```

## Key Configuration

In `package.json`:
```json
{
  "scripts": {
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
```

In `jest.config.js`:
```javascript
module.exports = {
  testEnvironment: 'node',
  collectCoverageFrom: ['src/**/*.js'],
  coveragePathIgnorePatterns: ['/node_modules/']
};
```

Jest is configured for Node.js environment with coverage collection.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/test-command/node-frameworks/jest

# Install dependencies
npm install

# Run tests
npm test

# Expected output:
# PASS  __tests__/math.test.js
#   Math operations
#     ✓ adds numbers (5ms)
#     ✓ multiplies numbers (2ms)
# PASS  __tests__/string.test.js
#   String operations
#     ✓ trims whitespace (3ms)
# 
# Test Suites: 2 passed, 2 total
# Tests: 4 passed, 4 total
# Snapshots: 0 total
# Time: 1.234s
```

### With azd app test
```bash
# From workspace root
azd app test --service jest

# Expected output:
# Testing jest service...
# 
# PASS  __tests__/math.test.js
#   Math operations
#     ✓ adds numbers
#     ✓ multiplies numbers
# PASS  __tests__/string.test.js
#   String operations
#     ✓ trims whitespace
# 
# Summary: 4 tests passed
```

### Coverage Report
```bash
npm run test:coverage

# Generates ./coverage/index.html for detailed report
```

### Automated Tests
This project is tested via:
- `cli/src/cmd/app/commands/test_command_integration_test.go` - Framework detection
- `cli/src/internal/executor/jest_executor_test.go` - Jest execution
- CI/CD pipeline test coverage validation

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct detection of Jest as the test runner
- Jest configuration parsing
- Test file discovery patterns
- Coverage report generation
- Proper integration with azd test command
- The most popular test framework (95M downloads)

### Real-World Scenario
Jest is the default test framework for modern JavaScript projects. Nearly all new Node.js projects use Jest. This test ensures production-ready support for the most common testing setup.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Framework | Jest 29+ | ✅ |
| Test Files | __tests__/**/*.test.js | ✅ |
| Configuration | jest.config.js | ✅ |
| Coverage | Supported | ✅ |
| Watch Mode | Supported | ✅ |
| Snapshots | Supported | ✅ |
| Mocking | Built-in | ✅ |

## Troubleshooting

**"Jest not found"**
- Install Jest: `npm install jest --save-dev`
- Verify: `npx jest --version`

**"Tests not found"**
- Ensure test files match pattern: `**/__tests__/**/*.test.js` or `**/*.test.js`
- Verify jest.config.js testMatch pattern
- Check file locations relative to package.json

**"Coverage not generated"**
- Run with coverage flag: `jest --coverage`
- Check collectCoverageFrom pattern in jest.config.js
- Verify src/ directory structure

**"Tests fail in azd app test but pass locally"**
- Check environment differences (CI vs local)
- Verify NODE_ENV is set appropriately
- Check working directory in azd config

## Related Test Projects

- [mocha/](../mocha/) - Mocha + Chai (flexible alternative)
- [vitest/](../vitest/) - Vitest (Vite-native, very fast)
- [jasmine/](../jasmine/) - Jasmine (BDD-style)
