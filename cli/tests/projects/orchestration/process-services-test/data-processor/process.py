"""
Data Processor - Process service with task mode
One-time execution that processes data and exits
"""

import json
import sys
import time

def process_data():
    print("Data Processor starting...")
    print("Loading input data...")
    
    # Simulate data processing
    for i in range(5):
        print(f"Processing batch {i + 1}/5...")
        time.sleep(0.5)
    
    result = {
        "status": "completed",
        "records_processed": 1000,
        "duration_seconds": 2.5
    }
    
    print(f"Processing complete: {json.dumps(result)}")
    return 0

if __name__ == "__main__":
    sys.exit(process_data())
