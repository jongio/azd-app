/**
 * YAML parsing and serialization utilities for azure.yaml editor
 * Uses yaml package (by eemeli) which preserves comments and formatting
 */

import { parseDocument, Document, isMap, isSeq } from 'yaml'
import type { DocumentOptions, ParseOptions, ToStringOptions, YAMLMap, YAMLSeq, Scalar, Pair } from 'yaml'

export interface YamlParseResult<T = unknown> {
  success: boolean
  data?: T
  error?: string
  document?: Document // Preserved document with comments
  originalYaml?: string // Original YAML string for reference
}

export interface YamlStringifyOptions {
  indent?: number
  lineWidth?: number
  sortKeys?: boolean
  preserveComments?: boolean // If true, attempts to preserve comments from original document
  originalDocument?: Document // Original document to preserve comments from
}

/**
 * Parse YAML string to JavaScript object while preserving comments
 * @param yamlString - The YAML string to parse
 * @returns Parse result with data, document, and original YAML
 */
export function parseYaml<T = unknown>(yamlString: string): YamlParseResult<T> {
  if (yamlString.trim() === '') {
    return {
      success: true,
      data: null as T,
      originalYaml: yamlString,
    }
  }

  try {
    const parseOptions: ParseOptions = {
      keepSourceTokens: true, // Preserve comments and formatting
      strict: false,
    }

    const doc = parseDocument(yamlString, parseOptions)
    
    // Check for parse errors
    if (doc.errors && doc.errors.length > 0) {
      return {
        success: false,
        error: doc.errors.map(e => e.message).join('; '),
        originalYaml: yamlString,
      }
    }
    
    const data = doc.toJS() as T

    return {
      success: true,
      data,
      document: doc,
      originalYaml: yamlString,
    }
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown parsing error',
      originalYaml: yamlString,
    }
  }
}

/**
 * Stringify JavaScript object to YAML with optional comment preservation
 * @param data - The data to stringify
 * @param options - Stringify options including optional original document
 * @returns YAML string
 */
export function stringifyYaml(data: unknown, options?: YamlStringifyOptions): string {
  const defaultOptions: YamlStringifyOptions = {
    indent: 2,
    lineWidth: 120,
    sortKeys: false, // Maintain key order
    preserveComments: false,
    ...options,
  }

  // If we have an original document and want to preserve comments, update it
  if (defaultOptions.preserveComments && defaultOptions.originalDocument) {
    try {
      const doc = defaultOptions.originalDocument
      // Update the document contents with new data while preserving structure and comments
      doc.contents = doc.createNode(data, {
        keepUndefined: false,
      })

      const toStringOptions: ToStringOptions = {
        indent: defaultOptions.indent ?? 2,
        lineWidth: defaultOptions.lineWidth ?? 120,
        minContentWidth: 0,
        simpleKeys: false,
        doubleQuotedAsJSON: false,
      }

      return doc.toString(toStringOptions)
    } catch {
      // Fall back to regular stringify if update fails
    }
  }

  // Create new document (comments will be lost, but data is preserved)
  const doc = new Document(data, {
    indent: defaultOptions.indent ?? 2,
    lineWidth: defaultOptions.lineWidth ?? 120,
  } as DocumentOptions)

  const toStringOptions: ToStringOptions = {
    indent: defaultOptions.indent ?? 2,
    lineWidth: defaultOptions.lineWidth ?? 120,
    minContentWidth: 0,
    simpleKeys: false,
    doubleQuotedAsJSON: false,
  }

  return doc.toString(toStringOptions)
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
 * Preserves comments and formatting by using the original document
 * @param yamlString - The YAML string to format
 * @param options - Stringify options
 * @returns Formatted YAML string or original if parsing fails
 */
export function formatYaml(yamlString: string, options?: YamlStringifyOptions): string {
  const result = parseYaml(yamlString)
  if (!result.success || result.data === undefined || !result.document) {
    return yamlString // Return original if parsing fails
  }
  
  // Check for document errors before stringifying
  if (result.document.errors && result.document.errors.length > 0) {
    return yamlString // Return original if document has errors
  }
  
  // Use the original document to preserve comments, just reformat
  return result.document.toString({
    indent: options?.indent ?? 2,
    lineWidth: options?.lineWidth ?? 120,
    minContentWidth: 0,
  })
}

/**
 * Update YAML document while preserving comments
 * This is the recommended way to modify YAML while keeping comments
 */
export function updateYamlPreservingComments(
  yamlString: string,
  updates: (doc: Document) => void
): string {
  try {
    const doc = parseDocument(yamlString, { keepSourceTokens: true })
    updates(doc)
    return doc.toString({
      indent: 2,
      lineWidth: 120,
      minContentWidth: 0,
    })
  } catch {
    return yamlString // Return original on error
  }
}

/**
 * Update a specific field in YAML while preserving comments
 * @param yamlString - Original YAML string with comments
 * @param path - Dot-separated path (e.g., 'services.api.port')
 * @param value - New value to set
 * @returns Updated YAML string with comments preserved
 */
export function updateYamlField(
  yamlString: string,
  path: string,
  value: unknown
): string {
  try {
    const doc = parseDocument(yamlString, { keepSourceTokens: true })
    if (!doc.contents) {
      doc.contents = doc.createNode({})
    }
    
    const parts = path.split('.')
    let current: YAMLMap | YAMLSeq | Scalar | Pair | null = doc.contents
    
    // Navigate to the target node, creating path if needed
    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i]
      if (isMap(current)) {
        const existing = current.get(part, true)
        if (existing && (isMap(existing) || isSeq(existing))) {
          current = existing
        } else {
          // Create new map node for this path segment
          const newNode = doc.createNode({})
          current.set(part, newNode)
          current = newNode
        }
      } else {
        // Current is not a map, can't navigate further
        return yamlString // Return original on error
      }
    }
    
    // Set the final value
    const key = parts[parts.length - 1]
    if (isMap(current)) {
      current.set(key, doc.createNode(value))
    }
    
    return doc.toString({
      indent: 2,
      lineWidth: 120,
      minContentWidth: 0,
    })
  } catch {
    // Fallback: parse to object, update, stringify (loses comments but preserves data)
    const parsed = parseYaml(yamlString)
    if (parsed.success && parsed.data) {
      const obj = parsed.data as Record<string, unknown>
      const parts = path.split('.')
      let current: unknown = obj
      for (let i = 0; i < parts.length - 1; i++) {
        if (typeof current === 'object' && current !== null && parts[i] in current) {
          current = (current as Record<string, unknown>)[parts[i]]
        } else if (typeof current === 'object' && current !== null) {
          (current as Record<string, unknown>)[parts[i]] = {}
          current = (current as Record<string, unknown>)[parts[i]]
        }
      }
      if (typeof current === 'object' && current !== null) {
        (current as Record<string, unknown>)[parts[parts.length - 1]] = value
      }
      return stringifyYaml(obj, { preserveComments: false })
    }
    return yamlString
  }
}

