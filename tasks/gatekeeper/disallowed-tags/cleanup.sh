#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-disallowed-tags" --ignore-not-found
