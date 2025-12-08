const express = require('express');
const app = express();
const port = process.env.PORT || 3000;

// Extract all Azure-related environment variables
function getAzureEnv() {
    const azureVars = {};
    for (const [key, value] of Object.entries(process.env)) {
        if (key.startsWith('AZURE_') || key.startsWith('AZD_') || key.startsWith('SERVICE_')) {
            // Mask sensitive values
            if (key.includes('TOKEN') || key.includes('SECRET') || key.includes('KEY')) {
                azureVars[key] = value ? `${value.substring(0, 10)}...` : '(not set)';
            } else {
                azureVars[key] = value || '(not set)';
            }
        }
    }
    return azureVars;
}

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

// Main page showing Azure environment
app.get('/', (req, res) => {
    const azureEnv = getAzureEnv();
    const hasAzureContext = Object.keys(azureEnv).length > 0;

    let html = `
<!DOCTYPE html>
<html>
<head>
    <title>Azure Deploy Test</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #0078d4; }
        .status { padding: 10px; border-radius: 4px; margin: 20px 0; }
        .success { background: #dff6dd; border: 1px solid #5db356; }
        .warning { background: #fff4ce; border: 1px solid #ffd335; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { text-align: left; padding: 12px; border-bottom: 1px solid #ddd; }
        th { background: #f4f4f4; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: 'Consolas', monospace; }
        .empty { color: #666; font-style: italic; }
    </style>
</head>
<body>
    <h1>🚀 Azure Deploy Test</h1>
    <p>This page displays Azure environment variables to verify <code>azd</code> context inheritance.</p>
    
    <div class="status ${hasAzureContext ? 'success' : 'warning'}">
        ${hasAzureContext 
            ? `✅ Found ${Object.keys(azureEnv).length} Azure/AZD environment variables` 
            : '⚠️ No Azure/AZD environment variables found. Run with <code>azd app run</code> or deploy with <code>azd up</code>.'
        }
    </div>

    <h2>Environment Variables</h2>
    ${hasAzureContext ? `
    <table>
        <thead>
            <tr><th>Variable</th><th>Value</th></tr>
        </thead>
        <tbody>
            ${Object.entries(azureEnv).sort().map(([key, value]) => 
                `<tr><td><code>${key}</code></td><td>${value}</td></tr>`
            ).join('')}
        </tbody>
    </table>
    ` : '<p class="empty">No Azure environment variables detected.</p>'}

    <h2>Expected Variables</h2>
    <table>
        <thead>
            <tr><th>Variable</th><th>Source</th><th>Description</th></tr>
        </thead>
        <tbody>
            <tr><td><code>AZURE_ENV_NAME</code></td><td>azd env</td><td>Environment name (dev, staging, prod)</td></tr>
            <tr><td><code>AZURE_SUBSCRIPTION_ID</code></td><td>azd env</td><td>Azure subscription ID</td></tr>
            <tr><td><code>AZURE_RESOURCE_GROUP_NAME</code></td><td>azd env</td><td>Resource group name</td></tr>
            <tr><td><code>AZURE_LOCATION</code></td><td>azd env</td><td>Azure region</td></tr>
            <tr><td><code>SERVICE_WEB_URL</code></td><td>azd up</td><td>Deployed service URL</td></tr>
            <tr><td><code>AZD_SERVER</code></td><td>azd extension</td><td>gRPC server for extension API</td></tr>
            <tr><td><code>AZD_ACCESS_TOKEN</code></td><td>azd extension</td><td>JWT token for extension API</td></tr>
        </tbody>
    </table>

    <h2>Quick Commands</h2>
    <pre><code># Local development
azd app run

# Deploy to Azure
azd up

# View environment values
azd env get-values</code></pre>

    <footer style="margin-top: 40px; color: #666; font-size: 0.9em;">
        Server started at ${new Date().toISOString()} | Port: ${port}
    </footer>
</body>
</html>
`;
    res.send(html);
});

// JSON endpoint for programmatic access
app.get('/api/env', (req, res) => {
    res.json({
        azure: getAzureEnv(),
        timestamp: new Date().toISOString(),
        port: port
    });
});

app.listen(port, () => {
    console.log(`🚀 Server running at http://localhost:${port}`);
    console.log('');
    console.log('Azure Environment Variables:');
    const azureEnv = getAzureEnv();
    if (Object.keys(azureEnv).length === 0) {
        console.log('  (none detected - run with azd app run or azd up)');
    } else {
        for (const [key, value] of Object.entries(azureEnv).sort()) {
            console.log(`  ${key}: ${value}`);
        }
    }
    console.log('');
});
