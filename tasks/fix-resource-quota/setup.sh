#!/usr/bin/env bash
set -e

NAMESPACE="billing"
DEPLOYMENT="invoice-api"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# The quota must exist before the deployment so that pod creation is rejected
kubectl apply -f artifacts/quota.yaml

# Wait for the quota controller to populate usage; until then all creations are refused anyway
echo "Waiting for the ResourceQuota to be initialized..."
for _ in $(seq 1 30); do
  USED=$(kubectl get resourcequota billing-compute-quota -n "$NAMESPACE" -o jsonpath='{.status.used.pods}' 2>/dev/null || true)
  if [[ -n "$USED" ]]; then
    break
  fi
  sleep 1
done

kubectl apply -f artifacts/deployment.yaml

# The ReplicaSet should be failing to create pods because of the quota
echo "Waiting for the ReplicaSet to report quota failures..."
for _ in $(seq 1 60); do
  FAILED_CREATE=$(kubectl get events -n "$NAMESPACE" --field-selector reason=FailedCreate \
    -o jsonpath='{.items[*].message}' 2>/dev/null || true)
  if [[ "$FAILED_CREATE" == *"failed quota"* ]]; then
    echo "Setup successful: pod creation for $DEPLOYMENT is being rejected by the quota"
    exit 0
  fi
  sleep 2
done

echo "Error: timed out waiting for quota-related FailedCreate events"
kubectl get events -n "$NAMESPACE"
exit 1
