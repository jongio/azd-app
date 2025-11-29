const net = require('net');
const port = 5432;

let startTime = Date.now();
let connectionCount = 0;

// Create TCP server (simulates database)
const server = net.createServer((socket) => {
  connectionCount++;
  console.log(`   Connection #${connectionCount} from ${socket.remoteAddress}:${socket.remotePort}`);
  
  // Send simulated PostgreSQL startup message
  socket.write('Database connection established\n');
  
  // Close connection after a short delay
  setTimeout(() => {
    socket.end();
  }, 100);
});

server.listen(port, () => {
  console.log(`✅ Database service (TCP) listening on port ${port}`);
  console.log(`   Simulating PostgreSQL TCP protocol`);
  console.log(`   No HTTP endpoint - will use port check for health`);
  console.log(`   Started at: ${new Date().toISOString()}`);
});

// Handle shutdown gracefully
process.on('SIGINT', () => {
  const uptime = Math.floor((Date.now() - startTime) / 1000);
  console.log(`\n   Database service shutting down after ${uptime}s`);
  console.log(`   Total connections served: ${connectionCount}`);
  server.close(() => {
    process.exit(0);
  });
});
