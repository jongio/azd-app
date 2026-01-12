# Configuring CORS with Custom URLs

## Overview

When you configure custom URLs for a service using `local.url` or `azure.url` in `azure.yaml`, you may need to add these URL origins to your service's CORS (Cross-Origin Resource Sharing) configuration. This allows your frontend application to make API requests through the custom URLs.

> 💡 **New to URL configuration?** See [Service URL Configuration](../cli/docs/schema/azure.yaml.md#service-url-configuration) for complete documentation on configuring custom URLs.

## What is a Custom URL?

A custom URL allows you to access a service through a different endpoint than its default system-generated URL. Common use cases include:

- **Reverse proxies** (nginx, Caddy)
- **Tunneling services** (ngrok, Cloudflare Tunnel, localtunnel)
- **Custom domains** (local DNS configuration or Azure custom domains)
- **Load balancers**
- **CDN endpoints** (Azure Front Door, Cloudflare)

## Configuration Example

```yaml
# azure.yaml
services:
  api:
    project: ./src/api
    language: node
    ports: ["3000"]
    local:
      url: https://api.ngrok-free.app  # ngrok tunnel for local development
    azure:
      url: https://cdn-api.myapp.example.com  # CDN override on Azure
  
  web:
    project: ./src/web
    language: typescript
    ports: ["5173"]
    local:
      url: https://web.ngrok-free.app
```

> 📖 **Learn more:** See [Service URL Configuration](../cli/docs/schema/azure.yaml.md#service-url-configuration) for detailed examples including custom domain auto-detection.

## CORS Configuration by Language

### Node.js / Express

Install the CORS package:
```bash
npm install cors
```

Configure allowed origins:
```javascript
const express = require('express');
const cors = require('cors');
const app = express();

// Define allowed origins (include both local and custom URLs)
const allowedOrigins = [
  'http://localhost:3000',           // Local development
  'http://localhost:5173',           // Vite dev server
  'https://myapp.example.com',       // Custom URL
  'https://web.myapp.example.com'    // Another service's custom URL
].filter(Boolean);  // Remove any undefined values

// Configure CORS
app.use(cors({
  origin: function(origin, callback) {
    // Allow requests with no origin (like mobile apps or curl)
    if (!origin) return callback(null, true);
    
    if (allowedOrigins.indexOf(origin) !== -1) {
      callback(null, true);
    } else {
      callback(new Error('Not allowed by CORS'));
    }
  },
  credentials: true  // Enable if you need to send cookies
}));

// Your routes here
app.get('/api/data', (req, res) => {
  res.json({ message: 'Hello from API' });
});

app.listen(3000, () => {
  console.log('API server running on port 3000');
});
```

### Python / Flask

Install Flask-CORS:
```bash
pip install flask-cors
```

Configure allowed origins:
```python
from flask import Flask
from flask_cors import CORS

app = Flask(__name__)

# Define allowed origins
allowed_origins = [
    "http://localhost:3000",
    "http://localhost:5173",
    "https://myapp.example.com",
    "https://web.myapp.example.com"
]

# Configure CORS
CORS(app, origins=allowed_origins, supports_credentials=True)

@app.route('/api/data')
def get_data():
    return {"message": "Hello from API"}

if __name__ == '__main__':
    app.run(port=5000, debug=True)
```

### Python / FastAPI

FastAPI includes built-in CORS support:

```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI()

# Define allowed origins
allowed_origins = [
    "http://localhost:3000",
    "http://localhost:5173",
    "https://myapp.example.com",
    "https://web.myapp.example.com"
]

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=allowed_origins,
    allow_credentials=True,
    allow_methods=["*"],  # Allow all methods (GET, POST, etc.)
    allow_headers=["*"],  # Allow all headers
)

@app.get("/api/data")
async def get_data():
    return {"message": "Hello from API"}
```

### ASP.NET Core

Configure CORS in `Program.cs`:

```csharp
var builder = WebApplication.CreateBuilder(args);

// Add CORS policy
builder.Services.AddCors(options =>
{
    options.AddPolicy("AllowedOrigins", policy =>
    {
        policy.WithOrigins(
                "http://localhost:3000",
                "http://localhost:5173",
                "https://myapp.example.com",
                "https://web.myapp.example.com"
            )
            .AllowAnyMethod()
            .AllowAnyHeader()
            .AllowCredentials();
    });
});

var app = builder.Build();

// Use CORS policy
app.UseCors("AllowedOrigins");

app.MapGet("/api/data", () => new { message = "Hello from API" });

app.Run();
```

## Loading URLs from Configuration

Instead of hardcoding custom URLs, load them from your `azure.yaml` configuration:

### Node.js Example

```javascript
// config-loader.js
const fs = require('fs');
const yaml = require('js-yaml');
const path = require('path');

function loadAzureConfig() {
  try {
    const configPath = path.join(__dirname, '../../azure.yaml');
    const config = yaml.load(fs.readFileSync(configPath, 'utf8'));
    return config;
  } catch (e) {
    console.warn('Failed to load azure.yaml:', e.message);
    return null;
  }
}

function getAllowedOrigins(serviceName) {
  const config = loadAzureConfig();
  const origins = [
    'http://localhost:3000',
    'http://localhost:5173'
  ];

  if (config?.services) {
    // Add custom URLs from all services that might call this API
    Object.values(config.services).forEach(svc => {
      // Add local.url if configured
      if (svc.local?.url) {
        try {
          const url = new URL(svc.local.url);
          const origin = `${url.protocol}//${url.host}`;
          if (!origins.includes(origin)) {
            origins.push(origin);
          }
        } catch (e) {
          console.warn('Invalid local.url:', svc.local.url);
        }
      }
      
      // Add azure.url if configured
      if (svc.azure?.url) {
        try {
          const url = new URL(svc.azure.url);
          const origin = `${url.protocol}//${url.host}`;
          if (!origins.includes(origin)) {
            origins.push(origin);
          }
        } catch (e) {
          console.warn('Invalid azure.url:', svc.azure.url);
        }
      }
    });
  }

  return origins;
}

