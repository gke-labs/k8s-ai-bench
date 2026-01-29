#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-must-have-key" --ignore-not-found
