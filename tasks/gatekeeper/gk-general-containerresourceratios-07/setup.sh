#!/usr/bin/env bash
set -e
kubectl delete namespace gk-test-007 --ignore-not-found --wait=true
kubectl create namespace gk-test-007
sleep 2  # Allow namespace to stabilize
kubectl apply -f artifacts/resource-alpha.yaml -n gk-test-007
kubectl apply -f artifacts/resource-beta.yaml -n gk-test-007
sleep 3  # Allow pods to be scheduled
echo "Waiting for resources to be ready..."
kubectl wait --for=condition=Ready pod/resource-alpha -n gk-test-007 --timeout=180s
kubectl wait --for=condition=Ready pod/resource-beta -n gk-test-007 --timeout=180s
