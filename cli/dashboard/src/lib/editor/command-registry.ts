/**
 * Command Registry
 * Default command definitions for the command palette
 */

import type { Command } from './command-types'

/**
 * Create navigation command
 */
export function createNavigationCommand(
  id: string,
  label: string,
  path: string,
  description?: string,
  keywords?: string[]
): Command {
  return {
    id,
    label,
    description,
    category: 'navigation',
    icon: 'ArrowRight',
    keywords,
    action: { type: 'navigate', path },
  }
}

/**
 * Create action command
 */
export function createActionCommand(
  id: string,
  label: string,
  handler: () => void,
  description?: string,
  keywords?: string[],
  shortcut?: string
): Command {
  return {
    id,
    label,
    description,
    category: 'action',
    icon: 'Zap',
    keywords,
    shortcut,
    action: { type: 'execute', handler },
  }
}

/**
 * Create field jump command
 */
export function createFieldCommand(
  id: string,
  label: string,
  fieldPath: string,
  description?: string,
  keywords?: string[]
): Command {
  return {
    id,
    label,
    description,
    category: 'field',
    icon: 'Edit3',
    keywords,
    action: { type: 'jump-to-field', fieldPath },
  }
}

/**
 * Create help command
 */
export function createHelpCommand(
  id: string,
  label: string,
  topic: string,
  description?: string,
  keywords?: string[]
): Command {
  return {
    id,
    label,
    description,
    category: 'help',
    icon: 'HelpCircle',
    keywords,
    action: { type: 'open-help', topic },
  }
}

/**
 * Get default command registry
 */
export function getDefaultCommands(): Command[] {
  return [
    // Navigation commands
    createNavigationCommand(
      'nav.overview',
      'Go to Overview',
      'overview',
      'View application overview and metadata',
      ['overview', 'home', 'app', 'name', 'metadata']
    ),
    createNavigationCommand(
      'nav.services',
      'Go to Services',
      'services',
      'View and manage services',
      ['services', 'containers', 'apps']
    ),
    createNavigationCommand(
      'nav.resources',
      'Go to Resources',
      'resources',
      'View and manage Azure resources',
      ['resources', 'azure', 'infrastructure']
    ),
    createNavigationCommand(
      'nav.hooks',
      'Go to Hooks',
      'hooks',
      'Configure lifecycle hooks',
      ['hooks', 'lifecycle', 'events', 'scripts']
    ),
    createNavigationCommand(
      'nav.pipeline',
      'Go to Pipeline',
      'pipeline',
      'Configure CI/CD pipeline',
      ['pipeline', 'cicd', 'deployment', 'github', 'actions']
    ),
    createNavigationCommand(
      'nav.metadata',
      'Go to Metadata',
      'metadata',
      'Edit application metadata',
      ['metadata', 'info', 'description']
    ),
  ]
}
