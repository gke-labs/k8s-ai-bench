#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-pod-disruption-budget" --ignore-not-found
