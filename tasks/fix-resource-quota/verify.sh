#!/usr/bin/env bash

NAMESPACE="billing"
DEPLOYMENT="invoice-api"
QUOTA="billing-compute-quota"
TIMEOUT="120s"

# The quota must still exist with exactly the same limits (it must be worked within, not loosened)
if ! kubectl get resourcequota "$QUOTA" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: ResourceQuota '$QUOTA' was deleted"
  exit 1
fi

LIVE_HARD=$(kubectl get resourcequota "$QUOTA" -n "$NAMESPACE" -o json | jq -S -c '.spec.hard')
DESIRED_HARD='{"limits.cpu":"2","limits.memory":"2Gi","pods":"4","requests.cpu":"1","requests.memory":"1Gi"}'
if [[ "$LIVE_HARD" != "$DESIRED_HARD" ]]; then
  echo "Failure: ResourceQuota '$QUOTA' was modified"
  echo "  live:     $LIVE_HARD"
  echo "  expected: $DESIRED_HARD"
  exit 1
fi

# There must be no other quota that could have loosened things either
QUOTA_COUNT=$(kubectl get resourcequota -n "$NAMESPACE" -o name | wc -l)
if [[ "$QUOTA_COUNT" -ne 1 ]]; then
  echo "Failure: expected exactly 1 ResourceQuota in '$NAMESPACE', found $QUOTA_COUNT"
  exit 1
fi

if ! kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: deployment '$DEPLOYMENT' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

REPLICAS=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [[ "$REPLICAS" != "2" ]]; then
  echo "Failure: deployment '$DEPLOYMENT' should have 2 replicas, has '$REPLICAS'"
  exit 1
fi

echo "Waiting for deployment '$DEPLOYMENT' to roll out..."
if ! kubectl rollout status deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: deployment '$DEPLOYMENT' did not roll out"
  kubectl get pods -n "$NAMESPACE"
  kubectl get events -n "$NAMESPACE" --field-selector reason=FailedCreate
  exit 1
fi

READY=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
if [[ "$READY" != "2" ]]; then
  echo "Failure: expected 2 ready replicas, got '$READY'"
  exit 1
fi

# Every container in the pod template must declare cpu/memory requests and limits (what the quota demands)
for FIELD in requests.cpu requests.memory limits.cpu limits.memory; do
  VALUE=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath="{.spec.template.spec.containers[0].resources.${FIELD}}")
  if [[ -z "$VALUE" ]]; then
    echo "Failure: container does not declare resources.$FIELD"
    exit 1
  fi
done

echo "Success: $DEPLOYMENT has 2 ready replicas within the unchanged ResourceQuota '$QUOTA'"
exit 0
