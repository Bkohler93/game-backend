#!/bin/bash
set -e

cd "$(dirname "$0")/.."

VALID_ARGS=$(getopt -o vcp:r: --long verbose,coverage,coverage-profile:,run: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1
fi

MATCHMAKE_CONTAINER_NAME="matchmake-redis-test-$(date +%s)"
GAMEPLAY_CONTAINER_NAME="gameplay-redis-test-$(date +%s)"

REDIS_MATCHMAKE_PORT=6379              
REDIS_GAMEPLAY_PORT=6380

COVERAGE_FILE="coverage.out"
APPLY_COVERAGE=0
APPLY_VERBOSE=0
APPLY_COVERAGE_PROFILE=0
APPLY_RUN=0
RUN_COMMAND=""

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
    -r|--run)
      APPLY_RUN=1
      RUN_COMMAND="$2"
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

echo "--- Starting Matchmake Redis container: $MATCHMAKE_CONTAINER_NAME ---"
docker run -d \
  --name "$MATCHMAKE_CONTAINER_NAME" \
  -p "$REDIS_MATCHMAKE_PORT:$REDIS_MATCHMAKE_PORT" \
  redis/redis-stack:latest
  
echo "--- Starting Gameplay Redis container: $GAMEPLAY_CONTAINER_NAME ---"
docker run -d \
  --name "$GAMEPLAY_CONTAINER_NAME" \
  -p "$REDIS_GAMEPLAY_PORT:$REDIS_GAMEPLAY_PORT" \
  redis/redis-stack:latest

echo "Waiting for Redis to be ready (up to 10 seconds)..."
TIMEOUT=10
for i in $(seq 1 $TIMEOUT); do
  if docker exec "$GAMEPLAY_CONTAINER_NAME" redis-cli ping &>/dev/null; then
    echo "Redis containers are ready!"
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
export REDIS_MATCHMAKE_ADDR="localhost:$REDIS_MATCHMAKE_PORT"
export REDIS_GAMEPLAY_ADDR="localhost:$REDIS_GAMEPLAY_PORT"

echo "--- Running Go Tests ---"

GO_TEST_CMD=(go test ./...)

if [[ $APPLY_VERBOSE -eq 1 ]]; then
  GO_TEST_CMD+=(-v)
fi

if [[ $APPLY_COVERAGE -eq 1 ]]; then
  GO_TEST_CMD+=(-cover)
fi

if [[ $APPLY_COVERAGE_PROFILE -eq 1 ]]; then
  if [[ $APPLY_COVERAGE -eq 0 ]]; then
    echo "Error: --coverage or -c for code coverage must be applied along with coverage profile" >&2 
    exit 1
  fi
  GO_TEST_CMD+=(-coverprofile="$COVERAGE_FILE")
fi

if [[ $APPLY_RUN -eq 1 ]]; then
  GO_TEST_CMD+=(-run "$RUN_COMMAND")
fi

GO_TEST_CMD+=("$@")

echo "Executing: ${GO_TEST_CMD[*]}"
"${GO_TEST_CMD[@]}"

if [[ $APPLY_COVERAGE -eq 1 ]]; then
  echo "--- Coverage Summary ---"
  go tool cover -func=coverage.out | grep total
fi

echo "--- Tests Completed Successfully ---"
