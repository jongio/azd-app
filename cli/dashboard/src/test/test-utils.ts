/**
 * Test Utilities and Helpers
 * Shared utilities for testing the Azure YAML Editor
 */

/**
 * Generate test YAML configuration
 */
export function generateTestConfig(options?: {
  serviceCount?: number
  includeResources?: boolean
  includeHooks?: boolean
}): string {
  const { serviceCount = 3, includeResources = false, includeHooks = false } = options || {}

  const services: Record<string, any> = {}
  
  for (let i = 0; i < serviceCount; i++) {
    services[`service-${i}`] = {
      host: '0.0.0.0',
      port: 8000 + i,
      language: ['node', 'python', 'go'][i % 3],
      project: `./service-${i}`,
    }

    if (includeHooks) {
      services[`service-${i}`].hooks = {
        prestart: `echo "Starting service-${i}"`,
      }
    }
  }

  const config: any = {
    name: 'test-app',
    version: '1.0.0',
    services,
  }

  if (includeResources) {
    config.resources = {
      database: {
        type: 'postgres',
      },
      cache: {
        type: 'redis',
      },
    }
  }

  return JSON.stringify(config, null, 2)
}

/**
 * Create mock schema for testing
 */
export function createMockSchema(): Record<string, unknown> {
  return {
    $schema: 'http://json-schema.org/draft-07/schema#',
    type: 'object',
    properties: {
      name: {
        type: 'string',
        minLength: 1,
      },
      version: {
        type: 'string',
        pattern: '^\\d+\\.\\d+\\.\\d+$',
      },
      services: {
        type: 'object',
        additionalProperties: {
          type: 'object',
          properties: {
            host: { type: 'string' },
            port: { type: 'number', minimum: 1, maximum: 65535 },
            language: {
              type: 'string',
              enum: ['node', 'python', 'go', 'dotnet', 'java'],
            },
            project: { type: 'string' },
            uses: {
              oneOf: [
                { type: 'string' },
                { type: 'array', items: { type: 'string' } },
              ],
            },
          },
        },
      },
      resources: {
        type: 'object',
        additionalProperties: {
          type: 'object',
          properties: {
            type: { type: 'string' },
          },
        },
      },
    },
    required: ['name'],
  }
}

/**
 * Wait for async validation to complete
 */
export async function waitForValidation(delay = 500): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, delay))
}

/**
 * Measure execution time of a function
 */
export async function measureTime<T>(fn: () => Promise<T>): Promise<{ result: T; duration: number }> {
  const start = performance.now()
  const result = await fn()
  const duration = performance.now() - start
  return { result, duration }
}

/**
 * Generate large configuration for performance testing
 */
export function generateLargeConfig(serviceCount: number): Record<string, unknown> {
  const services: Record<string, any> = {}

  for (let i = 0; i < serviceCount; i++) {
    services[`service-${i}`] = {
      host: '0.0.0.0',
      port: 8000 + (i % 1000),
      language: ['node', 'python', 'go', 'dotnet', 'java'][i % 5],
      project: `./service-${i}`,
      environment: {
        NODE_ENV: 'production',
        PORT: String(8000 + i),
      },
    }

    // Add dependencies to some services
    if (i > 0 && i % 5 === 0) {
      services[`service-${i}`].uses = [`service-${i - 1}`]
    }
  }

  return {
    name: 'large-app',
    version: '1.0.0',
    services,
  }
}

/**
 * Create validation error fixture
 */
export function createValidationError(overrides?: Partial<{
  level: 'error' | 'warning' | 'info'
  message: string
  path: string
  rule: string
}>): any {
  return {
    level: 'error',
    message: 'Validation failed',
    path: 'services.api',
    rule: 'schema',
    ...overrides,
  }
}

/**
 * Mock fetch responses for testing
 */
export function createFetchMock(responses: Record<string, any>): typeof fetch {
  return ((url: string) => {
    const response = responses[url]
    if (response) {
      return Promise.resolve({
        ok: true,
        json: async () => response,
        text: async () => JSON.stringify(response),
      })
    }
    return Promise.reject(new Error(`No mock for ${url}`))
  }) as unknown as typeof fetch
}

/**
 * Assert performance metric is within threshold
 */
export function assertPerformance(
  actualMs: number,
  thresholdMs: number,
  label: string
): void {
  if (actualMs > thresholdMs) {
    throw new Error(
      `${label} took ${actualMs}ms, exceeding threshold of ${thresholdMs}ms`
    )
  }
}

/**
 * Create mock backup data
 */
export function createMockBackup(timestamp: string = new Date().toISOString()): any {
  return {
    path: `/workspace/azure.yaml.backup.${timestamp.replace(/[:.]/g, '')}`,
    timestamp,
    size: 1024,
    content: 'name: backup-app\nservices:\n  api: {}',
  }
}

/**
 * Simulate user typing with realistic delays
 */
export async function simulateTyping(
  input: HTMLInputElement | HTMLTextAreaElement,
  text: string,
  delayMs: number = 50
): Promise<void> {
  for (const char of text) {
    input.value += char
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await new Promise(resolve => setTimeout(resolve, delayMs))
  }
}

/**
 * Check if element is visible in viewport
 */
