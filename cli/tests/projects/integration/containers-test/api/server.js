// Test API server that connects to container services
const http = require('http');
const { BlobServiceClient } = require('@azure/storage-blob');
const { CosmosClient } = require('@azure/cosmos');
const { createClient: createRedisClient } = require('redis');
const { Pool } = require('pg');

const PORT = process.env.PORT || 3000;

// Connection strings from environment
const STORAGE_CONN = process.env.AZURE_STORAGE_CONNECTION_STRING || 'UseDevelopmentStorage=true';
const COSMOS_ENDPOINT = process.env.COSMOS_ENDPOINT || 'https://localhost:8081';
const COSMOS_KEY = process.env.COSMOS_KEY || 'C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==';
const REDIS_URL = process.env.REDIS_URL || 'redis://localhost:6379';
const DATABASE_URL = process.env.DATABASE_URL || 'postgresql://postgres:postgres@localhost:5432/testdb';

// Service status tracking with detailed data
const serviceStatus = {
  azurite: { connected: false, lastOp: null, error: null, data: {} },
  cosmos: { connected: false, lastOp: null, error: null, data: {} },
  redis: { connected: false, lastOp: null, error: null, data: {} },
  postgres: { connected: false, lastOp: null, error: null, data: {} }
};

// Log environment for debugging
console.log('Environment variables:');
console.log('  AZURE_STORAGE_CONNECTION_STRING:', STORAGE_CONN ? 'set' : 'not set');
console.log('  COSMOS_ENDPOINT:', COSMOS_ENDPOINT);
console.log('  REDIS_URL:', REDIS_URL);
console.log('  DATABASE_URL:', DATABASE_URL ? 'set' : 'not set');

// ========== Azurite (Blob Storage) ==========
async function testAzurite() {
  try {
    const blobServiceClient = BlobServiceClient.fromConnectionString(STORAGE_CONN);
    const containerName = 'test-container';
    const containerClient = blobServiceClient.getContainerClient(containerName);
    
    // Create container if not exists
    const createResult = await containerClient.createIfNotExists();
    
    // Upload a test blob
    const blobName = `test-blob-${Date.now()}.txt`;
    const blockBlobClient = containerClient.getBlockBlobClient(blobName);
    const content = `Test content created at ${new Date().toISOString()}`;
    const uploadResult = await blockBlobClient.upload(content, content.length);
    
    // List all blobs with details
    const blobs = [];
    for await (const blob of containerClient.listBlobsFlat({ includeMetadata: true })) {
      blobs.push({
        name: blob.name,
        size: blob.properties.contentLength,
        created: blob.properties.createdOn,
        lastModified: blob.properties.lastModified,
        contentType: blob.properties.contentType
      });
    }
    
    // List all containers
    const containers = [];
    for await (const container of blobServiceClient.listContainers()) {
      containers.push({
        name: container.name,
        lastModified: container.properties.lastModified
      });
    }
    
    serviceStatus.azurite = {
      connected: true,
      lastOp: new Date().toISOString(),
      error: null,
      data: {
        endpoint: STORAGE_CONN.includes('azurite') ? 'http://azurite:10000' : 'http://localhost:10000',
        containerCreated: createResult.succeeded ? containerName : `${containerName} (already existed)`,
        blobUploaded: {
          name: blobName,
          content: content,
          size: content.length,
          etag: uploadResult.etag
        },
        allContainers: containers,
        allBlobs: blobs.slice(-10), // Last 10 blobs
        summary: {
          totalContainers: containers.length,
          totalBlobs: blobs.length
        }
      }
    };
  } catch (err) {
    serviceStatus.azurite = {
      connected: false,
      lastOp: new Date().toISOString(),
      error: err.message,
      data: { connectionString: STORAGE_CONN.substring(0, 50) + '...' }
    };
  }
}