/**
 * Merge updates into YAML while preserving comments
 * Uses a smarter approach: parse original, merge data, then update document nodes
 * @param yamlString - Original YAML string with comments
 * @param updates - Object with updates to merge
 * @returns Updated YAML string with comments preserved
 */
export function mergeYamlUpdates(
  yamlString: string,
  updates: Record<string, unknown>
): string {
  try {
    const doc = parseDocument(yamlString, { keepSourceTokens: true })
    
    if (!doc.contents) {
      doc.contents = doc.createNode(updates)
    } else if (isMap(doc.contents)) {
      // Update each key in the document
      for (const [key, value] of Object.entries(updates)) {
        const existing = doc.contents.get(key, true)
        
        // For nested objects, try to preserve structure
        if (existing && isMap(existing) && typeof value === 'object' && value !== null && !Array.isArray(value)) {
          // Update nested properties
          for (const [subKey, subValue] of Object.entries(value as Record<string, unknown>)) {
            existing.set(subKey, doc.createNode(subValue))
          }
        } else if (isMap(doc.contents)) {
          // Replace or add the key (yaml Map.set accepts string keys)
          doc.contents.set(key, doc.createNode(value))
        }
      }
    } else {
      // Contents is not a map, replace it
      doc.contents = doc.createNode(updates)
    }
    
    return doc.toString({
      indent: 2,
      lineWidth: 120,
      minContentWidth: 0,
    })
  } catch {
    // Fallback: parse to object, merge, stringify (loses comments but preserves data)
    const parsed = parseYaml(yamlString)
    if (parsed.success && parsed.data) {
      const current = parsed.data as Record<string, unknown>
      const merged = { ...current }
      for (const [key, value] of Object.entries(updates)) {
        if (typeof value === 'object' && value !== null && !Array.isArray(value) &&
            current[key] && typeof current[key] === 'object' && !Array.isArray(current[key])) {
          merged[key] = { ...(current[key] as Record<string, unknown>), ...(value as Record<string, unknown>) }
        } else {
          merged[key] = value
        }
      }
      return stringifyYaml(merged, { preserveComments: false })
    }
    return yamlString
  }
}

/**
 * Delete a dot-path from YAML while preserving comments
 * @param yamlString - Original YAML string with comments
 * @param path - Dot-separated path to delete (e.g., 'services.api')
 * @returns Updated YAML string
 */
export function deleteYamlPath(yamlString: string, path: string): string {
  try {
    const doc = parseDocument(yamlString, { keepSourceTokens: true })
    if (!doc.contents) {
      return yamlString
    }

    const parts = path.split('.').filter(Boolean)
    if (parts.length === 0) {
      return yamlString
    }

    let current: YAMLMap | YAMLSeq | Scalar | Pair | null = doc.contents
    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i]
      if (!isMap(current)) {
        return yamlString
      }

      const next = current.get(part, true)
      if (!next) {
        return yamlString
      }
      current = next
    }

    const key = parts[parts.length - 1]
    if (isMap(current)) {
      current.delete(key)
    }

    return doc.toString({
      indent: 2,
      lineWidth: 120,
      minContentWidth: 0,
    })
  } catch {
    // Fallback: parse to object, delete, stringify (loses comments but preserves data)
    const parsed = parseYaml(yamlString)
    if (parsed.success && parsed.data && typeof parsed.data === 'object') {
      const obj = parsed.data as Record<string, unknown>
      const parts = path.split('.').filter(Boolean)
      let cursor: unknown = obj
      for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i]
        if (typeof cursor !== 'object' || cursor === null || !(part in cursor)) {
          return yamlString
        }
        cursor = (cursor as Record<string, unknown>)[part]
      }
      if (cursor && typeof cursor === 'object') {
        delete (cursor as Record<string, unknown>)[parts[parts.length - 1]]
      }
      return stringifyYaml(obj, { preserveComments: false })
    }
    return yamlString
  }
}
