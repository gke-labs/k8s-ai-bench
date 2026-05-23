#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

NAMESPACE=payments

kubectl delete namespace ${NAMESPACE} --ignore-not-found
