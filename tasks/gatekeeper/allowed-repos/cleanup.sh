#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-allowed-repos" --ignore-not-found
