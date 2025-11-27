// Simulated in-memory cache service (Redis-style)

const operations = ['GET', 'SET', 'DEL', 'INCR', 'EXPIRE', 'EXISTS'];
const keys = ['user:session', 'product', 'counter:views', 'cache:data', 'temp:token'];

let operationCount = 0;
let memoryUsage = 20; // Simulated memory usage percentage

function log(level, message) {
  const timestamp = new Date().toTimeString().split(' ')[0];
  const ms = new Date().getMilliseconds().toString().padStart(3, '0');
  console.log(`[${level}] ${timestamp}.${ms} - ${message}`);
  
  if (level === 'ERROR') {
    console.error(`[${level}] ${timestamp}.${ms} - ${message}`);
  }
}

function simulateOperation() {
  operationCount++;
  
  const operation = operations[Math.floor(Math.random() * operations.length)];
  const key = keys[Math.floor(Math.random() * keys.length)];
  const id = Math.floor(Math.random() * 10000);
  
  switch (operation) {
    case 'GET':
      const hit = Math.random() > 0.3;
      log('INFO', `GET ${key}:${id} (${hit ? 'hit' : 'miss'})`);
      break;
    case 'SET':
      const ttl = Math.floor(Math.random() * 7200) + 300;
      log('INFO', `SET ${key}:${id} (ttl: ${ttl}s)`);
      break;
    case 'DEL':
      log('INFO', `DEL expired_key_${id}`);
      break;
    case 'INCR':
      log('INFO', `INCR ${key}:${id} (value: ${Math.floor(Math.random() * 1000)})`);
      break;
    case 'EXPIRE':
      log('INFO', `EXPIRE ${key}:${id} ${Math.floor(Math.random() * 3600)}s`);
      break;
    case 'EXISTS':
      log('INFO', `EXISTS ${key}:${id} (${Math.random() > 0.5 ? 'true' : 'false'})`);
      break;
  }
  
  // Simulate memory growth
  if (operationCount % 20 === 0) {
    memoryUsage += Math.random() * 5;
    if (memoryUsage > 85) {
      log('WARN', `Memory usage: ${memoryUsage.toFixed(1)}% - eviction triggered`);
      memoryUsage = 65; // Simulate eviction
    } else {
      log('INFO', `Memory usage: ${memoryUsage.toFixed(1)}%`);
    }
  }
  
  // Random warnings and errors
  if (operationCount % 30 === 0) {
    log('WARN', `Slow command detected: KEYS * (${(Math.random() * 2 + 0.5).toFixed(2)}s)`);
  }
  
  if (operationCount % 50 === 0) {
    const clientIp = `192.168.1.${Math.floor(Math.random() * 255)}`;
    log('ERROR', `Connection refused from client ${clientIp} - max connections reached`);
  }
}

function main() {
  log('INFO', '💾 Cache Service starting...');
  log('INFO', 'Redis-compatible server initialized on port 6379');
  log('INFO', '0 keys loaded from disk');
  
  // Run operations at varying intervals
  setInterval(() => {
    simulateOperation();
  }, Math.random() * 1000 + 500); // 500-1500ms intervals
}

main();
