#!/usr/bin/env bash
set -e

NAMESPACE="storefront"
DEPLOYMENT="frontend"
BAD_IMAGE="nginx:1.27.0-rc.does-not-exist"
TIMEOUT="180s"

# Delete namespace if it exists and create a fresh one
kubectl delete namespace "$NAMESPACE" --ignore-not-found
kubectl create namespace "$NAMESPACE"

# Revision 1: the known-good release
kubectl apply -f artifacts/deployment-v1.yaml
kubectl annotate deployment "$DEPLOYMENT" -n "$NAMESPACE" kubernetes.io/change-cause="release 1.25.3" --overwrite

echo "Waiting for the good release to roll out..."
kubectl rollout status deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT

# Revision 2: a release with a non-existent image tag; the rollout gets stuck in ImagePullBackOff
kubectl set image deployment/"$DEPLOYMENT" -n "$NAMESPACE" frontend="$BAD_IMAGE"
kubectl annotate deployment "$DEPLOYMENT" -n "$NAMESPACE" kubernetes.io/change-cause="release 1.27.0" --overwrite

echo "Waiting for the new release to get stuck on an image pull failure..."
for _ in $(seq 1 60); do
  REASONS=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" \
    -o jsonpath='{.items[*].status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)
  if [[ "$REASONS" == *"ErrImagePull"* || "$REASONS" == *"ImagePullBackOff"* ]]; then
    echo "Setup successful: rollout of $DEPLOYMENT is stuck on $BAD_IMAGE"
    exit 0
  fi
  sleep 2
done

echo "Error: timed out waiting for the broken rollout"
kubectl get pods -n "$NAMESPACE"
exit 1
