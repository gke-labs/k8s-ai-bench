#!/usr/bin/env bash
set -e

NAMESPACE="inventory"
TIMEOUT="120s"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# Headless service + StatefulSet whose serviceName does not match it
kubectl apply -f artifacts/manifest.yaml

echo "Waiting for the db-0 pod to be ready..."
for _ in $(seq 1 30); do
  if kubectl get pod db-0 -n "$NAMESPACE" &>/dev/null; then
    break
  fi
  sleep 2
done
kubectl wait --for=condition=Ready pod/db-0 -n "$NAMESPACE" --timeout=$TIMEOUT

echo "Setup completed: db-0 is running but has no db-0.inventory-db DNS record"
