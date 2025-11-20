import datetime
import logging
import azure.functions as func


def main(myTimer: func.TimerRequest) -> None:
    timestamp = datetime.datetime.now().isoformat()
    
    logging.info(f'Python timer trigger function executed at: {timestamp}')
    
    if myTimer.past_due:
        logging.info('Timer is running late!')