// ========== Cosmos DB ==========
async function testCosmos() {
  try {
    // Cosmos emulator uses self-signed cert
    process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
    
    const client = new CosmosClient({
      endpoint: COSMOS_ENDPOINT,
      key: COSMOS_KEY
    });
    
    const dbName = 'testdb';
    const containerName = 'testcontainer';
    
    // Create database if not exists
    const { database } = await client.databases.createIfNotExists({ id: dbName });
    
    // List all databases
    const { resources: databases } = await client.databases.readAll().fetchAll();
    
    // Create container if not exists
    const { container } = await database.containers.createIfNotExists({
      id: containerName,
      partitionKey: { paths: ['/id'] }
    });
    
    // List all containers in database
    const { resources: dbContainers } = await database.containers.readAll().fetchAll();
    
    // Create a test item
    const itemId = `item-${Date.now()}`;
    const item = {
      id: itemId,
      type: 'test-record',
      message: 'Test record from container integration',
      timestamp: new Date().toISOString(),
      metadata: {
        source: 'azd-app-integration-test',
        version: '1.0'
      }
    };
    const { resource: createdItem } = await container.items.create(item);
    
    // Query all items (last 10)
    const { resources: items } = await container.items
      .query('SELECT c.id, c.type, c.message, c.timestamp FROM c ORDER BY c._ts DESC')
      .fetchAll();
    
    serviceStatus.cosmos = {
      connected: true,
      lastOp: new Date().toISOString(),
      error: null,
      data: {
        endpoint: COSMOS_ENDPOINT,
        databaseCreated: dbName,
        containerCreated: containerName,
        documentInserted: {
          id: createdItem.id,
          content: item
        },
        allDatabases: databases.map(d => d.id),
        allContainers: dbContainers.map(c => c.id),
        recentDocuments: items.slice(0, 10),
        summary: {
          totalDatabases: databases.length,
          totalContainers: dbContainers.length,
          totalDocuments: items.length
        }
      }
    };
  } catch (err) {
    serviceStatus.cosmos = {
      connected: false,
      lastOp: new Date().toISOString(),
      error: err.message,
      data: { endpoint: COSMOS_ENDPOINT }
    };
  }
}

// ========== Redis ==========
async function testRedis() {
  let redisClient = null;
  try {
    redisClient = createRedisClient({ url: REDIS_URL });
    redisClient.on('error', () => {}); // Suppress connection errors during test
    
    await redisClient.connect();
    
    // Get server info
    const info = await redisClient.info('server');
    const memoryInfo = await redisClient.info('memory');
    
    // Set a test key with expiration
    const key = `test-key-${Date.now()}`;
    const value = JSON.stringify({
      message: 'Test value from container integration',
      timestamp: new Date().toISOString(),
      source: 'azd-app'
    });
    await redisClient.set(key, value, { EX: 3600 }); // 1 hour expiration
    
    // Get the value back
    const retrieved = await redisClient.get(key);
    
    // Set a hash
    const hashKey = `test-hash-${Date.now()}`;
    await redisClient.hSet(hashKey, {
      field1: 'value1',
      field2: 'value2',
      created: new Date().toISOString()
    });
    const hashData = await redisClient.hGetAll(hashKey);
    
    // Get all test keys with their values
    const testKeys = await redisClient.keys('test-*');
    const keyValues = [];
    for (const k of testKeys.slice(-10)) { // Last 10 keys
      const type = await redisClient.type(k);
      let val;
      if (type === 'string') {
        val = await redisClient.get(k);
      } else if (type === 'hash') {
        val = await redisClient.hGetAll(k);
      }
      keyValues.push({ key: k, type, value: val });
    }
    
    // Get database size
    const dbSize = await redisClient.dbSize();
    
    serviceStatus.redis = {
      connected: true,
      lastOp: new Date().toISOString(),
      error: null,
      data: {
        endpoint: REDIS_URL,
        serverVersion: info.match(/redis_version:(\S+)/)?.[1] || 'unknown',
        stringKeyCreated: {
          key: key,
          value: JSON.parse(value),
          expiresIn: '1 hour'
        },
        hashKeyCreated: {
          key: hashKey,
          fields: hashData
        },
        recentKeys: keyValues,
        summary: {
          totalKeys: dbSize,
          testKeys: testKeys.length,
          usedMemory: memoryInfo.match(/used_memory_human:(\S+)/)?.[1] || 'unknown'
        }
      }
    };
  } catch (err) {
    serviceStatus.redis = {
      connected: false,
      lastOp: new Date().toISOString(),
      error: err.message,
      data: { endpoint: REDIS_URL }
    };
  } finally {
    if (redisClient) {
      try { await redisClient.quit(); } catch {}
    }
  }
}

