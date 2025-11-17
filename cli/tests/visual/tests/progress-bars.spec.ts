import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Load metadata
const metadataPath = path.join(__dirname, '..', 'output', 'metadata.json');
let metadata: any = { captures: [] };

if (fs.existsSync(metadataPath)) {
  const metadataContent = fs.readFileSync(metadataPath, 'utf8');
  metadata = JSON.parse(metadataContent);
}

test.describe('Progress Bar Visual Tests', () => {
  test.beforeAll(async () => {
    // Ensure output directory exists
    const outputDir = path.join(__dirname, '..', 'output');
    if (!fs.existsSync(outputDir)) {
      throw new Error('No captured outputs found. Run capture-outputs.ps1 first.');
    }
  });

  for (const capture of metadata.captures || []) {
    test(`Terminal width ${capture.Width} - Visual rendering`, async ({ page }) => {
      const width = capture.Width;
      const height = metadata.height || 30;
      const filename = capture.FileName;

      // Navigate to terminal renderer with this output
      await page.goto(`/terminal?file=${filename}&width=${width}&height=${height}`);

      // Wait for content to load
      await page.waitForSelector('#terminal', { state: 'visible' });
      await page.waitForTimeout(1000); // Give time for rendering

      // Take screenshot
      const screenshotPath = path.join(__dirname, '..', 'screenshots', `width_${width}.png`);
      await page.screenshot({ 
        path: screenshotPath,
        fullPage: true 
      });

      console.log(`✓ Screenshot saved: width_${width}.png`);

      // Verify terminal rendered
      const terminal = page.locator('#terminal');
      await expect(terminal).toBeVisible();
      
      // Check that content exists
      const content = await terminal.textContent();
      expect(content).toBeTruthy();
      expect(content!.length).toBeGreaterThan(0);
    });
  }

  test('Compare all widths - Duplicate detection', async ({ page }) => {
    const results: any[] = [];

    for (const capture of metadata.captures || []) {
      const width = capture.Width;
      const height = metadata.height || 30;
      const filename = capture.FileName;

      await page.goto(`/terminal?file=${filename}&width=${width}&height=${height}`);
      await page.waitForSelector('#terminal');
      await page.waitForTimeout(500);

      // Extract text content
      const terminal = page.locator('#terminal');
      const content = await terminal.textContent();

      // Count progress bar lines (looking for common patterns)
      const lines = content?.split('\n') || [];
      const progressLines = lines.filter(line => 
        line.includes('[') && 
        (line.includes('⠋') || line.includes('⠙') || line.includes('⠹') || 
         line.includes('⠸') || line.includes('⠼') || line.includes('⠴') ||
         line.includes('⠦') || line.includes('⠧') || line.includes('⠇') || line.includes('⠏') ||
         line.includes('✓') || line.includes('100'))
      );

      // Count project mentions
      const webCount = (content?.match(/web/gi) || []).length;
      const apiCount = (content?.match(/api/gi) || []).length;

      results.push({
        width,
        totalLines: lines.length,
        progressLines: progressLines.length,
        webMentions: webCount,
        apiMentions: apiCount,
      });

      console.log(`Width ${width}: ${progressLines.length} progress lines, web: ${webCount}, api: ${apiCount}`);
    }

    // Analyze for duplicates
    // Wide terminals should have similar counts to narrow ones
    // If narrow terminals have 2-3x more lines, it suggests duplication
    
    const baseline = results.find(r => r.width === 120) || results[results.length - 1];
    
    for (const result of results) {
      const progressRatio = result.progressLines / baseline.progressLines;
      const webRatio = result.webMentions / baseline.webMentions;
      
      console.log(`Width ${result.width}: progress ratio ${progressRatio.toFixed(2)}, web ratio ${webRatio.toFixed(2)}`);
      
      // Fail if narrow terminal has significantly more output (duplication)
      if (progressRatio > 2.5) {
        throw new Error(`Width ${result.width} has ${progressRatio.toFixed(1)}x more progress lines than baseline (possible duplication)`);
      }
    }

    // Save comparison report
    const reportPath = path.join(__dirname, '..', 'test-results', 'comparison-report.json');
    const reportDir = path.dirname(reportPath);
    if (!fs.existsSync(reportDir)) {
      fs.mkdirSync(reportDir, { recursive: true });
    }
    fs.writeFileSync(reportPath, JSON.stringify({ results, baseline, timestamp: new Date().toISOString() }, null, 2));
    
    console.log(`✓ Comparison report saved to comparison-report.json`);
  });

  test('Screenshot comparison - All widths side-by-side', async ({ page }) => {
    // Create a comparison HTML page
    let html = `
<!DOCTYPE html>
<html>
<head>
    <title>Progress Bar Width Comparison</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        h1 { color: #333; }
        .comparison { display: flex; flex-wrap: wrap; gap: 20px; }
        .capture { background: white; padding: 15px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .capture h3 { margin-top: 0; }
        .capture img { max-width: 100%; border: 1px solid #ddd; }
        .stats { font-size: 12px; color: #666; margin-top: 10px; }
    </style>
</head>
<body>
    <h1>Progress Bar Visual Comparison</h1>
    <p>Generated: ${new Date().toISOString()}</p>
    <div class="comparison">
`;

    for (const capture of metadata.captures || []) {
      const screenshotFile = `width_${capture.Width}.png`;
      const screenshotPath = path.join(__dirname, '..', 'screenshots', screenshotFile);
      
      if (fs.existsSync(screenshotPath)) {
        html += `
        <div class="capture">
            <h3>Width: ${capture.Width} chars</h3>
            <img src="../screenshots/${screenshotFile}" alt="Width ${capture.Width}">
            <div class="stats">
                File: ${capture.FileName}<br>
                Captured: ${metadata.timestamp}
            </div>
        </div>
`;
      }
    }

    html += `
    </div>
</body>
</html>
`;

    const comparisonPath = path.join(__dirname, '..', 'test-results', 'visual-comparison.html');
    fs.writeFileSync(comparisonPath, html);
    
    console.log(`✓ Visual comparison page created: visual-comparison.html`);
  });
});
