#!/usr/bin/env bash

NAMESPACE="catalog"
SERVICE="catalog-api"
TIMEOUT="120s"

if ! kubectl get service "$SERVICE" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: service '$SERVICE' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

# Clients call port 80 on the service; that contract must not change.
SERVICE_PORT=$(kubectl get service "$SERVICE" -n "$NAMESPACE" -o jsonpath='{.spec.ports[0].port}')
if [[ "$SERVICE_PORT" != "80" ]]; then
  echo "Failure: service port changed to '$SERVICE_PORT'; clients expect port 80"
  exit 1
fi

echo "Waiting for the catalog-api pods to be ready..."
if ! kubectl wait --for=condition=Ready pod -l app=catalog-api -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: catalog-api pods are not ready"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

ENDPOINTS=$(kubectl get endpoints "$SERVICE" -n "$NAMESPACE" -o jsonpath='{.subsets[0].addresses[*].ip}')
if [[ -z "$ENDPOINTS" ]]; then
  echo "Failure: service '$SERVICE' has no endpoints"
  exit 1
fi

# The real test: an HTTP request through the service must succeed.
echo "Checking HTTP connectivity through the service..."
RESPONSE=$(kubectl run -n "$NAMESPACE" verify-client --image=busybox:1.36 --restart=Never --rm -i --quiet \
  --wait --timeout=180s -- \
  sh -c 'for i in $(seq 1 10); do wget -qO- --timeout=5 http://catalog-api.catalog.svc.cluster.local:80/ && exit 0; sleep 3; done; exit 1' 2>/dev/null)

if [[ "$RESPONSE" != *"nginx"* ]]; then
  echo "Failure: HTTP request through service '$SERVICE' did not succeed"
  kubectl describe service "$SERVICE" -n "$NAMESPACE"
  exit 1
fi

echo "Success: service '$SERVICE' routes traffic to the pods on port 80"
exit 0
