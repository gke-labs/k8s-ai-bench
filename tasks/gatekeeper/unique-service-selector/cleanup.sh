#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-unique-service-selector" --ignore-not-found
