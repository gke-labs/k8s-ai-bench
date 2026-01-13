#!/usr/bin/env bash
set -e
kubectl delete namespace gk-test-015 --ignore-not-found --wait=true
kubectl create namespace gk-test-015
sleep 2  # Allow namespace to stabilize
kubectl apply -f artifacts/resource-alpha.yaml -n gk-test-015
kubectl apply -f artifacts/resource-beta.yaml -n gk-test-015
sleep 3  # Allow pods to be scheduled
echo "Waiting for resources to be ready..."
kubectl wait --for=condition=Available deployment/resource-alpha -n gk-test-015 --timeout=180s
kubectl wait --for=condition=Available deployment/resource-beta -n gk-test-015 --timeout=180s
