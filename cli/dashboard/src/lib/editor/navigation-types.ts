/**
 * Navigation Types - Type definitions for navigation tree
 */

import type { LucideIcon } from 'lucide-react'
import { 
  LayoutDashboard, 
  Server, 
  Package, 
  Tag, 
  Info, 
  Circle, 
  Box,
  GitBranch,
  Workflow,
  CheckCircle,
  Database,
  FileText,
} from 'lucide-react'

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

  // Overview section (name, resourceGroup, metadata, test)
  const overviewChildren: NavigationNode[] = []
  if ('name' in config) {
    overviewChildren.push({ id: 'name', label: 'Application Name', type: 'property', icon: Tag })
  }
  if ('resourceGroup' in config) {
    overviewChildren.push({ id: 'resourceGroup', label: 'Resource Group', type: 'property', icon: Package })
  }
  if ('metadata' in config) {
    overviewChildren.push({ id: 'metadata', label: 'Metadata', type: 'property', icon: Info })
  }
  if ('test' in config) {
    overviewChildren.push({ id: 'test', label: 'Test Configuration', type: 'property', icon: CheckCircle })
  }
  
  nodes.push({
    id: 'overview',
    label: 'Overview',
    type: 'section',
    icon: LayoutDashboard,
    children: overviewChildren,
  })

  // Services section
  const services = config.services as Record<string, unknown> | undefined
  if (services) {
    const serviceNodes: NavigationNode[] = Object.keys(services).map((serviceName) => {
      const service = services[serviceName] as Record<string, unknown> | undefined
      const hasTest = service && 'test' in service
      
      const children: NavigationNode[] = []
      if (hasTest) {
        children.push({
          id: 'test',
          label: 'Test',
          type: 'property',
          icon: CheckCircle,
        })
      }
      
      return {
        id: serviceName,
        label: serviceName,
        type: 'item',
        icon: Server,
        children: children.length > 0 ? children : undefined,
        collapsible: children.length > 0,
      }
    })

    nodes.push({
      id: 'services',
      label: 'Services',
      type: 'section',
      icon: Server,
      children: serviceNodes,
      collapsible: true,
    })
  } else {
    nodes.push({
      id: 'services',
      label: 'Services',
      type: 'section',
      icon: Server,
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
      icon: Box,
    }))

    nodes.push({
      id: 'resources',
      label: 'Resources',
      type: 'section',
      icon: Package,
      children: resourceNodes,
      collapsible: true,
    })
  } else {
    nodes.push({
      id: 'resources',
      label: 'Resources',
      type: 'section',
      icon: Package,
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
      icon: GitBranch,
    })
  }

  // Pipeline section
  if ('pipeline' in config) {
    nodes.push({
      id: 'pipeline',
      label: 'Pipeline',
      type: 'section',
      icon: Workflow,
    })
  }

  // Required Versions section
  if ('requiredVersions' in config) {
    nodes.push({
      id: 'requiredVersions',
      label: 'Required Versions',
      type: 'section',
      icon: CheckCircle,
    })
  }

  // State section
  if ('state' in config) {
    nodes.push({
      id: 'state',
      label: 'State',
      type: 'section',
      icon: Database,
    })
  }

  return nodes
}
