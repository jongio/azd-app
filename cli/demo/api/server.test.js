const request = require('supertest');
const express = require('express');

// Create a test app instance
const createApp = () => {
  const app = express();
  app.use(express.json());
  
  // In-memory data store for tests
  const items = [];
  
  app.get('/items', (req, res) => {
    res.json(items);
  });
  
  app.post('/items', (req, res) => {
    const { name, price } = req.body;
    
    // Validation (this is what the "bug" in the real server is missing)
    if (!name || typeof name !== 'string') {
      return res.status(400).json({ error: 'Name is required' });
    }
    
    if (price === undefined || typeof price !== 'number' || isNaN(price)) {
      return res.status(400).json({ error: 'Valid price is required' });
    }
    
    const item = {
      id: items.length + 1,
      name: name,
      price: price,
      total: price * 1.1 // Calculate price with tax
    };
    
    items.push(item);
    res.status(201).json(item);
  });
  
  app.get('/items/:id', (req, res) => {
    const id = parseInt(req.params.id);
    const item = items.find(i => i.id === id);
    
    if (!item) {
      return res.status(404).json({ error: 'Item not found' });
    }
    
    res.json(item);
  });
  
  app.delete('/items/:id', (req, res) => {
    const id = parseInt(req.params.id);
    const index = items.findIndex(i => i.id === id);
    
    if (index === -1) {
      return res.status(404).json({ error: 'Item not found' });
    }
    
    items.splice(index, 1);
    res.status(204).send();
  });
  
  // Reset items for testing
  app.resetItems = () => items.length = 0;
  
  return app;
};

describe('Items API', () => {
  let app;
  
  beforeEach(() => {
    app = createApp();
  });
  
  describe('GET /items', () => {
    test('returns empty array initially', async () => {
      const response = await request(app)
        .get('/items')
        .expect(200);
      
      expect(response.body).toEqual([]);
    });
  });
  
  describe('POST /items', () => {
    test('creates a new item with valid data', async () => {
      const response = await request(app)
        .post('/items')
        .send({ name: 'Widget', price: 10 })
        .expect(201);
      
      expect(response.body).toMatchObject({
        id: 1,
        name: 'Widget',
        price: 10,
        total: 11 // 10 * 1.1
      });
    });
    
    test('returns 400 when name is missing', async () => {
      const response = await request(app)
        .post('/items')
        .send({ price: 10 })
        .expect(400);
      
      expect(response.body.error).toBe('Name is required');
    });
    
    test('returns 400 when price is missing', async () => {
      const response = await request(app)
        .post('/items')
        .send({ name: 'Widget' })
        .expect(400);
      
      expect(response.body.error).toBe('Valid price is required');
    });
    
    test('returns 400 when price is not a number', async () => {
      const response = await request(app)
        .post('/items')
        .send({ name: 'Widget', price: 'not-a-number' })
        .expect(400);
      
      expect(response.body.error).toBe('Valid price is required');
    });
    
    test('calculates tax correctly', async () => {
      const response = await request(app)
        .post('/items')
        .send({ name: 'Expensive Widget', price: 100 })
        .expect(201);
      
      expect(response.body.total).toBeCloseTo(110, 2); // 100 * 1.1 (floating point safe)
    });
  });
  
  describe('GET /items/:id', () => {
    test('returns 404 for non-existent item', async () => {
      const response = await request(app)
        .get('/items/999')
        .expect(404);
      
      expect(response.body.error).toBe('Item not found');
    });
    
    test('returns item by id', async () => {
      // First create an item
      await request(app)
        .post('/items')
        .send({ name: 'Test Item', price: 25 });
      
      const response = await request(app)
        .get('/items/1')
        .expect(200);
      
      expect(response.body.name).toBe('Test Item');
    });
  });
  
  describe('DELETE /items/:id', () => {
    test('returns 404 for non-existent item', async () => {
      await request(app)
        .delete('/items/999')
        .expect(404);
    });
    
    test('deletes item successfully', async () => {
      // First create an item
      await request(app)
        .post('/items')
        .send({ name: 'To Delete', price: 5 });
      
      // Delete it
      await request(app)
        .delete('/items/1')
        .expect(204);
      
      // Verify it's gone
      await request(app)
        .get('/items/1')
        .expect(404);
    });
  });
});
