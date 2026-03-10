#!/usr/bin/env bash
set -e
NAMESPACE="canary-deployment-ns"

# Delete the namespace if it exists to ensure a clean state
kubectl delete namespace limits-test --ignore-not-found

# Create the namespace
kubectl create namespace $NAMESPACE

# Apply the initial stable deployment and the service pointing only to it
kubectl apply -n $NAMESPACE -f artifacts/deployment-v1.yaml
kubectl apply -n $NAMESPACE -f artifacts/service.yaml
