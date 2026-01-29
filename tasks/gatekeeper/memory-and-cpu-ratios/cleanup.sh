#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-memory-and-cpu-ratios" --ignore-not-found