// ========== PostgreSQL ==========
async function testPostgres() {
  const pool = new Pool({ connectionString: DATABASE_URL });
  try {
    // Get server version
    const versionResult = await pool.query('SELECT version()');
    
    // Create test table if not exists
    await pool.query(`
      CREATE TABLE IF NOT EXISTS test_records (
        id SERIAL PRIMARY KEY,
        message TEXT,
        metadata JSONB,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      )
    `);
    
    // Insert a test record with metadata
    const message = `Test record created at ${new Date().toISOString()}`;
    const metadata = {
      source: 'azd-app-integration-test',
      version: '1.0',
      testRun: Date.now()
    };
    const insertResult = await pool.query(
      'INSERT INTO test_records (message, metadata) VALUES ($1, $2) RETURNING *',
      [message, JSON.stringify(metadata)]
    );
    
    // Get all records (last 10)
    const recordsResult = await pool.query(
      'SELECT * FROM test_records ORDER BY created_at DESC LIMIT 10'
    );
    
    // Get table info
    const tableInfoResult = await pool.query(`
      SELECT column_name, data_type, is_nullable
      FROM information_schema.columns
      WHERE table_name = 'test_records'
      ORDER BY ordinal_position
    `);
    
    // Get all tables in database
    const tablesResult = await pool.query(`
      SELECT table_name 
      FROM information_schema.tables 
      WHERE table_schema = 'public'
    `);
    
    // Get total count
    const countResult = await pool.query('SELECT COUNT(*) FROM test_records');
    
    serviceStatus.postgres = {
      connected: true,
      lastOp: new Date().toISOString(),
      error: null,
      data: {
        endpoint: DATABASE_URL.replace(/:[^:@]+@/, ':****@'), // Hide password
        serverVersion: versionResult.rows[0].version.split(' ').slice(0, 2).join(' '),
        tableCreated: 'test_records',
        tableSchema: tableInfoResult.rows,
        recordInserted: insertResult.rows[0],
        recentRecords: recordsResult.rows,
        allTables: tablesResult.rows.map(r => r.table_name),
        summary: {
          totalTables: tablesResult.rows.length,
          totalRecords: parseInt(countResult.rows[0].count)
        }
      }
    };
  } catch (err) {
    serviceStatus.postgres = {
      connected: false,
      lastOp: new Date().toISOString(),
      error: err.message,
      data: { endpoint: DATABASE_URL.replace(/:[^:@]+@/, ':****@') }
    };
  } finally {
    await pool.end();
  }
}

// Run all tests
async function runAllTests() {
  await Promise.all([
    testAzurite(),
    testCosmos(),
    testRedis(),
    testPostgres()
  ]);
}

// Initial test run
runAllTests();

// Calculate overall status
function getOverallStatus() {
  const services = [
    serviceStatus.azurite,
    serviceStatus.cosmos,
    serviceStatus.redis,
    serviceStatus.postgres
  ];
  const connectedCount = services.filter(s => s.connected).length;
  const total = services.length;
  
  if (connectedCount === total) {
    return { status: 'healthy', color: '#22c55e', label: 'All Services Connected', icon: 'OK' };
  } else if (connectedCount >= total / 2) {
    return { status: 'degraded', color: '#eab308', label: `${connectedCount}/${total} Services Connected`, icon: 'WARN' };
  } else {
    return { status: 'unhealthy', color: '#ef4444', label: `${connectedCount}/${total} Services Connected`, icon: 'FAIL' };
  }
}

