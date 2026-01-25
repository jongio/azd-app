import http from 'http';
import fs from 'fs';
import path from 'path';

const port = process.env.PORT || 3000;

// Get all Azure and AZD environment variables
const azureEnvVars = Object.entries(process.env)
  .filter(([key]) => key.startsWith('AZURE_') || key.startsWith('AZD_'))
  .reduce((acc, [key, value]) => {
    acc[key] = value;
    return acc;
  }, {});

const envInfo = {
  envName: process.env.AZURE_ENV_NAME || 'NONE',
  testValue: process.env.TEST_ENV_VALUE || 'NOT SET',
  allVars: azureEnvVars,
  timestamp: new Date().toISOString()
};

// Write to file for verification
const outputPath = path.join(process.cwd(), 'env-loaded.json');
fs.writeFileSync(outputPath, JSON.stringify(envInfo, null, 2));

console.log('\n========================================');
console.log('🔍 Environment Variable Test');
console.log('========================================');
console.log('Environment loaded:', process.env.AZURE_ENV_NAME || 'NONE');
console.log('Test value:', process.env.TEST_ENV_VALUE || 'NOT SET');
console.log(`Wrote environment info to: ${outputPath}`);
console.log('========================================\n');

const server = http.createServer((req, res) => {
  if (req.url === '/api/env') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      envName: process.env.AZURE_ENV_NAME || 'NONE',
      testValue: process.env.TEST_ENV_VALUE || 'NOT SET',
      allAzureVars: azureEnvVars,
      timestamp: new Date().toISOString(),
    }, null, 2));
  } else {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`
<!DOCTYPE html>
<html>
<head>
  <title>Environment Override Test</title>
  <style>
    body { font-family: monospace; padding: 20px; background: #1e1e1e; color: #d4d4d4; }
    h1 { color: #4fc3f7; }
    .env-info { background: #252526; padding: 15px; border-left: 3px solid #4fc3f7; margin: 10px 0; }
    .value { color: #ce9178; }
    pre { background: #2d2d30; padding: 10px; overflow-x: auto; }
  </style>
</head>
<body>
  <h1>🔍 Environment Override Test</h1>
  <div class="env-info">
    <strong>Environment Name:</strong> <span class="value">${process.env.AZURE_ENV_NAME || 'NONE'}</span>
  </div>
  <div class="env-info">
    <strong>Test Value (should be different per env):</strong> <span class="value">${process.env.TEST_ENV_VALUE || 'NOT SET'}</span>
  </div>
  <h2>All Azure/AZD Environment Variables:</h2>
  <pre>${JSON.stringify(azureEnvVars, null, 2)}</pre>
</body>
</html>
    `);
  }
});

server.listen(port, () => {
  console.log(`🚀 Server running at http://localhost:${port}`);
  console.log(`   API endpoint: http://localhost:${port}/api/env`);
  console.log('');
});
