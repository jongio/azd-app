/**
 * Large Configuration Generator
 * Generates large azure.yaml configurations for performance testing
 */

export interface Service {
  host: 'containerapp' | 'appservice' | 'function' | 'staticwebapp' | 'springapp' | 'aks'
  language?: 'node' | 'python' | 'dotnet' | 'java' | 'go'
  project?: string
  image?: string
  ports?: string[]
  environment?: Record<string, string>
  healthcheck?: {
    test: string
    interval: string
    timeout: string
    retries: number
  }
}

export interface AzureYamlConfig {
  name: string
  services: Record<string, Service>
  metadata?: {
    template?: string
  }
}

/**
 * Generate a large configuration with specified number of services
 */
export function generateLargeConfig(serviceCount: number): AzureYamlConfig {
  const services: Record<string, Service> = {}

  for (let i = 0; i < serviceCount; i++) {
    const serviceName = `service-${i.toString().padStart(4, '0')}`
    
    // Vary service types
    const types = ['containerapp', 'appservice', 'function', 'staticwebapp']
    const host = types[i % types.length] as Service['host']
    
    // Vary languages
    const languages = ['node', 'python', 'dotnet', 'java', 'go']
    const language = languages[i % languages.length] as Service['language']
    
    services[serviceName] = {
      host,
      language,
      project: `./src/${serviceName}`,
      ports: [`${8000 + i}:${8000 + i}`],
      environment: generateEnvironmentVariables(i),
      healthcheck: {
        test: `curl -f http://localhost:${8000 + i}/health || exit 1`,
        interval: '30s',
        timeout: '10s',
        retries: 3,
      },
    }
  }

  return {
    name: `large-app-${serviceCount}`,
    services,
    metadata: {
      template: 'performance-test',
    },
  }
}

/**
 * Generate environment variables for a service
 */
function generateEnvironmentVariables(index: number): Record<string, string> {
  const vars: Record<string, string> = {}
  
  // Generate 10-20 environment variables per service
  const count = 10 + (index % 10)
  
  for (let i = 0; i < count; i++) {
    vars[`VAR_${i}`] = `value-${index}-${i}`
  }
  
  vars['NODE_ENV'] = 'development'
  vars['PORT'] = (8000 + index).toString()
  vars['SERVICE_NAME'] = `service-${index.toString().padStart(4, '0')}`
  vars['LOG_LEVEL'] = index % 2 === 0 ? 'info' : 'debug'
  
  return vars
}

/**
 * Generate validation errors for a large configuration
 */
export function generateValidationErrors(serviceCount: number) {
  const errors = []
  const warnings = []
  const info = []
  
  // Add some errors
  for (let i = 0; i < serviceCount / 10; i++) {
    errors.push({
      level: 'error' as const,
      message: `Service 'service-${i.toString().padStart(4, '0')}' has invalid configuration`,
      path: `services.service-${i.toString().padStart(4, '0')}`,
    })
  }
  
  // Add some warnings
  for (let i = 0; i < serviceCount / 5; i++) {
    warnings.push({
      level: 'warning' as const,
      message: `Port ${8000 + i} may be in use by another service`,
      path: `services.service-${i.toString().padStart(4, '0')}.ports`,
    })
  }
  
  // Add some info
  for (let i = 0; i < serviceCount / 3; i++) {
    info.push({
      level: 'info' as const,
      message: `Consider adding resource limits for service-${i.toString().padStart(4, '0')}`,
      path: `services.service-${i.toString().padStart(4, '0')}`,
    })
  }
  
  return { errors, warnings, info }
}

/**
 * Generate navigation nodes for a large configuration
 */
export function generateNavigationNodes(serviceCount: number) {
  const serviceNodes = []
  
  for (let i = 0; i < serviceCount; i++) {
    serviceNodes.push({
      id: `service-${i.toString().padStart(4, '0')}`,
      label: `Service ${i.toString().padStart(4, '0')}`,
      type: 'service' as const,
    })
  }
  
  return [
    {
      id: 'overview',
      label: 'Overview',
      type: 'section' as const,
    },
    {
      id: 'services',
      label: 'Services',
      type: 'section' as const,
      children: serviceNodes,
    },
    {
      id: 'resources',
      label: 'Resources',
      type: 'section' as const,
    },
    {
      id: 'hooks',
      label: 'Hooks',
      type: 'section' as const,
    },
  ]
}

/**
 * Generate command palette commands for a large configuration
 */
export function generateCommands(serviceCount: number) {
  const commands = []
  
  // Navigation commands
  for (let i = 0; i < serviceCount; i++) {
    commands.push({
      id: `nav-service-${i}`,
      label: `Go to Service ${i.toString().padStart(4, '0')}`,
      description: `Navigate to service-${i.toString().padStart(4, '0')} configuration`,
      category: 'navigation' as const,
      action: {
        type: 'navigate' as const,
        path: `services.service-${i.toString().padStart(4, '0')}`,
      },
    })
  }
  
  // Action commands
  commands.push(
    {
      id: 'add-service',
      label: 'Add Service',
      description: 'Add a new service to the configuration',
      category: 'action' as const,
      action: {
        type: 'execute' as const,
        handler: () => { /* Add service logic */ },
      },
    },
    {
      id: 'validate',
      label: 'Validate Configuration',
      description: 'Run full validation on the configuration',
      category: 'action' as const,
      action: {
        type: 'execute' as const,
        handler: () => { /* Validate logic */ },
      },
    }
  )
  
  return commands
}

/**
 * Performance test scenarios
 */
export const PERFORMANCE_SCENARIOS = {
  small: { serviceCount: 10, name: 'Small (10 services)' },
  medium: { serviceCount: 50, name: 'Medium (50 services)' },
  large: { serviceCount: 100, name: 'Large (100 services)' },
  xlarge: { serviceCount: 200, name: 'Extra Large (200 services)' },
}
