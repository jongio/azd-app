/**
 * Navigation Types - Type definitions for navigation tree
 */

import type { LucideIcon } from 'lucide-react'

/**
 * Validation issue level
 */
export type ValidationLevel = 'error' | 'warning' | 'info'

/**
 * Validation issue
 */
export interface ValidationIssue {
  level: ValidationLevel
  message: string
  path: string
}

/**
 * Navigation node type
 */
export type NavigationNodeType = 
  | 'section'    // Top-level section (Services, Resources, etc.)
  | 'item'       // Individual item (service name, resource name)
  | 'property'   // Property (name, resourceGroup, etc.)

/**
 * Navigation tree node
 */
export interface NavigationNode {
  /** Unique identifier */
  id: string
  /** Display label */
  label: string
  /** Node type */
  type: NavigationNodeType
  /** Optional icon */
  icon?: LucideIcon
  /** Child nodes */
  children?: NavigationNode[]
  /** Whether node is collapsible */
  collapsible?: boolean
  /** Optional metadata */
  metadata?: Record<string, unknown>
}

/**
 * Build navigation tree from azure.yaml configuration
 */
export function buildNavigationTree(
  config: Record<string, unknown> | null
): NavigationNode[] {
  if (!config) {
    return []
  }

  const nodes: NavigationNode[] = []

  // Overview section (name, resourceGroup, metadata)
  nodes.push({
    id: 'overview',
    label: 'Overview',
    type: 'section',
    children: [
      { id: 'name', label: 'Application Name', type: 'property' },
      { id: 'resourceGroup', label: 'Resource Group', type: 'property' },
      { id: 'metadata', label: 'Metadata', type: 'property' },
    ].filter((node) => node.id in config),
  })

  // Services section
  const services = config.services as Record<string, unknown> | undefined
  if (services) {
    const serviceNodes: NavigationNode[] = Object.keys(services).map((serviceName) => ({
      id: serviceName,
      label: serviceName,
      type: 'item',
    }))

    nodes.push({
      id: 'services',
      label: 'Services',
      type: 'section',
      children: serviceNodes,
      collapsible: true,
    })
  } else {
    nodes.push({
      id: 'services',
      label: 'Services',
      type: 'section',
      children: [],
      collapsible: true,
    })
  }

  // Resources section
  const resources = config.resources as Record<string, unknown> | undefined
  if (resources) {
    const resourceNodes: NavigationNode[] = Object.keys(resources).map((resourceName) => ({
      id: resourceName,
      label: resourceName,
      type: 'item',
    }))

    nodes.push({
      id: 'resources',
      label: 'Resources',
      type: 'section',
      children: resourceNodes,
      collapsible: true,
    })
  } else {
    nodes.push({
      id: 'resources',
      label: 'Resources',
      type: 'section',
      children: [],
      collapsible: true,
    })
  }

  // Hooks section
  if ('hooks' in config) {
    nodes.push({
      id: 'hooks',
      label: 'Hooks',
      type: 'section',
    })
  }

  // Pipeline section
  if ('pipeline' in config) {
    nodes.push({
      id: 'pipeline',
      label: 'Pipeline',
      type: 'section',
    })
  }

  // Required Versions section
  if ('requiredVersions' in config) {
    nodes.push({
      id: 'requiredVersions',
      label: 'Required Versions',
      type: 'section',
    })
  }

  // State section
  if ('state' in config) {
    nodes.push({
      id: 'state',
      label: 'State',
      type: 'section',
    })
  }

  return nodes
}