export function isInViewport(element: Element): boolean {
  const rect = element.getBoundingClientRect()
  return (
    rect.top >= 0 &&
    rect.left >= 0 &&
    rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
    rect.right <= (window.innerWidth || document.documentElement.clientWidth)
  )
}

/**
 * Wait for element to be visible
 */
export async function waitForVisible(
  selector: string,
  timeoutMs: number = 5000
): Promise<Element> {
  const start = Date.now()

  while (Date.now() - start < timeoutMs) {
    const element = document.querySelector(selector)
    if (element && isInViewport(element)) {
      return element
    }
    await new Promise(resolve => setTimeout(resolve, 100))
  }

  throw new Error(`Element ${selector} not visible after ${timeoutMs}ms`)
}

/**
 * Get all focusable elements
 */
export function getFocusableElements(root: Element = document.body): Element[] {
  const selector = [
    'a[href]',
    'button:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
  ].join(',')

  return Array.from(root.querySelectorAll(selector))
}

/**
 * Test keyboard navigation
 */
export async function testKeyboardNav(
  startElement: Element,
  expectedElements: string[]
): Promise<boolean> {
  ;(startElement as HTMLElement).focus()

  for (const expectedSelector of expectedElements) {
    // Simulate Tab
    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true })
    document.activeElement?.dispatchEvent(event)

    await new Promise(resolve => setTimeout(resolve, 100))

    const matches = document.activeElement?.matches(expectedSelector)
    if (!matches) {
      console.error(`Expected ${expectedSelector}, got ${document.activeElement?.tagName}`)
      return false
    }
  }

  return true
}

/**
 * Generate test data for array fields
 */
export function generateArrayTestData(count: number): string[] {
  return Array.from({ length: count }, (_, i) => `item-${i}`)
}

/**
 * Generate test data for object fields
 */
export function generateObjectTestData(depth: number = 3): Record<string, any> {
  if (depth === 0) {
    return { value: 'leaf' }
  }

  return {
    [`level-${depth}`]: generateObjectTestData(depth - 1),
    data: `data-${depth}`,
  }
}

/**
 * Verify YAML round-trip integrity
 */
export async function verifyYamlRoundTrip(
  parseYaml: (yaml: string) => any,
  stringifyYaml: (data: any) => string,
  testData: Record<string, unknown>
): Promise<boolean> {
  const yaml = stringifyYaml(testData)
  const parsed = parseYaml(yaml)

  return JSON.stringify(parsed) === JSON.stringify(testData)
}

/**
 * Create performance measurement wrapper
 */
export class PerformanceMonitor {
  private measurements: Map<string, number[]> = new Map()

  measure(label: string, fn: () => void): number {
    const start = performance.now()
    fn()
    const duration = performance.now() - start

    if (!this.measurements.has(label)) {
      this.measurements.set(label, [])
    }
    this.measurements.get(label)!.push(duration)

    return duration
  }

  async measureAsync(label: string, fn: () => Promise<void>): Promise<number> {
    const start = performance.now()
    await fn()
    const duration = performance.now() - start

    if (!this.measurements.has(label)) {
      this.measurements.set(label, [])
    }
    this.measurements.get(label)!.push(duration)

    return duration
  }

  getStats(label: string): { min: number; max: number; avg: number; count: number } | null {
    const measurements = this.measurements.get(label)
    if (!measurements || measurements.length === 0) {
      return null
    }

    return {
      min: Math.min(...measurements),
      max: Math.max(...measurements),
      avg: measurements.reduce((a, b) => a + b, 0) / measurements.length,
      count: measurements.length,
    }
  }

  reset(): void {
    this.measurements.clear()
  }

  report(): void {
    console.log('\n=== Performance Report ===')
    for (const [label] of this.measurements) {
      const stats = this.getStats(label)
      if (stats) {
        console.log(
          `${label}: avg=${stats.avg.toFixed(2)}ms, min=${stats.min.toFixed(2)}ms, max=${stats.max.toFixed(2)}ms (n=${stats.count})`
        )
      }
    }
  }
}

/**
 * Accessibility test helpers
 */
export const a11y = {
  /**
   * Check if element has accessible name
   */
  hasAccessibleName(element: Element): boolean {
    const ariaLabel = element.getAttribute('aria-label')
    const ariaLabelledBy = element.getAttribute('aria-labelledby')
    const title = element.getAttribute('title')
    const text = element.textContent?.trim()

    return !!(ariaLabel || ariaLabelledBy || title || text)
  },

  /**
   * Check if form field has label
   */
  hasLabel(input: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement): boolean {
    const id = input.id
    const ariaLabel = input.getAttribute('aria-label')
    const ariaLabelledBy = input.getAttribute('aria-labelledby')
    const label = id ? document.querySelector(`label[for="${id}"]`) : null

    return !!(ariaLabel || ariaLabelledBy || label)
  },

  /**
   * Get focus order of elements
   */
  getFocusOrder(root: Element = document.body): Element[] {
    return getFocusableElements(root)
  },

  /**
   * Check if element has sufficient color contrast
   */
  async checkContrast(element: Element): Promise<boolean> {
    // This would require actual color contrast calculation
    // For now, just check if element has color styles
    const styles = window.getComputedStyle(element)
    const color = styles.color
    const bgColor = styles.backgroundColor

    return color !== 'rgba(0, 0, 0, 0)' && bgColor !== 'rgba(0, 0, 0, 0)'
  },
}
