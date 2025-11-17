# Visual Testing for azd Progress Bars

## Overview

This visual testing framework captures terminal output from `azd deps` at different terminal widths and generates **screenshot-based visual regression tests** using Playwright.

## Features

✅ **Automated Terminal Capture** - Captures raw ANSI output at multiple widths  
✅ **Screenshot Generation** - Renders terminal output as images  
✅ **Duplicate Detection** - Analyzes for progress bar duplication  
✅ **Visual Comparison** - Side-by-side comparison of all widths  
✅ **HTML Reports** - Beautiful visual reports with screenshots  

## Quick Start

### 1. Install Dependencies

```powershell
cd C:\code\azd-app\cli\tests\visual
npm install
npx playwright install chromium
```

### 2. Capture Terminal Outputs

```powershell
.\capture-outputs.ps1
```

This runs `azd deps` at widths: 40, 50, 60, 80, 100, 120, 140 chars

### 3. Run Visual Tests

```powershell
npm test
```

### 4. View Results

```powershell
npm run show-report
```

## How It Works

### Architecture

```
1. capture-outputs.ps1
   ├── Builds azd.exe
   ├── Sets $env:COLUMNS for each width
   ├── Runs azd deps --clean
   └── Saves raw ANSI output to output/*.txt

2. Playwright Tests
   ├── Loads output files
   ├── Renders in HTML terminal emulator
   ├── Takes screenshots
   └── Analyzes for duplicates

3. Reports
   ├── HTML comparison page
   ├── JSON analysis report
   └── Visual screenshots
```

### File Structure

```
tests/visual/
├── package.json              # Dependencies
├── playwright.config.ts      # Playwright configuration
├── server.js                 # Local web server
├── terminal.html             # ANSI terminal renderer
├── capture-outputs.ps1       # Output capture script
├── tests/
│   └── progress-bars.spec.ts # Visual tests
├── output/                   # Captured terminal outputs
│   ├── width_40_*.txt
│   ├── width_50_*.txt
│   ├── ...
│   └── metadata.json
├── screenshots/              # Generated screenshots
│   ├── width_40.png
│   ├── width_50.png
│   └── ...
└── test-results/            # Test reports
    ├── visual-comparison.html
    ├── comparison-report.json
    └── html-report/
```

## Captured Outputs

Terminal outputs are captured with:
- **Full ANSI codes** - Colors, cursor movements, etc.
- **Raw text** - Exactly as terminal receives it
- **Multiple widths** - From 40 to 140 characters

Example output file (`width_80_143522.txt`):
```
[Progress bars with ANSI escape codes]
⠋ web (npm)     [          ]   5% Installing packages...
⠙ api (pip)     [          ]   3% Installing dependencies...
```

## Terminal Renderer

The `terminal.html` page renders ANSI codes as HTML:

- **ANSI color codes** → CSS classes
- **Cursor movements** → Stripped (static view)
- **Progress bars** → Visible with colors
- **Spinners** → Visible as Unicode characters

### URL Parameters

```
/terminal?file=width_80_143522.txt&width=80&height=30
```

- `file` - Output filename from `/output/` directory
- `width` - Terminal width in characters
- `height` - Terminal height in lines

## Visual Tests

### Test 1: Individual Width Screenshots

```typescript
test(`Terminal width ${width} - Visual rendering`, async ({ page }) => {
  await page.goto(`/terminal?file=${filename}&width=${width}&height=${height}`);
  await page.screenshot({ path: `screenshots/width_${width}.png` });
});
```

**Purpose**: Captures how progress bars look at each width

### Test 2: Duplicate Detection

```typescript
test('Compare all widths - Duplicate detection', async ({ page }) => {
  // Count progress bar lines at each width
  // Compare to baseline (wide terminal)
  // Fail if narrow terminal has 2.5x+ more lines
});
```

**Purpose**: Automated detection of duplication

### Test 3: Visual Comparison Page

```typescript
test('Screenshot comparison - All widths side-by-side', async ({ page }) => {
  // Generates HTML page with all screenshots
});
```

**Purpose**: Easy visual inspection

## Duplicate Detection Logic

