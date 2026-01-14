/**
 * Axe-core accessibility testing utilities
 */

import axe, { type Result } from 'axe-core'

export interface A11yViolation {
  id: string
  impact: 'minor' | 'moderate' | 'serious' | 'critical'
  description: string
  help: string
  helpUrl: string
  nodes: Array<{
    html: string
    target: any
    failureSummary: string
  }>
}

export interface A11yTestResult {
  violations: A11yViolation[]
  passes: number
  incomplete: number
  inaccessible: boolean
}



/**
 * Run axe accessibility test on a container
 */
export async function runAxe(
  container: HTMLElement,
  options: {
    rules?: Record<string, { enabled: boolean }>
    runOnly?: string[]
  } = {}
): Promise<A11yTestResult> {
  const { rules = {}, runOnly } = options

  const config: any = {
    rules: {
      // Default rules
      'color-contrast': { enabled: true },
      'aria-allowed-attr': { enabled: true },
      'aria-required-attr': { enabled: true },
      'aria-valid-attr': { enabled: true },
      'button-name': { enabled: true },
      'label': { enabled: true },
      'link-name': { enabled: true },
      ...rules,
    },
  }

  if (runOnly) {
    config.runOnly = {
      type: 'tag',
      values: runOnly,
    }
  }

  const results: any = await axe.run(container, config)

  return {
    violations: results.violations.map(formatViolation),
    passes: results.passes.length,
    incomplete: results.incomplete.length,
    inaccessible: results.violations.length > 0,
  }
}

/**
 * Format axe violation for easier reading
 */
function formatViolation(result: Result): A11yViolation {
  return {
    id: result.id,
    impact: result.impact as any,
    description: result.description,
    help: result.help,
    helpUrl: result.helpUrl,
    nodes: result.nodes.map((node) => ({
      html: node.html,
      target: node.target,
      failureSummary: node.failureSummary || '',
    })),
  }
}

/**
 * Run WCAG AA test
 */
export async function runWCAGAA(container: HTMLElement): Promise<A11yTestResult> {
  return runAxe(container, {
    runOnly: ['wcag2a', 'wcag2aa'],
  })
}

/**
 * Run critical accessibility tests only
 */
export async function runCriticalTests(container: HTMLElement): Promise<A11yTestResult> {
  return runAxe(container, {
    rules: {
      'color-contrast': { enabled: true },
      'aria-allowed-attr': { enabled: true },
      'aria-required-attr': { enabled: true },
      'button-name': { enabled: true },
      'label': { enabled: true },
      'link-name': { enabled: true },
    },
  })
}

/**
 * Assert no accessibility violations
 * Throws error if violations found
 */
export function assertNoViolations(result: A11yTestResult) {
  if (result.violations.length > 0) {
    const message = formatViolationsMessage(result.violations)
    throw new Error(`Accessibility violations found:\n${message}`)
  }
}

/**
 * Format violations for error message
 */
function formatViolationsMessage(violations: A11yViolation[]): string {
  return violations
    .map((violation, index) => {
      const nodes = violation.nodes
        .map((node, i) => `    ${i + 1}. ${node.html}\n       Target: ${node.target.join(', ')}`)
        .join('\n')

      return `${index + 1}. [${violation.impact.toUpperCase()}] ${violation.help}\n   ${violation.helpUrl}\n${nodes}`
    })
    .join('\n\n')
}

/**
 * Filter violations by impact level
 */
export function filterByImpact(
  violations: A11yViolation[],
  minImpact: 'minor' | 'moderate' | 'serious' | 'critical'
): A11yViolation[] {
  const impactLevels = { minor: 0, moderate: 1, serious: 2, critical: 3 }
  const threshold = impactLevels[minImpact]

  return violations.filter((v) => impactLevels[v.impact] >= threshold)
}

/**
 * Group violations by impact
 */
export function groupByImpact(violations: A11yViolation[]): Record<string, A11yViolation[]> {
  return violations.reduce((acc, violation) => {
    const impact = violation.impact
    if (!acc[impact]) acc[impact] = []
    acc[impact].push(violation)
    return acc
  }, {} as Record<string, A11yViolation[]>)
}
