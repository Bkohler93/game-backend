#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Rebuilding services..."
docker compose build --no-cache --progress=plain