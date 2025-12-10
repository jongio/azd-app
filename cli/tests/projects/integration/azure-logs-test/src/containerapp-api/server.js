const express = require('express');

const app = express();
const port = process.env.PORT || 9847;
const serviceName = process.env.SERVICE_NAME || 'containerapp-api';

// Request logging middleware
app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${req.method} ${req.path} - ${serviceName}`);
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
  const count = parseInt(req.query.count) || 5;
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
function autoGenerateLogs() {
  logCounter++;
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
  console.log(`[${level}] ${message} - ${serviceName}`);
  
  // Occasionally log errors/warnings for variety
  if (logCounter % 10 === 0) {
    console.warn(`[WARN] High memory usage detected at iteration ${logCounter} - ${serviceName}`);
  }
  if (logCounter % 25 === 0) {
    console.error(`[ERROR] Transient connection timeout at iteration ${logCounter} - ${serviceName} (auto-retry succeeded)`);
  }
}

app.listen(port, () => {
  console.log(`[INFO] ${serviceName} started on port ${port}`);
  console.log(`[INFO] Health check: http://localhost:${port}/health`);
  console.log(`[INFO] Generate logs: http://localhost:${port}/generate-logs?count=10`);
  console.log(`[INFO] Auto-logging enabled - generating logs every 5 seconds`);
  
  // Start auto-logging after 2 second delay
  setTimeout(() => {
    setInterval(autoGenerateLogs, 5000);
    autoGenerateLogs(); // Generate first log immediately
  }, 2000);
});
