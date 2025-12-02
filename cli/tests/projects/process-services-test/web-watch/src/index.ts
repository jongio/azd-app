import express from 'express';

const app = express();
const PORT = process.env.PORT || 3001;

app.get('/health', (req, res) => {
  res.json({ status: 'healthy', service: 'web-watch', mode: 'watch' });
});

app.get('/', (req, res) => {
  res.json({ message: 'Web Watch Service - HTTP service with nodemon watch mode' });
});

app.listen(PORT, () => {
  console.log(`web-watch running on port ${PORT}`);
});
