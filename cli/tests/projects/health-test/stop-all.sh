#!/bin/bash

# Stop all services for health monitoring test

echo "Stopping all health-test services..."

if [ -f logs/web.pid ]; then
    kill $(cat logs/web.pid) 2>/dev/null && echo "  Stopped web service"
fi

if [ -f logs/api.pid ]; then
    kill $(cat logs/api.pid) 2>/dev/null && echo "  Stopped API service"
fi

if [ -f logs/database.pid ]; then
    kill $(cat logs/database.pid) 2>/dev/null && echo "  Stopped database service"
fi

if [ -f logs/worker.pid ]; then
    kill $(cat logs/worker.pid) 2>/dev/null && echo "  Stopped worker service"
fi

if [ -f logs/admin.pid ]; then
    kill $(cat logs/admin.pid) 2>/dev/null && echo "  Stopped admin service"
fi

# Clean up PID files
rm -f logs/*.pid

echo ""
echo "All services stopped"
