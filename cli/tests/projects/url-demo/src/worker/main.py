import http.server
import os
import socketserver

PORT = int(os.environ.get("PORT") or os.environ.get("AZD_PORT") or "8000")
SERVICE = os.environ.get("SERVICE_NAME", "worker")


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        body = f'{{"service":"{SERVICE}","path":"{self.path}"}}'.encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        return


with socketserver.TCPServer(("", PORT), Handler) as httpd:
    print(f"{SERVICE} listening on {PORT}")
    httpd.serve_forever()
