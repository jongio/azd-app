// Test API server that connects to container services
const http = require('http');

const PORT = process.env.PORT || 3000;

// Log environment for debugging
console.log('Environment variables:');
console.log('  AZURE_STORAGE_CONNECTION_STRING:', process.env.AZURE_STORAGE_CONNECTION_STRING ? 'set' : 'not set');
console.log('  COSMOS_ENDPOINT:', process.env.COSMOS_ENDPOINT || 'not set');
console.log('  REDIS_URL:', process.env.REDIS_URL || 'not set');
console.log('  DATABASE_URL:', process.env.DATABASE_URL ? 'set' : 'not set');

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'healthy',
      services: {
        azurite: process.env.AZURE_STORAGE_CONNECTION_STRING ? 'configured' : 'missing',
        cosmos: process.env.COSMOS_ENDPOINT ? 'configured' : 'missing',
        redis: process.env.REDIS_URL ? 'configured' : 'missing',
        postgres: process.env.DATABASE_URL ? 'configured' : 'missing'
      }
    }));
    return;
  }

  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ message: 'Container test API running' }));
});

server.listen(PORT, () => {
  console.log(`API server listening on port ${PORT}`);
});
