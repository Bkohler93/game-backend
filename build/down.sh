#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Stopping and removing containers..."
docker compose down