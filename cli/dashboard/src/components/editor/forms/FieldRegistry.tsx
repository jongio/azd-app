/**
 * Field Registry - Centralized field component registry to avoid circular dependencies
 * 
 * This file breaks the circular dependency between FieldRenderer and complex fields
 * (ArrayField, ObjectField) by providing a registry that can be populated after all
 * modules are loaded.
 */

import type { ComponentType } from 'react'
import type { FieldRendererProps } from './FieldRenderer'

type FieldComponent = ComponentType<FieldRendererProps>

interface FieldRegistry {
  array?: FieldComponent
  object?: FieldComponent
}

const registry: FieldRegistry = {}

export function registerField(type: 'array' | 'object', component: FieldComponent) {
  registry[type] = component
}

export function getFieldComponent(type: 'array' | 'object'): FieldComponent | undefined {
  return registry[type]
}
