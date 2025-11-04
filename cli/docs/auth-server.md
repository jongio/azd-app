# Authentication Server for Container Networks

The `azd app auth` command group provides secure authentication for distributed applications running in containerized environments (Docker, Podman, Kubernetes, etc.).

## Overview

When running distributed applications across multiple containers, each container typically needs Azure credentials to access Azure resources. The authentication server solves this by:

- Centralizing Azure credentials in one container (auth server)
- Providing secure token distribution to client containers
- Eliminating the need to duplicate credentials across containers
- Supporting both HTTP and HTTPS communication
- Implementing rate limiting and security best practices

## Architecture

```
┌─────────────────────────────────────────┐
│   Auth Server Container (azd + ext)    │
│  ┌──────────────────────────────────┐  │
│  │  azd app auth server             │  │
│  │  - Uses azd auth login tokens    │  │
│  │  - HTTP/HTTPS token endpoint     │  │
│  │  - Token caching/refresh         │  │
│  │  - JWT signing                   │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
              ↑ HTTPS (internal network)
              │
    ┌─────────┴─────────┬─────────────┐
    │                   │             │
┌───┴────┐        ┌─────┴──┐    ┌────┴────┐
│Client 1│        │Client 2│    │Client N │
└────────┘        └────────┘    └─────────┘
```

## Quick Start

### 1. Server Setup

Start the authentication server in one container:

```bash
# Basic HTTP server (development)
azd app auth server start --secret mysecret

# Production with TLS
azd app auth server start \
  --secret mysecret \
  --tls \
  --cert /path/to/server.crt \
  --key /path/to/server.key
```

### 2. Client Usage

Fetch tokens from client containers:

```bash
# Get token for Azure Resource Manager
export TOKEN=$(azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret \
  -o json | jq -r .access_token)

# Use token with Azure CLI or SDKs
curl -H "Authorization: Bearer $TOKEN" \
  https://management.azure.com/subscriptions?api-version=2020-01-01
```

## Server Commands

### `azd app auth server start`

Start the authentication server.

**Flags:**
- `--port` - Server port (default: 8080)
- `--tls` - Enable TLS/HTTPS (default: false)
- `--cert` - TLS certificate file path
- `--key` - TLS key file path
- `--secret` - Shared secret for authentication (or use `AZD_AUTH_SECRET` env var)
- `--token-expiry` - Token expiry in seconds (default: 900 = 15 minutes)
- `--bind` - Network interface to bind to (default: 0.0.0.0)
- `--rate-limit` - Max requests per minute per client (default: 10)

**Examples:**

```bash
# Start with custom port
azd app auth server start --secret mysecret --port 9000

# Start with TLS and custom token expiry
azd app auth server start \
  --secret mysecret \
  --tls \
  --cert server.crt \
  --key server.key \
  --token-expiry 1800

# Start bound to specific interface
azd app auth server start \
  --secret mysecret \
  --bind 192.168.1.10
```

### `azd app auth server status`

Check the status of the authentication server.

```bash
azd app auth server status --server http://localhost:8080
```

## Client Commands

### `azd app auth token get`

Fetch an access token from the authentication server.

**Flags:**
- `--server` - Authentication server URL (or use `AUTH_SERVER_URL` env var)
- `--secret` - Shared secret (or use `AZD_AUTH_SECRET` env var)
- `--scope` - Token scope (default: `https://management.azure.com/.default`)
- `--health-check` - Only check server health, don't fetch token

**Examples:**

```bash
# Get token for Azure Resource Manager
azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret

# Get token for Azure Storage
azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret \
  --scope https://storage.azure.com/.default

# Health check
azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret \
  --health-check

# Use in scripts
export TOKEN=$(azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret \
  -o json | jq -r .access_token)
```

## Docker Compose Example

### Standalone Server

```yaml
version: '3.8'

services:
  auth-server:
    image: myapp/auth-server
    command: azd app auth server start --secret ${AUTH_SECRET}
    environment:
      - AZD_AUTH_SECRET=${AUTH_SECRET}
    networks:
      - app-network
    ports:
      - "8080"  # Internal only
    volumes:
      - ${HOME}/.azd:/root/.azd:ro  # Mount azd credentials

  api-service:
    image: myapp/api
    environment:
      - AUTH_SERVER_URL=http://auth-server:8080
      - AZD_AUTH_SECRET=${AUTH_SECRET}
    networks:
      - app-network
    depends_on:
      - auth-server

networks:
  app-network:
    driver: bridge
```

