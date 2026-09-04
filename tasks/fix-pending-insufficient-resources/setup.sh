#!/usr/bin/env bash
set -e

NAMESPACE="analytics"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# Deployment whose memory request cannot be satisfied by any node
kubectl apply -f artifacts/deployment.yaml

echo "Waiting for the pod to be unschedulable..."
for _ in $(seq 1 60); do
  REASON=$(kubectl get pods -n "$NAMESPACE" -l app=analytics-worker \
    -o jsonpath='{.items[0].status.conditions[?(@.type=="PodScheduled")].reason}' 2>/dev/null || true)
  if [[ "$REASON" == "Unschedulable" ]]; then
    echo "Setup successful: analytics-worker pod is Pending (Unschedulable)"
    exit 0
  fi
  sleep 2
done

echo "Error: timed out waiting for the pod to become Unschedulable"
kubectl get pods -n "$NAMESPACE"
exit 1
