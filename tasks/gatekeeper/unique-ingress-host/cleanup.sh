#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-unique-ingress-host" --ignore-not-found
