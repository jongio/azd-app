/**
 * Azure provenance detection utilities for verifying logs come from Azure-hosted environments.
 * 
 * These utilities help detect and parse Azure-specific environment markers that prove
 * logs originated from Azure rather than local development environments.
 */

/**
 * Azure Container Apps provenance structure.
 */
export interface ContainerAppsProvenance {
  provider: 'azure';
  service: 'container-apps';
  appName: string;
  revision: string;
  replicaName: string;
  environmentName: string;
  region: string;
  hostname: string;
}

/**
 * Azure Functions provenance structure.
 */
export interface FunctionsProvenance {
  provider: 'azure';
  service: 'functions';
  siteName: string;
  region: string;
  hostname: string;
  runtime: string;
  sku: string;
  instanceId: string;
}

/**
 * Azure App Service provenance structure.
 */
export interface AppServiceProvenance {
  provider: 'azure';
  service: 'app-service';
  siteName: string;
  region: string;
  hostname: string;
  sku: string;
  instanceId: string;
}

export type AzureProvenance = ContainerAppsProvenance | FunctionsProvenance | AppServiceProvenance;

/**
 * Environment variables that indicate Azure Container Apps hosting.
 */
export const CONTAINER_APPS_ENV_MARKERS = [
  'CONTAINER_APP_NAME',
  'CONTAINER_APP_REVISION',
  'CONTAINER_APP_REPLICA_NAME',
  'CONTAINER_APP_ENVIRONMENT_NAME',
  'CONTAINER_APP_ENV_DNS_SUFFIX',
  'CONTAINER_APP_HOSTNAME',
] as const;

/**
 * Environment variables that indicate Azure Functions hosting.
 */
export const FUNCTIONS_ENV_MARKERS = [
  'WEBSITE_SITE_NAME',
  'FUNCTIONS_WORKER_RUNTIME',
  'WEBSITE_HOSTNAME',
  'WEBSITE_INSTANCE_ID',
  'WEBSITE_SKU',
  'REGION_NAME',
] as const;

/**
 * Environment variables that indicate Azure App Service hosting.
 */
export const APP_SERVICE_ENV_MARKERS = [
  'WEBSITE_SITE_NAME',
  'WEBSITE_HOSTNAME',
  'WEBSITE_INSTANCE_ID',
  'WEBSITE_SKU',
  'REGION_NAME',
] as const;

/**
 * Log field patterns that indicate Azure Container Apps provenance.
 */
export const CONTAINER_APPS_LOG_PATTERNS = {
  provider: /azure_provider=azure/,
  service: /azure_service=container-apps/,
  appName: /azure_app=([^\s]+)/,
  revision: /azure_revision=([^\s]+)/,
  replica: /azure_replica=([^\s]+)/,
  env: /azure_env=([^\s]+)/,
  region: /azure_region=([^\s]+)/,
  hostname: /azure_hostname=([^\s]+)/,
} as const;

/**
 * Log field patterns that indicate Azure Functions provenance.
 */
export const FUNCTIONS_LOG_PATTERNS = {
  provider: /azure_provider=azure/,
  service: /azure_service=functions/,
  siteName: /azure_site=([^\s]+)/,
  region: /azure_region=([^\s]+)/,
  hostname: /azure_hostname=([^\s]+)/,
  runtime: /azure_runtime=([^\s]+)/,
  sku: /azure_sku=([^\s]+)/,
  instanceId: /azure_instance=([^\s]+)/,
} as const;

/**
 * Detects if a log line contains Azure Container Apps provenance markers.
 */
export function hasContainerAppsProvenance(logLine: string): boolean {
  return (
    CONTAINER_APPS_LOG_PATTERNS.provider.test(logLine) &&
    CONTAINER_APPS_LOG_PATTERNS.service.test(logLine) &&
    CONTAINER_APPS_LOG_PATTERNS.appName.test(logLine)
  );
}

/**
 * Detects if a log line contains Azure Functions provenance markers.
 */
export function hasFunctionsProvenance(logLine: string): boolean {
  return (
    FUNCTIONS_LOG_PATTERNS.provider.test(logLine) &&
    FUNCTIONS_LOG_PATTERNS.service.test(logLine) &&
    FUNCTIONS_LOG_PATTERNS.siteName.test(logLine)
  );
}

