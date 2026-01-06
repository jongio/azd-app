#!/usr/bin/env node
/**
 * Simple test web service
 * Health endpoint returns healthy to show different states
 */

const http = require('http');

const PORT = 3000;

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'healthy',
      uptime: process.uptime(),
      timestamp: new Date().toISOString()
    }));
  } else if (req.url === '/') {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<h1>Web Service</h1><p>Health endpoint is healthy</p>');
  } else {
    res.writeHead(404);
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  console.log(`Web serving at http://localhost:${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health (healthy)`);
});
