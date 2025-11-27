const http = require('http');

const PORT = 3002;
const startTime = Date.now();

// Configuration for flaky behavior
const HEALTHY_DURATION = 20000;   // 20 seconds healthy
const UNHEALTHY_DURATION = 15000; // 15 seconds unhealthy
const CYCLE_TIME = HEALTHY_DURATION + UNHEALTHY_DURATION;

let isHealthy = true;
let lastTransition = Date.now();
let cycleCount = 0;

// Logging helper with timestamps
function log(level, message, meta = {}) {
  const timestamp = new Date().toISOString();
  const uptime = ((Date.now() - startTime) / 1000).toFixed(1);
  console.log(JSON.stringify({
    timestamp,
    level,
    service: 'flaky-service',
    uptime: `${uptime}s`,
    message,
    ...meta
  }));
}

// Simulate state transitions
function updateHealthState() {
  const elapsed = Date.now() - lastTransition;
  const previousState = isHealthy;
  
  if (isHealthy && elapsed > HEALTHY_DURATION) {
    isHealthy = false;
    lastTransition = Date.now();
    cycleCount++;
    log('error', '🔴 SERVICE DEGRADED - Database connection lost', {
      error: 'ECONNREFUSED',
      previousState: 'healthy',
      newState: 'unhealthy',
      cycle: cycleCount,
      reason: 'Simulated database connection failure'
    });
    log('error', 'Failed to connect to database at localhost:5432', {
      retries: 3,
      lastError: 'Connection refused'
    });
    log('warn', 'Service entering degraded state, requests may fail');
  } else if (!isHealthy && elapsed > UNHEALTHY_DURATION) {
    isHealthy = true;
    lastTransition = Date.now();
    log('info', '🟢 SERVICE RECOVERED - Database connection restored', {
      previousState: 'unhealthy',
      newState: 'healthy',
      cycle: cycleCount,
      recoveryTime: `${(UNHEALTHY_DURATION / 1000).toFixed(0)}s`
    });
    log('info', 'Successfully reconnected to database', {
      connectionPool: 5,
      latency: '12ms'
    });
    log('info', 'Service fully operational, resuming normal operations');
  }
}

const server = http.createServer((req, res) => {
  updateHealthState();
  
  if (req.url === '/health') {
    if (isHealthy) {
      log('info', 'Health check passed', { status: 'healthy' });
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: 'healthy',
        service: 'flaky-service',
        uptime: Date.now() - startTime,
        cycle: cycleCount,
        nextTransition: `${Math.max(0, (HEALTHY_DURATION - (Date.now() - lastTransition)) / 1000).toFixed(0)}s`
      }));
    } else {
      log('error', 'Health check FAILED', { 
        status: 'unhealthy',
        reason: 'Database connection unavailable'
      });
      res.writeHead(503, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: 'unhealthy',
        service: 'flaky-service',
        error: 'Database connection unavailable',
        cycle: cycleCount,
        expectedRecovery: `${Math.max(0, (UNHEALTHY_DURATION - (Date.now() - lastTransition)) / 1000).toFixed(0)}s`
      }));
    }
  } else if (req.url === '/') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      message: 'Flaky service', 
      healthy: isHealthy,
      cycle: cycleCount 
    }));
  } else {
    res.writeHead(404);
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  log('info', `Flaky service started on port ${PORT}`, { port: PORT });
  log('info', 'Service will cycle between healthy/unhealthy states', {
    healthyDuration: `${HEALTHY_DURATION / 1000}s`,
    unhealthyDuration: `${UNHEALTHY_DURATION / 1000}s`
  });
  
  // Periodic state check
  setInterval(() => {
    updateHealthState();
    const stateEmoji = isHealthy ? '🟢' : '🔴';
    const timeInState = ((Date.now() - lastTransition) / 1000).toFixed(0);
    log(isHealthy ? 'info' : 'warn', `${stateEmoji} Current state: ${isHealthy ? 'healthy' : 'unhealthy'}`, {
      timeInState: `${timeInState}s`,
      cycle: cycleCount
    });
  }, 10000);
});

process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down');
  server.close(() => process.exit(0));
});
