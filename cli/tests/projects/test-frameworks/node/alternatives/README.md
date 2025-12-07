# Alternative Node.js Testing Frameworks

This project consolidates testing with alternative Node.js frameworks: **Mocha** and **Jasmine**.

## What is Being Tested

- **Mocha** (+ Chai assertions)
  - Popular flexible test runner (~8M downloads/month)
  - Used with external assertion library (Chai)
  - Supports multiple assertion styles
  - Command: `npm run test:mocha`

- **Jasmine**
  - BDD-style testing (~3M downloads/month)
  - Built-in expectations (no external assertions needed)
  - Focused on readability and simplicity
  - Command: `npm run test:jasmine`

## Project Structure

```
alternatives/
├── src/
│   └── arrays.js           # Shared implementation
├── mocha/
│   └── arrays.test.js      # Mocha test (with Chai)
├── jasmine/
│   ├── jasmine.json        # Jasmine configuration
│   ├── tests/
│   │   └── arrays.test.js  # Jasmine test
│   └── src/                # Required by jasmine.json
├── package.json
└── README.md               # This file
```

## Running Tests

### Run both frameworks
```bash
npm test
```

### Run Mocha only
```bash
npm run test:mocha
```

### Run Jasmine only  
```bash
npm run test:jasmine
```

### Generate coverage
```bash
npm run test:coverage
```

## Why These Tests Exist

Both Mocha and Jasmine are common but less mainstream than Jest/Vitest. They're tested together here to validate that `azd app test` can detect and run both frameworks with standard commands (`npm test`).

**Test matrix:**
- ✅ Framework detection (Mocha vs Jasmine)
- ✅ Command execution (npm test → correct framework)
- ✅ Test output parsing
- ✅ Coverage reporting
- ✅ Exit codes

## Key Differences from Jest/Vitest

- **No built-in assertions** (Mocha needs Chai or other library)
- **BDD vs TDD syntax** (Jasmine is purely BDD)
- **Manual runner selection** (no zero-config like Vitest)
- **Lower adoption** than Jest (~8M vs ~95M downloads)

## Troubleshooting

### Mocha tests not found
Check that test files match glob pattern: `'tests/**/*.test.js'`

### Jasmine tests not found  
Verify `jasmine.json` points to correct spec_dir: `"spec_dir": "tests"`

### Coverage not generated
Ensure c8 is installed: `npm install --save-dev c8`
