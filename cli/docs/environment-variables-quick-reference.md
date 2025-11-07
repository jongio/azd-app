# Environment Variables Quick Reference

## Three Ways to Define Environment Variables

### 1. Map Format ⭐ Recommended
```yaml
services:
  api:
    environment:
      NODE_ENV: production
      PORT: "3000"
```

### 2. Array of Strings
```yaml
services:
  api:
    environment:
      - NODE_ENV=production
      - PORT=3000
```

### 3. Array of Objects
```yaml
services:
  api:
    environment:
      - name: API_KEY
        value: my-key
      - name: SECRET
        secret: super-secret
```

## Variable Substitution

```yaml
environment:
  DB_HOST: localhost
  DB_PORT: "5432"
  DATABASE_URL: postgresql://${DB_HOST}:${DB_PORT}/db
```

## Priority

1. 🥇 Service environment (azure.yaml)
2. 🥈 .env file
3. 🥉 Azure environment
4. OS environment

## Special Characters

All formats handle special characters:
```yaml
environment:
  PASSWORD: "p@ssw0rd!"
  CONNECTION: "Server=localhost;Pass=abc=123"
```

## Examples

### Full Stack App
```yaml
services:
  web:
    environment:
      NODE_ENV: production
      API_URL: http://localhost:5000
  
  api:
    environment:
      - FLASK_ENV=production
      - DATABASE_URL=postgresql://localhost:5432/db
```

### With Substitution
```yaml
services:
  api:
    environment:
      HOST: localhost
      PORT: "5000"
      API_URL: http://${HOST}:${PORT}
```

## See Also

📖 Full documentation: [docs/environment-variables.md](./environment-variables.md)
