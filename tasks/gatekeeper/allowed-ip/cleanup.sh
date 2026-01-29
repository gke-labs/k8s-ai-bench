#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-allowed-ip" --ignore-not-found
