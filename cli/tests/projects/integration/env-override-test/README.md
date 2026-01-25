# Environment Override Test

This test project verifies which environment is loaded when both:
1. The `.azure/config.json` specifies a default environment (`dev`)
2. The `-e` flag specifies a different environment (`prod`)

## Test Setup

Two environments are configured:
- **dev** (default in config.json): `TEST_ENV_VALUE = "THIS_IS_DEV_ENVIRONMENT"`
- **prod**: `TEST_ENV_VALUE = "THIS_IS_PROD_ENVIRONMENT"`

## Running the Tests

### Test 1: Without -e flag (should use dev from config.json)
```powershell
cd c:\code\azd-app\cli\tests\projects\integration\env-override-test
azd app run
```

Expected output:
```
Environment loaded: dev
Test value: THIS_IS_DEV_ENVIRONMENT
```

### Test 2: With -e prod flag (should override config.json and use prod)
```powershell
azd app run -e prod
```

Expected output:
```
Environment loaded: prod
Test value: THIS_IS_PROD_ENVIRONMENT
```

## Verification

The server will:
1. Print the environment name and test value to the console on startup
2. Serve them at http://localhost:3000
3. Provide JSON API at http://localhost:3000/api/env

Check which `TEST_ENV_VALUE` is displayed to verify which environment was actually loaded.
