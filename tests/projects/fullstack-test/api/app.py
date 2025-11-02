from flask import Flask, jsonify
from flask_cors import CORS
import os

app = Flask(__name__)
CORS(app)

@app.route('/')
def home():
    return jsonify({
        'message': 'Welcome to the API',
        'status': 'running',
        'service': 'api'
    })

@app.route('/api/data')
def get_data():
    return jsonify({
        'items': [
            {'id': 1, 'name': 'Item 1', 'description': 'First item'},
            {'id': 2, 'name': 'Item 2', 'description': 'Second item'},
            {'id': 3, 'name': 'Item 3', 'description': 'Third item'}
        ],
        'count': 3
    })

@app.route('/api/health')
def health():
    return jsonify({
        'status': 'healthy',
        'service': 'api'
    })

if __name__ == '__main__':
    port = int(os.getenv('PORT', 5000))
    print(f'🚀 API server starting on http://localhost:{port}')
    app.run(host='0.0.0.0', port=port, debug=True)
