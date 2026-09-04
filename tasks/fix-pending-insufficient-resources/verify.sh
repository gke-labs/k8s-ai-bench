#!/usr/bin/env bash

NAMESPACE="analytics"
DEPLOYMENT="analytics-worker"
TIMEOUT="120s"

if ! kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: deployment '$DEPLOYMENT' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

echo "Waiting for deployment '$DEPLOYMENT' to become available..."
if ! kubectl wait --for=condition=Available deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: deployment '$DEPLOYMENT' is not available"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

if ! kubectl wait --for=condition=Ready pod -l app="$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: pods for '$DEPLOYMENT' are not ready"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

# There must be no leftover Pending pods from the broken revision
PENDING=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" --field-selector=status.phase=Pending -o name | wc -l)
if [[ "$PENDING" -ne 0 ]]; then
  echo "Failure: $PENDING pod(s) for '$DEPLOYMENT' are still Pending"
  exit 1
fi

# The fix should be a sane request, not simply stripping resource management: a memory request must remain.
MEMORY_REQUEST=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}')
if [[ -z "$MEMORY_REQUEST" ]]; then
  echo "Failure: the memory request was removed instead of corrected"
  exit 1
fi

echo "Success: $DEPLOYMENT is running with memory request '$MEMORY_REQUEST'"
exit 0
