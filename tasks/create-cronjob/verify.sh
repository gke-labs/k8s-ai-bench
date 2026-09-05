#!/usr/bin/env bash

NAMESPACE="platform-ops"
CRONJOB="log-rotate"
TIMEOUT="120s"

if ! kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" &>/dev/null; then
  echo "Failure: CronJob '$CRONJOB' does not exist in namespace '$NAMESPACE'"
  exit 1
fi

CJ_JSON=$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o json)

SCHEDULE=$(echo "$CJ_JSON" | jq -r '.spec.schedule')
# Accept the common spellings of "every 15 minutes"
case "$SCHEDULE" in
  "*/15 * * * *"|"0,15,30,45 * * * *"|"0/15 * * * *"|"@every 15m") ;;
  *)
    echo "Failure: schedule is '$SCHEDULE', expected every 15 minutes (e.g. '*/15 * * * *')"
    exit 1
    ;;
esac

CONCURRENCY=$(echo "$CJ_JSON" | jq -r '.spec.concurrencyPolicy')
if [[ "$CONCURRENCY" != "Forbid" ]]; then
  echo "Failure: concurrencyPolicy is '$CONCURRENCY', expected 'Forbid' so runs never overlap"
  exit 1
fi

SUCCESS_HISTORY=$(echo "$CJ_JSON" | jq -r '.spec.successfulJobsHistoryLimit')
if [[ "$SUCCESS_HISTORY" != "3" ]]; then
  echo "Failure: successfulJobsHistoryLimit is '$SUCCESS_HISTORY', expected 3"
  exit 1
fi

FAILED_HISTORY=$(echo "$CJ_JSON" | jq -r '.spec.failedJobsHistoryLimit')
if [[ "$FAILED_HISTORY" != "1" ]]; then
  echo "Failure: failedJobsHistoryLimit is '$FAILED_HISTORY', expected 1"
  exit 1
fi

SUSPENDED=$(echo "$CJ_JSON" | jq -r '.spec.suspend // false')
if [[ "$SUSPENDED" == "true" ]]; then
  echo "Failure: the CronJob is suspended"
  exit 1
fi

IMAGE=$(echo "$CJ_JSON" | jq -r '.spec.jobTemplate.spec.template.spec.containers[0].image')
if [[ "$IMAGE" != "busybox:1.36" ]]; then
  echo "Failure: container image is '$IMAGE', expected 'busybox:1.36'"
  exit 1
fi

# The command may be expressed via command and/or args; the job must print "rotating logs"
COMMAND_LINE=$(echo "$CJ_JSON" | jq -r '.spec.jobTemplate.spec.template.spec.containers[0] | ((.command // []) + (.args // [])) | join(" ")')
if [[ "$COMMAND_LINE" != *"echo rotating logs"* && "$COMMAND_LINE" != *"echo \"rotating logs\""* && "$COMMAND_LINE" != *"echo 'rotating logs'"* ]]; then
  echo "Failure: container command is '$COMMAND_LINE', expected it to run 'echo rotating logs'"
  exit 1
fi

# Prove the job template actually works by triggering a run from the CronJob and waiting for it
TEST_JOB="${CRONJOB}-verify-run"
kubectl delete job "$TEST_JOB" -n "$NAMESPACE" --ignore-not-found &>/dev/null
if ! kubectl create job "$TEST_JOB" -n "$NAMESPACE" --from=cronjob/"$CRONJOB"; then
  echo "Failure: could not create a Job from CronJob '$CRONJOB'"
  exit 1
fi

echo "Waiting for the manually triggered job to complete..."
if ! kubectl wait --for=condition=Complete job/"$TEST_JOB" -n "$NAMESPACE" --timeout=$TIMEOUT; then
  echo "Failure: job created from the CronJob did not complete"
  kubectl get pods -n "$NAMESPACE"
  kubectl delete job "$TEST_JOB" -n "$NAMESPACE" --ignore-not-found &>/dev/null
  exit 1
fi

LOGS=$(kubectl logs -n "$NAMESPACE" job/"$TEST_JOB" 2>/dev/null)
kubectl delete job "$TEST_JOB" -n "$NAMESPACE" --ignore-not-found &>/dev/null
if [[ "$LOGS" != *"rotating logs"* ]]; then
  echo "Failure: job output was '$LOGS', expected 'rotating logs'"
  exit 1
fi

echo "Success: CronJob '$CRONJOB' is configured correctly and its job template runs"
exit 0
