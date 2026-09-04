#!/usr/bin/env bash

NAMESPACE="checkout"
DEPLOYMENT="checkout-api"
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

# The dependency check must be fixed, not deleted: the init container has to still exist.
INIT_COUNT=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.initContainers[*].name}' | wc -w)
if [[ "$INIT_COUNT" -lt 1 ]]; then
  echo "Failure: the init container was removed instead of being fixed"
  exit 1
fi

# The database service the application relies on must still be there and reachable by its real name.
if ! kubectl get service checkout-db -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: service 'checkout-db' no longer exists"
  exit 1
fi

DB_ENDPOINTS=$(kubectl get endpoints checkout-db -n "$NAMESPACE" -o jsonpath='{.subsets[0].addresses[*].ip}')
if [[ -z "$DB_ENDPOINTS" ]]; then
  echo "Failure: service 'checkout-db' has no endpoints"
  exit 1
fi

# The init container must have actually completed (not been bypassed) on the running pod.
POD=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" \
  --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')
INIT_EXIT_CODE=$(kubectl get pod "$POD" -n "$NAMESPACE" \
  -o jsonpath='{.status.initContainerStatuses[0].state.terminated.exitCode}')
if [[ "$INIT_EXIT_CODE" != "0" ]]; then
  echo "Failure: init container on pod '$POD' did not terminate successfully (exit code: '$INIT_EXIT_CODE')"
  exit 1
fi

echo "Success: checkout-api is running, its init container completed, and checkout-db is still in place"
exit 0
