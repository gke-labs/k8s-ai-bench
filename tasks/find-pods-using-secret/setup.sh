#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

NAMESPACE=payments

kubectl delete namespace ${NAMESPACE} --ignore-not-found
kubectl create namespace ${NAMESPACE}
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=60s namespace ${NAMESPACE}

kubectl apply -f "$(dirname "$0")/artifacts/manifest.yaml"

# Wait for the workloads to be observed. We don't need them to be Ready -
# the model only needs to read their specs to answer the question.
kubectl rollout status deployment/payments-api -n ${NAMESPACE} --timeout=120s || true
kubectl rollout status deployment/payments-worker -n ${NAMESPACE} --timeout=120s || true
kubectl rollout status deployment/payments-cache -n ${NAMESPACE} --timeout=120s || true
kubectl rollout status deployment/metrics-exporter -n ${NAMESPACE} --timeout=120s || true
