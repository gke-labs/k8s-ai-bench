#!/usr/bin/env bash
kubectl delete namespace analytics --ignore-not-found
kubectl create namespace analytics

kubectl apply -f artifacts/postgres-deployment.yaml
kubectl apply -f artifacts/postgres-service.yaml

if ! kubectl wait --for=condition=Available deployment/postgres -n analytics --timeout=180s; then
    echo "postgres did not become available during setup"
    kubectl describe deployment/postgres -n analytics
    exit 1
fi

kubectl apply -f artifacts/api-deployment.yaml
