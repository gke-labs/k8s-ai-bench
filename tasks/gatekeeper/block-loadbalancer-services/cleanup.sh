#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-block-loadbalancer-services" --ignore-not-found
