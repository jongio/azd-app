const net = require('net');

const PORT = process.env.PORT || 9000;

const server = net.createServer((socket) => {
  console.log('Client connected');
  
  socket.on('data', (data) => {
    console.log('Received:', data.toString());
    socket.write('ACK: ' + data.toString());
  });
  
  socket.on('end', () => {
    console.log('Client disconnected');
  });
});

server.listen(PORT, () => {
  console.log(`TCP server listening on port ${PORT}`);
});
