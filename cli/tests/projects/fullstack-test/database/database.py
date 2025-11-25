import time
import random
import sys
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

def execute_query(query_type, duration_ms):
    """Simulate SQL query execution"""
    query_templates = {
        'SELECT': 'SELECT * FROM {} WHERE id={}',
        'INSERT': 'INSERT INTO {} (name, data) VALUES (...)',
        'UPDATE': 'UPDATE {} SET status={} WHERE id={}',
        'DELETE': 'DELETE FROM {} WHERE expires < NOW()'
    }
    
    tables = ['users', 'orders', 'products', 'sessions', 'logs', 'analytics']
    table = random.choice(tables)
    record_id = random.randint(1, 10000)
    
    query = query_templates.get(query_type, 'SELECT').format(table, record_id)
    
    if duration_ms > 1000:
        log("WARN", f"Slow query detected: {query} ({duration_ms}ms)")
    else:
        log("INFO", f"{query} (duration: {duration_ms}ms)")

def main():
    log("INFO", "🗄️ Database Service starting...")
    log("INFO", "PostgreSQL 15.3 starting on port 5432")
    log("INFO", "Database system ready to accept connections")
    
    query_count = 0
    connection_count = 0
    
    while True:
        try:
            query_count += 1
            
            # Random query type
            query_type = random.choice(['SELECT', 'SELECT', 'SELECT', 'INSERT', 'UPDATE', 'DELETE'])
            duration = random.randint(5, 150)
            
            execute_query(query_type, duration)
            
            # Connection management
            if query_count % 10 == 0:
                connection_count = random.randint(15, 50)
                log("INFO", f"Active connections: {connection_count}/100")
            
            # Index optimization suggestions
            if query_count % 25 == 0:
                log("WARN", "Missing index on frequently queried column - performance degraded")
            
            # Transaction errors
            if query_count % 30 == 0:
                log("ERROR", "Deadlock detected: transaction rolled back")
            
            # Replication lag
            if query_count % 40 == 0:
                lag_ms = random.randint(100, 5000)
                if lag_ms > 1000:
                    log("WARN", f"Replication lag: {lag_ms}ms")
                else:
                    log("INFO", f"Replication lag: {lag_ms}ms")
            
            # Vacuum and maintenance
            if query_count % 50 == 0:
                log("INFO", "Auto-vacuum running on table: users")
            
            # Occasional errors
            if query_count % 60 == 0:
                log("ERROR", "Connection lost to replica database - failover initiated")
            
            # Wait before next query
            time.sleep(random.uniform(1.5, 3.5))
            
        except KeyboardInterrupt:
            log("INFO", "Database shutting down gracefully...")
            break
        except Exception as e:
            log("ERROR", f"Database error: {str(e)}")
            time.sleep(5)

if __name__ == '__main__':
    main()
