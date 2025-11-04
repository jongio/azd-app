# Auth Server Example - Complete Setup Guide

This guide walks you through setting up and running the authentication server example from scratch.

## Prerequisites

Before you begin, ensure you have the following installed:

1. **Docker** (or Podman)
   ```bash
   docker --version  # Should be 20.0.0 or later
   ```

2. **Docker Compose**
   ```bash
   docker compose version  # Should be 2.0.0 or later
   ```

3. **Azure Developer CLI (azd)**
   ```bash
   azd version  # Should be 1.0.0 or later
   ```

4. **Azure CLI** (optional, for verification)
   ```bash
   az --version
   ```

5. **An active Azure subscription**

## Step-by-Step Setup

### Step 1: Install azd app Extension

First, install the azd app extension that includes the auth server:

```bash
# Add the extension registry
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app/main/registry.json

# Install the extension
azd extension install app

# Verify installation
azd app version
```

### Step 2: Authenticate with Azure

Authenticate with Azure using azd:

```bash
# Login to Azure
azd auth login

# Verify you're logged in
azd auth login --check-status

# Optional: Set default subscription
az account set --subscription "Your Subscription Name"
```

### Step 3: Generate a Secure Secret

Generate a secure random secret for authentication:

```bash
# On Linux/macOS
export AZD_AUTH_SECRET=$(openssl rand -base64 32)

# On Windows (PowerShell)
$env:AZD_AUTH_SECRET = [Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))

# Verify secret is set
echo $AZD_AUTH_SECRET
```

**Important:** Save this secret securely. All containers must use the same secret.

### Step 4: Navigate to the Example Directory

```bash
cd cli/tests/projects/auth-example
```

### Step 5: Start the Services

Start all services using Docker Compose:

```bash
# Start in detached mode
docker compose up -d

# Or start with logs visible
docker compose up
```

This will start three containers:
1. **auth-server**: The authentication server (port 8080, internal)
2. **api**: The Python FastAPI backend (port 8000)
3. **frontend**: The Node.js frontend (port 3000)

### Step 6: Verify Services are Running

Check that all services are healthy:

```bash
# Check container status
docker compose ps

# View logs
docker compose logs auth-server
docker compose logs api
docker compose logs frontend
```

Expected output:
```
NAME                    IMAGE                    STATUS      PORTS
auth-example-auth-server   auth-example-auth-server   Up (healthy)  8080/tcp
auth-example-api          auth-example-api          Up          0.0.0.0:8000->8000/tcp
auth-example-frontend     auth-example-frontend     Up          0.0.0.0:3000->3000/tcp
```

### Step 7: Test the Authentication Flow

#### Option A: Use the Web UI

1. Open your browser to http://localhost:3000

2. You should see the "Auth Server Demo" page

3. Click **"Test Authentication"** to verify the auth flow works

4. Click **"List Subscriptions"** to fetch your Azure subscriptions

5. Click **"List Storage Accounts"** to see storage accounts in your subscription

#### Option B: Use curl to Test the API

Test the auth endpoint directly:

```bash
# Test authentication
curl http://localhost:8000/api/auth/test | jq

# Expected response:
# {
#   "success": true,
#   "message": "Successfully authenticated with auth server",
#   "expires_on": 1234567890,
#   "token_length": 500
# }

# List subscriptions
curl http://localhost:8000/api/subscriptions | jq

# Expected response:
# {
#   "count": 1,
#   "subscriptions": [
#     {
#       "id": "12345678-1234-1234-1234-123456789012",
#       "name": "My Subscription",
#       "state": "Enabled",
#       "tenantId": "87654321-4321-4321-4321-210987654321"
#     }
#   ],
#   "authMethod": "AuthServerCredential"
# }
```

#### Option C: Test the Auth Server Directly

Test the auth server's health endpoint:

```bash
# From your host machine (auth server is internal only, so exec into a container)
docker compose exec api curl http://auth-server:8080/health

# Expected response:
# {
#   "status": "healthy",
#   "version": "1.0.0"
# }
```

## Understanding the Flow

Let's trace through what happens when you click "List Subscriptions":

### 1. Frontend → API Request

Browser sends request to frontend:
```
GET http://localhost:3000/api/subscriptions
```

Frontend proxies to API:
```
GET http://api:8000/api/subscriptions
```

### 2. API → Auth Server Request

API's `AuthServerCredential` requests a token:

```python
# In api/auth_credential.py
token = credential.get_token("https://management.azure.com/.default")
```

This makes an HTTP request:
```
GET http://auth-server:8080/token?scope=https://management.azure.com/.default
Authorization: ******
```

### 3. Auth Server → Azure

Auth server uses your azd credentials:
- Reads `~/.azd/` credentials (mounted as volume)
- Uses `DefaultAzureCredential` from Azure SDK
- Requests token from Azure AD
- Wraps token in a JWT
- Returns to API

Response:
```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "https://management.azure.com/.default"
}
```

### 4. API → Azure Management API

API uses the token with Azure SDK:

```python
from azure.mgmt.resource import SubscriptionClient

client = SubscriptionClient(credential)
subscriptions = list(client.subscriptions.list())
```

