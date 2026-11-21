#!/usr/bin/env bash
kubectl delete namespace storefront --ignore-not-found
kubectl create namespace storefront

kubectl apply -f artifacts/configmap.yaml
kubectl apply -f artifacts/deployment.yaml
