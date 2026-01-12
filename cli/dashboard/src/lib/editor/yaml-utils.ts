/**
 * YAML parsing and serialization utilities for azure.yaml editor
 * Uses js-yaml for parsing and stringification
 */

import * as yaml from 'js-yaml'

export interface YamlParseResult<T = unknown> {
  success: boolean
  data?: T
  error?: string
}

export interface YamlStringifyOptions {
  indent?: number
  lineWidth?: number
  noRefs?: boolean
  sortKeys?: boolean
}

/**
 * Parse YAML string to JavaScript object
 * @param yamlString - The YAML string to parse
 * @returns Parse result with data or error
 */
export function parseYaml<T = unknown>(yamlString: string): YamlParseResult<T> {
  try {
    const data = yaml.load(yamlString, {
      // Strict JSON compatibility
      json: false,
      // Don't execute code in YAML
      schema: yaml.DEFAULT_SCHEMA,
    }) as T

    return {
      success: true,
      data,
    }
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown parsing error',
    }
  }
}

/**
 * Stringify JavaScript object to YAML
 * @param data - The data to stringify
 * @param options - Stringify options
 * @returns YAML string
 */
export function stringifyYaml(data: unknown, options?: YamlStringifyOptions): string {
  const defaultOptions: YamlStringifyOptions = {
    indent: 2,
    lineWidth: 120,
    noRefs: true, // Don't use YAML references/anchors
    sortKeys: false, // Maintain key order
    ...options,
  }

  return yaml.dump(data, {
    indent: defaultOptions.indent,
    lineWidth: defaultOptions.lineWidth,
    noRefs: defaultOptions.noRefs,
    sortKeys: defaultOptions.sortKeys,
    // Use block style for better readability
    styles: {
      '!!null': 'canonical',
    },
  })
}

/**
 * Validate YAML string without parsing to object
 * @param yamlString - The YAML string to validate
 * @returns Validation result with error if invalid
 */
export function validateYaml(yamlString: string): { valid: boolean; error?: string } {
  const result = parseYaml(yamlString)
  return {
    valid: result.success,
    error: result.error,
  }
}

/**
 * Safe YAML parse with error handling
 * Returns null if parsing fails instead of throwing
 */
export function safeParseYaml<T = unknown>(yamlString: string): T | null {
  const result = parseYaml<T>(yamlString)
  return result.success && result.data !== undefined ? result.data : null
}

/**
 * Check if a string is valid YAML
 */
export function isValidYaml(yamlString: string): boolean {
  return validateYaml(yamlString).valid
}

/**
 * Format YAML string (parse and re-stringify with consistent formatting)
 * @param yamlString - The YAML string to format
 * @param options - Stringify options
 * @returns Formatted YAML string or original if parsing fails
 */
export function formatYaml(yamlString: string, options?: YamlStringifyOptions): string {
  const result = parseYaml(yamlString)
  if (!result.success || result.data === undefined) {
    return yamlString // Return original if parsing fails
  }
  return stringifyYaml(result.data, options)
}
