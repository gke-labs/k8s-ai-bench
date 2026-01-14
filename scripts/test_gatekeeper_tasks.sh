#!/usr/bin/env bash
set -u

# Test all gatekeeper tasks
TASKS_DIR="tasks/gatekeeper"
RESULTS_FILE="gatekeeper_test_results.txt"
TIMEOUT=180  # 3 minutes per task

# Clear results file
> "$RESULTS_FILE"

echo "Testing all gatekeeper tasks..."
echo "================================"

passed=0
failed=0

for task_dir in $(ls -d "$TASKS_DIR"/gk-* | sort); do
    task_name=$(basename "$task_dir")
    echo -n "Testing $task_name... "

    cd "$task_dir"

    # Run setup.sh with timeout
    TIMEOUT_CMD="timeout"
    if command -v gtimeout &> /dev/null; then
        TIMEOUT_CMD="gtimeout"
    fi

    if $TIMEOUT_CMD $TIMEOUT ./setup.sh > setup_output.log 2>&1; then
        setup_status="PASS"
    else
        setup_status="FAIL (exit $?)"
    fi

    # Run cleanup.sh
    ./cleanup.sh > cleanup_output.log 2>&1 || true

    cd - > /dev/null

    if [[ "$setup_status" == "PASS" ]]; then
        echo "PASS"
        ((passed++))
        echo "$task_name: PASS" >> "$RESULTS_FILE"
    else
        echo "FAIL"
        ((failed++))
        echo "$task_name: $setup_status" >> "$RESULTS_FILE"
        echo "  Setup output:" >> "$RESULTS_FILE"
        sed 's/^/    /' "$task_dir/setup_output.log" >> "$RESULTS_FILE"
        echo "" >> "$RESULTS_FILE"
    fi
done

echo "================================"
echo "Results: $passed passed, $failed failed"
echo ""
echo "Details saved to $RESULTS_FILE"
