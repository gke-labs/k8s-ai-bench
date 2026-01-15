# k8s-ai-bench

`k8s-ai-bench` is a benchmark for assessing the performance of LLM models for Kubernetes related tasks. It evaluates AI agents (like `kubectl-ai`) on their ability to perform real-world Kubernetes operations such as creating deployments, debugging crash loops, and scaling applications.

## 📊 Live Dashboard

See [k8s-ai-bench live leaderboard](https://gke-labs.github.io/k8s-ai-bench/) for the latest benchmark results.

The leaderboard shows run results for widely used proprietary and open models over a few run types:

* **Pass@1**: Can the agent solve the task on the first try? This measures raw capability and immediate correctness.
* **Pass@5**: Can the agent solve the task at least once in 5 attempts? This shows if the agent can eventually find a solution.
* **Pass^5**: Does the agent solve the task every single time? This measures reliability and consistency, which is crucial for autonomous usage.



## 🚀 Quick Start

### 1. Build the Binary
```sh
go build
```

### 2. Run an Evaluation
Run the benchmark against your agent binary. Results will be saved to the `.build` directory.
```sh
# Basic usage
./k8s-ai-bench run --agent-bin <path/to/kubectl-ai> --output-dir .build/k8s-ai-bench

# Run a specific task type (e.g., scaling tasks)
./k8s-ai-bench run --agent-bin <path/to/kubectl-ai> --task-pattern "scale" --output-dir .build/k8s-ai-bench
```

## 🛠 Usage Guide

### `run` Subcommand
The `run` subcommand executes the benchmark evaluations. It creates ephemeral Kind clusters (by default) to ensure test isolation.

```sh
# Run with specific LLM provider and model
./k8s-ai-bench run \
  --agent-bin <path/to/kubectl-ai> \
  --llm-provider gemini \
  --models gemini-2.5-pro-preview-03-25 \
  --task-pattern fix \
  --output-dir .build/k8s-ai-bench
```

**Common Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--agent-bin` | Path to kubectl-ai binary (Required) | - |
| `--output-dir` | Directory to write results (Required) | - |
| `--task-pattern` | RegEx pattern to filter tasks (e.g. 'pod', 'fix') | - |
| `--llm-provider` | LLM provider ID (e.g. 'gemini', 'openai') | gemini |
| `--models` | Comma-separated list of models | gemini-2.5-pro... |
| `--concurrency` | Number of parallel tasks (0 = auto) | 0 |

### `analyze` Subcommand
Process and summarize results from previous runs.

```sh
# Generate a Markdown report
./k8s-ai-bench analyze --input-dir .build/k8s-ai-bench --results-filepath report.md

# Generate JSONL for visualization
./k8s-ai-bench analyze --input-dir .build/k8s-ai-bench --output-format jsonl --results-filepath site/combined_results.jsonl
```

## 💻 Development Scripts
For a streamlined development loop, use the scripts in `dev/ci/periodics/`:

- **Run Evaluation Loop**: Runs evaluations multiple times to test consistency.
  ```sh
  ./dev/ci/periodics/run-eval-loop.sh --iterations 5 --task-pattern "create"
  ```
- **Run Single Evaluation**:
  ```sh
  TEST_ARGS="--task-pattern=fix-probes" ./dev/ci/periodics/run-evals.sh
  ```
- **Analyze Results**:
  ```sh
  ./dev/ci/periodics/analyze-evals.sh --show-failures
  ```

## 📈 Visualizing Results Locally

The `site` directory contains a static website (Vue.js based) for visualizing benchmark results.

1.  **Generate Data**:
    ```sh
    ./k8s-ai-bench analyze --input-dir .build/k8s-ai-bench --output-format jsonl --results-filepath site/combined_results.jsonl
    ```
2.  **Serve Locally**:
    ```sh
    cd site
    python3 -m http.server
    ```
3.  **View**: Open [http://localhost:8000](http://localhost:8000)

## 🤝 Contributions
We welcome contributions! Please check out the [contributions guide](contributing.md).
