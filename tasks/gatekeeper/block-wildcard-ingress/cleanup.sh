#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-block-wildcard-ingress" --ignore-not-found