### Sidecar Pattern

```yaml
version: '3.8'

services:
  app-with-auth:
    image: myapp/application
    networks:
      - app-network
    
  # Auth sidecar
  auth-sidecar:
    image: myapp/auth-server
    command: azd app auth server start --secret ${AUTH_SECRET} --bind 127.0.0.1
    environment:
      - AZD_AUTH_SECRET=${AUTH_SECRET}
    network_mode: "service:app-with-auth"  # Share network namespace
    volumes:
      - ${HOME}/.azd:/root/.azd:ro

networks:
  app-network:
    driver: bridge
```

## Podman Compatibility

The authentication server works seamlessly with Podman:

```bash
# Podman Compose
podman-compose up

# Podman pod (sidecar pattern)
podman pod create --name app-pod
podman run -d --pod app-pod \
  -e AZD_AUTH_SECRET=mysecret \
  -v ~/.azd:/root/.azd:ro \
  myapp/auth-server azd app auth server start --secret mysecret

podman run -d --pod app-pod \
  -e AUTH_SERVER_URL=http://localhost:8080 \
  -e AZD_AUTH_SECRET=mysecret \
  myapp/api
```

## Kubernetes Example

### Standalone Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: auth-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: auth-server
  template:
    metadata:
      labels:
        app: auth-server
    spec:
      containers:
      - name: auth-server
        image: myapp/auth-server
        command: ["azd", "app", "auth", "server", "start"]
        env:
        - name: AZD_AUTH_SECRET
          valueFrom:
            secretKeyRef:
              name: auth-secret
              key: secret
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: azd-credentials
          mountPath: /root/.azd
          readOnly: true
      volumes:
      - name: azd-credentials
        secret:
          secretName: azd-credentials
---
apiVersion: v1
kind: Service
metadata:
  name: auth-server
spec:
  selector:
    app: auth-server
  ports:
  - port: 8080
    targetPort: 8080
```

### Sidecar Pattern

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-with-auth
spec:
  replicas: 1
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      # Main application
      - name: app
        image: myapp/application
        env:
        - name: AUTH_SERVER_URL
          value: "http://localhost:8080"
        - name: AZD_AUTH_SECRET
          valueFrom:
            secretKeyRef:
              name: auth-secret
              key: secret
      
      # Auth sidecar
      - name: auth-sidecar
        image: myapp/auth-server
        command: ["azd", "app", "auth", "server", "start", "--bind", "127.0.0.1"]
        env:
        - name: AZD_AUTH_SECRET
          valueFrom:
            secretKeyRef:
              name: auth-secret
              key: secret
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: azd-credentials
          mountPath: /root/.azd
          readOnly: true
      volumes:
      - name: azd-credentials
        secret:
          secretName: azd-credentials
```

## Security Best Practices

### 1. Network Isolation

**Standalone Mode:**
- Bind to internal network interfaces only
- Use network policies in Kubernetes
- Never expose the auth server to the public internet

```bash
# Bind to internal interface only
azd app auth server start --secret mysecret --bind 10.0.0.10
```

**Sidecar Mode:**
- Bind to localhost (127.0.0.1) only
- Containers share network namespace
- No external network access needed

```bash
# Sidecar: bind to localhost
azd app auth server start --secret mysecret --bind 127.0.0.1
```

### 2. Secret Management

**Use Environment Variables:**

```bash
# Set secret via environment variable
export AZD_AUTH_SECRET=$(openssl rand -base64 32)
azd app auth server start
```

**Use Secret Managers:**

```bash
# Docker secrets
echo "my-secret" | docker secret create auth_secret -
docker service create \
  --secret auth_secret \
  -e AZD_AUTH_SECRET_FILE=/run/secrets/auth_secret \
  myapp/auth-server

# Kubernetes secrets
kubectl create secret generic auth-secret --from-literal=secret=$(openssl rand -base64 32)
```

### 3. TLS Encryption

**Generate Self-Signed Certificates (Development):**

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
azd app auth server start --secret mysecret --tls --cert cert.pem --key key.pem
```

**Use Let's Encrypt (Production):**

```bash
certbot certonly --standalone -d auth.example.com
azd app auth server start \
  --secret mysecret \
  --tls \
  --cert /etc/letsencrypt/live/auth.example.com/fullchain.pem \
  --key /etc/letsencrypt/live/auth.example.com/privkey.pem
```

### 4. Token Scope Limiting

Request only the minimum required scope:

```bash
# Azure Storage only
azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret \
  --scope https://storage.azure.com/.default

