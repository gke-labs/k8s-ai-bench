#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-tls-required" --ignore-not-found
