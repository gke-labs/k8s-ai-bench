#!/usr/bin/env bash
set -e

NAMESPACE="reporting"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# A suspended CronJob that never schedules anything
kubectl apply -f artifacts/cronjob.yaml

echo "Setup completed"
