import os
import logging
import random
from datetime import datetime
from flask import Flask, jsonify, request

app = Flask(__name__)

# Configure logging
log_level = os.environ.get('LOG_LEVEL', 'INFO')
logging.basicConfig(
    level=getattr(logging, log_level),
    format='[%(asctime)s] [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

SERVICE_NAME = os.environ.get('SERVICE_NAME', 'appservice-web')


@app.before_request
def log_request():
    logger.info(f'{request.method} {request.path} - {SERVICE_NAME}')


@app.route('/health')
def health():
    return jsonify({
        'status': 'healthy',
        'service': SERVICE_NAME,
        'timestamp': datetime.utcnow().isoformat()
    })


@app.route('/')
def root():
    logger.info('Root endpoint accessed - generating sample logs')
    return jsonify({
        'service': SERVICE_NAME,
        'host': 'appservice',
        'message': 'Azure App Service log streaming test service',
        'timestamp': datetime.utcnow().isoformat()
    })


@app.route('/generate-logs')
def generate_logs():
    count = request.args.get('count', 5, type=int)
    levels = ['INFO', 'WARNING', 'ERROR', 'DEBUG']
    
    for i in range(count):
        level = random.choice(levels)
        message = f'Sample log message {i + 1} of {count} from {SERVICE_NAME}'
        
        if level == 'INFO':
            logger.info(message)
        elif level == 'WARNING':
            logger.warning(message)
        elif level == 'ERROR':
            logger.error(message)
        else:
            logger.debug(message)
    
    return jsonify({'generated': count, 'service': SERVICE_NAME})


@app.route('/error')
def error():
    logger.error(f'Simulated error in {SERVICE_NAME} - this is a test error for log streaming')
    return jsonify({'error': 'Simulated error', 'service': SERVICE_NAME}), 500


if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8293))
    logger.info(f'{SERVICE_NAME} started on port {port}')
    logger.info(f'Health check: http://localhost:{port}/health')
    logger.info(f'Generate logs: http://localhost:{port}/generate-logs?count=10')
    app.run(host='0.0.0.0', port=port)