### 5. Response Chain

Data flows back:
```
Azure → API → Frontend → Browser
```

## Troubleshooting

### Issue: "Credential not initialized"

**Cause:** Missing environment variables

**Solution:**
```bash
# Check environment variables are set
docker compose exec api env | grep AUTH

# Should see:
# AUTH_SERVER_URL=http://auth-server:8080
# AZD_AUTH_SECRET=your-secret

# If not, restart with correct environment
export AZD_AUTH_SECRET=your-secret
docker compose down
docker compose up -d
```

### Issue: "Authentication failed"

**Cause:** Secret mismatch or azd not logged in

**Solution:**
```bash
# Verify azd login
azd auth login --check-status

# Check secrets match
docker compose exec auth-server printenv AZD_AUTH_SECRET
docker compose exec api printenv AZD_AUTH_SECRET

# They should be identical
```

### Issue: "Failed to fetch token from auth server"

**Cause:** Network connectivity or auth server not running

**Solution:**
```bash
# Check auth server health
docker compose exec api curl http://auth-server:8080/health

# Check auth server logs
docker compose logs auth-server

# Restart auth server
docker compose restart auth-server
```

### Issue: "No subscriptions found"

**Cause:** Azure credentials don't have access to any subscriptions

**Solution:**
```bash
# Verify your Azure access
az account list

# Re-login if needed
azd auth login
az login

# Restart services
docker compose restart
```

### Issue: Port conflicts

**Cause:** Ports 3000, 8000, or 8080 already in use

**Solution:**
```bash
# Check what's using the ports
lsof -i :3000
lsof -i :8000

# Kill the processes or change ports in docker-compose.yml
# Edit docker-compose.yml ports section:
#   - "3001:3000"  # Frontend on 3001
#   - "8001:8000"  # API on 8001
```

## Viewing Logs

### All services
```bash
docker compose logs -f
```

### Specific service
```bash
docker compose logs -f auth-server
docker compose logs -f api
docker compose logs -f frontend
```

### Just errors
```bash
docker compose logs --tail=100 | grep -i error
```

## Stopping the Services

```bash
# Stop containers (preserves data)
docker compose stop

# Stop and remove containers
docker compose down

# Stop, remove containers, and remove volumes
docker compose down -v
```

## Next Steps

### Experiment with Different Scopes

Modify `api/main.py` to request different token scopes:

```python
# For Azure Storage
token = credential.get_token("https://storage.azure.com/.default")

# For Key Vault
token = credential.get_token("https://vault.azure.net/.default")

# For Microsoft Graph
token = credential.get_token("https://graph.microsoft.com/.default")
```

### Add TLS Support

Generate self-signed certificates:

```bash
# Generate certificates
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Update docker-compose.yml
# auth-server:
#   command: ./bin/app auth server start --tls --cert /certs/cert.pem --key /certs/key.pem
#   volumes:
#     - ./certs:/certs:ro
```

### Deploy to Kubernetes

See the Kubernetes examples in the [main documentation](../../../../docs/auth-server.md#kubernetes-example).

### Implement in Other Languages

The same pattern works in any language with Azure SDK support:

- **.NET**: Implement `TokenCredential` interface
- **Java**: Implement `TokenCredential` interface  
- **JavaScript**: Implement credential for `@azure/identity`
- **Go**: Implement `azcore.TokenCredential`

## Understanding the Custom Credential

The `AuthServerCredential` in `api/auth_credential.py` is a standard Azure SDK credential that:

1. **Implements `TokenCredential` interface** - Works with all Azure SDKs
2. **Fetches tokens from auth server** - Instead of directly from Azure
3. **Caches tokens locally** - Reduces requests to auth server
4. **Handles expiration** - Automatically refreshes before expiration
5. **Uses standard Azure SDK types** - Returns `AccessToken` objects

This means you can use it anywhere you would use `DefaultAzureCredential`:

```python
# Before (requires Azure credentials in container)
from azure.identity import DefaultAzureCredential
credential = DefaultAzureCredential()

# After (uses auth server)
from auth_credential import AuthServerCredential
credential = AuthServerCredential()

# Both work exactly the same with Azure SDKs!
client = SomeAzureClient(credential)
```

## Security Best Practices

1. **Never commit secrets** - Use environment variables or Azure Key Vault
2. **Use TLS in production** - Enable `--tls` with proper certificates
3. **Rotate secrets regularly** - Update `AZD_AUTH_SECRET` periodically
4. **Limit network access** - Bind auth server to internal network only
5. **Monitor token usage** - Check logs for unusual patterns
6. **Use short-lived tokens** - Default 15 minutes is recommended

## Support and Feedback

- **Issues**: https://github.com/jongio/azd-app/issues
- **Documentation**: See `../../../docs/auth-server.md`
- **Questions**: Open a GitHub Discussion

## Summary

You now have a working example of:
- ✅ Auth server distributing Azure tokens
- ✅ Custom credential that works with Azure SDKs
- ✅ Multi-container application with secure authentication
- ✅ No credential duplication across containers
- ✅ Standard Azure SDK integration

The same pattern can be used in production environments with proper security hardening!
