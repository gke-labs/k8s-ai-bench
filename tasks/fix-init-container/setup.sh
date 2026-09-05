#!/usr/bin/env bash
set -e

NAMESPACE="checkout"
TIMEOUT="120s"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# Database stand-in, its service, and the API whose init container waits on the wrong hostname
kubectl apply -f artifacts/manifest.yaml

# The database must be up so that the only problem is the API's init container
echo "Waiting for the database to be ready..."
kubectl wait --for=condition=Available deployment/checkout-db -n "$NAMESPACE" --timeout=$TIMEOUT

# The API pod should be stuck in Init (the init container never completes)
echo "Waiting for the checkout-api pod to be stuck in its init container..."
for _ in $(seq 1 60); do
  INIT_READY=$(kubectl get pods -n "$NAMESPACE" -l app=checkout-api \
    -o jsonpath='{.items[0].status.initContainerStatuses[0].state.running}' 2>/dev/null || true)
  if [[ -n "$INIT_READY" ]]; then
    echo "Setup successful: checkout-api is stuck waiting in its init container"
    exit 0
  fi
  sleep 2
done

echo "Error: timed out waiting for the checkout-api init container to start"
kubectl get pods -n "$NAMESPACE"
exit 1
