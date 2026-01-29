#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-required-probes" --ignore-not-found
