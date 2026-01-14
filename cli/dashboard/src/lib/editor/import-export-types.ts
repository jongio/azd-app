/**
 * Import/Export Types
 * Type definitions for import/export functionality
 */

export type MergeStrategy = 'replace' | 'merge' | 'cherry-pick'
export type ImportSource = 'template' | 'file' | 'paste'
export type ExportFormat = 'yaml' | 'json' | 'template'

export interface TemplateMetadata {
  id: string
  name: string
  description: string
  category: string
  tags: string[]
  icon?: string
}

export interface ImportOptions {
  source: ImportSource
  strategy: MergeStrategy
  selectedSections?: string[]
}

export interface ExportOptions {
  format: ExportFormat
  includeComments?: boolean
  minify?: boolean
  includeSecrets?: boolean
  templateMode?: boolean
}

export interface ImportPreview {
  current: string
  imported: string
  merged: string
  diff: DiffSection[]
}

export interface DiffSection {
  path: string
  type: 'added' | 'removed' | 'changed' | 'unchanged'
  currentValue?: unknown
  importedValue?: unknown
}

export interface CherryPickSection {
  id: string
  name: string
  description: string
  type: 'service' | 'resource' | 'hooks' | 'pipeline' | 'metadata'
  selected: boolean
}

export interface SecurityWarning {
  type: 'secrets' | 'sensitive-data' | 'overwrite'
  message: string
  severity: 'warning' | 'error'
  requiresConfirmation: boolean
}
