#!/usr/bin/env bash
set -e

NAMESPACE="notifications"
DEPLOYMENT="notification-service"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# ConfigMap + Deployment whose env var references a ConfigMap key that does not exist
kubectl apply -f artifacts/manifest.yaml

# Wait for the pod to hit CreateContainerConfigError (the kubelet cannot resolve the env var)
echo "Waiting for the pod to fail with CreateContainerConfigError..."
for _ in $(seq 1 60); do
  REASON=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" \
    -o jsonpath='{.items[0].status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)
  if [[ "$REASON" == "CreateContainerConfigError" ]]; then
    echo "Setup successful: pod is in CreateContainerConfigError"
    exit 0
  fi
  sleep 2
done

echo "Error: timed out waiting for the pod to enter CreateContainerConfigError"
kubectl get pods -n "$NAMESPACE"
exit 1
