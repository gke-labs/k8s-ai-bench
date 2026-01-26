#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
TASK_NAMESPACE="gk-required-probes"
kubectl delete namespace "gk-required-probes" --ignore-not-found
kubectl create namespace "gk-required-probes"
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=120s namespace "gk-required-probes"
ARTIFACTS_DIR="$(dirname "$0")/artifacts"
# Apply alpha/beta pod resources
for file in "$ARTIFACTS_DIR"/alpha-*.yaml; do
  kubectl apply -f "$file"
done
for file in "$ARTIFACTS_DIR"/beta-*.yaml; do
  kubectl apply -f "$file"
done
kubectl get pods -n "$TASK_NAMESPACE" 2>/dev/null || true
