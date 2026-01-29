#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-container-limits" --ignore-not-found
