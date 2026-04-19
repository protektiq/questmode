#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

eval "$(minikube docker-env)"

echo "Building quest-mode/backend:latest from ${ROOT}/backend"
docker build -t quest-mode/backend:latest "${ROOT}/backend"

echo "Building quest-mode/frontend:latest from ${ROOT}/frontend"
docker build -t quest-mode/frontend:latest "${ROOT}/frontend"

kubectl rollout restart deployment/frontend deployment/backend -n quest-mode