// Generate HTML status page
function generateHtmlStatus() {
  const overall = getOverallStatus();
  
  const getStatusIndicator = (connected) => {
    if (connected) {
      return '<span class="status-dot green"></span><span class="status-text green">Connected</span>';
    }
    return '<span class="status-dot red"></span><span class="status-text red">Disconnected</span>';
  };

  const formatJson = (obj) => JSON.stringify(obj, null, 2)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Container Integration Status</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #0f172a; color: #e2e8f0; }
    h1 { color: #38bdf8; margin-bottom: 10px; }
    .timestamp { color: #64748b; font-size: 14px; margin-bottom: 20px; }
    
    /* Overall Status Banner */
    .overall-status { 
      background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
      border-radius: 12px; 
      padding: 24px; 
      margin: 20px 0; 
      border: 2px solid ${overall.color};
      display: flex;
      align-items: center;
      gap: 20px;
    }
    .overall-indicator {
      width: 80px;
      height: 80px;
      border-radius: 50%;
      background: ${overall.color};
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: bold;
      font-size: 16px;
      color: #0f172a;
      box-shadow: 0 0 20px ${overall.color}40;
    }
    .overall-details { flex: 1; }
    .overall-title { font-size: 24px; font-weight: bold; color: #f1f5f9; margin-bottom: 8px; }
    .overall-subtitle { color: #94a3b8; font-size: 14px; }
    .service-badges { display: flex; gap: 12px; margin-top: 12px; flex-wrap: wrap; }
    .service-badge { 
      padding: 6px 12px; 
      border-radius: 20px; 
      font-size: 12px; 
      font-weight: 500;
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .service-badge.green { background: #22c55e20; color: #22c55e; border: 1px solid #22c55e40; }
    .service-badge.red { background: #ef444420; color: #ef4444; border: 1px solid #ef444440; }
    
    /* Status indicators */
    .status-dot { 
      display: inline-block; 
      width: 10px; 
      height: 10px; 
      border-radius: 50%; 
      margin-right: 8px;
    }
    .status-dot.green { background: #22c55e; box-shadow: 0 0 8px #22c55e80; }
    .status-dot.yellow { background: #eab308; box-shadow: 0 0 8px #eab30880; }
    .status-dot.red { background: #ef4444; box-shadow: 0 0 8px #ef444480; }
    .status-text { font-weight: 600; }
    .status-text.green { color: #22c55e; }
    .status-text.yellow { color: #eab308; }
    .status-text.red { color: #ef4444; }
    
    /* Service cards */
    .service { background: #1e293b; border-radius: 8px; padding: 20px; margin: 15px 0; border-left: 4px solid #334155; }
    .service.connected { border-left-color: #22c55e; }
    .service.disconnected { border-left-color: #ef4444; }
    .service-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
    .service-name { font-size: 18px; font-weight: bold; color: #f1f5f9; }
    .endpoint { color: #94a3b8; font-size: 13px; font-family: monospace; background: #0f172a; padding: 4px 8px; border-radius: 4px; margin-top: 8px; display: inline-block; }
    
    /* Summary grid */
    .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 10px; margin: 15px 0; }
    .summary-item { background: #334155; padding: 12px; border-radius: 6px; text-align: center; }
    .summary-value { font-size: 24px; font-weight: bold; color: #38bdf8; }
    .summary-label { font-size: 11px; color: #94a3b8; margin-top: 4px; text-transform: uppercase; }
    
    /* Sections */
    .section { margin: 15px 0; }
    .section-title { color: #cbd5e1; font-weight: 600; margin-bottom: 8px; font-size: 13px; text-transform: uppercase; letter-spacing: 0.5px; }
    pre { background: #0f172a; padding: 15px; border-radius: 6px; overflow-x: auto; font-size: 12px; border: 1px solid #334155; margin: 0; }
    .error-box { background: #7f1d1d; border: 1px solid #991b1b; color: #fecaca; padding: 15px; border-radius: 6px; }
    
    /* Actions */
    .actions { margin: 20px 0; display: flex; gap: 10px; flex-wrap: wrap; }
    .btn { background: #2563eb; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; font-size: 14px; text-decoration: none; display: inline-block; }
    .btn:hover { background: #1d4ed8; }
    .btn-secondary { background: #475569; }
    .btn-secondary:hover { background: #64748b; }
    
    /* Tables */
    table { width: 100%; border-collapse: collapse; margin: 10px 0; }
    th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #334155; }
    th { color: #94a3b8; font-weight: 500; font-size: 11px; text-transform: uppercase; }
    td { color: #e2e8f0; font-size: 13px; }
    .mono { font-family: monospace; font-size: 12px; }
  </style>
</head>
<body>
  <h1>Container Integration Status</h1>
  <div class="timestamp">Last updated: ${new Date().toISOString()}</div>
  
  <!-- Overall Status Banner -->
  <div class="overall-status">
    <div class="overall-indicator">${overall.icon}</div>
    <div class="overall-details">
      <div class="overall-title">${overall.label}</div>
      <div class="overall-subtitle">Integration test status for container services</div>
      <div class="service-badges">
        <span class="service-badge ${serviceStatus.azurite.connected ? 'green' : 'red'}">
          <span class="status-dot ${serviceStatus.azurite.connected ? 'green' : 'red'}"></span>
          Azurite
        </span>
        <span class="service-badge ${serviceStatus.cosmos.connected ? 'green' : 'red'}">
          <span class="status-dot ${serviceStatus.cosmos.connected ? 'green' : 'red'}"></span>
          Cosmos DB
        </span>
        <span class="service-badge ${serviceStatus.redis.connected ? 'green' : 'red'}">
          <span class="status-dot ${serviceStatus.redis.connected ? 'green' : 'red'}"></span>
          Redis
        </span>
        <span class="service-badge ${serviceStatus.postgres.connected ? 'green' : 'red'}">
          <span class="status-dot ${serviceStatus.postgres.connected ? 'green' : 'red'}"></span>
          PostgreSQL
        </span>
      </div>
    </div>
  </div>
  
  <div class="actions">
    <a class="btn" href="/test">Run Fresh Tests</a>
    <a class="btn btn-secondary" href="/status">JSON Status</a>
    <a class="btn btn-secondary" href="/health">Health Check</a>
  </div>

  <!-- Azurite -->
  <div class="service ${serviceStatus.azurite.connected ? 'connected' : 'disconnected'}">
    <div class="service-header">
      <span class="service-name">Azurite (Blob Storage)</span>
      ${getStatusIndicator(serviceStatus.azurite.connected)}
    </div>
    ${serviceStatus.azurite.connected ? `
      <div class="endpoint">${serviceStatus.azurite.data.endpoint || 'N/A'}</div>
      <div class="summary">
        <div class="summary-item"><div class="summary-value">${serviceStatus.azurite.data.summary?.totalContainers || 0}</div><div class="summary-label">Containers</div></div>
        <div class="summary-item"><div class="summary-value">${serviceStatus.azurite.data.summary?.totalBlobs || 0}</div><div class="summary-label">Total Blobs</div></div>
      </div>
      <div class="section">
        <div class="section-title">Last Blob Uploaded</div>
        <pre>${formatJson(serviceStatus.azurite.data.blobUploaded || {})}</pre>
      </div>
      <div class="section">
        <div class="section-title">Recent Blobs</div>
        <table>
          <tr><th>Name</th><th>Size</th><th>Last Modified</th></tr>
          ${(serviceStatus.azurite.data.allBlobs || []).map(b => `<tr><td class="mono">${b.name}</td><td>${b.size} bytes</td><td>${b.lastModified || 'N/A'}</td></tr>`).join('')}
        </table>
      </div>
    ` : `<div class="error-box">Error: ${serviceStatus.azurite.error || 'Not connected'}</div>`}
  </div>

  <!-- Cosmos DB -->
  <div class="service ${serviceStatus.cosmos.connected ? 'connected' : 'disconnected'}">
    <div class="service-header">
      <span class="service-name">Cosmos DB</span>
      ${getStatusIndicator(serviceStatus.cosmos.connected)}
    </div>
    ${serviceStatus.cosmos.connected ? `
      <div class="endpoint">${serviceStatus.cosmos.data.endpoint || 'N/A'}</div>
      <div class="summary">
        <div class="summary-item"><div class="summary-value">${serviceStatus.cosmos.data.summary?.totalDatabases || 0}</div><div class="summary-label">Databases</div></div>
        <div class="summary-item"><div class="summary-value">${serviceStatus.cosmos.data.summary?.totalContainers || 0}</div><div class="summary-label">Containers</div></div>
        <div class="summary-item"><div class="summary-value">${serviceStatus.cosmos.data.summary?.totalDocuments || 0}</div><div class="summary-label">Documents</div></div>
      </div>
      <div class="section">
        <div class="section-title">Last Document Inserted</div>
        <pre>${formatJson(serviceStatus.cosmos.data.documentInserted || {})}</pre>
      </div>
      <div class="section">
        <div class="section-title">Recent Documents</div>
        <table>
          <tr><th>ID</th><th>Type</th><th>Message</th><th>Timestamp</th></tr>
          ${(serviceStatus.cosmos.data.recentDocuments || []).map(d => `<tr><td class="mono">${d.id}</td><td>${d.type || 'N/A'}</td><td>${d.message || 'N/A'}</td><td>${d.timestamp || 'N/A'}</td></tr>`).join('')}
        </table>
      </div>
    ` : `<div class="error-box">Error: ${serviceStatus.cosmos.error || 'Not connected'}</div>`}
  </div>

  <!-- Redis -->
  <div class="service ${serviceStatus.redis.connected ? 'connected' : 'disconnected'}">
    <div class="service-header">
      <span class="service-name">Redis</span>
      ${getStatusIndicator(serviceStatus.redis.connected)}
    </div>
    ${serviceStatus.redis.connected ? `
      <div class="endpoint">${serviceStatus.redis.data.endpoint || 'N/A'} (v${serviceStatus.redis.data.serverVersion || 'unknown'})</div>
      <div class="summary">
        <div class="summary-item"><div class="summary-value">${serviceStatus.redis.data.summary?.totalKeys || 0}</div><div class="summary-label">Total Keys</div></div>
        <div class="summary-item"><div class="summary-value">${serviceStatus.redis.data.summary?.testKeys || 0}</div><div class="summary-label">Test Keys</div></div>
        <div class="summary-item"><div class="summary-value">${serviceStatus.redis.data.summary?.usedMemory || 'N/A'}</div><div class="summary-label">Memory Used</div></div>
      </div>
      <div class="section">
        <div class="section-title">Last String Key Created</div>
        <pre>${formatJson(serviceStatus.redis.data.stringKeyCreated || {})}</pre>
      </div>
      <div class="section">
        <div class="section-title">Last Hash Key Created</div>
        <pre>${formatJson(serviceStatus.redis.data.hashKeyCreated || {})}</pre>
      </div>
      <div class="section">
        <div class="section-title">Recent Keys</div>
        <table>
          <tr><th>Key</th><th>Type</th><th>Value</th></tr>
          ${(serviceStatus.redis.data.recentKeys || []).map(k => `<tr><td class="mono">${k.key}</td><td>${k.type}</td><td class="mono">${typeof k.value === 'object' ? JSON.stringify(k.value) : (k.value || '').substring(0, 50)}</td></tr>`).join('')}
        </table>
      </div>
    ` : `<div class="error-box">Error: ${serviceStatus.redis.error || 'Not connected'}</div>`}
  </div>

  <!-- PostgreSQL -->
  <div class="service ${serviceStatus.postgres.connected ? 'connected' : 'disconnected'}">
    <div class="service-header">
      <span class="service-name">PostgreSQL</span>
      ${getStatusIndicator(serviceStatus.postgres.connected)}
    </div>
    ${serviceStatus.postgres.connected ? `
      <div class="endpoint">${serviceStatus.postgres.data.serverVersion || 'N/A'}</div>
      <div class="summary">
        <div class="summary-item"><div class="summary-value">${serviceStatus.postgres.data.summary?.totalTables || 0}</div><div class="summary-label">Tables</div></div>
        <div class="summary-item"><div class="summary-value">${serviceStatus.postgres.data.summary?.totalRecords || 0}</div><div class="summary-label">Records</div></div>
      </div>
      <div class="section">
        <div class="section-title">Table Schema: ${serviceStatus.postgres.data.tableCreated}</div>
        <table>
          <tr><th>Column</th><th>Type</th><th>Nullable</th></tr>
          ${(serviceStatus.postgres.data.tableSchema || []).map(c => `<tr><td class="mono">${c.column_name}</td><td>${c.data_type}</td><td>${c.is_nullable}</td></tr>`).join('')}
        </table>
      </div>
      <div class="section">
        <div class="section-title">Last Record Inserted</div>
        <pre>${formatJson(serviceStatus.postgres.data.recordInserted || {})}</pre>
      </div>
      <div class="section">
        <div class="section-title">Recent Records</div>
        <table>
          <tr><th>ID</th><th>Message</th><th>Created At</th></tr>
          ${(serviceStatus.postgres.data.recentRecords || []).map(r => `<tr><td>${r.id}</td><td>${(r.message || '').substring(0, 40)}...</td><td>${r.created_at || 'N/A'}</td></tr>`).join('')}
        </table>
      </div>
    ` : `<div class="error-box">Error: ${serviceStatus.postgres.error || 'Not connected'}</div>`}
  </div>

  <script>
    // Auto-refresh every 30 seconds
    setTimeout(function() { location.reload(); }, 30000);
  </script>
</body>
</html>`;
}

const server = http.createServer(async (req, res) => {
  // Root path shows HTML dashboard
  if (req.url === '/' || req.url === '/dashboard') {
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(generateHtmlStatus());
    return;
  }
  
  res.setHeader('Content-Type', 'application/json');
  
  if (req.url === '/health') {
    const overall = getOverallStatus();
    res.writeHead(200);
    res.end(JSON.stringify({
      overall: {
        status: overall.status,
        label: overall.label,
        connectedServices: [
          serviceStatus.azurite.connected ? 'azurite' : null,
          serviceStatus.cosmos.connected ? 'cosmos' : null,
          serviceStatus.redis.connected ? 'redis' : null,
          serviceStatus.postgres.connected ? 'postgres' : null
        ].filter(Boolean),
        disconnectedServices: [
          !serviceStatus.azurite.connected ? 'azurite' : null,
          !serviceStatus.cosmos.connected ? 'cosmos' : null,
          !serviceStatus.redis.connected ? 'redis' : null,
          !serviceStatus.postgres.connected ? 'postgres' : null
        ].filter(Boolean)
      },
      timestamp: new Date().toISOString(),
      services: {
        azurite: { 
          status: serviceStatus.azurite.connected ? 'connected' : 'disconnected',
          blobs: serviceStatus.azurite.data?.summary?.totalBlobs || 0,
          error: serviceStatus.azurite.error
        },
        cosmos: { 
          status: serviceStatus.cosmos.connected ? 'connected' : 'disconnected',
          documents: serviceStatus.cosmos.data?.summary?.totalDocuments || 0,
          error: serviceStatus.cosmos.error
        },
        redis: { 
          status: serviceStatus.redis.connected ? 'connected' : 'disconnected',
          keys: serviceStatus.redis.data?.summary?.totalKeys || 0,
          error: serviceStatus.redis.error
        },
        postgres: { 
          status: serviceStatus.postgres.connected ? 'connected' : 'disconnected',
          records: serviceStatus.postgres.data?.summary?.totalRecords || 0,
          error: serviceStatus.postgres.error
        }
      }
    }, null, 2));
    return;
  }

  if (req.url === '/status') {
    res.writeHead(200);
    res.end(JSON.stringify({
      timestamp: new Date().toISOString(),
      services: serviceStatus
    }, null, 2));
    return;
  }

  if (req.url === '/test') {
    // Run fresh tests
    await runAllTests();
    // Redirect to dashboard to show results
    res.writeHead(302, { 'Location': '/' });
    res.end();
    return;
  }

  // API info
  res.writeHead(200);
  res.end(JSON.stringify({
    message: 'Container test API running',
    endpoints: {
      '/': 'HTML Dashboard with detailed status',
      '/dashboard': 'Same as /',
      '/health': 'JSON health check with counts',
      '/status': 'Full JSON status of all connections',
      '/test': 'Run fresh tests and redirect to dashboard'
    }
  }, null, 2));
});

server.listen(PORT, () => {
  console.log(`API server listening on port ${PORT}`);
  console.log('Endpoints:');
  console.log(`  http://localhost:${PORT}/          - HTML Dashboard`);
  console.log(`  http://localhost:${PORT}/health    - JSON health check`);
  console.log(`  http://localhost:${PORT}/status    - Full JSON status`);
  console.log(`  http://localhost:${PORT}/test      - Run fresh tests`);
});
