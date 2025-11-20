import logging
import azure.functions as func
import json
import datetime


def main(req: func.HttpRequest) -> func.HttpResponse:
    logging.info('Python HTTP trigger function processed a request (v1 model).')

    name = req.params.get('name')
    if not name:
        try:
            req_body = req.get_json()
        except ValueError:
            pass
        else:
            name = req_body.get('name')

    if not name:
        name = 'World'

    response_data = {
        'message': f'Hello, {name}! (Python v1 legacy)',
        'timestamp': datetime.datetime.now().isoformat(),
        'method': req.method
    }

    return func.HttpResponse(
        body=json.dumps(response_data),
        status_code=200,
        mimetype='application/json'
    )
