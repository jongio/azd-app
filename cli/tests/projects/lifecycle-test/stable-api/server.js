const http = require('http');

const PORT = 3001;
const startTime = Date.now();

// Logging helper with timestamps
function log(level, message, meta = {}) {
  const timestamp = new Date().toISOString();
  const uptime = ((Date.now() - startTime) / 1000).toFixed(1);
  console.log(JSON.stringify({
    timestamp,
    level,
    service: 'stable-api',
    uptime: `${uptime}s`,
    message,
    ...meta
  }));
}

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    log('info', 'Health check passed', { endpoint: '/health', status: 'healthy' });
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'healthy',
      service: 'stable-api',
      uptime: Date.now() - startTime,
      timestamp: new Date().toISOString()
    }));
  } else if (req.url === '/') {
    log('info', 'Root endpoint accessed');
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ message: 'Stable API is running' }));
  } else {
    log('warn', 'Unknown endpoint accessed', { path: req.url });
    res.writeHead(404);
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  log('info', `Stable API started on port ${PORT}`, { port: PORT });
  log('info', 'Service is healthy and ready to accept requests');
  
  // Periodic status logs
  setInterval(() => {
    log('info', 'Service heartbeat - all systems operational', { 
      connections: server.connections || 0,
      memoryUsage: Math.round(process.memoryUsage().heapUsed / 1024 / 1024) + 'MB'
    });
  }, 30000);
});

process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down gracefully');
  server.close(() => {
    log('info', 'Server closed');
    process.exit(0);
  });
});
