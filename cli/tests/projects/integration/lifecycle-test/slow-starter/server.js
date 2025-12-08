const http = require('http');

const PORT = 3003;
const startTime = Date.now();

// Warmup configuration
const WARMUP_TIME = 25000; // 25 seconds before becoming healthy
let warmupComplete = false;
let warmupProgress = 0;

// Logging helper with timestamps
function log(level, message, meta = {}) {
  const timestamp = new Date().toISOString();
  const uptime = ((Date.now() - startTime) / 1000).toFixed(1);
  console.log(JSON.stringify({
    timestamp,
    level,
    service: 'slow-starter',
    uptime: `${uptime}s`,
    message,
    ...meta
  }));
}

// Simulate warmup tasks
const warmupTasks = [
  { name: 'Loading configuration', duration: 3000 },
  { name: 'Initializing database connection pool', duration: 5000 },
  { name: 'Loading ML models', duration: 8000 },
  { name: 'Warming up caches', duration: 5000 },
  { name: 'Running health self-checks', duration: 4000 },
];

async function performWarmup() {
  log('info', '🚀 Starting service warmup sequence...', { 
    totalTasks: warmupTasks.length,
    estimatedTime: `${WARMUP_TIME / 1000}s`
  });
  
  let completedTasks = 0;
  for (const task of warmupTasks) {
    log('info', `⏳ ${task.name}...`, { task: task.name, duration: `${task.duration / 1000}s` });
    
    await new Promise(resolve => setTimeout(resolve, task.duration));
    
    completedTasks++;
    warmupProgress = Math.round((completedTasks / warmupTasks.length) * 100);
    log('info', `✓ ${task.name} complete`, { 
      progress: `${warmupProgress}%`,
      completed: completedTasks,
      remaining: warmupTasks.length - completedTasks
    });
  }
  
  warmupComplete = true;
  log('info', '🟢 WARMUP COMPLETE - Service is now healthy and ready', {
    totalWarmupTime: `${((Date.now() - startTime) / 1000).toFixed(1)}s`,
    status: 'healthy'
  });
}

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    if (warmupComplete) {
      log('debug', 'Health check passed', { status: 'healthy' });
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: 'healthy',
        service: 'slow-starter',
        uptime: Date.now() - startTime,
        warmupComplete: true
      }));
    } else {
      log('warn', 'Health check FAILED - warmup in progress', { 
        status: 'unhealthy',
        progress: `${warmupProgress}%`
      });
      res.writeHead(503, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: 'unhealthy',
        service: 'slow-starter',
        error: 'Service is warming up',
        warmupProgress: `${warmupProgress}%`,
        estimatedReady: `${Math.max(0, (WARMUP_TIME - (Date.now() - startTime)) / 1000).toFixed(0)}s`
      }));
    }
  } else if (req.url === '/') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      message: 'Slow starter service',
      ready: warmupComplete,
      warmupProgress: `${warmupProgress}%`
    }));
  } else {
    res.writeHead(404);
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  log('info', `Slow starter service listening on port ${PORT}`, { port: PORT });
  log('warn', '⚠️ Service is NOT READY - warmup required', {
    estimatedWarmupTime: `${WARMUP_TIME / 1000}s`
  });
  
  // Start warmup process
  performWarmup();
});

process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down');
  server.close(() => process.exit(0));
});
