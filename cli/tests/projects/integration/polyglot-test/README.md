# Polyglot Test Project

This is a test project with multiple languages used to validate the `azd app test` command.

## Services

1. **node-api** - Node.js API with Vitest tests
2. **python-worker** - Python worker with pytest tests
3. **go-service** - Go service with go test
4. **dotnet-api** - .NET API with xUnit tests

## Running Tests

```bash
# Run all tests
azd app test

# Run with coverage
azd app test --coverage --threshold 70

# Run only unit tests
azd app test --type unit

# Run tests for specific service
azd app test --service node-api
```
