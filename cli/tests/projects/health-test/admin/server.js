const express = require('express');
const app = express();
const port = 4000;

let startTime = Date.now();
const ADMIN_TOKEN = 'test-token-123';

// Middleware to check authentication
const authenticate = (req, res, next) => {
  const authHeader = req.headers['authorization'];
  
  if (!authHeader) {
    return res.status(401).json({ error: 'No authorization header' });
  }
  
  const token = authHeader.replace('Bearer ', '');
  
  if (token !== ADMIN_TOKEN) {
    return res.status(403).json({ error: 'Invalid token' });
  }
  
  next();
};

// Health response function
const getHealthResponse = () => {
  const uptime = Math.floor((Date.now() - startTime) / 1000);
  return {
    status: 'healthy',
    service: 'admin',
    version: '1.0.0',
    uptime: `${uptime}s`,
    timestamp: new Date().toISOString()
  };
};

// Public health endpoint (no auth required)
app.get('/health', (req, res) => {
  res.json(getHealthResponse());
});

// Authenticated health endpoint
app.get('/api/health', authenticate, (req, res) => {
  const response = getHealthResponse();
  response.authenticated = true;
  res.json(response);
});

// Public endpoint (no auth)
app.get('/', (req, res) => {
  res.send('Admin Service - Health Monitoring Test');
});

app.listen(port, () => {
  console.log(`✅ Admin service listening on port ${port}`);
  console.log(`   Health endpoint: http://localhost:${port}/api/health (requires auth)`);
  console.log(`   Token: Bearer ${ADMIN_TOKEN}`);
  console.log(`   Started at: ${new Date().toISOString()}`);
});
