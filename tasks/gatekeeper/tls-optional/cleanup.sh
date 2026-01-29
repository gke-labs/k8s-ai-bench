#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-tls-optional" --ignore-not-found
