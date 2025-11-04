# Quick Reference - Auth Server Example

## Commands

### Start Everything
```bash
export AZD_AUTH_SECRET=$(openssl rand -base64 32)
azd auth login
docker compose up
```

### Open the Demo
```
http://localhost:3000
```

### Test Individual Components

#### Auth Server Health
```bash
docker compose exec api curl http://auth-server:8080/health
```

#### API Test
```bash
curl http://localhost:8000/api/auth/test | jq
curl http://localhost:8000/api/subscriptions | jq
```

#### Frontend
```bash
open http://localhost:3000
```

### View Logs
```bash
docker compose logs -f auth-server
docker compose logs -f api
docker compose logs -f frontend
```

### Cleanup
```bash
docker compose down
```

## Architecture Flow

```
Browser → Frontend (port 3000)
    ↓
  API (port 8000) uses AuthServerCredential
    ↓
  Auth Server (port 8080) 
    ↓
  Azure AD (gets real Azure token)
    ↓
  Returns JWT-wrapped token
    ↓
  API uses token → Azure Management API
    ↓
  Results → Frontend → Browser
```

## Key Files

- `api/auth_credential.py` - **Custom Azure SDK credential**
- `api/main.py` - FastAPI backend using the credential
- `frontend/public/index.html` - Web UI
- `docker-compose.yml` - Container orchestration
- `SETUP.md` - **Full setup guide** (start here!)

## Common Issues

### Secret not set
```bash
export AZD_AUTH_SECRET=$(openssl rand -base64 32)
docker compose restart
```

### Not logged into Azure
```bash
azd auth login
docker compose restart auth-server
```

### Port conflicts
Edit `docker-compose.yml` ports:
```yaml
ports:
  - "3001:3000"  # Use different host port
```

## What Makes This Special?

1. **Standard Azure SDK** - Uses TokenCredential interface
2. **No Custom Code** - Works with any Azure SDK client
3. **Language Agnostic** - Same pattern works in .NET, Java, JS, Go
4. **No Credential Duplication** - One auth source for all containers
5. **Automatic Caching** - Tokens cached at server and client
6. **Drop-in Replacement** - Replace DefaultAzureCredential

## Extending

### Add a new service
```yaml
services:
  worker:
    build: ./worker
    environment:
      - AUTH_SERVER_URL=http://auth-server:8080
      - AZD_AUTH_SECRET=${AZD_AUTH_SECRET}
```

### Use different Azure services
```python
from azure.mgmt.storage import StorageManagementClient
from auth_credential import AuthServerCredential

credential = AuthServerCredential()
client = StorageManagementClient(credential, subscription_id)
```

### Add TLS
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

Update docker-compose.yml:
```yaml
auth-server:
  command: ./bin/app auth server start --tls --cert /certs/cert.pem --key /certs/key.pem
  volumes:
    - ./certs:/certs:ro
```

## See Full Documentation

- **Setup Guide**: `SETUP.md` - Complete step-by-step instructions
- **Auth Server Docs**: `../../../docs/auth-server.md` - Full auth server documentation
- **Project README**: `README.md` - Architecture and how it works
