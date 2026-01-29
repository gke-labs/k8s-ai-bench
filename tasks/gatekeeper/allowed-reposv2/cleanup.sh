#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-allowed-reposv2" --ignore-not-found
