#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Parse command line arguments with getopt
VALID_ARGS=$(getopt -o vcp: --long verbose,coverage,coverage-profile: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1
fi

CONTAINER_NAME="my-redis-test-$(date +%s)" # Use a unique name for each test run
REDIS_PORT=6379 # Default Redis port

# calling parameters
COVERAGE_FILE="coverage.out"
APPLY_COVERAGE=0
APPLY_VERBOSE=0
APPLY_COVERAGE_PROFILE=0

eval set -- "$VALID_ARGS"
while true; do
  case "$1" in
    -v|--verbose)
      APPLY_VERBOSE=1
      shift
      ;;
    -c|--coverage)
      APPLY_COVERAGE=1
      shift
      ;;
    -p|--coverage-profile)
      APPLY_COVERAGE_PROFILE=1
      COVERAGE_FILE="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
  esac
done

cleanup() {
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

echo "--- Starting Redis container: $CONTAINER_NAME ---"
docker run -d \
  --name "$CONTAINER_NAME" \
  -p "$REDIS_PORT:$REDIS_PORT" \
  redis/redis-stack:latest

echo "Waiting for Redis to be ready (up to 10 seconds)..."
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

echo "--- Setting Env Variables ---"
export ENV="PROD"
export PORT=8083
export REDIS_ADDR="localhost:$REDIS_PORT"

echo "--- Running Go Tests ---"

GO_TEST_CMD=(go test .././...)

if [[ $APPLY_VERBOSE -eq 1 ]]; then
  GO_TEST_CMD+=(-v)
fi

if [[ $APPLY_COVERAGE -eq 1 ]]; then
  GO_TEST_CMD+=(-cover -coverprofile=coverage.out)
fi

if [[ $APPLY_COVERAGE_PROFILE -eq 1 ]]; then
  if [[ $APPLY_COVERAGE -eq 0 ]]; then
    echo "Error: --coverage or -c for code coverage must be applied along with coverage profile" >&2 # Redirect to standard error
    exit 1
  fi
  GO_TEST_CMD+=(-coverprofile="$COVERAGE_FILE")
fi

# Append any extra arguments
GO_TEST_CMD+=("$@")

echo "Executing: ${GO_TEST_CMD[*]}"
"${GO_TEST_CMD[@]}"

if [[ $APPLY_COVERAGE -eq 1 ]]; then
  echo "--- Coverage Summary ---"
  go tool cover -func=coverage.out | grep total
fi

echo "--- Tests Completed Successfully ---"
