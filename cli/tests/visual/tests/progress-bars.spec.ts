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

test.describe('Progress Bar Analysis Tests', () => {
  test.beforeAll(async () => {
    // Ensure output directory exists
    const outputDir = path.join(__dirname, '..', 'output');
    if (!fs.existsSync(outputDir)) {
      throw new Error('No captured outputs found. Run capture-outputs.ps1 first.');
    }
  });

  test('Compare all widths - Duplicate detection', async () => {
    const results: any[] = [];

    for (const capture of metadata.captures || []) {
      const width = capture.Width;
      const filePath = path.join(__dirname, '..', 'output', capture.FileName);
      
      if (!fs.existsSync(filePath)) {
        throw new Error(`Output file not found: ${capture.FileName}`);
      }

      // Read and analyze file content
      const content = fs.readFileSync(filePath, 'utf8');
      const lines = content.split('\n');
      
      // Count progress bar lines (looking for common patterns)
      // Strip ANSI codes for accurate counting
      const stripAnsi = (str: string) => str.replace(/\x1b\[[0-9;]*m/g, '').replace(/\x1b\[[0-9]*[A-K]/g, '').replace(/\r/g, '');
      
      const cleanLines = lines.map(stripAnsi);
      const progressLines = cleanLines.filter(line => 
        line.includes('[') && 
        (line.includes('⠋') || line.includes('⠙') || line.includes('⠹') || 
         line.includes('⠸') || line.includes('⠼') || line.includes('⠴') ||
         line.includes('⠦') || line.includes('⠧') || line.includes('⠇') || line.includes('⠏') ||
         line.includes('✓') || line.includes('100'))
      );

      // Count project mentions
      const webCount = (content.match(/web/gi) || []).length;
      const apiCount = (content.match(/api/gi) || []).length;

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

  test('Generate text-based comparison report', async () => {
    // Create a comparison report in markdown format
    let markdown = `# Progress Bar Width Comparison\n\n`;
    markdown += `Generated: ${new Date().toISOString()}\n\n`;
    markdown += `## Captured Outputs\n\n`;
    
    for (const capture of metadata.captures || []) {
      const filePath = path.join(__dirname, '..', 'output', capture.FileName);
      
      if (fs.existsSync(filePath)) {
        const stats = fs.statSync(filePath);
        markdown += `### Width: ${capture.Width} chars\n\n`;
        markdown += `- File: \`${capture.FileName}\`\n`;
        markdown += `- Size: ${stats.size} bytes\n`;
        markdown += `- Captured: ${metadata.timestamp}\n\n`;
      }
    }

    const comparisonPath = path.join(__dirname, '..', 'test-results', 'text-comparison.md');
    const reportDir = path.dirname(comparisonPath);
    if (!fs.existsSync(reportDir)) {
      fs.mkdirSync(reportDir, { recursive: true });
    }
    fs.writeFileSync(comparisonPath, markdown);
    
    console.log(`✓ Text comparison report created: text-comparison.md`);
  });
});