module.exports = { getAllowedOrigins };
```

Usage:
```javascript
const express = require('express');
const cors = require('cors');
const { getAllowedOrigins } = require('./config-loader');

const app = express();

app.use(cors({
  origin: getAllowedOrigins('api')
}));
```

## Important Notes

### Extract Origin Only

When using custom `url`, extract only the **origin** (protocol + host), not the full path:

```javascript
// ✓ Correct - origin only
const origin = 'https://myapp.example.com';

// ✗ Wrong - includes path
const wrong = 'https://myapp.example.com/api/v1';
```

CORS compares against the **Origin** header sent by browsers, which only includes protocol and host.

### Development vs Production

Consider using environment-based configuration:

```javascript
const allowedOrigins = process.env.NODE_ENV === 'production'
  ? [
      'https://myapp.example.com',
      'https://web.myapp.example.com'
    ]
  : [
      'http://localhost:3000',
      'http://localhost:5173',
      'https://myapp.example.com'  // Test with custom URL in dev
    ];
```

### Security Considerations

1. **Never use `origin: '*'` in production** - it allows any website to call your API
2. **Be specific** - only allow origins you control
3. **Use HTTPS** - custom URLs should use HTTPS in production
4. **Validate credentials** - if using `credentials: true`, ensure proper authentication

## Testing CORS Configuration

### Using curl

```bash
# Test CORS with custom URL origin
curl -i \
  -H "Origin: https://myapp.example.com" \
  -H "Access-Control-Request-Method: GET" \
  -X OPTIONS \
  http://localhost:3000/api/data

