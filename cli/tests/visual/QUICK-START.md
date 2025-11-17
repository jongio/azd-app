# Quick Start - Visual Testing

Get screenshots of progress bars at different terminal widths in **3 commands**.

## Prerequisites

- Node.js 18+ installed
- Go 1.21+ installed  
- Windows (PowerShell)

## Steps

### 1. Navigate to visual tests directory

```powershell
cd C:\code\azd-app\cli\tests\visual
```

### 2. Run the complete test suite

```powershell
.\run-visual-tests.ps1
```

**This will**:
- Install dependencies (npm + Playwright)
- Build azd.exe
- Capture terminal output at 7 different widths (40, 50, 60, 80, 100, 120, 140 chars)
- Generate screenshots
- Run duplicate detection analysis
- Open HTML reports in your browser

### 3. View results

Reports open automatically! Look for:

**Visual Comparison Page** - Side-by-side screenshots  
**Playwright Report** - Detailed test results  
**Screenshots folder** - Individual width screenshots

## What You'll See

### ✅ If Tests Pass

```
========================================
Summary
========================================
Screenshots generated: 7
Test results: PASSED

Analysis Results:
  ✓ Width 40: ratio 1.05
  ✓ Width 50: ratio 1.02
  ✓ Width 60: ratio 1.01
  ✓ Width 80: ratio 1.00
  ✓ Width 100: ratio 0.98
  ✓ Width 120: ratio 1.00
  ✓ Width 140: ratio 0.99

✓ Visual testing complete!
```

**Ratios close to 1.0 = Good!** Progress bars behave consistently across widths.

### ❌ If Tests Fail (Duplication Detected)

```
Analysis Results:
  ❌ Width 40: ratio 3.45
  ✓ Width 50: ratio 1.15
  ✓ Width 80: ratio 1.00

✗ Some tests failed. Review the reports for details.
```

**Ratio > 2.5 = Problem!** Narrow terminal has too many progress bar lines (duplication).

## Files Generated

```
tests/visual/
├── screenshots/
│   ├── width_40.png       ← Visual output at 40 chars
│   ├── width_50.png       ← Visual output at 50 chars
│   ├── width_60.png
│   ├── width_80.png
│   ├── width_100.png
│   ├── width_120.png
│   └── width_140.png
│
├── test-results/
│   ├── visual-comparison.html    ← Side-by-side view
│   ├── comparison-report.json    ← Analysis data
│   └── html-report/
│       └── index.html            ← Playwright report
│
└── output/
    ├── width_40_*.txt     ← Raw terminal output
    ├── width_50_*.txt
    └── metadata.json
```

## Common Tasks

### Test specific widths only

```powershell
.\run-visual-tests.ps1 -Widths 40,80,120
```

### Re-run tests without recapturing

```powershell
.\run-visual-tests.ps1 -SkipCapture
```

### Just capture, no tests

```powershell
.\capture-outputs.ps1
```

### Just tests, no capture

```powershell
npm test
```

## Next Steps

- **Review screenshots** in `screenshots/` folder
- **Open visual comparison** - `test-results/visual-comparison.html`
- **Check analysis** - Look at pass/fail ratios
- **Read full docs** - See [README.md](README.md) and [ARCHITECTURE.md](ARCHITECTURE.md)

## Troubleshooting

### "No captured outputs found"

```powershell
# Run capture first
.\capture-outputs.ps1

# Then run tests
npm test
```

### "Playwright not found"

```powershell
npm install
npx playwright install chromium
```

### "Build failed"

```powershell
cd C:\code\azd-app\cli
go build -o bin\azd.exe
```

## That's It!

You now have **visual proof** that progress bars work (or don't work) at all terminal widths.

The screenshots provide irrefutable evidence of how the output actually looks to users.
