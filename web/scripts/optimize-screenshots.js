import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const files = [
  'dashboard-console.png',
  'dashboard-resources-grid.png',
  'dashboard-resources-table.png',
  'dashboard-azure-logs.png',
  'dashboard-azure-logs-time-range.png',
  'dashboard-azure-logs-filters.png'
];

const dir = path.join(__dirname, '..', 'public', 'screenshots');

async function optimizeScreenshots() {
  console.log('🔧 Optimizing screenshots...\n');
  
  for (const file of files) {
    const filePath = path.join(dir, file);
    const tempPath = filePath + '.tmp';
    
    try {
      const before = fs.statSync(filePath).size;
      
      await sharp(filePath)
        .png({
          quality: 90,
          compressionLevel: 9,
          palette: true
        })
        .toFile(tempPath);
      
      fs.renameSync(tempPath, filePath);
      
      const after = fs.statSync(filePath).size;
      const savedKB = Math.round((before - after) / 1024 * 10) / 10;
      const beforeKB = Math.round(before / 1024);
      const afterKB = Math.round(after / 1024);
      
      console.log(`  ✓ ${file}: ${beforeKB}KB → ${afterKB}KB (saved ${savedKB}KB)`);
    } catch (error) {
      console.error(`  ❌ Failed to optimize ${file}:`, error.message);
    }
  }
  
  console.log('\n✨ Optimization complete!');
}

optimizeScreenshots().catch(console.error);
