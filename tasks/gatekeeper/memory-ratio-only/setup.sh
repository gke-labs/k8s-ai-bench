#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
TASK_NAMESPACE="gk-memory-ratio-only"
kubectl delete namespace "gk-memory-ratio-only" --ignore-not-found
kubectl create namespace "gk-memory-ratio-only"
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=120s namespace "gk-memory-ratio-only"
ARTIFACTS_DIR="$(dirname "$0")/artifacts"
# Apply alpha/beta pod resources
for file in "$ARTIFACTS_DIR"/alpha-*.yaml; do
  kubectl apply -f "$file"
done
for file in "$ARTIFACTS_DIR"/beta-*.yaml; do
  kubectl apply -f "$file"
done
kubectl get pods -n "$TASK_NAMESPACE" 2>/dev/null || true
