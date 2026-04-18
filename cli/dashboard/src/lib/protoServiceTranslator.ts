/**
 * Translation between proto ServiceInfo (Connect wire type) and the
 * dashboard's existing Service domain type.
 *
 * Why this file exists:
 * - The Service shape pre-dates the proto. Components consume `service.local.url`,
 *   `service.local.health`, `service.azure.customDomain` etc. and would
 *   require a sweeping rename to consume the proto enum/flat structure
 *   directly. During the parallel-stack window (Stages 1-3 of the ADR),
 *   the cheapest way to keep components untouched is to translate at the
 *   transport boundary.
 * - The Go server's `serviceInfoToProto` flattens the legacy Local/Azure
 *   structs into a flat proto and stashes overflow fields in a
 *   `metadata` Struct. This module reverses that flattening, including
 *   metadata.autoUrl precedence for the URL/customUrl pair.
 * - Stage 4 (REST removal) keeps this translator in place; Stage 5+ will
 *   gradually consume proto types directly and shrink this file.
 */
import type { ServiceInfo, AzureDeploymentInfo } from '@/gen/proto/azdapp/v1/common_pb.js'
import { ServiceStatus, HealthState } from '@/gen/proto/azdapp/v1/common_pb.js'
import type { Service, ServiceStatus as DashboardServiceStatus, AzureServiceInfo, LocalServiceInfo } from '@/types'

/**
 * Reverse of mapServiceStatus on the Go side. Returns a string value the
 * dashboard's existing UI logic understands. UNSPECIFIED is treated as
 * "stopped" so cards render predictably; the alternative (undefined)
 * would force every consumer to add a null check.
 */
function statusEnumToString(status: ServiceStatus): DashboardServiceStatus {
  switch (status) {
    case ServiceStatus.READY:
      return 'ready'
    case ServiceStatus.STARTING:
      return 'starting'
    case ServiceStatus.STOPPED:
      return 'stopped'
    case ServiceStatus.STOPPING:
      return 'stopping'
    case ServiceStatus.DEGRADED:
      return 'ready' // Dashboard surfaces degraded via the health field.
    case ServiceStatus.ERROR:
      return 'error'
    case ServiceStatus.UNSPECIFIED:
    default:
      return 'stopped'
  }
}

/**
 * Reverse of mapHealthState. Honors the metadata.health override the
 * server stashes for "degraded" so the dashboard can keep rendering the
 * existing degraded badge.
 */
function healthEnumToString(
  health: HealthState,
  override: string | undefined,
): LocalServiceInfo['health'] {
  if (override === 'degraded') {
    // Server signals degraded via metadata; surface it as a valid string.
    // The dashboard `HealthStatus` union accepts 'degraded' even though
    // LocalServiceInfo's narrower 'healthy'|'unhealthy'|'unknown' alias
    // doesn't -- preserve the richer string by escaping the type here.
    return 'degraded' as LocalServiceInfo['health']
  }
  switch (health) {
    case HealthState.HEALTHY:
      return 'healthy'
    case HealthState.UNHEALTHY:
      return 'unhealthy'
    case HealthState.STARTING:
      return 'starting' as LocalServiceInfo['health']
    case HealthState.UNKNOWN:
      return 'unknown'
    case HealthState.UNSPECIFIED:
    default:
      return 'unknown'
  }
}

/**
 * Convert protobuf Struct fields (which arrive as JSON `Value`s after
 * connect-web serialisation) to a plain key/value map of strings.
 * Non-string values are JSON-stringified so callers can still inspect
 * them; in practice the server only writes strings + nested string maps.
 */
function structToObject(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object') return {}
  // After JSON wire transport, Struct surfaces as a plain object whose
  // values are themselves Value objects with a `kind` discriminator.
  // connect-web's protobuf-es runtime exposes them via `.toJson()` on
  // the Struct message instance; here we receive the already-JSON form.
  return value as Record<string, unknown>
}

function metaString(meta: Record<string, unknown>, key: string): string | undefined {
  const v = meta[key]
  return typeof v === 'string' ? v : undefined
}

