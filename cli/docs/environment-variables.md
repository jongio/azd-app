# Environment Variables in azure.yaml

## Overview

The `azure.yaml` file supports Docker Compose-compatible environment variable configuration. You can define environment variables for your services using the `environment` field with multiple formats that match Docker Compose's flexibility.

## Supported Formats

### 1. Map Format (Recommended)

The simplest and most readable format. Recommended for most use cases.

```yaml
services:
  api:
    host: localhost
    language: python
    environment:
      NODE_ENV: production
      PORT: "3000"
      DATABASE_URL: postgresql://localhost:5432/db
```

**Benefits:**
- Clean, readable syntax
- Standard Docker Compose format
- Easy to maintain

### 2. Array of Strings Format

Compact format using `KEY=value` syntax. Useful when migrating from Docker Compose.

```yaml
services:
  web:
    host: localhost
    language: node
    environment:
      - NODE_ENV=production
      - API_URL=http://localhost:5000
      - PORT=3000
      - DEBUG=true
```

**Benefits:**
- Compatible with Docker Compose
- Compact representation
- Easy to copy from existing docker-compose.yml

**Notes:**
- Values after the first `=` are preserved (useful for connection strings with `=` in them)
- Variables without values (e.g., `VARIABLE` instead of `VARIABLE=value`) are set to empty strings

### 3. Array of Objects Format

Object-based format with explicit `name` and `value` fields.

```yaml
services:
  worker:
    host: localhost
    language: python
    environment:
      - name: QUEUE_URL
        value: redis://localhost:6379
      - name: LOG_LEVEL
        value: info
      - name: API_KEY
        secret: super-secret-key
```

**When to use:**
- When you need explicit secret handling (see Secrets section)

## Variable Substitution

All formats support variable substitution using `${VAR}` syntax:

```yaml
services:
  api:
    environment:
      DB_HOST: localhost
      DB_PORT: "5432"
      DATABASE_URL: postgresql://${DB_HOST}:${DB_PORT}/mydb
```

Variables are resolved from:
1. Azure environment (from `azd env`)
2. Service-specific variables defined earlier
3. OS environment variables

## Environment Variable Priority

When your service starts, environment variables are merged with the following priority (highest to lowest):

1. **Service-specific env** (from `azure.yaml`)
2. `.env` file (if using `--env-file`)
3. Azure environment (from `azd env`)
4. Auto-generated service URLs
5. OS environment

Example:

```yaml
# azure.yaml
services:
  api:
    environment:
      PORT: "8080"           # Highest priority
      DATABASE_URL: postgresql://localhost:5432/db
```

```bash
# .env file
PORT=3000                     # Overridden by azure.yaml
LOG_LEVEL=debug              # Used if not in azure.yaml
```

## Special Characters and Escaping

### Map Format
No escaping needed for most special characters:

```yaml
environment:
  DATABASE_URL: "postgresql://user:p@ssw0rd@localhost:5432/db"
  API_KEY: "abc123!@#$%^&*()"
  SPECIAL: "quotes 'and' \"escapes\""
```

### Array of Strings Format
Values after `=` are preserved as-is:

```yaml
env:
  - CONNECTION_STRING=Server=localhost;Password=abc=123
  - SPECIAL_CHARS=!@#$%^&*()
```

## Complete Examples

### Full Stack Application

```yaml
name: fullstack-app

services:
  # Frontend - Map format
  web:
    language: node
    host: containerapp
    project: ./web
    ports:
      - "3000"
    environment:
      NODE_ENV: production
      API_URL: http://localhost:5000
      PUBLIC_URL: https://myapp.com
  
  # Backend - Array of strings format
  api:
    language: python
    host: containerapp
    project: ./api
    ports:
      - "5000"
    environment:
      - FLASK_ENV=production
      - DATABASE_URL=postgresql://localhost:5432/db
      - REDIS_URL=redis://localhost:6379
  
  # Worker - Object format with substitution
  worker:
    language: python
    host: containerapp
    project: ./worker
    environment:
      - name: QUEUE_URL
        value: redis://localhost:6379
      - name: DATABASE_URL
        value: ${DATABASE_URL}
      - name: API_ENDPOINT
        value: http://localhost:5000
```

