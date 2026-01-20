/**
 * Utility function for building dynamic commands based on configuration
 */

import type { Command } from '@/lib/editor/command-types'
import { createFieldCommand } from '@/lib/editor/command-registry'

/**
 * Example: Building dynamic commands based on configuration
 */
export function buildDynamicCommands(config: {
  services?: Record<string, unknown>
  resources?: Record<string, unknown>
}): Command[] {
  const commands: Command[] = []
  
  // Add navigation commands for each service
  if (config.services) {
    Object.keys(config.services).forEach((serviceName) => {
      commands.push({
        id: `nav.services.${serviceName}`,
        label: `Go to ${serviceName}`,
        description: `View ${serviceName} service configuration`,
        category: 'navigation',
        icon: 'ArrowRight',
        keywords: [serviceName, 'service'],
        action: { type: 'navigate', path: `services.${serviceName}` },
      })
      
      // Add field commands for service properties
      commands.push(
        createFieldCommand(
          `field.services.${serviceName}.host`,
          `${serviceName} - Host Type`,
          `services.${serviceName}.host`,
          `Edit host type for ${serviceName}`,
          [serviceName, 'host', 'type']
        ),
        createFieldCommand(
          `field.services.${serviceName}.ports`,
          `${serviceName} - Ports`,
          `services.${serviceName}.ports`,
          `Configure ports for ${serviceName}`,
          [serviceName, 'ports', 'port']
        )
      )
    })
  }
  
  // Add navigation commands for each resource
  if (config.resources) {
    Object.keys(config.resources).forEach((resourceName) => {
      commands.push({
        id: `nav.resources.${resourceName}`,
        label: `Go to ${resourceName}`,
        description: `View ${resourceName} resource configuration`,
        category: 'navigation',
        icon: 'ArrowRight',
        keywords: [resourceName, 'resource'],
        action: { type: 'navigate', path: `resources.${resourceName}` },
      })
    })
  }
  
  return commands
}
