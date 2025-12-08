const express = require('express');
const app = express();
const port = 3000;

let startTime = Date.now();
let requestCount = 0;

// Health endpoint
app.get('/health', (req, res) => {
  requestCount++;
  const uptime = Math.floor((Date.now() - startTime) / 1000);
  
  res.json({
    status: 'healthy',
    service: 'web',
    version: '1.0.0',
    uptime: `${uptime}s`,
    requestCount: requestCount,
    timestamp: new Date().toISOString()
  });
});

// Main endpoint
app.get('/', (req, res) => {
  res.send('Web Service - Health Monitoring Test');
});

app.listen(port, () => {
  console.log(`✅ Web service listening on port ${port}`);
  console.log(`   Health endpoint: http://localhost:${port}/health`);
  console.log(`   Started at: ${new Date().toISOString()}`);
});
