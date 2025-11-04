const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3000;
const API_URL = process.env.API_URL || 'http://api:8000';

app.use(express.static('public'));
app.use(express.json());

// API proxy endpoints
app.get('/api/*', async (req, res) => {
    try {
        const apiPath = req.path;
        const response = await fetch(`${API_URL}${apiPath}`);
        const data = await response.json();
        res.json(data);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

// Serve index.html for root
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

app.listen(PORT, '0.0.0.0', () => {
    console.log(`✓ Frontend server running on port ${PORT}`);
    console.log(`  API URL: ${API_URL}`);
    console.log(`  Open http://localhost:${PORT} in your browser`);
});
