# Custom URL Demo

This project demonstrates the custom URL configuration feature for services.

## Configuration

The `azure.yaml` file configures custom URLs for services:

```yaml
services:
  web:
    language: node
    host: containerapp
    project: ./src/web
    url: https://myapp.example.com
  
  api:
    language: python
    host: appservice
    project: ./src/api
    url: https://api.myapp.example.com
```

## Testing Console Output

To test the console output:

1. Set up environment variables to simulate Azure deployment:
   ```bash
   $env:SERVICE_WEB_URL = "https://web-abc123.azurecontainerapps.io"
   $env:SERVICE_API_URL = "https://api-abc123.azurewebsites.net"
   ```

2. Run the info command:
   ```bash
   cd tests/projects/alt-url-demo
   azd app info
   ```

## Expected Output

The console output should display both the deployment URL and access URL:

```
Service: web
  Deployment URL: https://web-abc123.azurecontainerapps.io
  Access URL: https://myapp.example.com

Service: api
  Deployment URL: https://api-abc123.azurewebsites.net
  Access URL: https://api.myapp.example.com
```
