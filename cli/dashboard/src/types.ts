export interface LocalServiceInfo {
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health: 'healthy' | 'unhealthy' | 'unknown'
  url?: string
  port?: number
  pid?: number
  startTime?: string
  lastChecked?: string
}

export interface AzureServiceInfo {
  url?: string
  resourceName?: string
  imageName?: string
}

export interface Service {
  name: string
  language?: string
  framework?: string
  project?: string
  local?: LocalServiceInfo
  azure?: AzureServiceInfo
  environmentVariables?: Record<string, string>
  // Legacy fields for compatibility during transition
  status?: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health?: 'healthy' | 'unhealthy' | 'unknown'
  startTime?: string
  lastChecked?: string
  error?: string
}

export interface ServiceUpdate {
  type: 'update' | 'add' | 'remove'
  service: Service
}
