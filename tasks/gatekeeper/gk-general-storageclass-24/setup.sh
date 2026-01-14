#!/usr/bin/env bash
set -e
kubectl delete namespace gk-test-024 --ignore-not-found --wait=true
kubectl create namespace gk-test-024
sleep 2  # Allow namespace to stabilize
kubectl apply -f artifacts/resource-alpha.yaml -n gk-test-024
kubectl apply -f artifacts/resource-beta.yaml -n gk-test-024
sleep 3  # Allow pods to be scheduled
echo "Resources deployed. Waiting for stability..."
sleep 5
