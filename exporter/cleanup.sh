#!/bin/bash
echo "Stopping all Docker containers..."
docker stop $(docker ps -aq) 2>/dev/null

echo "Removing all Docker containers..."
docker rm $(docker ps -aq) 2>/dev/null

echo "Checking for process using port 2113..."
PID=$(netstat -ano | grep 2113 | awk '{print $5}' | head -n 1)

if [ ! -z "$PID" ]; then
    echo "Killing process with PID $PID using port 2113..."
    taskkill //F //PID $PID
else
    echo "No process using port 2113"
fi

echo "All containers removed and port 2113 freed!"
