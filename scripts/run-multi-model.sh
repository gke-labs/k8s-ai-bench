#!/bin/bash

# --- Configuration ---
# GCE VM Details (CHANGE THESE TO FIT YOUR OWN USAGE)
GCE_INSTANCE_NAME="your-gce-vm"
GCE_ZONE="us-west1-c"
# virtual env path, for use with uv
VENV_PATH="/home/user/.venv/bin/activate"
# tensor parallel number (the number of GPUs on your gce vm)
TP=8

# k8s-ai-bench configuration
ITERATIONS=5
CONCURRENCY=5
CLUSTER_CREATION_POLICY="AlwaysCreate"

# --- Local Evaluation Scripts ---
LOCAL_PROGRAM_1="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m openai/gpt-oss-20b -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_2="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m openai/gpt-oss-120b -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_3="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m Qwen/Qwen3-Next-80B-A3B-Instruct -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_4="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m zai-org/GLM-4.6 -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_5="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m zai-org/GLM-4.5 -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_6="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m Qwen/Qwen3-Coder-30B-A3B-Instruct -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_7="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p openai -m Qwen/Qwen3-Coder-480B-A35B-Instruct -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"


# --- Remote vLLM Server Commands ---
VLLM_START_1_CMD="source ${VENV_PATH}; nohup vllm serve openai/gpt-oss-20b -tp ${TP} > ~/vllm_server_gpt-oss-20b.log 2>&1 &"
VLLM_START_2_CMD="source ${VENV_PATH}; nohup vllm serve openai/gpt-oss-120b -tp ${TP} > ~/vllm_server_gpt-oss-120b.log 2>&1 &"
VLLM_START_3_CMD="source ${VENV_PATH}; nohup vllm serve Qwen/Qwen3-Next-80B-A3B-Instruct -tp ${TP} --enable-auto-tool-choice --tool-call-parser hermes --enforce-eager > ~/vllm_server_Qwen3-Next-80B.log 2>&1 &"
VLLM_START_4_CMD="source ${VENV_PATH}; nohup vllm serve zai-org/GLM-4.6 -tp ${TP} --enable-auto-tool-choice --tool-call-parser glm45 --reasoning-parser glm45 --enforce-eager > ~/vllm_server_zai-GLM-46.log 2>&1 &"
VLLM_START_5_CMD="source ${VENV_PATH}; nohup vllm serve zai-org/GLM-4.5 -tp ${TP} --enable-auto-tool-choice --tool-call-parser glm45 --reasoning-parser glm45 --enforce-eager > ~/vllm_server_zai-GLM-45.log 2>&1 &"
VLLM_START_6_CMD="source ${VENV_PATH}; nohup vllm serve Qwen/Qwen3-Coder-30B-A3B-Instruct -tp ${TP} --enable-auto-tool-choice --tool-call-parser qwen3_coder --enforce-eager > ~/vllm_server_Qwen3-Coder-30B.log 2>&1 &"
VLLM_START_7_CMD="source ${VENV_PATH}; nohup vllm serve Qwen/Qwen3-Coder-480B-A35B-Instruct -tp ${TP} --enable-auto-tool-choice --tool-call-parser qwen3_coder --enforce-eager --max-model-len 131072 > ~/vllm_server_Qwen3-Coder-480B.log 2>&1 &"
VLLM_STOP_CMD="pkill -f 'vllm serve'"

# --- Helper Function ---
# This function handles the entire process for a single model evaluation.
# Arguments:
#   $1: Model Name as it appears on huggingface (e.g., "openai/gpt-oss-20b", "Qwen/Qwen3-Next-80B-A3B-Instruct")
#   $2: The command to serve that model with vllm
#   $3: The command to run k8s-ai-bench
run_model_evaluation() {
    local model_name="$1"
    local start_cmd="$2"
    local eval_cmd="$3"

    echo "Starting server for Model ${model_name}..."
    gcloud compute ssh $GCE_INSTANCE_NAME --zone=$GCE_ZONE --command="${start_cmd}"
    echo "Waiting for the server to become healthy..."
    until curl --output /dev/null --silent --fail http://localhost:8000/health; do
        printf '.'
        sleep 30
    done
    echo -e "\nServer for Model ${model_name} is ready!"

    echo "Running local evaluation for Model ${model_name}."
    eval "${eval_cmd}"

    echo "Shutting down server for Model ${model_name}..."
    gcloud compute ssh $GCE_INSTANCE_NAME --zone=$GCE_ZONE --command="${VLLM_STOP_CMD}"
    sleep 30 # Give time for ports to free up
}


# --- Main Script Logic ---
# We assume your manual gcloud SSH tunnel is active in another terminal.
echo "PREP: Ensuring no vLLM servers are currently running..."
gcloud compute ssh $GCE_INSTANCE_NAME --zone=$GCE_ZONE --command="${VLLM_STOP_CMD}"
sleep 30

# Run evaluations for models that use Responses API
export OPENAI_USE_RESPONSES_API=true
run_model_evaluation "gpt-oss-20b" "${VLLM_START_1_CMD}" "${LOCAL_PROGRAM_1}"
run_model_evaluation "gpt-oss-120b" "${VLLM_START_2_CMD}" "${LOCAL_PROGRAM_2}"
unset OPENAI_USE_RESPONSES_API

# Run evaluations for the remaining models on Completions API
run_model_evaluation "Qwen/Qwen3-Next-80B-A3B-Instruct" "${VLLM_START_3_CMD}" "${LOCAL_PROGRAM_3}"
run_model_evaluation "zai-org/GLM-4.6" "${VLLM_START_4_CMD}" "${LOCAL_PROGRAM_4}"
run_model_evaluation "zai-org/GLM-4.5" "${VLLM_START_5_CMD}" "${LOCAL_PROGRAM_5}"
run_model_evaluation "Qwen/Qwen3-Coder-30B-A3B-Instruct" "${VLLM_START_6_CMD}" "${LOCAL_PROGRAM_6}"
run_model_evaluation "Qwen/Qwen3-Coder-480B-A35B-Instruct" "${VLLM_START_7_CMD}" "${LOCAL_PROGRAM_7}"

# Shut down GCE VM -- no longer needed
gcloud compute instances stop $GCE_INSTANCE_NAME --zone=$GCE_ZONE

# Gemini doesn't need a local setup, we run these against the gemini api
LOCAL_PROGRAM_GEM_PRO="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p gemini -m gemini-2.5-pro -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
LOCAL_PROGRAM_GEM_FLASH="./dev/ci/periodics/run-eval-loop.sh -i $ITERATIONS -p gemini -m gemini-2.5-flash -a http://localhost:8000/v1 -c $CONCURRENCY -k $CLUSTER_CREATION_POLICY"
eval "${LOCAL_PROGRAM_GEM_PRO}"
eval "${LOCAL_PROGRAM_GEM_FLASH}"

echo "Evaluations for all models finished successfully! You can now close your gcloud SSH tunnel."
exit 0
