const express = require('express');
const app = express();

const API_URL = process.env.API_URL || 'http://localhost:5000';
const PORT = process.env.PORT || 5001;

// Serve HTML page
app.get('/', (req, res) => {
  res.send(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fullstack Test App</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-top: 0;
        }
        .status {
            padding: 10px;
            border-radius: 4px;
            margin: 20px 0;
        }
        .success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        button {
            background: #007bff;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
            margin: 10px 5px 10px 0;
        }
        button:hover {
            background: #0056b3;
        }
        #data {
            margin-top: 20px;
        }
        .item {
            padding: 15px;
            margin: 10px 0;
            background: #f8f9fa;
            border-left: 4px solid #007bff;
            border-radius: 4px;
        }
        .item-name {
            font-weight: bold;
            color: #333;
            margin-bottom: 5px;
        }
        .item-desc {
            color: #666;
        }
        pre {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 4px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Fullstack Test Application</h1>
        <p><strong>Web Server:</strong> Node.js on port ${PORT}</p>
        <p><strong>API Server:</strong> Python Flask on ${API_URL}</p>
        
        <div id="status"></div>
        
        <button onclick="checkHealth()">Check API Health</button>
        <button onclick="loadData()">Load Data from API</button>
        
        <div id="data"></div>
    </div>

    <script>
        async function checkHealth() {
            const statusDiv = document.getElementById('status');
            try {
                const response = await fetch('/api/health');
                const data = await response.json();
                statusDiv.className = 'status success';
                statusDiv.innerHTML = '✅ API is healthy: ' + JSON.stringify(data, null, 2);
            } catch (error) {
                statusDiv.className = 'status error';
                statusDiv.innerHTML = '❌ API is not responding: ' + error.message;
            }
        }

        async function loadData() {
            const dataDiv = document.getElementById('data');
            dataDiv.innerHTML = '<p>Loading...</p>';
            
            try {
                const response = await fetch('/api/data');
                const result = await response.json();
                
                let html = '<h2>Data from API</h2>';
                html += '<p>Found ' + result.count + ' items:</p>';
                
                result.items.forEach(item => {
                    html += \`
                        <div class="item">
                            <div class="item-name">📦 \${item.name}</div>
                            <div class="item-desc">\${item.description}</div>
                        </div>
                    \`;
                });
                
                dataDiv.innerHTML = html;
            } catch (error) {
                dataDiv.innerHTML = '<div class="status error">❌ Failed to load data: ' + error.message + '</div>';
            }
        }

        // Auto-check health on load
        checkHealth();
    </script>
</body>
</html>
  `);
});

// Proxy API requests
app.get('/api/health', async (req, res) => {
  try {
    const response = await fetch(`${API_URL}/api/health`);
    const data = await response.json();
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to reach API', message: error.message });
  }
});

app.get('/api/data', async (req, res) => {
  try {
    const response = await fetch(`${API_URL}/api/data`);
    const data = await response.json();
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to reach API', message: error.message });
  }
});

app.listen(PORT, () => {
  console.log(`🚀 Web server started on http://localhost:${PORT}`);
  console.log(`   Connecting to API at ${API_URL}`);
});
