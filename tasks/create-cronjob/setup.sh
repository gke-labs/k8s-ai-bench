#!/usr/bin/env bash
set -e

NAMESPACE="platform-ops"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

echo "Setup completed"
