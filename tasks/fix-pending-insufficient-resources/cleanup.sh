#!/usr/bin/env bash

# Delete the namespace, which removes all resources created for this task
kubectl delete namespace analytics --ignore-not-found

echo "Cleanup completed"
