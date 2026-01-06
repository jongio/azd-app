#!/usr/bin/env python3
"""Simple test API service"""
import http.server
import socketserver
import json
from datetime import datetime

PORT = 8000

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(503)  # Unhealthy on purpose to test diagnostics
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {
                'status': 'unhealthy',
                'error': 'Database connection failed',
                'details': 'Connection timeout after 5s',
                'timestamp': datetime.now().isoformat()
            }
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/':
            self.send_response(200)
            self.send_header('Content-type', 'text/html')
            self.end_headers()
            self.wfile.write(b'<h1>API Service</h1><p>Health endpoint returns 503 for testing</p>')
        else:
            self.send_response(404)
            self.end_headers()

with socketserver.TCPServer(("", PORT), Handler) as httpd:
    print(f"API serving at http://localhost:{PORT}")
    print(f"Health check: http://localhost:{PORT}/health (returns 503)")
    httpd.serve_forever()
