#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-repo-must-not-be-k8s-gcr-io" --ignore-not-found
