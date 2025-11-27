import time
import random
import sys
import os
from datetime import datetime

# Set UTF-8 encoding for Windows console
if sys.platform == 'win32':
    sys.stdout.reconfigure(encoding='utf-8')
    sys.stderr.reconfigure(encoding='utf-8')

def log(level, message):
    timestamp = datetime.now().strftime('%H:%M:%S.%f')[:-3]
    print(f"[{level}] {timestamp} - {message}", flush=True)
    if level == "ERROR":
        print(f"[{level}] {timestamp} - {message}", file=sys.stderr, flush=True)

def process_job(job_id):
    """Simulate background job processing"""
    log("INFO", f"Processing job #{job_id}")
    
    # Simulate work
    time.sleep(random.uniform(0.5, 2.0))
    
    # Random outcomes
    outcome = random.choice(['success', 'success', 'success', 'warning', 'error'])
    
    if outcome == 'success':
        log("INFO", f"Job #{job_id} completed successfully")
    elif outcome == 'warning':
        log("WARN", f"Job #{job_id} completed with warnings - retry count exceeded threshold")
    else:
        log("ERROR", f"Job #{job_id} failed - unable to process task data")

def main():
    log("INFO", "⚙️ Worker Service starting...")
    log("INFO", "Connected to job queue")
    
    job_counter = 0
    
    while True:
        try:
            job_counter += 1
            
            # Batch processing
            batch_size = random.randint(1, 5)
            log("INFO", f"Processing job batch #{job_counter} ({batch_size} jobs)")
            
            for i in range(batch_size):
                process_job(f"{job_counter}.{i+1}")
            
            # Occasional system messages
            if job_counter % 10 == 0:
                log("INFO", f"Queue status: {random.randint(5, 50)} pending jobs")
            
            if job_counter % 15 == 0:
                log("WARN", "High queue depth detected - scaling workers recommended")
            
            # Wait before next batch
            time.sleep(random.uniform(2, 5))
            
        except KeyboardInterrupt:
            log("INFO", "Worker shutting down gracefully...")
            break
        except Exception as e:
            log("ERROR", f"Unexpected error in worker: {str(e)}")
            time.sleep(5)

if __name__ == '__main__':
    main()
