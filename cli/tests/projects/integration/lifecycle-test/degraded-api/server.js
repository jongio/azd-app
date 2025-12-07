const http = require('http');

const PORT = 3004;
const startTime = Date.now();

// Degradation cycle: fast → slow → degraded → critical → fast
// Each phase demonstrates a different health state
const PHASE_DURATION = 15000; // 15 seconds per phase
const phases = [
  { name: 'optimal', responseTime: 10, status: 'healthy' },     // ✅ healthy - fast response
  { name: 'normal', responseTime: 50, status: 'healthy' },      // ✅ healthy - normal response  
  { name: 'slow', responseTime: 200, status: 'healthy' },       // ✅ healthy - still within limits
  { name: 'degraded', responseTime: 800, status: 'degraded' },  // ⚠️ DEGRADED - slow but working
  { name: 'critical', responseTime: 1500, status: 'unhealthy' }, // ❌ unhealthy - too slow
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
    
    // Log phase transition with appropriate level
    if (newPhase.status === 'unhealthy') {
      log('error', `🔴 PERFORMANCE CRITICAL - Response times exceeding thresholds`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        previousStatus: previousPhase.status,
        newStatus: newPhase.status,
        responseTime: `${newPhase.responseTime}ms`
      });
    } else if (newPhase.status === 'degraded') {
      log('warn', `🟡 SERVICE DEGRADED - High latency detected`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        previousStatus: previousPhase.status,
        newStatus: newPhase.status,
        responseTime: `${newPhase.responseTime}ms`
      });
    } else if (previousPhase.status !== 'healthy' && newPhase.status === 'healthy') {
      log('info', `🟢 SERVICE RECOVERED - Response times normalized`, {
        previousPhase: previousPhase.name,
        newPhase: newPhase.name,
        previousStatus: previousPhase.status,
        newStatus: newPhase.status,
        responseTime: `${newPhase.responseTime}ms`
      });
    } else {
      log('info', `📊 Phase transition`, {
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
    // Simulate response time based on phase
    setTimeout(() => {
      // Return the phase's status directly - this triggers the correct health state
      // healthy → green, degraded → yellow, unhealthy → red
      const httpStatus = phase.status === 'unhealthy' ? 503 : 200;
      const logLevel = phase.status === 'unhealthy' ? 'error' : 
                       phase.status === 'degraded' ? 'warn' : 'info';
      
      log(logLevel, `Health check completed`, { 
        status: phase.status,
        phase: phase.name,
        responseTime: `${phase.responseTime}ms`
      });
      
      res.writeHead(httpStatus, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        status: phase.status,  // This is what the health monitor reads!
        service: 'degraded-api',
        phase: phase.name,
        responseTime: phase.responseTime,
        uptime: Date.now() - startTime
      }));
    }, phase.responseTime);
  } else if (req.url === '/') {
    setTimeout(() => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        message: 'Degraded API service',
        phase: phase.name,
        status: phase.status,
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
    const emoji = phase.status === 'healthy' ? '🟢' : 
                  phase.status === 'degraded' ? '🟡' : '🔴';
    const logLevel = phase.status === 'unhealthy' ? 'error' : 
                     phase.status === 'degraded' ? 'warn' : 'info';
    log(logLevel, `${emoji} Current phase: ${phase.name}`, {
      status: phase.status,
      responseTime: `${phase.responseTime}ms`,
      timeInPhase: `${((Date.now() - phaseStartTime) / 1000).toFixed(0)}s`
    });
  }, 10000);
});

process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down');
  server.close(() => process.exit(0));
});
