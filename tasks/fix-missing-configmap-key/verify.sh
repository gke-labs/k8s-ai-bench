#!/usr/bin/env bash
# Configuration
NAMESPACE="storefront"
DEPLOYMENT="checkout"
TIMEOUT="180s"
TASK_NAME="fix-missing-configmap-key"

echo "Starting verification for $TASK_NAME..."

if ! kubectl rollout status deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
    echo "Deployment '$DEPLOYMENT' did not complete its rollout within $TIMEOUT."
    kubectl describe pods -n "$NAMESPACE" -l app="$DEPLOYMENT"
    kubectl get events -n "$NAMESPACE" --sort-by=.lastTimestamp
    exit 1
fi

if ! kubectl wait --for=condition=Ready pod -l app="$DEPLOYMENT" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
    echo "Pods for '$DEPLOYMENT' did not become Ready within $TIMEOUT."
    kubectl describe pods -n "$NAMESPACE" -l app="$DEPLOYMENT"
    exit 1
fi

# The container must actually receive a resolved value, not just start.
gateway=$(kubectl exec -n "$NAMESPACE" deploy/"$DEPLOYMENT" -- printenv PAYMENT_GATEWAY_URL 2>/dev/null || true)
if [[ -z "$gateway" ]]; then
    echo "PAYMENT_GATEWAY_URL is still unresolved inside the '$DEPLOYMENT' container."
    kubectl get configmap checkout-config -n "$NAMESPACE" -o yaml
    exit 1
fi
echo "PAYMENT_GATEWAY_URL resolved to: $gateway"

echo "Verification successful for $TASK_NAME."
exit 0