# Azure Key Vault only
azd app auth token get \
  --server http://auth-server:8080 \
  --secret mysecret \
  --scope https://vault.azure.net/.default
```

### 5. Rate Limiting

Adjust rate limits based on your needs:

```bash
# Allow more requests for high-traffic scenarios
azd app auth server start --secret mysecret --rate-limit 50

# Stricter limits for sensitive environments
azd app auth server start --secret mysecret --rate-limit 5
```

## Troubleshooting

### Connection Refused

**Problem:** Client cannot connect to auth server.

**Solutions:**
1. Check server is running: `azd app auth token get --health-check`
2. Verify network connectivity: `ping auth-server`
3. Check firewall rules
4. Verify bind address is accessible from client

### Unauthorized Error

**Problem:** Server returns 401 Unauthorized.

**Solutions:**
1. Verify shared secret matches on server and client
2. Check environment variable `AZD_AUTH_SECRET` is set correctly
3. Ensure Authorization header is properly formatted

### Token Expired

**Problem:** Token has expired.

**Solutions:**
1. Fetch a new token (client automatically caches and refreshes)
2. Increase token expiry: `--token-expiry 1800` (30 minutes)
3. Implement automatic token refresh in your application

### Rate Limited

**Problem:** Too many requests, rate limit exceeded.

**Solutions:**
1. Implement client-side caching
2. Increase rate limit: `--rate-limit 50`
3. Use token caching to reduce requests

### Certificate Errors (TLS)

**Problem:** TLS certificate verification fails.

**Solutions:**
1. Use valid certificates from a CA
2. For self-signed certs, add to trust store or disable verification (development only)
3. Verify certificate and key files are readable

## API Reference

### Token Endpoint

**Request:**
```
GET /token?scope=<scope> HTTP/1.1
Host: auth-server:8080
Authorization: Bearer <shared-secret>
```

**Response:**
```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "https://management.azure.com/.default"
}
```

### Health Check Endpoint

**Request:**
```
GET /health HTTP/1.1
Host: auth-server:8080
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

## Implementation Modes

### Standalone Server

**Best for:**
- Multiple services need authentication
- Centralized credential management
- Existing infrastructure with service discovery

**Pros:**
- Single authentication point
- Easy to scale
- Independent lifecycle

**Cons:**
- Network dependency
- Additional service to manage
- Requires service discovery

### Sidecar Pattern

**Best for:**
- Per-service isolation
- Kubernetes/container orchestration
- No service discovery needed

**Pros:**
- No network latency (localhost)
- Automatic scaling with application
- Simplified networking

**Cons:**
- More containers to run
- Higher resource usage
- Credentials duplicated per pod/service

## Advanced Configuration

### Custom Token Expiry

```bash
# Short-lived tokens (5 minutes)
azd app auth server start --secret mysecret --token-expiry 300

# Long-lived tokens (1 hour)
azd app auth server start --secret mysecret --token-expiry 3600
```

### Multiple Scopes

```bash
# Get different tokens for different services
TOKEN_ARM=$(azd app auth token get --server http://auth:8080 --secret mysecret --scope https://management.azure.com/.default -o json | jq -r .access_token)
TOKEN_STORAGE=$(azd app auth token get --server http://auth:8080 --secret mysecret --scope https://storage.azure.com/.default -o json | jq -r .access_token)
```

### JSON Output

All commands support JSON output for scripting:

```bash
azd app auth server start --secret mysecret -o json
azd app auth token get --server http://auth:8080 --secret mysecret -o json
```

## Migration Guide

### From Shared Credentials

**Before:**
```yaml
services:
  api:
    environment:
      - AZURE_CLIENT_ID=${AZURE_CLIENT_ID}
      - AZURE_CLIENT_SECRET=${AZURE_CLIENT_SECRET}
      - AZURE_TENANT_ID=${AZURE_TENANT_ID}
```

**After:**
```yaml
services:
  auth-server:
    command: azd app auth server start --secret ${AUTH_SECRET}
    environment:
      - AZD_AUTH_SECRET=${AUTH_SECRET}
    volumes:
      - ${HOME}/.azd:/root/.azd:ro

  api:
    environment:
      - AUTH_SERVER_URL=http://auth-server:8080
      - AZD_AUTH_SECRET=${AUTH_SECRET}
```

## Additional Resources

- [Azure Developer CLI Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Podman Documentation](https://docs.podman.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
