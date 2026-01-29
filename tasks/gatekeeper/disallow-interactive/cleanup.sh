#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-disallow-interactive" --ignore-not-found
