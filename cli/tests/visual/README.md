# Progress Bar Analysis Tests

Automated testing for progress bar duplication detection across different terminal widths.

## Quick Start

```powershell
# 1. Capture outputs at different widths
.\capture-outputs.ps1

# 2. Run analysis tests
npm install  # First time only
npm test
```

## How It Works

1. **capture-outputs.ps1** - Runs `azd deps` at widths 50, 80, 120 and captures output
2. **Tests** - Analyzes captured files for duplicate progress bars
3. **Reports** - Generates JSON and markdown comparison reports

## Test Criteria

- ✅ **Pass**: Narrow terminals have ≤2.5x progress lines vs baseline (120 width)
- ❌ **Fail**: Narrow terminals have >2.5x progress lines (indicates duplication)

## Files

- `capture-outputs.ps1` - Output capture script
- `tests/progress-bars.spec.ts` - Analysis tests
- `output/` - Captured terminal outputs
- `test-results/` - Test reports

## Customization

```powershell
# Test specific widths
.\capture-outputs.ps1 -Widths 40,60,100

# Custom terminal height
.\capture-outputs.ps1 -Height 50
```

## CI/CD Integration

Tests are fully automated and require no browser/UI:

```yaml
- run: cd cli/tests/visual && .\capture-outputs.ps1
- run: cd cli/tests/visual && npm ci && npm test
```
