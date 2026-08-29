#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="resize-pv"
PVC_NAME="storage-pvc"

kubectl delete namespace "${NAMESPACE}" --ignore-not-found
kubectl create namespace "${NAMESPACE}"

# Use the cluster default storage class and enable volume expansion when supported.
DEFAULT_SC="$(kubectl get sc -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}')"
if [[ -n "${DEFAULT_SC}" ]]; then
  kubectl patch storageclass "${DEFAULT_SC}" -p '{"allowVolumeExpansion": true}' --type=merge || true
fi

kubectl apply -f artifacts/storage-pvc.yaml
kubectl apply -f artifacts/storage-pod.yaml

kubectl wait --for=condition=Bound "pvc/${PVC_NAME}" -n "${NAMESPACE}" --timeout=120s
