#!/usr/bin/env bash

NAMESPACE="inventory"
STS="db"
SERVICE="inventory-db"
FQDN="db-0.inventory-db.inventory.svc.cluster.local"
TIMEOUT="120s"

# The application's target hostname is derived from the headless service; it must still exist and be headless.
CLUSTER_IP=$(kubectl get service "$SERVICE" -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
if [[ "$CLUSTER_IP" != "None" ]]; then
  echo "Failure: headless service '$SERVICE' is missing or is no longer headless (clusterIP='$CLUSTER_IP')"
  exit 1
fi

# The database must still be a StatefulSet named db, governed by that service
if ! kubectl get statefulset "$STS" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: StatefulSet '$STS' does not exist in namespace '$NAMESPACE'"
  exit 1
fi
SERVICE_NAME=$(kubectl get statefulset "$STS" -n "$NAMESPACE" -o jsonpath='{.spec.serviceName}')
if [[ "$SERVICE_NAME" != "$SERVICE" ]]; then
  echo "Failure: StatefulSet '$STS' serviceName is '$SERVICE_NAME', expected '$SERVICE'"
  exit 1
fi

echo "Waiting for pod db-0 to be ready..."
for _ in $(seq 1 30); do
  if kubectl get pod db-0 -n "$NAMESPACE" &>/dev/null; then
    break
  fi
  sleep 2
done
if ! kubectl wait --for=condition=Ready pod/db-0 -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: pod db-0 is not ready"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

POD_IP=$(kubectl get pod db-0 -n "$NAMESPACE" -o jsonpath='{.status.podIP}')

# The real test: the per-pod DNS name must resolve to db-0's IP from inside the cluster.
echo "Resolving $FQDN from inside the cluster..."
RESOLVED=$(kubectl run -n "$NAMESPACE" verify-dns --image=busybox:1.36 --restart=Never --rm -i --quiet \
  --wait --timeout=180s -- \
  sh -c "for i in \$(seq 1 15); do nslookup $FQDN >/dev/null 2>&1 && nslookup $FQDN && exit 0; sleep 3; done; exit 1" 2>/dev/null)

if [[ "$RESOLVED" != *"$POD_IP"* ]]; then
  echo "Failure: $FQDN did not resolve to db-0's IP ($POD_IP)"
  echo "nslookup output: $RESOLVED"
  exit 1
fi

echo "Success: $FQDN resolves to $POD_IP"
exit 0
