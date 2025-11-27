const http = require('http');

const PORT = 3004;
const startTime = Date.now();

// Degradation cycle: fast → slow → degraded → fast
const PHASE_DURATION = 15000; // 15 seconds per phase
const phases = [
  { name: 'optimal', responseTime: 10, healthy: true },
  { name: 'normal', responseTime: 50, healthy: true },
  { name: 'slow', responseTime: 200, healthy: true },
  { name: 'degraded', responseTime: 800, healthy: true },  // Still healthy but slow
  { name: 'critical', responseTime: 2500, healthy: false }, // Too slow, unhealthy
];

let currentPhaseIndex = 0;
let phaseStartTime = Date.now();

// Logging helper with timestamps
function log(level, message, meta = {}) {
  const timestamp = new Date().toISOString();
  const uptime = ((Date.now() - startTime) / 1000).toFixed(1);
  console.log(JSON.stringify({
    timestamp,
    level,
    service: 'degraded-api',
    uptime: `${uptime}s`,
    message,
    ...meta
  }));
}

function getCurrentPhase() {
  const elapsed = Date.now() - phaseStartTime;
  if (elapsed > PHASE_DURATION) {
    const previousPhase = phases[currentPhaseIndex];
    currentPhaseIndex = (currentPhaseIndex + 1) % phases.length;
    phaseStartTime = Date.now();
    const newPhase = phases[currentPhaseIndex];
    
    // Log phase transition
    if (previousPhase.healthy && !newPhase.healthy) {
      log('error', `🔴 PERFORMANCE CRITICAL - Response times exceeding thresholds`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        responseTime: `${newPhase.responseTime}ms`
      });
    } else if (!previousPhase.healthy && newPhase.healthy) {
      log('info', `🟢 PERFORMANCE RECOVERED - Response times normalized`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        responseTime: `${newPhase.responseTime}ms`
      });
    } else if (newPhase.responseTime > previousPhase.responseTime) {
      log('warn', `⚠️ Performance degrading - increased latency detected`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        responseTime: `${newPhase.responseTime}ms`
      });
    } else {
      log('info', `📈 Performance improving`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        responseTime: `${newPhase.responseTime}ms`
      });
    }
  }
  return phases[currentPhaseIndex];
}

const server = http.createServer((req, res) => {
  const phase = getCurrentPhase();
  
  if (req.url === '/health') {
    // Simulate response time
    setTimeout(() => {
      if (phase.healthy) {
        const level = phase.responseTime > 500 ? 'warn' : 'info';
        log(level, `Health check completed`, { 
          status: 'healthy',
          phase: phase.name,
          responseTime: `${phase.responseTime}ms`
        });
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          status: 'healthy',
          service: 'degraded-api',
          phase: phase.name,
          responseTime: phase.responseTime,
          uptime: Date.now() - startTime,
          degraded: phase.responseTime > 200
        }));
      } else {
        log('error', `Health check FAILED - response time exceeded threshold`, { 
          status: 'unhealthy',
          phase: phase.name,
          responseTime: `${phase.responseTime}ms`,
          threshold: '2000ms'
        });
        res.writeHead(503, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          status: 'unhealthy',
          service: 'degraded-api',
          error: 'Response time exceeded threshold',
          phase: phase.name,
          responseTime: phase.responseTime
        }));
      }
    }, phase.responseTime);
  } else if (req.url === '/') {
    setTimeout(() => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        message: 'Degraded API service',
        phase: phase.name,
        responseTime: phase.responseTime
      }));
    }, phase.responseTime);
  } else {
    res.writeHead(404);
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  log('info', `Degraded API started on port ${PORT}`, { port: PORT });
  log('info', 'Service will cycle through performance phases', {
    phases: phases.map(p => p.name).join(' → '),
    phaseDuration: `${PHASE_DURATION / 1000}s`
  });
  
  // Periodic status log
  setInterval(() => {
    const phase = getCurrentPhase();
    const emoji = phase.healthy ? (phase.responseTime > 200 ? '🟡' : '🟢') : '🔴';
    log(phase.healthy ? 'info' : 'error', `${emoji} Current phase: ${phase.name}`, {
      responseTime: `${phase.responseTime}ms`,
      healthy: phase.healthy,
      timeInPhase: `${((Date.now() - phaseStartTime) / 1000).toFixed(0)}s`
    });
  }, 10000);
});

process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down');
  server.close(() => process.exit(0));
});
