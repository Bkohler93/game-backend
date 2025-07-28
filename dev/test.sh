#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

CONTAINER_NAME="my-redis-test-$(date +%s)" # Use a unique name for each test run
REDIS_PORT=6379 # Default Redis port

# Ensure the container name is clean for cleanup function
# trap cleanup EXIT is the correct mechanism to ensure cleanup runs on script exit
cleanup() {
  echo "--- Cleanup ---"
  echo "Stopping container $CONTAINER_NAME..."
  # Use 'docker stop' which is more graceful than 'docker kill'
  # Check if container exists before trying to stop/remove
  if docker ps -a --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}$"; then
    docker stop "$CONTAINER_NAME" || true # '|| true' to prevent script exit if stop fails for already stopped container
    docker rm "$CONTAINER_NAME" || true   # '|| true' to prevent script exit if rm fails
  else
    echo "Container $CONTAINER_NAME not found, nothing to clean up."
  fi
  echo "--- Cleanup Complete ---"
}

# Trap ensures cleanup() is called on script exit, even if errors occur
trap cleanup EXIT

echo "--- Starting Redis container: $CONTAINER_NAME ---"
# Run Redis container in detached mode (-d)
# We don't need a separate run_redis.sh if it's just docker run.
# If run_redis.sh does more, then adapt this.
docker run -d --name "$CONTAINER_NAME" -p "$REDIS_PORT:$REDIS_PORT" redis/redis-stack:latest

echo "Waiting for Redis to be ready (up to 10 seconds)..."
# Use a loop to check if Redis is ready inside the container
TIMEOUT=10
for i in $(seq 1 $TIMEOUT); do
  if docker exec "$CONTAINER_NAME" redis-cli ping &>/dev/null; then
    echo "Redis container is ready!"
    break
  fi
  echo "Waiting for Redis... ($i/$TIMEOUT)"
  sleep 1
  if [ "$i" -eq "$TIMEOUT" ]; then
    echo "Error: Redis container did not become ready in time."
    exit 1
  fi
done

echo "--- Running Go Tests ---"
# Run tests recursively.
# Important: ensure your Go tests are configured to connect to localhost:$REDIS_PORT
go test -v .././...

echo "--- Tests Completed Successfully ---"
# The trap will handle cleanup automatically when the script exits