#!/bin/bash

set -e

CONTAINER_NAME="my-redis-dev" 
REDIS_PORT=6379              
REDIS_IMAGE="redis/redis-stack:latest" 
REDIS_READY_TIMEOUT=15       

run_service() {
  name=$1
  shift
  ( "$@" 2>&1 | sed "s/^/[$name] /" ) &
}

cleanup() {
  echo "" 
  echo "--- Cleanup ---"
  
  echo "Stopping container $CONTAINER_NAME..."
  if docker ps -a --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
    docker stop "$CONTAINER_NAME" || true 
    docker rm "$CONTAINER_NAME" || true  
  else
    echo "Container $CONTAINER_NAME not found, nothing to clean up."
  fi
  echo "--- Cleanup Complete ---"
}

trap cleanup EXIT

echo "--- Preparing Redis container ---"
if docker ps --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
  echo "Container $CONTAINER_NAME is already running."
elif docker ps -a --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
  echo "Container $CONTAINER_NAME exists but is stopped. Starting it..."
  docker start "$CONTAINER_NAME"
else
  echo "Container $CONTAINER_NAME not found. Creating and starting new one..."
  docker run -d --name "$CONTAINER_NAME" -p "$REDIS_PORT:$REDIS_PORT" "$REDIS_IMAGE"
fi

echo "Waiting for Redis to be ready (up to $REDIS_READY_TIMEOUT seconds)..."
for i in $(seq 1 "$REDIS_READY_TIMEOUT"); do
  if docker exec "$CONTAINER_NAME" redis-cli ping &>/dev/null; then
    echo "Redis container is ready!"
    break
  fi
  echo "Waiting for Redis... ($i/$REDIS_READY_TIMEOUT)"
  sleep 1
  if [ "$i" -eq "$REDIS_READY_TIMEOUT" ]; then
    echo "Error: Redis container did not become ready in time. Exiting."
    exit 1 
  fi
done

echo "--- Starting other services ---"
run_service "gateway" ./matchmake/run_gateway.sh
run_service "matchmaker" ./matchmake/run_matchmaker.sh

echo "--- All services started. Press Ctrl+C to stop. ---"

wait