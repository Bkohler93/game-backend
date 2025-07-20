#!/bin/bash

set -e
CONTAINER_NAME=my-redis-dev

run_service() {
  name=$1
  shift
  # Run the remaining args as the command
  ( "$@" 2>&1 | sed "s/^/[$name] /" ) &
}

# Run each service
run_service "redis" ./matchmake/run_redis.sh "$CONTAINER_NAME"
sleep 1
run_service "match_gateway" ./matchmake/run_gateway.sh
run_service "matchmaker" ./matchmake/run_matchmaker.sh

cleanup() {
  echo "Stopping container $CONTAINER_NAME..."
  docker stop "$CONTAINER_NAME"
  docker remove "$CONTAINER_NAME"
}
trap cleanup EXIT

# Wait for all background processes
wait
