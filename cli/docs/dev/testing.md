# Dashboard Testing Guide

This guide covers the testing infrastructure for the azd-app dashboard.

## Test Coverage

Current test coverage: **97.85%**

- Statement Coverage: 97.85%
- Branch Coverage: 93.07%
- Function Coverage: 71.23%
- Line Coverage: 97.85%

## Test Infrastructure

### Unit Tests (Vitest)

We use [Vitest](https://vitest.dev/) for unit and component testing. Vitest is a fast unit test framework powered by Vite.

#### Running Unit Tests

```bash
# Run all tests
npm test

# Run tests with coverage
npm run test:coverage

# Run tests with UI
npm run test:ui
```

#### Test Structure

Unit tests are colocated with their source files:

```
src/
  components/
    ServiceCard.tsx
    ServiceCard.test.tsx  ← Test file
    URLCell.tsx
    URLCell.test.tsx      ← Test file
```

#### Test Coverage by Component

| Component | Coverage | Tests |
|-----------|----------|-------|
| StatusCell | 100% | 17 tests |
| URLCell | 100% | 33 tests |
| Tabs | 100% | 17 tests |
| ServiceCard | 100% | 20 tests |
| ServiceTable | 100% | 17 tests |
| LogsView | 91.44% | 21 tests |
| Sidebar | 100% | 12 tests |
| App | 100% | 14 tests |

### E2E Tests (Playwright)

We use [Playwright](https://playwright.dev/) for end-to-end testing. These tests simulate real user interactions with the dashboard.

#### Running E2E Tests

```bash
# Install Playwright browsers (one-time setup)
npx playwright install chromium

# Run E2E tests
npm run test:e2e

# Run E2E tests with UI
npm run test:e2e:ui
```

#### E2E Test Coverage

The E2E tests cover:

1. **Resources View**
   - Display project name
   - Show services in table view
   - Switch between table and grid view
   - Display service details and status
   - Filter services

2. **Console View**
   - Navigate to console
   - Display log controls
   - Filter logs by service

3. **Error States**
   - Loading state
   - Empty state (no services)
   - Error handling

4. **Accessibility**
   - Proper heading structure
   - Accessible buttons and controls
   - Keyboard navigation

## Writing Tests

### Unit Test Example

```typescript
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusCell } from '@/components/StatusCell'

describe('StatusCell', () => {
  it('should display Running status for healthy service', () => {
    render(<StatusCell status="ready" health="healthy" />)
    
    expect(screen.getByText('Running')).toBeInTheDocument()
  })
})
```

### E2E Test Example

```typescript
import { test, expect } from '@playwright/test'

test('should display services in table view', async ({ page }) => {
  await page.goto('/')
  
  await expect(page.getByText('api')).toBeVisible()
  await expect(page.getByText('web')).toBeVisible()
})
```

## Test Utilities

### Mock Data

Mock data for tests is located in `src/test/mocks.ts`:

```typescript
export const mockServices = [
  {
    name: 'api',
    language: 'python',
    framework: 'flask',
    local: {
      status: 'ready',
      health: 'healthy',
      url: 'http://localhost:5000',
      port: 5000,
    },
  },
  // ... more services
]
```

### Test Setup

Global test setup is in `src/test/setup.ts`:

```typescript
import '@testing-library/jest-dom'
```

## Best Practices

### Unit Tests

1. **Test behavior, not implementation** - Focus on what the component does, not how it does it
2. **Use meaningful test names** - Describe what is being tested and expected outcome
3. **Test edge cases** - Include tests for error states, empty states, and boundary conditions
4. **Keep tests isolated** - Each test should be independent and not rely on other tests
5. **Mock external dependencies** - Mock API calls, WebSocket connections, etc.

### E2E Tests

1. **Test user workflows** - Simulate real user interactions
2. **Use data-testid sparingly** - Prefer accessible queries (role, label, text)
3. **Mock API responses** - Use Playwright's `page.route()` to mock backend
4. **Keep tests maintainable** - Use page objects for complex flows
5. **Test accessibility** - Include tests for keyboard navigation and screen readers

## CI/CD Integration

Tests run automatically on:

- Pull requests
- Pushes to main branch
- Manual workflow triggers

### Coverage Threshold

Minimum coverage requirements:
- Statement: 80%
- Branch: 80%
- Line: 80%

Current coverage exceeds these thresholds at **97.85%**.

## Troubleshooting

### Common Issues

**Tests fail with "Cannot find module"**
```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

**E2E tests timeout**
```bash
# Increase timeout in playwright.config.ts
timeout: 120 * 1000, // 120 seconds
```

**Coverage not updating**
```bash
# Clear coverage cache
rm -rf coverage/
npm run test:coverage
```

## Resources

- [Vitest Documentation](https://vitest.dev/)
- [Playwright Documentation](https://playwright.dev/)
- [Testing Library Documentation](https://testing-library.com/)
- [React Testing Best Practices](https://kentcdodds.com/blog/common-mistakes-with-react-testing-library)
