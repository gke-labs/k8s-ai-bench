#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-memory-ratio-only" --ignore-not-found
