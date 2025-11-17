/**
 * Simple HTTP server to serve terminal rendering pages for visual testing
 */
import http from 'http';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PORT = 9999;

const server = http.createServer((req, res) => {
  // Serve terminal.html
  if (req.url === '/' || req.url === '/terminal') {
    const htmlPath = path.join(__dirname, 'terminal.html');
    const html = fs.readFileSync(htmlPath, 'utf8');
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(html);
    return;
  }

  // Serve terminal output data
  if (req.url && req.url.startsWith('/output/')) {
    const filename = req.url.split('/output/')[1];
    const filePath = path.join(__dirname, 'output', filename);
    
    if (fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath, 'utf8');
      res.writeHead(200, { 'Content-Type': 'text/plain' });
      res.end(content);
    } else {
      res.writeHead(404);
      res.end('Not found');
    }
    return;
  }

  res.writeHead(404);
  res.end('Not found');
});

server.listen(PORT, () => {
  console.log(`Terminal rendering server running at http://localhost:${PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});
