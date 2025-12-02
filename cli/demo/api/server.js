const express = require('express');
const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());

// In-memory data store
const items = [];

// GET /items - List all items
app.get('/items', (req, res) => {
  console.log('[API] GET /items - returning', items.length, 'items');
  res.json(items);
});

// POST /items - Create new item
app.post('/items', (req, res) => {
  const { name, price } = req.body;
  
  // BUG: Missing validation causes undefined values
  // This will log an error when price is not a number
  const item = {
    id: items.length + 1,
    name: name,
    price: price,
    total: price * 1.1 // Calculate price with tax
  };
  
  // BUG: This will fail if price is undefined or not a number
  if (isNaN(item.total)) {
    console.error('[API] ERROR: Invalid price calculation - price:', price, 'total:', item.total);
    console.error('[API] ERROR: Request body was:', JSON.stringify(req.body));
  }
  
  items.push(item);
  console.log('[API] POST /items - created item:', JSON.stringify(item));
  res.status(201).json(item);
});

// GET /items/:id - Get single item
app.get('/items/:id', (req, res) => {
  const id = parseInt(req.params.id);
  const item = items.find(i => i.id === id);
  
  if (!item) {
    console.error('[API] ERROR: Item not found with id:', id);
    return res.status(404).json({ error: 'Item not found' });
  }
  
  console.log('[API] GET /items/' + id + ' - returning item');
  res.json(item);
});

// DELETE /items/:id - Delete item
app.delete('/items/:id', (req, res) => {
  const id = parseInt(req.params.id);
  const index = items.findIndex(i => i.id === id);
  
  if (index === -1) {
    console.error('[API] ERROR: Cannot delete - item not found with id:', id);
    return res.status(404).json({ error: 'Item not found' });
  }
  
  items.splice(index, 1);
  console.log('[API] DELETE /items/' + id + ' - item deleted');
  res.status(204).send();
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('[API] UNHANDLED ERROR:', err.message);
  console.error('[API] Stack trace:', err.stack);
  res.status(500).json({ error: 'Internal server error' });
});

app.listen(port, () => {
  console.log('[API] Demo API server running on port', port);
  console.log('[API] Try: POST /items with {"name": "Test", "price": 10}');
  console.log('[API] Bug demo: POST /items with {"name": "Test"} (missing price)');
});
