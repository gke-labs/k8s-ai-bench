#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-container-cpu-requests-memory-limits-and-requests" --ignore-not-found