# Look for:
# Access-Control-Allow-Origin: https://myapp.example.com
```

### Using Browser DevTools

1. Open browser DevTools (F12)
2. Go to Console tab
3. Make a fetch request:
```javascript
fetch('http://localhost:3000/api/data', {
  method: 'GET',
  credentials: 'include'
})
.then(res => res.json())
.then(console.log)
.catch(console.error);
```

4. Check Network tab for CORS headers in the response

### Common CORS Errors

**Error:** `No 'Access-Control-Allow-Origin' header is present`
- **Solution:** Add the origin to `allowedOrigins` array

**Error:** `CORS policy: Credential is not supported if CORS header 'Access-Control-Allow-Origin' is '*'`
- **Solution:** Specify explicit origins instead of `'*'`

**Error:** `CORS policy: Request header field X is not allowed`
- **local:
      url: https://web.ngrok-free.app
    azure:
      url: https://app.example.com
  
  api:
    project: ./api
    language: node
    ports: ["3000"]
    local:
      url: https://api.ngrok-free.app
    azure:
      url: https://api.example.com
```

API CORS configuration:
```javascript
// api/server.js
const express = require('express');
const cors = require('cors');

const allowedOrigins = [
  'http://localhost:5173',           // Local frontend
  'https://web.ngrok-free.app',      // Frontend ngrok tunnel
  'https://app.example.com'          // Frontend custom domain on Azure
];

app.use(cors({ origin: allowedOrigins }));
```

### ngrok Tunnel

When using ngrok for testing:

```bash
# Start ngrok tunnel
ngrok http 3000
# Forwarding: https://abc123.ngrok-free.app -> http://localhost:3000
```

Update `azure.yaml`:
```yaml
services:
  api:
    project: ./api
    language: node
    ports: ["3000"]
    local:
      url: https://abc123.ngrok-free.app
```

Add to CORS:
```javascript
const allowedOrigins = [
  'http://localhost:3000',
  'https://abc123.ngrok-free.app
```yaml
services:
  api:
    project: ./api
    local:
      url: https://abc123.ngrok.io
```

Add to CORS:
```javascript
const allowedOrigins = [
  'http://localhost:3000',
  'https://abc123.ngrok.io'  // ngrok URL
];
```

## Automated CORS Configuration

### Using CORS Helper Utility (Go-based services)

For Go-based backend services, `azd app` provides a built-in CORS helper utility that automatically extracts all origins from service configuration.

**Usage:**

```go
import (
    "github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// Get service info
services, err := serviceinfo.GetServiceInfo(projectDir)
if err != nil {
    log.Fatal(err)
}

// Option 1: Get CORS origins for a specific service
for _, svc := range services {
    if svc.Name == "api" {
        origins := serviceinfo.CORSOrigins(svc)
        // origins = ["http://localhost:3000", "https://myapp.example.com", ...]
    }
}

// Option 2: Get all CORS origins from all services
allOrigins := serviceinfo.AllCORSOrigins(services)
// Useful for backend services that need to accept requests from all frontends
```

**Environment Variable Injection:**

The helper also provides an environment variable formatter for easy Docker integration:

```go
corsEnv := serviceinfo.CORSOriginsEnvVar(svc)
// corsEnv = "http://localhost:3000,https://myapp.example.com,https://api.example.com"

// Can be injected as ALLOWED_ORIGINS env var to your service container
```

> 📖 **Learn more:** See [Service URL Configuration](../cli/docs/schema/azure.yaml.md#service-url-configuration) for details on URL precedence and custom domain auto-detection.

## See Also

- [Service URL Configuration](../cli/docs/schema/azure.yaml.md#service-url-configuration) - Complete URL configuration documentation
- [Azure.yaml Reference](../cli/docs/schema/azure.yaml.md) - Full configuration reference
- ✅ Automatically deduplicates origins
- ✅ Extracts only origin (protocol + host), not full path
- ✅ Skips invalid URLs gracefully
- ✅ Returns sorted list for consistent output

## See Also

- [azure.yaml Reference](../reference/azure-yaml.md) - Configuration options
- [Service Configuration](../guides/service-configuration.md) - Service setup
- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) - CORS specification

