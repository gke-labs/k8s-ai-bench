#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-must-have-set-of-annotations" --ignore-not-found
