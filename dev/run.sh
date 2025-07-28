#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
CONTAINER_NAME="my-redis-dev" # This name is fine for a persistent dev container
REDIS_PORT=6379              # Default Redis port
REDIS_IMAGE="redis/redis-stack:latest" # Image for Redis with RediSearch/RedisJSON
REDIS_READY_TIMEOUT=15       # Seconds to wait for Redis to be ready

# --- Utility Function (Keep as is, it's good for dev logs) ---
run_service() {
  name=$1
  shift
  # Run the remaining args as the command in background,
  # redirect stderr to stdout, and prefix output lines.
  ( "$@" 2>&1 | sed "s/^/[$name] /" ) &
}

# --- Cleanup Function ---
cleanup() {
  echo "" # Newline for cleaner output after Ctrl+C
  echo "--- Cleanup ---"
  echo "Stopping container $CONTAINER_NAME..."
  # Check if container exists before trying to stop/remove
  if docker ps -a --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
    docker stop "$CONTAINER_NAME" || true # '|| true' prevents script exit if stop fails
    docker rm "$CONTAINER_NAME" || true   # '|| true' prevents script exit if rm fails
  else
    echo "Container $CONTAINER_NAME not found, nothing to clean up."
  fi
  echo "--- Cleanup Complete ---"
}

# Trap ensures cleanup() is called on script exit (e.g., Ctrl+C, or script error)
trap cleanup EXIT

# --- Service Startup ---

echo "--- Preparing Redis container ---"
# Check if container is already running
if docker ps --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
  echo "Container $CONTAINER_NAME is already running."
elif docker ps -a --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
  echo "Container $CONTAINER_NAME exists but is stopped. Starting it..."
  docker start "$CONTAINER_NAME"
else
  echo "Container $CONTAINER_NAME not found. Creating and starting new one..."
  # Run Redis container in detached mode. This command returns quickly.
  # If run_redis.sh does more than just 'docker run -d', you might need to
  # adjust this. But for a simple docker run, this is better.
  docker run -d --name "$CONTAINER_NAME" -p "$REDIS_PORT:$REDIS_PORT" "$REDIS_IMAGE"
fi

echo "Waiting for Redis to be ready (up to $REDIS_READY_TIMEOUT seconds)..."
# Use a loop to check if Redis is ready inside the container
for i in $(seq 1 "$REDIS_READY_TIMEOUT"); do
  # Use docker exec to ping Redis inside the container
  if docker exec "$CONTAINER_NAME" redis-cli ping &>/dev/null; then
    echo "Redis container is ready!"
    break
  fi
  echo "Waiting for Redis... ($i/$REDIS_READY_TIMEOUT)"
  sleep 1
  if [ "$i" -eq "$REDIS_READY_TIMEOUT" ]; then
    echo "Error: Redis container did not become ready in time. Exiting."
    exit 1 # Exit the script with an error code
  fi
done

# Now run your other services.
# `run_service` will put them in the background, their logs will be prefixed.
echo "--- Starting other services ---"
run_service "gateway" ./matchmake/run_gateway.sh
run_service "matchmaker" ./matchmake/run_matchmaker.sh

echo "--- All services started. Press Ctrl+C to stop. ---"

# Wait for all background processes.
# This keeps the script running and allows log output from services.
# When you Ctrl+C, 'wait' is interrupted, and the 'cleanup' trap is activated.
wait