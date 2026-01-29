#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-ephemeral-storage-limit" --ignore-not-found
