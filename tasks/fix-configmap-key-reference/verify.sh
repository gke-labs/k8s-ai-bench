#!/usr/bin/env bash

NAMESPACE="notifications"
DEPLOYMENT="notification-service"
EXPECTED_SMTP_HOST="smtp.internal.example.com"
TIMEOUT="120s"

# The deployment must exist and become available
if ! kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: deployment '$DEPLOYMENT' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

echo "Waiting for deployment '$DEPLOYMENT' to become available..."
if ! kubectl wait --for=condition=Available deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: deployment '$DEPLOYMENT' is not available"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

if ! kubectl wait --for=condition=Ready pod -l app="$DEPLOYMENT" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: pods for '$DEPLOYMENT' are not ready"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

# The application still has to receive the SMTP host from configuration.
# This accepts either fixing the key reference in the Deployment or renaming the key in the ConfigMap.
POD=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" \
  --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')
if [[ -z "$POD" ]]; then
  echo "Failure: no running pod found for '$DEPLOYMENT'"
  exit 1
fi

ACTUAL_SMTP_HOST=$(kubectl exec -n "$NAMESPACE" "$POD" -- sh -c 'printf "%s" "$SMTP_HOST"' 2>/dev/null)
if [[ "$ACTUAL_SMTP_HOST" != "$EXPECTED_SMTP_HOST" ]]; then
  echo "Failure: SMTP_HOST inside the container is '$ACTUAL_SMTP_HOST', expected '$EXPECTED_SMTP_HOST'"
  echo "The env var must still be sourced from the configuration, not removed or hardcoded to something else."
  exit 1
fi

# The value must still come from the ConfigMap (not a hardcoded literal in the pod spec)
SOURCE_CM=$(kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="SMTP_HOST")].valueFrom.configMapKeyRef.name}')
if [[ -z "$SOURCE_CM" ]]; then
  echo "Failure: SMTP_HOST is no longer sourced from a ConfigMap"
  exit 1
fi

echo "Success: deployment is available and SMTP_HOST is correctly resolved from ConfigMap '$SOURCE_CM'"
exit 0
