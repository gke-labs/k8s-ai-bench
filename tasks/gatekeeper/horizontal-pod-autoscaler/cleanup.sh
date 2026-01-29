#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-horizontal-pod-autoscaler" --ignore-not-found
