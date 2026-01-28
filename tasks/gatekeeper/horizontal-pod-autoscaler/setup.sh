#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
TASK_NAMESPACE="gk-horizontal-pod-autoscaler"
kubectl delete namespace "gk-horizontal-pod-autoscaler" --ignore-not-found
kubectl create namespace "gk-horizontal-pod-autoscaler"
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=120s namespace "gk-horizontal-pod-autoscaler"
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

