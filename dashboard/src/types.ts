export interface Service {
  name: string
  projectDir: string
  pid: number
  port: number
  url: string
  language: string
  framework: string
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error'
  health: 'healthy' | 'unhealthy' | 'unknown'
  startTime: string
  lastChecked: string
  error?: string
}

export interface ServiceUpdate {
  type: 'update' | 'add' | 'remove'
  service: Service
}
