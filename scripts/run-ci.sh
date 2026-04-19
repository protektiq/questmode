#!/usr/bin/env bash
set -euo pipefail

kubectl delete pipelinerun -n quest-mode -l app=quest-mode-ci --ignore-not-found
kubectl create -f k8s/tekton/pipeline-run.yaml -n quest-mode
tkn pipelinerun logs -n quest-mode --last -f
