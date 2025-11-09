import time
import os
import signal
import sys

running = True
start_time = time.time()
task_count = 0

def signal_handler(sig, frame):
    global running
    uptime = int(time.time() - start_time)
    print(f'\n   Worker shutting down after {uptime}s')
    print(f'   Total tasks processed: {task_count}')
    running = False
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)

print(f'✅ Worker service started (background mode)')
print(f'   No ports - will use process check for health')
print(f'   Started at: {time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}')
print(f'   PID: {os.getpid()}')
print(f'   Processing tasks every 10 seconds...')

while running:
    time.sleep(10)
    task_count += 1
    uptime = int(time.time() - start_time)
    print(f'   Task #{task_count} processed (uptime: {uptime}s)')
