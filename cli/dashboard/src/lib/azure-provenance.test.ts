import { describe, it, expect } from 'vitest'
import {
  CONTAINER_APPS_ENV_MARKERS,
  FUNCTIONS_ENV_MARKERS,
  extractEndpoint,
  extractMethod,
  extractRoute,
  hasAzureProvenance,
  hasContainerAppsProvenance,
  hasFunctionsProvenance,
  hasPublicEndpointInfo,
  parseAzureProvenance,
  parseContainerAppsProvenance,
  parseFunctionsProvenance,
} from './azure-provenance'

const logs = {
  containerAzure:
    '[2025-01-15T10:00:00Z] GET /health - containerapp-api ' +
    'method=GET route=/health endpoint=https://my-api.example.com/health ' +
    'azure_provider=azure azure_service=container-apps ' +
    'azure_app=my-api azure_revision=rev1 azure_replica=replica1 ' +
    'azure_env=prod azure_region=eastus azure_hostname=my-api.example.com',
  containerLocal:
    '[2025-01-15T10:00:00Z] GET /health - containerapp-api ' +
    'method=GET route=/health endpoint=http://localhost:9847/health ' +
    'xf_host=undefined xf_proto=undefined',
  functionsAzure:
    '[ENDPOINT] service=functions-worker method=GET route=/api/health ' +
    'endpoint=https://my-func.azurewebsites.net/api/health ' +
    'xf_host=my-func.azurewebsites.net xf_proto=https ' +
    'azure_provider=azure azure_service=functions ' +
    'azure_site=my-func azure_region=eastus azure_hostname=my-func.azurewebsites.net ' +
    'azure_runtime=node azure_sku=Y1 azure_instance=abc123',
  noEndpoint: '[INFO] Processing request batch #1',
} as const

describe('azure-provenance', () => {
  it('includes key env markers', () => {
    for (const marker of ['CONTAINER_APP_NAME', 'CONTAINER_APP_REVISION', 'CONTAINER_APP_REPLICA_NAME', 'CONTAINER_APP_ENVIRONMENT_NAME']) {
      expect(CONTAINER_APPS_ENV_MARKERS).toContain(marker)
    }

    for (const marker of ['WEBSITE_SITE_NAME', 'FUNCTIONS_WORKER_RUNTIME', 'WEBSITE_HOSTNAME', 'WEBSITE_INSTANCE_ID']) {
      expect(FUNCTIONS_ENV_MARKERS).toContain(marker)
    }
  })

  it('detects provenance types', () => {
    expect(hasContainerAppsProvenance(logs.containerAzure)).toBe(true)
    expect(hasContainerAppsProvenance(logs.functionsAzure)).toBe(false)
    expect(hasContainerAppsProvenance(logs.containerLocal)).toBe(false)
    expect(hasContainerAppsProvenance('')).toBe(false)

    expect(hasFunctionsProvenance(logs.functionsAzure)).toBe(true)
    expect(hasFunctionsProvenance(logs.containerAzure)).toBe(false)
    expect(hasFunctionsProvenance('')).toBe(false)

    expect(hasAzureProvenance(logs.containerAzure)).toBe(true)
    expect(hasAzureProvenance(logs.functionsAzure)).toBe(true)
    expect(hasAzureProvenance(logs.containerLocal)).toBe(false)
  })

  it('parses container apps provenance', () => {
    const parsed = parseContainerAppsProvenance(logs.containerAzure)
    expect(parsed).not.toBeNull()
    expect(parsed?.service).toBe('container-apps')
    expect(parsed?.appName).toBe('my-api')
    expect(parsed?.region).toBe('eastus')

    expect(parseContainerAppsProvenance(logs.containerLocal)).toBeNull()
    expect(parseContainerAppsProvenance('azure_provider=azure azure_service=container-apps azure_app=myapp')?.appName).toBe('myapp')
  })

  it('parses functions provenance', () => {
    const parsed = parseFunctionsProvenance(logs.functionsAzure)
    expect(parsed).not.toBeNull()
    expect(parsed?.service).toBe('functions')
    expect(parsed?.siteName).toBe('my-func')
    expect(parsed?.runtime).toBe('node')

    expect(parseFunctionsProvenance('[INFO] Running locally (no Azure provenance)')).toBeNull()
  })

  it('parses any azure provenance', () => {
    expect(parseAzureProvenance(logs.containerAzure)?.service).toBe('container-apps')
    expect(parseAzureProvenance(logs.functionsAzure)?.service).toBe('functions')
    expect(parseAzureProvenance(logs.containerLocal)).toBeNull()
  })

  it('extracts endpoint, method, and route', () => {
    expect(hasPublicEndpointInfo(logs.containerAzure)).toBe(true)
    expect(hasPublicEndpointInfo(logs.containerLocal)).toBe(true)
    expect(hasPublicEndpointInfo('[INFO] Public endpoint: GET https://example.com/health')).toBe(true)
    expect(hasPublicEndpointInfo(logs.noEndpoint)).toBe(false)

    expect(extractEndpoint(logs.containerAzure)).toBe('https://my-api.example.com/health')
    expect(extractEndpoint(logs.noEndpoint)).toBeNull()

    expect(extractMethod('method=get route=/health')).toBe('GET')
    expect(extractMethod(logs.noEndpoint)).toBeNull()

    expect(extractRoute('method=GET route=/api/health endpoint=http://localhost:7071/api/health')).toBe('/api/health')
    expect(extractRoute(logs.noEndpoint)).toBeNull()
  })
})
