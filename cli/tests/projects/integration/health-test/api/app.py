from flask import Flask, jsonify
import time

app = Flask(__name__)
start_time = time.time()
request_count = 0

@app.route('/healthz')
def health():
    global request_count
    request_count += 1
    uptime = int(time.time() - start_time)
    
    return jsonify({
        'status': 'ok',
        'service': 'api',
        'version': '1.0.0',
        'uptime': f'{uptime}s',
        'database': 'connected',
        'requestCount': request_count,
        'timestamp': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
    })

@app.route('/')
def index():
    return 'API Service - Health Monitoring Test'

if __name__ == '__main__':
    port = 5000
    print(f'✅ API service listening on port {port}')
    print(f'   Health endpoint: http://localhost:{port}/healthz')
    print(f'   Started at: {time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}')
    app.run(host='0.0.0.0', port=port, debug=False)
