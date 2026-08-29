#!/bin/bash
set -euo pipefail

PVC_NAME="storage-pvc"
NAMESPACE="resize-pv"
EXPECTED_SIZE="15Gi"
TIMEOUT="120s"

echo "Verifying PVC ${PVC_NAME} in namespace ${NAMESPACE} was resized to ${EXPECTED_SIZE}"

if ! kubectl get pvc "${PVC_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1; then
  echo "Error: PVC '${PVC_NAME}' not found in namespace '${NAMESPACE}'."
  exit 1
fi

if ! kubectl wait --for=jsonpath='{.spec.resources.requests.storage}'="${EXPECTED_SIZE}" \
  "pvc/${PVC_NAME}" -n "${NAMESPACE}" --timeout="${TIMEOUT}"; then
  ACTUAL="$(kubectl get pvc "${PVC_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.resources.requests.storage}')"
  echo "FAILURE: PVC request is '${ACTUAL}', expected '${EXPECTED_SIZE}'."
  exit 1
fi

echo "SUCCESS: PVC '${PVC_NAME}' request is '${EXPECTED_SIZE}'."
