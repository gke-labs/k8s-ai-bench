#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
TASK_NAMESPACE="gk-allowed-reposv2"
kubectl delete namespace "gk-allowed-reposv2" --ignore-not-found
kubectl create namespace "gk-allowed-reposv2"
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=120s namespace "gk-allowed-reposv2"
ARTIFACTS_DIR="$(dirname "$0")/artifacts"
# Apply inventory
for file in "$ARTIFACTS_DIR"/inventory-*.yaml; do
  kubectl apply -f "$file"
done
# Apply alpha/beta resources
for file in "$ARTIFACTS_DIR"/alpha-*.yaml; do
  kubectl apply -f "$file"
done
for file in "$ARTIFACTS_DIR"/beta-*.yaml; do
  kubectl apply -f "$file"
done
kubectl get pods -n "$TASK_NAMESPACE" 2>/dev/null || true
