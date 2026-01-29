#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-container-requests" --ignore-not-found
