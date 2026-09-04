#!/usr/bin/env bash

NAMESPACE="storefront"
DEPLOYMENT="frontend"
GOOD_IMAGE="nginx:1.25.3"
TIMEOUT="180s"

if ! kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: deployment '$DEPLOYMENT' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

# The deployment must be back on the previous working image
IMAGE=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
if [[ "$IMAGE" != "$GOOD_IMAGE" ]]; then
  echo "Failure: deployment image is '$IMAGE', expected the previous release '$GOOD_IMAGE'"
  exit 1
fi

REPLICAS=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [[ "$REPLICAS" != "3" ]]; then
  echo "Failure: deployment should still have 3 replicas, has '$REPLICAS'"
  exit 1
fi

echo "Waiting for the rollback to complete..."
if ! kubectl rollout status deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: deployment '$DEPLOYMENT' rollout did not complete"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

READY=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
if [[ "$READY" != "3" ]]; then
  echo "Failure: expected 3 ready replicas, got '$READY'"
  exit 1
fi

# No pod from the broken release may be left behind (allow a little time for terminating pods to go away)
for _ in $(seq 1 30); do
  BAD_PODS=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" \
    -o jsonpath='{.items[*].spec.containers[0].image}' | tr ' ' '\n' | grep -vc "^${GOOD_IMAGE}$" || true)
  if [[ "$BAD_PODS" -eq 0 ]]; then
    break
  fi
  sleep 2
done
if [[ "$BAD_PODS" -ne 0 ]]; then
  echo "Failure: $BAD_PODS pod(s) are still running an image other than $GOOD_IMAGE"
  kubectl get pods -n "$NAMESPACE" -o wide
  exit 1
fi

echo "Success: $DEPLOYMENT rolled back to $GOOD_IMAGE with 3/3 replicas ready"
exit 0
