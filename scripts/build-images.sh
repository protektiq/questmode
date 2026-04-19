#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Building quest-mode/backend:latest from ${ROOT}/backend"
docker build -t quest-mode/backend:latest "${ROOT}/backend"

echo "Building quest-mode/frontend:latest from ${ROOT}/frontend"
docker build -t quest-mode/frontend:latest "${ROOT}/frontend"

echo "Done. Images tagged: quest-mode/backend:latest quest-mode/frontend:latest"
