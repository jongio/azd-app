# Auth Server Example - End-to-End Demo

This is a complete end-to-end example demonstrating the authentication server for secure container-to-container communication.

## Architecture

```
┌─────────────┐     HTTP      ┌─────────────┐
│  Frontend   │──────────────>│   API       │
│  (Node.js)  │               │  (Python)   │
└─────────────┘               └──────┬──────┘
                                     │ Get Token
                                     ↓
                              ┌─────────────┐
                              │ Auth Server │
                              │  (azd app)  │
                              └──────┬──────┘
                                     │ Azure Token
                                     ↓
                              ┌─────────────┐
                              │   Azure     │
                              │  Services   │
                              └─────────────┘
```

## Components

1. **Frontend** (Node.js/Express): Simple web UI that calls the API
2. **API** (Python/FastAPI): Backend service that uses Azure SDK with custom credential
3. **Auth Server**: Distributes Azure tokens to the API container
4. **Custom Credential**: Python implementation of Azure SDK `TokenCredential` that fetches from auth server

## Prerequisites

- Docker and Docker Compose (or Podman)
- Azure subscription with active `azd auth login` session
- `azd app` extension installed

## Quick Start

### 1. Authenticate with Azure

```bash
azd auth login
```

### 2. Set the shared secret

```bash
export AZD_AUTH_SECRET="your-secret-key-here"
# Or use a secure random secret
export AZD_AUTH_SECRET=$(openssl rand -base64 32)
```

### 3. Start the services

```bash
docker compose up
```

### 4. Access the application

Open your browser to http://localhost:3000

The frontend will call the API, which will:
1. Use the custom credential to fetch a token from the auth server
2. Use that token to list Azure subscriptions
3. Return the data to the frontend

## Project Structure

```
auth-example/
├── README.md                    # This file
├── docker-compose.yml           # Docker Compose configuration
├── azure.yaml                   # Azure Developer CLI configuration
├── frontend/                    # Frontend application
│   ├── Dockerfile
│   ├── package.json
│   └── server.js
├── api/                        # Backend API
│   ├── Dockerfile
│   ├── requirements.txt
│   ├── main.py                 # FastAPI application
│   └── auth_credential.py      # Custom TokenCredential implementation
└── .env.example                # Environment variables template
```

## How It Works

### Custom Credential (`auth_credential.py`)

The `AuthServerCredential` class implements Azure SDK's `TokenCredential` interface:

```python
class AuthServerCredential:
    def get_token(self, *scopes, **kwargs):
        # Fetch token from auth server
        response = requests.get(
            f"{self.server_url}/token",
            headers={"Authorization": f"******"},
            params={"scope": scopes[0]}
        )
        # Return AccessToken that Azure SDK expects
        return AccessToken(token, expires_on)
```

This credential can be used with **any** Azure SDK client:

```python
from azure.identity import AuthServerCredential
from azure.mgmt.resource import SubscriptionClient

credential = AuthServerCredential(
    server_url=os.environ["AUTH_SERVER_URL"],
    secret=os.environ["AZD_AUTH_SECRET"]
)

# Works with any Azure SDK client!
client = SubscriptionClient(credential)
subscriptions = client.subscriptions.list()
```

### Benefits of This Approach

1. **Language Agnostic**: Works with Azure SDKs in any language (Python, .NET, Java, JavaScript)
2. **No Code Changes**: Drop-in replacement for `DefaultAzureCredential`
3. **Standard Interface**: Implements the same `TokenCredential` interface
4. **Automatic Caching**: Both server and credential cache tokens
5. **Automatic Refresh**: Tokens are refreshed before expiration

## Security Notes

- **Never commit secrets**: Use environment variables or secret managers
- **Use TLS in production**: Enable `--tls` flag with proper certificates
- **Network isolation**: Bind auth server to internal network only
- **Rotate secrets regularly**: Update `AZD_AUTH_SECRET` periodically

## Troubleshooting

### Container can't reach auth server

```bash
# Check if auth server is running
docker compose ps

# Check logs
docker compose logs auth-server

# Test connectivity from API container
docker compose exec api curl http://auth-server:8080/health
```

### Authentication errors

```bash
# Verify you're logged in with azd
azd auth login --check-status

# Check if secret matches
echo $AZD_AUTH_SECRET
```

### Token errors

```bash
# Check API logs
docker compose logs api

# Check auth server logs
docker compose logs auth-server
```

## Extending This Example

### Adding More Services

Add another service that needs Azure credentials:

```yaml
services:
  worker:
    build: ./worker
    environment:
      - AUTH_SERVER_URL=http://auth-server:8080
      - AZD_AUTH_SECRET=${AZD_AUTH_SECRET}
    depends_on:
      - auth-server
```

### Using Different Scopes

Request tokens for different Azure services:

```python
# Storage
credential.get_token("https://storage.azure.com/.default")

# Key Vault
credential.get_token("https://vault.azure.net/.default")

# Microsoft Graph
credential.get_token("https://graph.microsoft.com/.default")
```

### Sidecar Pattern

Instead of a standalone auth server, run it as a sidecar:

```yaml
services:
  api:
    build: ./api
    environment:
      - AUTH_SERVER_URL=http://localhost:8080
    
  api-auth-sidecar:
    image: ghcr.io/jongio/azd-app:latest
    command: azd app auth server start --bind 127.0.0.1
    network_mode: "service:api"
    volumes:
      - ${HOME}/.azd:/root/.azd:ro
```

## Next Steps

- Explore the [full auth server documentation](../../../docs/auth-server.md)
- Try deploying to Kubernetes with the provided manifests
- Implement the credential in other languages (.NET, Java, JavaScript)

## Support

For issues or questions:
- [GitHub Issues](https://github.com/jongio/azd-app/issues)
- [Documentation](../../../docs/auth-server.md)
