#!/usr/bin/env bash

NAMESPACE="payments"
DEPLOYMENT="payments-api"
TIMEOUT="120s"

# The deployment itself must be untouched and healthy
REPLICAS=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null)
if [[ "$REPLICAS" != "3" ]]; then
  echo "Failure: deployment '$DEPLOYMENT' should still have 3 replicas, has '$REPLICAS'"
  exit 1
fi
if ! kubectl wait --for=condition=Available deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: deployment '$DEPLOYMENT' is not available"
  exit 1
fi

PDB_COUNT=$(kubectl get pdb -n "$NAMESPACE" -o name | wc -l)
if [[ "$PDB_COUNT" -eq 0 ]]; then
  echo "Failure: no PodDisruptionBudget found in namespace '$NAMESPACE'"
  exit 1
fi

# Find a PDB whose selector actually covers the payments-api pods: the PDB controller
# reports expectedPods for the matched set, so a PDB that selects all 3 pods is the one we want.
PDB_NAME=""
for _ in $(seq 1 30); do
  PDB_NAME=$(kubectl get pdb -n "$NAMESPACE" -o json | jq -r '.items[] | select(.status.expectedPods == 3) | .metadata.name' | head -n1)
  if [[ -n "$PDB_NAME" ]]; then
    break
  fi
  sleep 2
done
if [[ -z "$PDB_NAME" ]]; then
  echo "Failure: no PodDisruptionBudget in '$NAMESPACE' selects the 3 payments-api pods"
  kubectl get pdb -n "$NAMESPACE" -o wide
  exit 1
fi

PDB_JSON=$(kubectl get pdb "$PDB_NAME" -n "$NAMESPACE" -o json)
MIN_AVAILABLE=$(echo "$PDB_JSON" | jq -r '.spec.minAvailable // empty')
MAX_UNAVAILABLE=$(echo "$PDB_JSON" | jq -r '.spec.maxUnavailable // empty')

# "At least 2 of 3 must stay up" == minAvailable 2 (or an equivalent maxUnavailable of 1)
case "$MIN_AVAILABLE|$MAX_UNAVAILABLE" in
  "2|"|"|1") ;;
  *)
    echo "Failure: PDB '$PDB_NAME' has minAvailable='$MIN_AVAILABLE' maxUnavailable='$MAX_UNAVAILABLE'; expected minAvailable=2 (or maxUnavailable=1)"
    exit 1
    ;;
esac

# With 3 healthy pods and a floor of 2, exactly 1 disruption must be allowed
echo "Waiting for the PDB status to settle..."
DISRUPTIONS_ALLOWED=""
for _ in $(seq 1 30); do
  CURRENT_HEALTHY=$(kubectl get pdb "$PDB_NAME" -n "$NAMESPACE" -o jsonpath='{.status.currentHealthy}')
  DISRUPTIONS_ALLOWED=$(kubectl get pdb "$PDB_NAME" -n "$NAMESPACE" -o jsonpath='{.status.disruptionsAllowed}')
  if [[ "$CURRENT_HEALTHY" == "3" && "$DISRUPTIONS_ALLOWED" == "1" ]]; then
    break
  fi
  sleep 2
done
if [[ "$CURRENT_HEALTHY" != "3" || "$DISRUPTIONS_ALLOWED" != "1" ]]; then
  echo "Failure: PDB '$PDB_NAME' reports currentHealthy='$CURRENT_HEALTHY' disruptionsAllowed='$DISRUPTIONS_ALLOWED'; expected 3 and 1"
  kubectl describe pdb "$PDB_NAME" -n "$NAMESPACE"
  exit 1
fi

echo "Success: PDB '$PDB_NAME' keeps at least 2 payments-api pods available"
exit 0
