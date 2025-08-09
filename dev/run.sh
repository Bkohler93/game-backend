#!/bin/bash
set -e

cd "$(dirname "$0")/.."

MATCHMAKE_CONTAINER_NAME="matchmake-redis-test-$(date +%s)"
GAMEPLAY_CONTAINER_NAME="gameplay-redis-test-$(date +%s)"
REDIS_MATCHMAKE_PORT=6379              
REDIS_GAMEPLAY_PORT=6380
REDIS_IMAGE="redis/redis-stack:latest" 
REDIS_READY_TIMEOUT=15       

run_service() {
  name=$1
  shift
  ( "$@" 2>&1 | sed "s/^/[$name] /" ) &
}

cleanup() {
  echo "--- Cleanup ---"
  echo "Stopping container $MATCHMAKE_CONTAINER_NAME..."
  if docker ps -a --format '{{.Names}}' | grep -Eq "^${MATCHMAKE_CONTAINER_NAME}$"; then
    docker stop "$MATCHMAKE_CONTAINER_NAME" || true
    docker rm "$MATCHMAKE_CONTAINER_NAME" || true
  else
    echo "Container $MATCHMAKE_CONTAINER_NAME not found, nothing to clean up."
  fi
  echo "Stopping container $GAMEPLAY_CONTAINER_NAME..."
    if docker ps -a --format '{{.Names}}' | grep -Eq "^${GAMEPLAY_CONTAINER_NAME}$"; then
      docker stop "$GAMEPLAY_CONTAINER_NAME" || true
      docker rm "$GAMEPLAY_CONTAINER_NAME" || true
    else
      echo "Container $GAMEPLAY_CONTAINER_NAME not found, nothing to clean up."
    fi
  echo "--- Cleanup Complete ---"
}

trap cleanup EXIT

echo "--- Preparing Matchmake Redis container ---"
if docker ps --format '{{.Names}}' | grep -Eq "^${MATCHMAKE_CONTAINER_NAME}$"; then
  echo "Container $MATCHMAKE_CONTAINER_NAME is already running."
elif docker ps -a --format '{{.Names}}' | grep -Eq "^${MATCHMAKE_CONTAINER_NAME}$"; then
  echo "Container $MATCHMAKE_CONTAINER_NAME exists but is stopped. Starting it..."
  docker start "$MATCHMAKE_CONTAINER_NAME"
else
  echo "Container $MATCHMAKE_CONTAINER_NAME not found. Creating and starting new one..."
  docker run -d --name "$MATCHMAKE_CONTAINER_NAME" -p "$REDIS_MATCHMAKE_PORT:$REDIS_MATCHMAKE_PORT" "$REDIS_IMAGE"
fi

echo "Waiting for Redis to be ready (up to $REDIS_READY_TIMEOUT seconds)..."
for i in $(seq 1 "$REDIS_READY_TIMEOUT"); do
  if docker exec "$MATCHMAKE_CONTAINER_NAME" redis-cli ping &>/dev/null; then
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

echo "--- Preparing Gameplay Redis container ---"
if docker ps --format '{{.Names}}' | grep -Eq "^${GAMEPLAY_CONTAINER_NAME}$"; then
  echo "Container $GAMEPLAY_CONTAINER_NAME is already running."
elif docker ps -a --format '{{.Names}}' | grep -Eq "^${GAMEPLAY_CONTAINER_NAME}$"; then
  echo "Container $GAMEPLAY_CONTAINER_NAME exists but is stopped. Starting it..."
  docker start "$GAMEPLAY_CONTAINER_NAME"
else
  echo "Container $GAMEPLAY_CONTAINER_NAME not found. Creating and starting new one..."
  docker run -d --name "$GAMEPLAY_CONTAINER_NAME" -p "$REDIS_GAMEPLAY_PORT:$REDIS_GAMEPLAY_PORT" "$REDIS_IMAGE"
fi

echo "Waiting for Redis to be ready (up to $REDIS_READY_TIMEOUT seconds)..."
for i in $(seq 1 "$REDIS_READY_TIMEOUT"); do
  if docker exec "$GAMEPLAY_CONTAINER_NAME" redis-cli ping &>/dev/null; then
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