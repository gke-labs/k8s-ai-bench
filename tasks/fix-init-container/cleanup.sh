#!/usr/bin/env bash

# Delete the namespace, which removes all resources created for this task
kubectl delete namespace checkout --ignore-not-found

echo "Cleanup completed"
