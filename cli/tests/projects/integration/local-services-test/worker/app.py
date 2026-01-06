#!/usr/bin/env python3
"""Simple test worker service"""
import http.server
import socketserver
import json
import time
import random
from datetime import datetime

PORT = 8080

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            # Randomly return degraded (slow response) for testing
            time.sleep(random.uniform(1.0, 2.0))  # Slow response
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {
                'status': 'degraded',
                'warning': 'Slow response time',
                'responseTime': '1500ms',
                'timestamp': datetime.now().isoformat()
            }
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/':
            self.send_response(200)
            self.send_header('Content-type', 'text/html')
            self.end_headers()
            self.wfile.write(b'<h1>Worker Service</h1><p>Health endpoint is slow (degraded)</p>')
        else:
            self.send_response(404)
            self.end_headers()

with socketserver.TCPServer(("", PORT), Handler) as httpd:
    print(f"Worker serving at http://localhost:{PORT}")
    print(f"Health check: http://localhost:{PORT}/health (slow/degraded)")
    httpd.serve_forever()
