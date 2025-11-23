const express = require('express');
const _ = require('lodash');

const app = express();

app.get('/', (req, res) => {
  res.json({ message: _.capitalize('hello from api workspace') });
});

const port = process.env.PORT || 3001;
app.listen(port, () => {
  console.log(`API listening on port ${port}`);
});
