# xUnit .NET Test Framework Test Project

## Purpose

This test project validates that `azd app test` correctly **detects and runs xUnit test framework** for .NET applications, ensuring comprehensive test discovery and execution with the modern .NET testing framework.

## What Is Being Tested

### Test Framework Detection
When running `azd app test`:
1. xUnit is correctly identified as the test runner
2. Test classes inheriting from test base are discovered
3. Test methods marked with [Fact] and [Theory] are identified
4. xUnit configuration (xunit.runner.json) is parsed
5. `azd app test` executes `dotnet test` with proper reporting
6. Parametrized tests via [Theory] and [InlineData] work
7. Test results are aggregated across services

### Validation Points
- ✅ xUnit is the most popular modern .NET testing framework (~150M downloads)
- ✅ xUnit.net package is recognized
- ✅ Test methods with [Fact] attribute are discovered
- ✅ [Theory] with [InlineData] parametrization works
- ✅ xunit.runner.json configuration is recognized
- ✅ `dotnet test` command executes correctly
- ✅ Test output in JSON, TRX, and text formats
- ✅ Proper exit codes on test pass/fail

## Project Structure

```
xunit/
├── MathLibrary.csproj        # Class library to test
├── MathLibrary.Tests.csproj  # xUnit test project
├── src/
│   ├── Calculator.cs        # Source code to test
│   └── StringHelper.cs      # Source code to test
├── tests/
│   ├── CalculatorTests.cs   # xUnit test file
│   └── StringHelperTests.cs # xUnit test file
└── README.md                # This file
```

## Key Configuration

In `.csproj` (test project):
```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <IsTestProject>true</IsTestProject>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="xunit" Version="2.6.0" />
    <PackageReference Include="xunit.runner.visualstudio" Version="2.5.0" />
    <PackageReference Include="Microsoft.NET.Test.Sdk" Version="17.8.0" />
  </ItemGroup>

  <ItemGroup>
    <ProjectReference Include="../MathLibrary.csproj" />
  </ItemGroup>
</Project>
```

In `xunit.runner.json`:
```json
{
  "$schema": "https://xunit.net/schema/current/xunit.runner.schema.json",
  "appDomain": "denied",
  "methodDisplay": "method",
  "parallelizeAssembly": true,
  "parallelizeTestCollections": true
}
```

xUnit is configured with parallel execution and verbose output.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/test-command/dotnet-frameworks

# Build solution
dotnet build

# Run tests
dotnet test

# Expected output:
# Test Run Successful.
# Total tests: 4
#   Passed: 4
#   Failed: 0
#   Skipped: 0
# Test execution time: 1.234 Seconds
```

### With azd app test
```bash
# From workspace root
azd app test --service xunit-service

# Expected output:
# Testing xunit-service...
# 
# CalculatorTests::TestAdd PASSED
# CalculatorTests::TestMultiply PASSED
# StringHelperTests::TestTrim PASSED
# StringHelperTests::TestUpper PASSED
# 
# Summary: 4 tests passed
```

### Test with Theory and InlineData
```csharp
[Theory]
[InlineData(2, 3, 5)]
[InlineData(5, 5, 10)]
public void Addition(int a, int b, int expected)
{
    Assert.Equal(expected, Calculator.Add(a, b));
}
```

### Coverage Report
```bash
dotnet test /p:CollectCoverage=true /p:CoverageFormat=opencover

# Generates coverage.opencover.xml for analysis
```

### Automated Tests
This project is tested via:
- `cli/src/cmd/app/commands/test_command_integration_test.go` - Framework detection
- `cli/src/internal/executor/dotnet_executor_test.go` - dotnet test execution
- CI/CD pipeline test coverage validation

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct detection of xUnit as the test runner
- xUnit project structure and configuration
- Test method discovery via attributes
- Theory/parametrization support
- Coverage report generation
- Proper integration with azd test command
- The most popular modern .NET framework (150M downloads)

### Real-World Scenario
xUnit is the standard testing framework for modern .NET projects. Nearly all new C# projects use xUnit. This test ensures production-ready support for the most common .NET testing setup.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Framework | xUnit 2.6+ | ✅ |
| Test Classes | Public + [Fact] | ✅ |
| Test Methods | [Fact] and [Theory] | ✅ |
| Parametrization | [Theory][InlineData] | ✅ |
| Configuration | xunit.runner.json | ✅ |
| Parallel | Enabled | ✅ |
| Coverage | Supported via SDK | ✅ |
| Output Formats | JSON, TRX, text | ✅ |

## Troubleshooting

**"xUnit not found"**
- Install via NuGet: `dotnet add package xunit`
- Or via Package Manager: `Install-Package xunit`
- Verify: `dotnet package search xunit`

**"Tests not discovered"**
- Ensure test class is public
- Ensure test methods are public and have [Fact] or [Theory]
- Ensure project has `<IsTestProject>true</IsTestProject>`
- Check TargetFramework compatibility

**"InlineData not working"**
- Use [Theory] attribute instead of [Fact]
- Add [InlineData] attributes with correct parameter types
- Ensure parameter types match test method signature

**"Coverage not generated"**
- Install coverage tool: `dotnet add package coverlet.collector`
- Run with coverage: `dotnet test /p:CollectCoverage=true`
- Check coverage format: opencover, json, lcov

**"Tests hang or timeout"**
- Check for deadlocks in async code
- Set timeout: `[Fact(Timeout = 5000)]` (5 seconds)
- Reduce parallel execution if needed

## Related Test Projects

- [nunit/](../nunit/) - NUnit (original .NET framework)
- [mstest/](../mstest/) - MSTest (Microsoft's framework)
