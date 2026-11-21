#!/usr/bin/env bash
# Configuration
NAMESPACE="analytics"
API_DEPLOYMENT="api"
DB_SERVICE="postgres"
TIMEOUT="180s"
TASK_NAME="fix-init-container-dependency"

echo "Starting verification for $TASK_NAME..."

# The dependency service must actually select a backing pod, otherwise the
# init container can never resolve it.
endpoints=$(kubectl get endpoints "$DB_SERVICE" -n "$NAMESPACE" -o jsonpath='{.subsets[0].addresses[*].ip}')
if [[ -z "$endpoints" ]]; then
    echo "Service '$DB_SERVICE' in namespace '$NAMESPACE' has no endpoints."
    kubectl get endpoints "$DB_SERVICE" -n "$NAMESPACE" -o yaml
    kubectl get pods -n "$NAMESPACE" --show-labels
    exit 1
fi
echo "Service '$DB_SERVICE' has endpoints: $endpoints"

if ! kubectl rollout status deployment/"$API_DEPLOYMENT" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
    echo "Deployment '$API_DEPLOYMENT' did not complete its rollout within $TIMEOUT."
    echo "--- pods"
    kubectl describe pods -n "$NAMESPACE" -l app="$API_DEPLOYMENT"
    echo "--- events"
    kubectl get events -n "$NAMESPACE" --sort-by=.lastTimestamp
    exit 1
fi

# Every replica must be Ready, not merely created.
if ! kubectl wait --for=condition=Ready pod -l app="$API_DEPLOYMENT" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
    echo "Pods for '$API_DEPLOYMENT' did not become Ready within $TIMEOUT."
    kubectl describe pods -n "$NAMESPACE" -l app="$API_DEPLOYMENT"
    exit 1
fi

ready=$(kubectl get deployment "$API_DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
desired=$(kubectl get deployment "$API_DEPLOYMENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
if [[ "${ready:-0}" != "$desired" ]]; then
    echo "Only ${ready:-0} of $desired replicas of '$API_DEPLOYMENT' are Ready."
    exit 1
fi

echo "All $desired replicas of '$API_DEPLOYMENT' are Ready. Verification successful for $TASK_NAME."
exit 0
