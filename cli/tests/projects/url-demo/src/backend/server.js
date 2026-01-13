const http = require("http");

const port = process.env.PORT || process.env.AZD_PORT || 3000;
const service = process.env.SERVICE_NAME || "backend";

const handler = (req, res) => {
  const payload = JSON.stringify({ service, path: req.url });
  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(payload);
};

http
  .createServer(handler)
  .listen(port, "0.0.0.0", () => {
    console.log(`${service} listening on ${port}`);
  });
