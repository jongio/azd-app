const express = require('express');

const app = express();
const port = process.env.PORT || 9847;
const serviceName = process.env.SERVICE_NAME || 'containerapp-api';

function getEffectiveRequestUrl(req) {
  const forwardedHost = req.headers['x-forwarded-host'] || req.headers['x-original-host'];
  const forwardedProto = req.headers['x-forwarded-proto'];
  const host = forwardedHost || req.headers.host;
  const proto = forwardedProto || req.protocol || 'http';
  return `${proto}://${host}${req.originalUrl || req.url}`;
}

function describeEnv() {
  const entries = [
    ['WEBSITE_HOSTNAME', process.env.WEBSITE_HOSTNAME],
    ['CONTAINER_APP_NAME', process.env.CONTAINER_APP_NAME],
    ['CONTAINER_APP_REVISION', process.env.CONTAINER_APP_REVISION],
    ['PORT', process.env.PORT],
  ].filter(([, v]) => Boolean(v));

  if (entries.length === 0) return '';
  return entries.map(([k, v]) => `${k}=${v}`).join(' ');
}

/**
 * Returns true when running on Azure Container Apps (CONTAINER_APP_NAME set).
 * Local runs will not have this env var.
 */
function isAzureEnvironment() {
  return Boolean(process.env.CONTAINER_APP_NAME);
}

/**
 * Builds structured Azure provenance object for logging.
 * Returns null when not running on Azure.
 */
function buildAzureProvenance() {
  if (!isAzureEnvironment()) return null;

  // Parse region from CONTAINER_APP_ENV_DNS_SUFFIX (e.g., "eastus.azurecontainerapps.io")
  const dnsSuffix = process.env.CONTAINER_APP_ENV_DNS_SUFFIX || '';
  const region = dnsSuffix.split('.')[0] || process.env.AZURE_REGION || 'unknown';

  return {
    provider: 'azure',
    service: 'container-apps',
    appName: process.env.CONTAINER_APP_NAME,
    revision: process.env.CONTAINER_APP_REVISION || 'unknown',
    replicaName: process.env.CONTAINER_APP_REPLICA_NAME || 'unknown',
    environmentName: process.env.CONTAINER_APP_ENVIRONMENT_NAME || 'unknown',
    region,
    hostname: process.env.CONTAINER_APP_HOSTNAME || process.env.WEBSITE_HOSTNAME || 'unknown',
  };
}

/**
 * Formats Azure provenance as a log-friendly string.
 */
function formatAzureProvenance(provenance) {
  if (!provenance) return '';
  return (
    `azure_provider=${provenance.provider} ` +
    `azure_service=${provenance.service} ` +
    `azure_app=${provenance.appName} ` +
    `azure_revision=${provenance.revision} ` +
    `azure_replica=${provenance.replicaName} ` +
    `azure_env=${provenance.environmentName} ` +
    `azure_region=${provenance.region} ` +
    `azure_hostname=${provenance.hostname}`
  );
}

// Request logging middleware
app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  const azureProvenance = buildAzureProvenance();
  const azureFields = formatAzureProvenance(azureProvenance);
  console.log(
    `[${timestamp}] ${req.method} ${req.path} - ${serviceName} ` +
      `method=${req.method} route=${req.path} ` +
      `endpoint=${getEffectiveRequestUrl(req)} ` +
      `xf_host=${req.headers['x-forwarded-host']} ` +
      `xf_proto=${req.headers['x-forwarded-proto']} ` +
      `${describeEnv()}` +
      (azureFields ? ` ${azureFields}` : '')
  );
  next();
});

// Health endpoint
app.get('/health', (req, res) => {
  console.log(`[INFO] Health endpoint hit - ${serviceName} is healthy`);
  res.json({ status: 'healthy', service: serviceName, timestamp: new Date().toISOString() });
});

// Root endpoint
app.get('/', (req, res) => {
  console.log(`[INFO] Root endpoint hit - Welcome to ${serviceName}`);
  res.json({
    service: serviceName,
    host: 'containerapp',
    message: 'Azure Container Apps log streaming test service',
    timestamp: new Date().toISOString()
  });
});

// Generate logs endpoint - for testing log streaming
app.get('/generate-logs', (req, res) => {
  const count = Number.parseInt(req.query.count, 10) || 5;
  const levels = ['INFO', 'WARN', 'ERROR', 'DEBUG'];
  
  for (let i = 0; i < count; i++) {
    const level = levels[Math.floor(Math.random() * levels.length)];
    const message = `Sample log message ${i + 1} of ${count} from ${serviceName}`;
    console.log(`[${level}] ${message}`);
  }
  
  res.json({ generated: count, service: serviceName });
});

// Error simulation endpoint
app.get('/error', (req, res) => {
  console.error(`[ERROR] Simulated error in ${serviceName} - this is a test error for log streaming`);
  res.status(500).json({ error: 'Simulated error', service: serviceName });
});

// Auto-generate logs every 5 seconds for testing log streaming
let logCounter = 0;
function autoGenerateLogs(azureProvenance) {
  logCounter++;
  const azureFields = formatAzureProvenance(azureProvenance);
  const levels = ['INFO', 'INFO', 'INFO', 'WARN', 'DEBUG']; // Weight towards INFO
  const level = levels[Math.floor(Math.random() * levels.length)];
  const messages = [
    `Processing request batch #${logCounter}`,
    `Container app handling traffic - iteration ${logCounter}`,
    `API endpoint activity detected - cycle ${logCounter}`,
    `Service heartbeat #${logCounter} - all systems operational`,
    `Background task completed - run ${logCounter}`,
  ];
  const message = messages[Math.floor(Math.random() * messages.length)];
  const azureSuffix = azureFields ? ` ${azureFields}` : '';
  console.log(`[${level}] ${message} - ${serviceName}${azureSuffix}`);
  
  // Occasionally log errors/warnings for variety
  if (logCounter % 10 === 0) {
    console.warn(`[WARN] High memory usage detected at iteration ${logCounter} - ${serviceName}${azureSuffix}`);
  }
  if (logCounter % 25 === 0) {
    console.error(`[ERROR] Transient connection timeout at iteration ${logCounter} - ${serviceName} (auto-retry succeeded)${azureSuffix}`);
  }
}

app.listen(port, () => {
  const azureProvenance = buildAzureProvenance();
  const azureFields = formatAzureProvenance(azureProvenance);
  const baseUrl = azureProvenance
    ? `https://${azureProvenance.hostname}`
    : `http://localhost:${port}`;

  console.log(`[INFO] ${serviceName} started on port ${port}`);
  console.log(`[INFO] Public endpoint: GET ${baseUrl}/ (root)`);
  console.log(`[INFO] Public endpoint: GET ${baseUrl}/health (health check)`);
  console.log(`[INFO] Public endpoint: GET ${baseUrl}/generate-logs?count=N (generate logs)`);
  console.log(`[INFO] Public endpoint: GET ${baseUrl}/error (error simulation)`);

  if (azureProvenance) {
    console.log(
      `[INFO] Azure Container Apps provenance: ${azureFields}`
    );
  } else {
    console.log(`[INFO] Running locally (no Azure provenance)`);
  }

  const provenanceSuffix = azureFields ? ' with Azure provenance' : '';
  console.log(`[INFO] Auto-logging enabled - generating logs every 60 seconds${provenanceSuffix}`);

  // Start auto-logging after 2 second delay
  setTimeout(() => {
    setInterval(() => autoGenerateLogs(azureProvenance), 60000);
    autoGenerateLogs(azureProvenance); // Generate first log immediately
  }, 2000);
});
