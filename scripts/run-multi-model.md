# Running k8s-ai-bench for multiple locally-served models with `run-multi-model.sh`

This document outlines the steps to run the `run-multi-model.sh` script to evaluate multiple local models with `k8s-ai-bench`.

## Prerequisites

Before running the script, note the following:

1.  **Run from Repository Base:** The script must be executed from the root of the repository.

2.  **GCE VM for Model Serving:** You need a Google Compute Engine (GCE) virtual machine to serve the models. This VM must meet the following criteria:
    *   **Sufficient Disk Space:** The VM's boot disk or an attached persistent disk must have enough space to store the models you intend to use. For example, the GLM 4.6 model requires approximately 800GB of storage.
    *   **NVIDIA GPUs:** The VM must be equipped with NVIDIA GPUs that have enough VRAM to load and serve the models with vllm.
    *   **SSH Tunnel:** An active SSH tunnel must be established from your local machine to the GCE VM. You can create the tunnel using the following `gcloud` command. This command forwards traffic from your local port 8000 to the VM's port 8000.

        ```bash
        gcloud compute ssh your-gce-vm --zone=us-west1-c -- -L 8000:localhost:8000
        ```
    *   **If you'd like to use a setup other than GCE vm, you will have to modify `run-multi-model.sh` accordingly.**
    *   **Spot VMs:** Pro tip: If you're having availability or stockout issues acquiring a GCE vm, or you're looking to lower its costs, spot vms are a good option, but risk getting preempted.

3.  **Ensure correct settings in `run-multi-model.sh`:** Update any variables inside the script as necessary:
    *   **GCE vm configuration:** Replace the defaults (`your-gce-vm`, `us-west1-c`, `8`) with your GCE vm's name, location, and tensor parallel size (number of NVIDIA GPUs), respectively.
    *   **Benchmark configuration:** Change any options as you see fit: number of iterations, eval concurrency, cluster creation policy...
    *   **Model configuration:** Add any missing models you want to benchmark, and remove any included models you do not want to benchmark. Change any vllm flags as needed (e.g., max-model-len, gpu-memory-utilization, etc.)
    *   **Virtual environment configuration:** This script assumes your GCE vm has a virtual env with vllm and necessary dependencies installed. Such a setup might look like this:
        ```bash
        curl -LsSf https://astral.sh/uv/install.sh | sh
        source $HOME/.local/bin/env
        uv venv --python 3.12 --seed
        source .venv/bin/acivate
        uv pip install vllm
        # You may need to install other packages, on Debian for example:
        sudo apt-get update
        sudo apt-get install build-essential python3.12-dev -y
        ```

## Usage

Navigate to the root of the repository and execute the script:

```bash
./k8s-ai-bench/scripts/run-multi-model.sh
```

## Output

The script will run the `k8s-ai-bench` evaluation for each model defined within it.

*   **Multiple Runs:** For each run, the script creates a new output directory in .build/.
*   **Directory Structure:** The output directories will contain the results of the `k8s-ai-bench analyze` command for both markdown and json format, which includes detailed information about the model's performance on the benchmark tasks.
