#!/bin/bash
set -e

cd "$(dirname "$0")/.."

ENVIRONMENT=${ENV:-DEV}

echo "Starting Docker Compose in $ENVIRONMENT mode..."
ENV=$ENVIRONMENT docker compose up --build