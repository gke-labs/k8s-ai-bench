#!/usr/bin/env bash
set -e

NAMESPACE="payments"
TIMEOUT="120s"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

kubectl apply -f artifacts/deployment.yaml

echo "Waiting for payments-api to be ready..."
kubectl wait --for=condition=Available deployment/payments-api -n "$NAMESPACE" --timeout=$TIMEOUT

echo "Setup completed"
