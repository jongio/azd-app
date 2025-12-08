import express from 'express';
import { add, subtract, multiply, divide } from './calculator.js';

const app = express();
app.use(express.json());

app.get('/health', (_req, res) => {
  res.json({ status: 'healthy' });
});

app.post('/calculate', (req, res) => {
  const { operation, a, b } = req.body;
  
  try {
    let result: number;
    switch (operation) {
      case 'add':
        result = add(a, b);
        break;
      case 'subtract':
        result = subtract(a, b);
        break;
      case 'multiply':
        result = multiply(a, b);
        break;
      case 'divide':
        result = divide(a, b);
        break;
      default:
        res.status(400).json({ error: 'Unknown operation' });
        return;
    }
    res.json({ result });
  } catch (error) {
    res.status(400).json({ error: (error as Error).message });
  }
});

const PORT = process.env.PORT || 3000;

if (process.env.NODE_ENV !== 'test') {
  app.listen(PORT, () => {
    console.log(`Server running on port ${PORT}`);
  });
}

export { app };
