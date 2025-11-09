#!/bin/bash

# Start all services for health monitoring test

echo "Starting all health-test services..."
echo ""

# Create logs directory
mkdir -p logs

# Start web service
echo "Starting web service (port 3000)..."
cd web
npm install --silent 2>&1 > ../logs/web-install.log
nohup npm start > ../logs/web.log 2>&1 &
echo $! > ../logs/web.pid
cd ..

# Start API service
echo "Starting API service (port 5000)..."
cd api
pip install -q -r requirements.txt 2>&1 > ../logs/api-install.log
nohup python app.py > ../logs/api.log 2>&1 &
echo $! > ../logs/api.pid
cd ..

# Start database service
echo "Starting database service (port 5432)..."
cd database
npm install --silent 2>&1 > ../logs/database-install.log
nohup npm start > ../logs/database.log 2>&1 &
echo $! > ../logs/database.pid
cd ..

# Start worker service
echo "Starting worker service (background)..."
cd worker
pip install -q -r requirements.txt 2>&1 > ../logs/worker-install.log
nohup python worker.py > ../logs/worker.log 2>&1 &
echo $! > ../logs/worker.pid
cd ..

# Start admin service
echo "Starting admin service (port 4000)..."
cd admin
npm install --silent 2>&1 > ../logs/admin-install.log
nohup npm start > ../logs/admin.log 2>&1 &
echo $! > ../logs/admin.pid
cd ..

echo ""
echo "All services started!"
echo ""
echo "PIDs:"
echo "  Web:      $(cat logs/web.pid)"
echo "  API:      $(cat logs/api.pid)"
echo "  Database: $(cat logs/database.pid)"
echo "  Worker:   $(cat logs/worker.pid)"
echo "  Admin:    $(cat logs/admin.pid)"
echo ""
echo "Logs in logs/ directory"
echo ""
echo "Wait 30 seconds for services to start, then run:"
echo "  ../../../azd-app health"
echo ""
echo "To stop all services:"
echo "  ./stop-all.sh"
