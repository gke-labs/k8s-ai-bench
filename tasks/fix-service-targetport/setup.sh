#!/usr/bin/env bash
set -e

NAMESPACE="catalog"
TIMEOUT="120s"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# Deployment and a Service whose targetPort does not match the container port
kubectl apply -f artifacts/manifest.yaml

echo "Waiting for the catalog-api pods to be ready..."
kubectl wait --for=condition=Available deployment/catalog-api -n "$NAMESPACE" --timeout=$TIMEOUT

# The service must have endpoints: the failure is the port, not the selector
echo "Waiting for the service to have endpoints..."
for _ in $(seq 1 30); do
  ENDPOINTS=$(kubectl get endpoints catalog-api -n "$NAMESPACE" -o jsonpath='{.subsets[0].addresses[*].ip}' 2>/dev/null || true)
  if [[ -n "$ENDPOINTS" ]]; then
    echo "Setup successful: service has endpoints ($ENDPOINTS) but targets the wrong port"
    exit 0
  fi
  sleep 2
done

echo "Error: timed out waiting for service endpoints"
kubectl get endpoints -n "$NAMESPACE"
exit 1
