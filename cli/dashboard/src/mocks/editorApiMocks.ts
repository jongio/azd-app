import type { WellKnownService } from '@/lib/editor/wellknown-types'

const demoConfig = {
  name: 'azure-yaml-editor',
  services: {
    api: {
      host: 'containerapp',
      project: './src/api',
      language: 'node',
      image: 'mcr.microsoft.com/azuredocs/azure-vote-front:latest',
      ports: ['8080:80'],
    },
    web: {
      host: 'staticwebapp',
      project: './src/web',
      language: 'node',
      image: 'httpd:alpine',
      ports: ['8080:80'],
    },
  },
  resources: {
    storage: {
      type: 'storage',
      uses: ['api'],
    },
  },
}

const demoConfigYaml = `name: azure-yaml-editor
services:
  api:
    host: containerapp
    project: ./src/api
    language: node
    image: mcr.microsoft.com/azuredocs/azure-vote-front:latest
    ports:
      - "8080:80"
  web:
    host: staticwebapp
    project: ./src/web
    language: node
    image: httpd:alpine
    ports:
      - "8080:80"
resources:
  storage:
    type: storage
    uses:
      - api
`

const wellKnownServices: WellKnownService[] = [
  {
    name: 'azurite',
    displayName: 'Azurite',
    description: 'Azure Storage emulator',
    category: 'storage',
    host: 'containerapp',
    image: 'mcr.microsoft.com/azure-storage/azurite',
    ports: ['10000:10000'],
    environment: {
      AZURITE_ACCOUNTS: 'devstoreaccount1:key',
    },
    healthcheck: {
      test: ['CMD', 'curl', '-f', 'http://localhost:10000'],
      interval: '30s',
      timeout: '3s',
      retries: 3,
    },
  },
  {
    name: 'redis',
    displayName: 'Redis Cache',
    description: 'In-memory data store',
    category: 'database',
    host: 'containerapp',
    image: 'redis:7-alpine',
    ports: ['6379:6379'],
    healthcheck: {
      test: ['CMD', 'redis-cli', 'ping'],
      interval: '30s',
      timeout: '3s',
      retries: 3,
    },
  },
  {
    name: 'postgres',
    displayName: 'PostgreSQL',
    description: 'Relational database',
    category: 'database',
    host: 'containerapp',
    image: 'postgres:16-alpine',
    ports: ['5432:5432'],
    environment: {
      POSTGRES_PASSWORD: 'localdevpassword',
    },
    healthcheck: {
      test: ['CMD-SHELL', 'pg_isready -U postgres'],
      interval: '30s',
      timeout: '3s',
      retries: 3,
    },
  },
  {
    name: 'mongo',
    displayName: 'MongoDB',
    description: 'Document database',
    category: 'database',
    host: 'containerapp',
    image: 'mongo:7',
    ports: ['27017:27017'],
    healthcheck: {
      test: ['CMD', 'mongosh', '--eval', 'db.adminCommand("ping")'],
      interval: '30s',
      timeout: '3s',
      retries: 3,
    },
  },
]

const mockBackups = [
  {
    timestamp: '2026-01-01T00:00:00Z',
    path: '/backups/azure-yaml-editor-2026-01-01.yaml',
    size: 2048,
  },
  {
    timestamp: '2026-01-02T00:00:00Z',
    path: '/backups/azure-yaml-editor-2026-01-02.yaml',
    size: 4096,
  },
]

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json',
    },
  })
}

export function setupEditorApiMocks(): void {
  if (typeof window === 'undefined' || typeof window.fetch !== 'function') {
    return
  }

  // Avoid double-patching
  if ((window as unknown as { __editorApiMocksInstalled?: boolean }).__editorApiMocksInstalled) {
    return
  }

  (window as unknown as { __editorApiMocksInstalled?: boolean }).__editorApiMocksInstalled = true

  const originalFetch = window.fetch.bind(window)

  window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' || input instanceof URL
      ? new URL(input, window.location.origin)
      : new URL(input.url, window.location.origin)

    const method = (init?.method || 'GET').toUpperCase()

    if (url.pathname === '/api/editor/config' && method === 'GET') {
      return jsonResponse({
        path: './azure.yaml',
        content: demoConfigYaml,
        lastModified: new Date().toISOString(),
      })
    }

    if (url.pathname === '/api/editor/config' && method === 'POST') {
      return jsonResponse({ success: true, backup: 'azure.yaml.bak', written: true })
    }

    if (url.pathname === '/api/editor/wellknown') {
      return jsonResponse({ services: wellKnownServices })
    }

    if (url.pathname.startsWith('/api/editor/wellknown/')) {
      const name = url.pathname.split('/').pop()
      const service = wellKnownServices.find((s) => s.name === name)
      if (service) {
        return jsonResponse(service)
      }
      return jsonResponse({ error: 'Not found' }, 404)
    }

    if (url.pathname === '/api/editor/backups' && method === 'GET') {
      return jsonResponse({ backups: mockBackups })
    }

    if (url.pathname.startsWith('/api/editor/backups/') && method === 'GET') {
      const timestamp = url.pathname.split('/').pop() || mockBackups[0].timestamp
      return jsonResponse({ content: demoConfigYaml, timestamp })
    }

    if (url.pathname.includes('/restore') && method === 'POST') {
      const timestamp = url.pathname.split('/').slice(-2)[0]
      return jsonResponse({ success: true, restoredFrom: timestamp, backupCreated: mockBackups[0].timestamp })
    }

    if (url.pathname.startsWith('/api/editor/backups/') && method === 'DELETE') {
      return jsonResponse({ success: true })
    }

    return originalFetch(input as RequestInfo, init)
  }
}

export const editorDemoConfig = demoConfig
export const editorDemoConfigYaml = demoConfigYaml
export const editorDemoWellKnown = wellKnownServices
export const editorDemoBackups = mockBackups
