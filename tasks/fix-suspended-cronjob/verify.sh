#!/usr/bin/env bash

NAMESPACE="reporting"
CRONJOB="usage-report"
TIMEOUT="120s"

if ! kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: CronJob '$CRONJOB' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

SUSPENDED=$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o jsonpath='{.spec.suspend}')
if [[ "$SUSPENDED" == "true" ]]; then
  echo "Failure: CronJob '$CRONJOB' is still suspended"
  exit 1
fi

# The CronJob controller must actually schedule a Job on its own (schedule is every minute).
# Only Jobs owned by the CronJob count; a manually created Job would not prove the schedule works.
echo "Waiting for the CronJob controller to schedule a job..."
JOB=""
for _ in $(seq 1 75); do
  JOB=$(kubectl get jobs -n "$NAMESPACE" -o json | jq -r --arg cj "$CRONJOB" \
    '.items[] | select(.metadata.ownerReferences[]? | (.kind == "CronJob" and .name == $cj)) | .metadata.name' | head -n1)
  if [[ -n "$JOB" ]]; then
    break
  fi
  sleep 2
done
if [[ -z "$JOB" ]]; then
  echo "Failure: no Job was scheduled by CronJob '$CRONJOB'"
  kubectl describe cronjob "$CRONJOB" -n "$NAMESPACE"
  exit 1
fi

echo "Waiting for job '$JOB' to complete..."
if ! kubectl wait --for=condition=Complete job/"$JOB" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: job '$JOB' did not complete"
  kubectl get pods -n "$NAMESPACE"
  exit 1
fi

LAST_SCHEDULE=$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o jsonpath='{.status.lastScheduleTime}')
if [[ -z "$LAST_SCHEDULE" ]]; then
  echo "Failure: CronJob '$CRONJOB' has no lastScheduleTime"
  exit 1
fi

echo "Success: CronJob '$CRONJOB' is active again and job '$JOB' completed"
exit 0
