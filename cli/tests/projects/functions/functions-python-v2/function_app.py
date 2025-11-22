import azure.functions as func
import datetime
import json
import logging

app = func.FunctionApp()

@app.route(route="httpTrigger", methods=["GET", "POST"], auth_level=func.AuthLevel.ANONYMOUS)
async def http_trigger(req: func.HttpRequest) -> func.HttpResponse:
    logging.info('Python HTTP trigger function processed a request (v2 model).')

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
        'message': f'Hello, {name}! (Python v2)',
        'timestamp': datetime.datetime.now().isoformat(),
        'method': req.method
    }

    return func.HttpResponse(
        body=json.dumps(response_data),
        status_code=200,
        mimetype='application/json'
    )


@app.timer_trigger(schedule="0 */5 * * * *", arg_name="myTimer", run_on_startup=False)
def timer_trigger(myTimer: func.TimerRequest) -> None:
    timestamp = datetime.datetime.now().isoformat()
    
    logging.info(f'Python timer trigger function executed at: {timestamp}')
    
    if myTimer.past_due:
        logging.info('Timer is running late!')