```javascript
const baseline = results.find(r => r.width === 120);

for (const result of results) {
  const progressRatio = result.progressLines / baseline.progressLines;
  
  if (progressRatio > 2.5) {
    // FAIL: Duplication detected!
    throw new Error(`Width ${width} has too many progress lines`);
  }
}
```

**Threshold**: 2.5x more progress lines indicates duplication

## Custom Widths

```powershell
# Test specific widths
.\capture-outputs.ps1 -Widths 45,75,110

# Test with different height
.\capture-outputs.ps1 -Height 50

# Clean old outputs first
.\capture-outputs.ps1 -CleanFirst
```

## Advanced Usage

### Headless vs Headed

```powershell
# Headless (default)
npm test

# Headed (watch browser)
npm run test:headed

# Debug mode
npm run test:debug
```

### UI Mode

```powershell
npm run test:ui
```

Interactive test runner with time travel debugging!

### Single Test

```powershell
npx playwright test --grep "width 80"
```

### Update Baselines

```powershell
# Capture new baseline outputs
.\capture-outputs.ps1 -CleanFirst

# Run tests
npm test

# Review screenshots
npm run show-report
```

## CI/CD Integration

```yaml
# .github/workflows/visual-tests.yml
- name: Capture terminal outputs
  shell: pwsh
  run: |
    cd cli/tests/visual
    .\capture-outputs.ps1

- name: Run visual tests
  run: |
    cd cli/tests/visual
    npm ci
    npx playwright install --with-deps chromium
    npm test

- name: Upload screenshots
  uses: actions/upload-artifact@v3
  with:
    name: visual-test-results
    path: cli/tests/visual/screenshots/
```

## Screenshot Analysis

Screenshots show:
- ✅ **Spinner animations** (captured frame)
- ✅ **Progress bar formatting**
- ✅ **Color coding** (success=green, error=red)
- ✅ **Text alignment**
- ✅ **Terminal width handling**
- ✅ **Line wrapping** (or lack thereof)

## Interpreting Results

### ✅ Good Result

```
Width 40: progress ratio 1.02, web ratio 1.05
Width 80: progress ratio 1.00, web ratio 1.00
Width 120: progress ratio 1.00, web ratio 1.00
```

**Narrow terminals have similar line counts to wide ones**

### ❌ Bad Result (Duplication)

```
Width 40: progress ratio 3.45, web ratio 4.21
Width 80: progress ratio 1.15, web ratio 1.08
Width 120: progress ratio 1.00, web ratio 1.00
```

**Narrow terminal has 3.45x more progress lines → duplication detected!**

## Troubleshooting

### No outputs captured

```powershell
# Check azd builds
cd C:\code\azd-app\cli
go build -o bin\azd.exe

# Manually test
cd tests\projects\fullstack-test
$env:COLUMNS = 80
..\..\..\bin\azd.exe deps --clean
```

### Screenshots are blank

- Ensure server is running (`node server.js`)
- Check `output/metadata.json` exists
- Verify output files have content

### Tests fail with "No captured outputs"

```powershell
# Run capture first
.\capture-outputs.ps1

# Then run tests
npm test
```

## Comparison with Other Tests

| Test | Method | Visual? | Automated? | CI-Ready? |
|------|--------|---------|------------|-----------|
| `test-progress-duplicates.ps1` | Text analysis | ❌ | ✅ | ✅ |
| `test-terminal-resize.ps1` | $env:COLUMNS | ❌ | ✅ | ✅ |
| **Visual tests** | Playwright screenshots | ✅ | ✅ | ✅ |

**Visual tests provide the most comprehensive verification**

## Example Output

### Visual Comparison Page

![Visual Comparison](./docs/visual-comparison-example.png)

Shows all widths side-by-side for easy inspection.

### Individual Screenshot

![Width 50](./docs/width-50-example.png)

Shows exact terminal rendering at 50 characters wide.

## Future Enhancements

- [ ] Baseline comparison (detect regressions)
- [ ] Animation capture (GIF/video of spinner)
- [ ] Real-time resize simulation
- [ ] Color accuracy verification
- [ ] Cross-platform testing (macOS, Linux terminals)

## References

- [Playwright Visual Comparisons](https://playwright.dev/docs/test-snapshots)
- [ANSI Escape Codes](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [Terminal Emulation](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
