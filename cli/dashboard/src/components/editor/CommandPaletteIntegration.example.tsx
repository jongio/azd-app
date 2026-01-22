/**
 * Command Palette Integration Example
 * 
 * This example demonstrates how to integrate the CommandPalette component
 * into the Azure YAML Editor.
 */

/* eslint-disable react-refresh/only-export-components */
import * as React from 'react'
import { CommandPalette } from './CommandPalette'
import { useCommandPalette } from '@/hooks/useCommandPalette'
import { getDefaultCommands, createActionCommand, createFieldCommand } from '@/lib/editor/command-registry'
import type { Command } from '@/lib/editor/command-types'

/**
 * Example integration in the main editor component
 */
export function EditorWithCommandPalette() {
  // Use the hook to manage palette state and Cmd/Ctrl+K shortcut
  const { isOpen, close } = useCommandPalette()
  
  // Build command list with default commands + dynamic ones
  const commands = React.useMemo<Command[]>(() => {
    const defaultCommands = getDefaultCommands()
    
    // Add dynamic action commands
    const actionCommands: Command[] = [
      createActionCommand(
        'action.add-service',
        'Add Service',
        () => {
          // Open add service modal
        },
        'Add a new service to your application',
        ['add', 'create', 'new', 'service'],
        'Cmd+N'
      ),
      createActionCommand(
        'action.save',
        'Save Configuration',
        () => {
          // Save configuration
        },
        'Save changes to azure.yaml',
        ['save', 'write'],
        'Cmd+S'
      ),
      createActionCommand(
        'action.validate',
        'Validate Configuration',
        () => {
          // Run validation
        },
        'Check configuration for errors',
        ['validate', 'check', 'verify']
      ),
    ]
    
    // Add dynamic field commands based on current form
    const fieldCommands: Command[] = [
      createFieldCommand(
        'field.name',
        'Application Name',
        'name',
        'Jump to application name field',
        ['name', 'title', 'app']
      ),
      createFieldCommand(
        'field.services.api.host',
        'API Service Host',
        'services.api.host',
        'Jump to API service host configuration',
        ['api', 'host', 'service']
      ),
    ]
    
    return [...defaultCommands, ...actionCommands, ...fieldCommands]
  }, [])
  
  // Navigation handler
  const handleNavigate = React.useCallback((_path: string) => {
    // Implement navigation logic here
    // e.g., scroll to section, activate tab, etc.
  }, [])
  
  // Jump to field handler
  const handleJumpToField = React.useCallback((fieldPath: string) => {
    // Implement field focus logic here
    // e.g., find input by data-field-path attribute and focus it
    const field = document.querySelector(`[data-field-path="${fieldPath}"]`) as HTMLElement
    if (field) {
      field.scrollIntoView({ behavior: 'smooth', block: 'center' })
      field.focus()
    }
  }, [])
  
  // Open help handler
  const handleOpenHelp = React.useCallback((_topic: string) => {
    // Open help documentation
    // Implement help panel logic here
    // e.g., open documentation modal, navigate to help section, etc.
  }, [])
  
  return (
    <div>
      {/* Your editor UI here */}
      <div className="p-4">
        <h1>Azure YAML Editor</h1>
        <p className="text-sm text-slate-600">
          Press <kbd className="px-2 py-1 bg-slate-100 rounded">Cmd/Ctrl+K</kbd> to open the command palette
        </p>
      </div>
      
      {/* Command Palette */}
      <CommandPalette
        isOpen={isOpen}
        onClose={close}
        commands={commands}
        onNavigate={handleNavigate}
        onJumpToField={handleJumpToField}
        onOpenHelp={handleOpenHelp}
      />
    </div>
  )
}

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
