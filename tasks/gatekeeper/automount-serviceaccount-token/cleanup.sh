#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-automount-serviceaccount-token" --ignore-not-found