/**
 * Detects if a log line contains any Azure provenance markers.
 */
export function hasAzureProvenance(logLine: string): boolean {
  return hasContainerAppsProvenance(logLine) || hasFunctionsProvenance(logLine);
}

/**
 * Parses Container Apps provenance from a log line.
 * Returns null if the log line doesn't contain valid provenance.
 */
export function parseContainerAppsProvenance(logLine: string): ContainerAppsProvenance | null {
  if (!hasContainerAppsProvenance(logLine)) return null;

  const appNameMatch = CONTAINER_APPS_LOG_PATTERNS.appName.exec(logLine);
  const revisionMatch = CONTAINER_APPS_LOG_PATTERNS.revision.exec(logLine);
  const replicaMatch = CONTAINER_APPS_LOG_PATTERNS.replica.exec(logLine);
  const envMatch = CONTAINER_APPS_LOG_PATTERNS.env.exec(logLine);
  const regionMatch = CONTAINER_APPS_LOG_PATTERNS.region.exec(logLine);
  const hostnameMatch = CONTAINER_APPS_LOG_PATTERNS.hostname.exec(logLine);

  return {
    provider: 'azure',
    service: 'container-apps',
    appName: appNameMatch?.[1] ?? 'unknown',
    revision: revisionMatch?.[1] ?? 'unknown',
    replicaName: replicaMatch?.[1] ?? 'unknown',
    environmentName: envMatch?.[1] ?? 'unknown',
    region: regionMatch?.[1] ?? 'unknown',
    hostname: hostnameMatch?.[1] ?? 'unknown',
  };
}

/**
 * Parses Functions provenance from a log line.
 * Returns null if the log line doesn't contain valid provenance.
 */
export function parseFunctionsProvenance(logLine: string): FunctionsProvenance | null {
  if (!hasFunctionsProvenance(logLine)) return null;

  const siteNameMatch = FUNCTIONS_LOG_PATTERNS.siteName.exec(logLine);
  const regionMatch = FUNCTIONS_LOG_PATTERNS.region.exec(logLine);
  const hostnameMatch = FUNCTIONS_LOG_PATTERNS.hostname.exec(logLine);
  const runtimeMatch = FUNCTIONS_LOG_PATTERNS.runtime.exec(logLine);
  const skuMatch = FUNCTIONS_LOG_PATTERNS.sku.exec(logLine);
  const instanceIdMatch = FUNCTIONS_LOG_PATTERNS.instanceId.exec(logLine);

  return {
    provider: 'azure',
    service: 'functions',
    siteName: siteNameMatch?.[1] ?? 'unknown',
    region: regionMatch?.[1] ?? 'unknown',
    hostname: hostnameMatch?.[1] ?? 'unknown',
    runtime: runtimeMatch?.[1] ?? 'unknown',
    sku: skuMatch?.[1] ?? 'unknown',
    instanceId: instanceIdMatch?.[1] ?? 'unknown',
  };
}

/**
 * Parses any Azure provenance from a log line.
 * Returns null if the log line doesn't contain valid provenance.
 */
export function parseAzureProvenance(logLine: string): AzureProvenance | null {
  return parseContainerAppsProvenance(logLine) ?? parseFunctionsProvenance(logLine);
}

/**
 * Validates that a log line contains the expected public endpoint information.
 */
export function hasPublicEndpointInfo(logLine: string): boolean {
  return /endpoint=https?:\/\/[^\s]+/.test(logLine) || /Public endpoint:/.test(logLine);
}

/**
 * Extracts the endpoint URL from a log line.
 */
export function extractEndpoint(logLine: string): string | null {
  const match = /endpoint=(https?:\/\/[^\s]+)/.exec(logLine);
  return match?.[1] ?? null;
}

/**
 * Extracts the HTTP method from a log line.
 */
export function extractMethod(logLine: string): string | null {
  const match = /method=(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)/i.exec(logLine);
  return match?.[1]?.toUpperCase() ?? null;
}

/**
 * Extracts the route from a log line.
 */
export function extractRoute(logLine: string): string | null {
  const match = /route=([^\s]+)/.exec(logLine);
  return match?.[1] ?? null;
}
