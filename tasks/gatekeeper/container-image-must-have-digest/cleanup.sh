#!/usr/bin/env bash
set -euo pipefail
kubectl delete namespace "gk-container-image-must-have-digest" --ignore-not-found
