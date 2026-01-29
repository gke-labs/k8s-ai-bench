#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-replica-limit" --ignore-not-found
