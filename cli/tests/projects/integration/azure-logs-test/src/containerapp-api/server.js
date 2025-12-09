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
  res.json({ status: 'healthy', service: serviceName, timestamp: new Date().toISOString() });
});

// Root endpoint
app.get('/', (req, res) => {
  console.log(`[INFO] Root endpoint accessed - generating sample logs`);
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

app.listen(port, () => {
  console.log(`[INFO] ${serviceName} started on port ${port}`);
  console.log(`[INFO] Health check: http://localhost:${port}/health`);
  console.log(`[INFO] Generate logs: http://localhost:${port}/generate-logs?count=10`);
});
