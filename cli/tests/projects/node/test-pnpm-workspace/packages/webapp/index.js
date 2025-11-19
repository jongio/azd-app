const express = require('express');
const axios = require('axios');
const app = express();
const PORT = process.env.PORT || 3002;
const API_URL = process.env.API_URL || 'http://localhost:3001';

app.get('/', async (req, res) => {
  try {
    const apiResponse = await axios.get(`${API_URL}/`);
    res.json({
      service: 'webapp',
      message: 'pnpm workspace Web App service',
      apiData: apiResponse.data,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.json({
      service: 'webapp',
      message: 'pnpm workspace Web App service',
      apiError: 'Could not reach API service',
      timestamp: new Date().toISOString()
    });
  }
});

app.get('/health', (req, res) => {
  res.json({ status: 'healthy' });
});

app.listen(PORT, () => {
  console.log(`WebApp service listening on http://localhost:${PORT}`);
});
