# Using vCluster with k8s-ai-bench

This guide demonstrates how to run `k8s-ai-bench` using [vCluster](https://www.vcluster.com/) (Virtual Clusters). This allows you to run benchmark tasks in isolated virtual clusters created on a host Kubernetes cluster, providing better tenant isolation and scalability.

## Prerequisites

Before running the benchmark, ensure you have the following tools installed and configured:

1.  **vCluster CLI (0.20.0+)**
    *   Used to create and manage virtual clusters.
    *   Install: `curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-amd64" && sudo install -c -m 0755 vcluster /usr/local/bin && rm -f vcluster`
    *   Verify: `vcluster --version`

2.  **kubectl**
    *   Standard Kubernetes CLI.
    *   Verify: `kubectl version --client`

3.  **Helm (Optional but recommended)**
    *   Useful if you need to manually inspect vCluster charts.
    *   Verify: `helm version`

4.  **Google Cloud CLI (gcloud)**
    *   Required for creating and managing GKE clusters.
    *   Verify: `gcloud version`

5.  **Host Cluster Credentials**
    *   You must have a `kubeconfig` configured with access to a real "Host" Kubernetes cluster (e.g., GKE, EKS, or a large local Kind cluster).
    *   The user context used for the host cluster must have permissions to:
        *   Create Namespaces
        *   Create Services (LoadBalancers)
        *   Create Pods/Deployments
        *   Manage PVCs/PVs

## Setup Steps

### 1. Prepare Host Cluster (GKE)

For this demo, we recommend using a **Google Kubernetes Engine (GKE) Standard** cluster.

**Why Standard over Autopilot?**
*   **Performance**: Benchmarks via vCluster often require rapid provisioning of resources (Pods, LoadBalancers).
*   **Latency**: GKE Autopilot scales aggressively and may have higher startup latency for temporary workloads like vClusters, leading to timeouts in `setup.sh` scripts.
*   **Control**: Standard clusters allow you to pre-provision a node pool to ensure there are always resources ready for the vCluster pods to land on immediately.

**Create the Cluster:**
You can create a zonal standard cluster using `gcloud`:

```bash
# Create a standard GKE regional cluster.
gcloud container clusters create host-standard \
    --region us-central1 \
    --project <your-project-id> \
    --num-nodes 8 \
    --machine-type e2-standard-2
```

**Get Credentials:**
Once the cluster is ready (or if you are using an existing one), configure your local `kubectl`:

```bash
gcloud container clusters get-credentials host-standard --region us-central1 --project <your-project-id>
```

To verify, check that your current context is set correctly:
```bash
kubectl config current-context
```

### 2. Run k8s-ai-bench

The `k8s-ai-bench` tool handles the lifecycle of the vCluster automatically. You need to specify:
- `--cluster-provider vcluster`: Tells the harness to use vCluster.
- `--host-cluster-context <context-name>`: The context name in your `kubeconfig` to use as the host.

**Example Command:**

```bash
# Run all tasks
./k8s-ai-bench run \
  --cluster-provider vcluster \
  --host-cluster-context gke_<your-project-id>_us-central1_host-standard \
  --agent-bin /usr/local/bin/kubectl-ai \
  --output-dir .build/results \

# Analyze results
./k8s-ai-bench analyze --input-dir .build/results --results-filepath report.md
```

### Observations & Known Issues

When running benchmarking tasks on vCluster, we observed some unique behaviors compared to using `kind` clusters:

1.  **LLM "Self-Awareness"**:
    *   Sometimes, LLMs are able to deduce they are running inside a vCluster.
    *   **Impact**: Occasionally, an LLM might try to "break out" or troubleshoot the virtual environment itself (e.g., checking syncer logs) rather than focusing purely on the task.

2.  **Storage Class Sync**:
    *   vCluster does not have its own PV provisioner by default; it syncs PVCs to the host.
    *   LLMs that hardcode `storageClassName: standard` usually succeed (if mapped), but those inventing random storage class names (`premium-ssd`) will fail unless `vcluster.yaml` is configured to sync SCs and the host actually has them.
    *   **Fix**: We explicitly enable `storageClasses`, `PersistentVolumeClaims`, `PersistentVolumes` sync in our `vcluster.yaml`.

3.  **Connection to vCluster**:
    *   We are currently connecting to vCluster using a local background-proxy to do port-forwarding on a docker container. This can be unstable and cause connection issues at scale.
    *   **Potential Fix**: Have a ingress controller running on the host cluster and connect to vCluster via ingress.