### Microservices with Shared Config

```yaml
name: microservices-app

services:
  auth:
    language: node
    host: containerapp
    environment:
      SERVICE_NAME: auth
      PORT: "3001"
      DATABASE_URL: postgresql://${DB_HOST}:${DB_PORT}/auth
      JWT_SECRET: ${JWT_SECRET}
  
  users:
    language: node
    host: containerapp
    environment:
      SERVICE_NAME: users
      PORT: "3002"
      DATABASE_URL: postgresql://${DB_HOST}:${DB_PORT}/users
      AUTH_SERVICE_URL: http://localhost:3001
```

## Best Practices

### 1. Use Map Format for Readability
```yaml
# ✅ Recommended
environment:
  NODE_ENV: production
  PORT: "3000"

# ❌ Less readable for simple cases
env:
  - NODE_ENV=production
  - PORT=3000
```

### 2. Quote Numeric Values
```yaml
# ✅ Explicit string
environment:
  PORT: "3000"

# ⚠️ May be interpreted as number
environment:
  PORT: 3000
```

### 3. Use Variable Substitution for DRY
```yaml
# ✅ Single source of truth
environment:
  DB_HOST: localhost
  DB_PORT: "5432"
  DB_NAME: mydb
  DATABASE_URL: postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}

# ❌ Duplication
environment:
  DATABASE_URL: postgresql://localhost:5432/mydb
```

### 4. Group Related Variables
```yaml
environment:
  # Database configuration
  DATABASE_URL: postgresql://localhost:5432/db
  DATABASE_POOL_SIZE: "20"
  DATABASE_TIMEOUT: "30"
  
  # Redis configuration
  REDIS_URL: redis://localhost:6379
  REDIS_MAX_RETRIES: "3"
  
  # Application settings
  LOG_LEVEL: info
  DEBUG: "false"
```

### 5. Use .env Files for Secrets
```yaml
# azure.yaml - Non-sensitive configuration
services:
  api:
    environment:
      PORT: "3000"
      LOG_LEVEL: info
```

```bash
# .env - Sensitive values (gitignored)
DATABASE_PASSWORD=secret123
API_KEY=xyz789
```

## Migration from Docker Compose

Azure.yaml is designed to be compatible with Docker Compose. You can often copy environment variables directly:

### From docker-compose.yml
```yaml
services:
  web:
    image: myapp
    environment:
      NODE_ENV: production
      PORT: 3000
      API_URL: http://api:5000
```

### To azure.yaml
```yaml
services:
  web:
    language: node
    host: containerapp
    project: ./web
    environment:
      NODE_ENV: production
      PORT: "3000"
      API_URL: http://api:5000
```

### Array Format Migration
```yaml
# docker-compose.yml
services:
  api:
    environment:
      - NODE_ENV=production
      - PORT=5000

# azure.yaml (identical syntax works!)
services:
  api:
    language: node
    host: containerapp
    environment:
      - NODE_ENV=production
      - PORT=5000
```

## Troubleshooting

### Variable Not Set
If a variable isn't being set:
1. Check spelling and capitalization
2. Verify YAML indentation (2 spaces)
3. Check variable priority (azure.yaml overrides everything)
4. Use `azd app info` to see resolved environment variables

### Variable Substitution Not Working
```yaml
# ❌ Wrong - trying to use undefined variable
environment:
  DATABASE_URL: postgresql://${UNDEFINED_VAR}:5432/db

# ✅ Correct - define or ensure it exists in Azure env
environment:
  DB_HOST: localhost
  DATABASE_URL: postgresql://${DB_HOST}:5432/db
```

### Special Characters Issues
```yaml
# ✅ Quote strings with special characters
environment:
  PASSWORD: "p@ssw0rd!"
  
# ✅ For array format, no quotes needed
env:
  - PASSWORD=p@ssw0rd!
```

## See Also

- [Docker Compose Environment Variables](https://docs.docker.com/compose/environment-variables/)
- [azd app run command](./commands/run.md)
- [Azure Environment Integration](./dev/azd-context-inheritance.md)