function azureFromProto(
  azure: AzureDeploymentInfo | undefined,
  azureMeta: Record<string, unknown>,
): AzureServiceInfo | undefined {
  if (!azure && Object.keys(azureMeta).length === 0) return undefined
  const out: AzureServiceInfo = {}
  if (azure?.resourceId) out.resourceName = azure.resourceId
  if (azure?.resourceType) out.resourceType = azure.resourceType
  if (azure?.resourceGroup) out.resourceGroup = azure.resourceGroup
  if (azure?.subscriptionId) out.subscriptionId = azure.subscriptionId
  if (azure?.region) out.location = azure.region
  if (azure?.workspaceId) out.logAnalyticsId = azure.workspaceId
  const url = metaString(azureMeta, 'url')
  if (url) out.url = url
  const customUrl = metaString(azureMeta, 'customUrl')
  if (customUrl) out.customUrl = customUrl
  const customDomain = metaString(azureMeta, 'customDomain')
  if (customDomain) out.customDomain = customDomain
  const cds = metaString(azureMeta, 'customDomainSource')
  if (cds === 'user' || cds === 'azure-sdk') out.customDomainSource = cds
  const imageName = metaString(azureMeta, 'imageName')
  if (imageName) out.imageName = imageName
  return Object.keys(out).length > 0 ? out : undefined
}

/**
 * Convert a proto ServiceInfo back into the dashboard's Service shape.
 *
 * Field-precedence rules that mirror the Go-side flattener:
 * - The proto `url` field carries the override (CustomURL) when present,
 *   the auto-discovered URL otherwise. Metadata.autoUrl signals the
 *   override case so we can split them back into local.url + local.customUrl.
 * - Top-level legacy fields (status, health, startTime) are duplicated
 *   from the local block for components that still read the flat shape.
 */
export function protoServiceToService(info: ServiceInfo): Service {
  const meta = info.metadata ? structToObject(info.metadata.toJson()) : {}
  const azureMetaRaw = meta.azure
  const azureMeta = azureMetaRaw && typeof azureMetaRaw === 'object'
    ? (azureMetaRaw as Record<string, unknown>)
    : {}

  const healthOverride = metaString(meta, 'health')
  const lastChecked = metaString(meta, 'lastChecked')
  const serviceType = metaString(meta, 'serviceType')
  const serviceMode = metaString(meta, 'serviceMode')
  const autoUrl = metaString(meta, 'autoUrl')

  const statusStr = statusEnumToString(info.status)
  const healthStr = healthEnumToString(info.health, healthOverride)

  // Determine the local block. The server emits one whenever there's any
  // local-side data; we infer presence from non-zero fields rather than
  // a dedicated flag (the proto doesn't carry one).
  const hasLocal =
    info.status !== ServiceStatus.UNSPECIFIED ||
    info.health !== HealthState.UNSPECIFIED ||
    info.url !== '' ||
    info.port !== 0 ||
    info.pid !== 0 ||
    info.startTime !== undefined ||
    lastChecked !== undefined

  let local: LocalServiceInfo | undefined
  if (hasLocal) {
    local = {
      status: statusStr,
      health: healthStr,
    }
    if (autoUrl !== undefined) {
      // Split: proto url is the override, autoUrl is the discovered one.
      local.url = autoUrl
      local.customUrl = info.url
    } else if (info.url !== '') {
      local.url = info.url
    }
    if (info.port !== 0) local.port = info.port
    if (info.pid !== 0) local.pid = info.pid
    if (info.startTime) local.startTime = info.startTime.toDate().toISOString()
    if (lastChecked) local.lastChecked = lastChecked
    if (serviceType) local.serviceType = serviceType as LocalServiceInfo['serviceType']
    if (serviceMode) local.serviceMode = serviceMode as LocalServiceInfo['serviceMode']
  }

  const service: Service = {
    name: info.name,
  }
  if (info.kind) service.host = info.kind
  if (info.language) service.language = info.language
  if (info.framework) service.framework = info.framework
  if (info.projectDir) service.project = info.projectDir
  if (Object.keys(info.environment).length > 0) service.environmentVariables = { ...info.environment }
  if (local) service.local = local
  const azureInfo = azureFromProto(info.azure, azureMeta)
  if (azureInfo) service.azure = azureInfo

  // Legacy flat fields some components still read.
  service.status = statusStr
  if (healthStr === 'healthy' || healthStr === 'unhealthy' || healthStr === 'unknown') {
    service.health = healthStr
  }
  if (info.startTime) service.startTime = info.startTime.toDate().toISOString()
  if (lastChecked) service.lastChecked = lastChecked

  return service
}
