#!/usr/bin/env python3
import os
import yaml
import requests
from pathlib import Path
import subprocess
import shutil

GITHUB_API = "https://api.github.com/repos/open-policy-agent/gatekeeper-library/contents"
RAW_BASE = "https://raw.githubusercontent.com/open-policy-agent/gatekeeper-library/master"
CATEGORIES = ["library/general", "library/pod-security-policy"]
OUTPUT_DIR = Path(__file__).parent.parent / "tasks" / "gatekeeper"
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY", "")

EXCLUDED_POLICIES = [
    "verifydeprecatedapi",
    "ephemeralstoragelimit",
    "forbidden-sysctls",  # Excluded: requires complex sysctl values that are hard to patch safely
    "flexvolume-drivers", # Excluded: test drivers don't exist on standard clusters
    "proc-mount",         # Excluded: requires Kubelet Featuregate explicitly enabled
    "allowedrepos",       # Excluded: requires complex image swapping & args stripping
    "allowedreposv2",     # Excluded: requires complex image swapping & args stripping
    "disallowedrepos",    # Excluded: requires complex image swapping & args stripping
    "requiredprobes",     # Excluded: requires complex probe port patching
    "imagedigests",       # Excluded: requires complex fake digest injection
    "apparmor",           # Excluded: AppArmor not enabled on host
    "privileged-containers", # Excluded: initContainers hang without complex patching
]

WAITABLE_KINDS = {
    "Pod": "condition=Ready",
    "Deployment": "condition=Available",
    "StatefulSet": "condition=Ready",
    "DaemonSet": "condition=Ready",
    "ReplicaSet": "condition=Ready",
    "Job": "condition=Complete",
}

def get_wait_command(manifest_str: str, namespace: str) -> list[str]:
    """Generate kubectl wait commands for supported resources."""
    cmds = []
    try:
        docs = yaml.safe_load_all(manifest_str)
        for doc in docs:
            if not doc or "kind" not in doc:
                continue
            kind = doc["kind"]
            name = doc["metadata"]["name"]

            if kind in WAITABLE_KINDS:
                condition = WAITABLE_KINDS[kind]
                cmds.append(f"kubectl wait --for={condition} {kind.lower()}/{name} -n {namespace} --timeout=180s")
    except Exception:
        pass
    return cmds


def generate_description(constraint: str) -> str:
    """Use Gemini to generate a natural language description of the constraint."""
    if not GEMINI_API_KEY:
        return ""
    resp = requests.post(
        f"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key={GEMINI_API_KEY}",
        json={
            "contents": [{"parts": [{"text": f"""Describe this Gatekeeper constraint policy in plain English. Be concise (2-3 sentences). Focus on what the policy requires/forbids. Don't mention Gatekeeper or Kubernetes jargon.

{constraint}"""}]}]
        },
        timeout=30,
    )
    if resp.ok:
        return resp.json()["candidates"][0]["content"]["parts"][0]["text"].strip()
    return ""

REPO_URL = "https://github.com/open-policy-agent/gatekeeper-library.git"
LOCAL_REPO = Path(__file__).parent.parent / ".gatekeeper-library"
LIBRARY_PATH = LOCAL_REPO / "library"

def clone_repo():
    """Clone or update the repository."""
    if not LOCAL_REPO.exists():
        print(f"Cloning {REPO_URL} into {LOCAL_REPO}...")
        subprocess.run(["git", "clone", "--depth", "1", REPO_URL, str(LOCAL_REPO)], check=True)
    else:
        print(f"Updating {REPO_URL} in {LOCAL_REPO}...")
        subprocess.run(["git", "pull"], cwd=LOCAL_REPO, check=True)

def list_dirs(path: Path) -> list[dict]:
    """List directories in a local path."""
    if not path.exists():
        return []
    return [{"name": p.name, "path": str(p), "type": "dir"} for p in path.iterdir() if p.is_dir() and not p.name.startswith(".")]

def read_file(path: Path) -> str:
    """Read local file content."""
    return path.read_text()


def neutralize_manifest(manifest: str, suffix: str) -> str:
    """Neutralize resource names, container names, and labels to avoid leaking info."""
    docs = list(yaml.safe_load_all(manifest))
    for doc in docs:
        if not doc or "metadata" not in doc:
            continue
        # Neutralize metadata.name
        doc["metadata"]["name"] = f"resource-{suffix}"
        # Remove usage of specific namespace so we can apply to any namespace
        doc["metadata"].pop("namespace", None)
        # Neutralize app labels if present
        labels = doc["metadata"].get("labels", {})
        if "app" in labels:
            labels["app"] = f"app-{suffix}"
        
        # Track renames for annotation fixes
        renames = {}
        # Neutralize container names
        for key in ["containers", "initContainers"]:
            prefix = "init-container" if key == "initContainers" else "container"
            containers = doc.get("spec", {}).get(key, [])
            if not isinstance(containers, list):
                continue
            for i, c in enumerate(containers):
                old_name = c.get("name", "")
                new_name = f"{prefix}-{suffix}-{i}" if i else f"{prefix}-{suffix}"
                c["name"] = new_name
                if old_name:
                    renames[old_name] = new_name
        
        # Generic Fix: Update annotations that reference container names (e.g. apparmor, seccomp)
        annotations = doc["metadata"].get("annotations", {})
        if annotations and renames:
            new_annotations = {}
            for k, v in annotations.items():
                updated_k = k
                for old, new in renames.items():
                    # Check for container.apparmor.security.beta.kubernetes.io/OLD_NAME
                    if k.endswith("/" + old):
                         updated_k = k.replace("/" + old, "/" + new)
                         break
                new_annotations[updated_k] = v
            doc["metadata"]["annotations"] = new_annotations

    return yaml.dump_all(docs, default_flow_style=False)

def patch_manifest(manifest_str: str) -> str:
    """Patch manifest to fix common issues using YAML parsing for reliability."""
    import re

    # 1. Text-based replacements for specific bad images
    manifest_str = manifest_str.replace("safe-images.com/nginx", "nginx:latest")
    manifest_str = manifest_str.replace("safeimages.com/nginx", "nginx:latest")
    manifest_str = manifest_str.replace("openpolicyagent/opa:0.9.2", "nginx:latest")
    manifest_str = manifest_str.replace("openpolicyagent/opa", "nginx:latest")
    manifest_str = manifest_str.replace("localhost/custom", "runtime/default")
    # Fix invalid/non-existent images
    manifest_str = manifest_str.replace("nginx-exempt", "nginx:latest")
    manifest_str = manifest_str.replace("unnginx:latest", "nginx:latest")
    manifest_str = manifest_str.replace("nginx:latest:latest", "nginx:latest")  # Double tag
    manifest_str = manifest_str.replace("image: exempt", "image: nginx:latest")

    # 2. Use nginx-unprivileged for non-root contexts
    manifest_str = manifest_str.replace("image: nginx\n", "image: nginxinc/nginx-unprivileged:latest\n")

    # 3. Parse YAML for more complex fixes
    try:
        docs = list(yaml.safe_load_all(manifest_str))
        modified = False
        for doc in docs:
            if not doc or "spec" not in doc:
                continue

            # Check if any container has readOnlyRootFilesystem: true
            needs_tmp_volume = False
            for key in ["containers", "initContainers"]:
                containers = doc.get("spec", {}).get(key, [])
                if not isinstance(containers, list):
                    continue
                for container in containers:
                    # Remove args that look like OPA server args
                    if "args" in container:
                        args = container["args"]
                        if isinstance(args, list) and any("--server" in str(a) or "--addr" in str(a) for a in args):
                            del container["args"]
                            modified = True
                        elif isinstance(args, list) and "run" in args:
                            del container["args"]
                            modified = True

                    # Check for readOnlyRootFilesystem
                    sc = container.get("securityContext", {})

                    # Fix localhost seccomp profiles that don't exist on standard clusters
                    if "seccompProfile" in sc:
                        profile = sc["seccompProfile"]
                        if profile.get("type") == "Localhost":
                            # Change to RuntimeDefault since localhost profiles aren't available
                            profile["type"] = "RuntimeDefault"
                            profile.pop("localhostProfile", None)
                            modified = True

                    # Scale down large memory requests/limits to fit on kind clusters
                    resources = container.get("resources", {})
                    for res_type in ["requests", "limits"]:
                        res = resources.get(res_type, {})
                        if "memory" in res:
                            mem = res["memory"]
                            # Convert Gi to Mi if >= 1Gi
                            if isinstance(mem, str) and "Gi" in mem:
                                try:
                                    gi_val = float(mem.replace("Gi", ""))
                                    if gi_val >= 1:
                                        # Scale down by 4x (2Gi -> 512Mi)
                                        new_val = int(gi_val * 256)
                                        res["memory"] = f"{new_val}Mi"
                                        modified = True
                                except ValueError:
                                    pass

                    if sc.get("readOnlyRootFilesystem") == True:
                        needs_tmp_volume = True
                        # Add volumeMount for /tmp
                        if "volumeMounts" not in container:
                            container["volumeMounts"] = []
                        # Check if /tmp mount already exists
                        if not any(vm.get("mountPath") == "/tmp" for vm in container["volumeMounts"]):
                            container["volumeMounts"].append({
                                "name": "tmp-volume",
                                "mountPath": "/tmp"
                            })
                            modified = True

            # Add emptyDir volume for /tmp if needed
            if needs_tmp_volume:
                if "volumes" not in doc["spec"]:
                    doc["spec"]["volumes"] = []
                if not any(v.get("name") == "tmp-volume" for v in doc["spec"]["volumes"]):
                    doc["spec"]["volumes"].append({
                        "name": "tmp-volume",
                        "emptyDir": {}
                    })
                    modified = True

        if modified:
            manifest_str = yaml.dump_all(docs, default_flow_style=False)
    except Exception:
        pass  # If YAML parsing fails, return original

    return manifest_str


def process_sample(policy_path: str, sample_name: str, policy_name: str) -> dict | None:
    """Process a single sample directory."""
    # policy_path is like "library/general/allowedrepos" (string from previous logic)
    # Be careful: policy_path passed from main loop is RELATIVE to LOCAL_REPO
    
    full_sample_path = LOCAL_REPO / policy_path / "samples" / sample_name
    
    if not full_sample_path.exists():
        return None

    files = {p.name: p for p in full_sample_path.iterdir() if p.is_file()}

    # Find constraint and examples
    constraint_file = files.get("constraint.yaml")
    allowed_files = [n for n in files if n.startswith("example_allowed")]
    disallowed_files = [n for n in files if n.startswith("example_disallowed") and "both" not in n]

    if not constraint_file or not allowed_files or not disallowed_files:
        return None

    constraint = read_file(constraint_file)
    allowed = read_file(files[allowed_files[0]])
    disallowed = read_file(files[disallowed_files[0]])
    # Fix: Forbidden Sysctls safe values
    # (Excluded: Logic removed as it was fragile)

    # Neutralize names, container names, and labels
    allowed = neutralize_manifest(allowed, "alpha")
    disallowed = neutralize_manifest(disallowed, "beta")

    # Apply global patching (initContainers, safe-images, etc.)
    allowed = patch_manifest(allowed)
    disallowed = patch_manifest(disallowed)

    return {
        "constraint": constraint,
        "allowed": allowed,
        "disallowed": disallowed,
        "sample_name": sample_name,
    }


def generate_benchmark(policy_name: str, category: str, sample: dict, idx: int):
    """Generate benchmark files for a sample."""
    task_name = f"gk-{category}-{policy_name}-{idx:02d}"
    task_dir = OUTPUT_DIR / task_name
    artifacts_dir = task_dir / "artifacts"
    artifacts_dir.mkdir(parents=True, exist_ok=True)

    namespace = f"gk-test-{idx:03d}"

    # Write artifacts
    (artifacts_dir / "resource-alpha.yaml").write_text(sample["allowed"])
    (artifacts_dir / "resource-beta.yaml").write_text(sample["disallowed"])
    (artifacts_dir / "constraint.yaml").write_text(sample["constraint"])

    # Prepare wait commands
    wait_cmds = []
    wait_cmds.extend(get_wait_command(sample["allowed"], namespace))
    wait_cmds.extend(get_wait_command(sample["disallowed"], namespace))

    if wait_cmds:
        wait_block = 'echo "Waiting for resources to be ready..."\n' + "\n".join(wait_cmds)
    else:
        wait_block = 'echo "Resources deployed. Waiting for readiness..."\nsleep 3'

    # Add privileged label for PSP tasks
    extra_setup = ""
    if "pod-security-policy" in category:
        extra_setup = f"kubectl label namespace {namespace} pod-security.kubernetes.io/enforce=privileged\n"

    # setup.sh
    setup = f"""#!/usr/bin/env bash
set -e
kubectl delete namespace {namespace} --ignore-not-found --wait=true
kubectl create namespace {namespace}
{extra_setup}sleep 2  # Allow namespace to stabilize
kubectl apply -f artifacts/resource-alpha.yaml -n {namespace}
kubectl apply -f artifacts/resource-beta.yaml -n {namespace}
sleep 3  # Allow pods to be scheduled
{wait_block}
"""
    (task_dir / "setup.sh").write_text(setup)
    os.chmod(task_dir / "setup.sh", 0o755)

    # cleanup.sh
    cleanup = f"""#!/usr/bin/env bash
kubectl delete namespace {namespace} --ignore-not-found
exit 0
"""
    (task_dir / "cleanup.sh").write_text(cleanup)
    os.chmod(task_dir / "cleanup.sh", 0o755)

    # Generate natural language description
    description = generate_description(sample["constraint"])
    if not description:
        description = "A compliance policy is in effect for this cluster."

    # task.yaml
    task = {
        "setup": "setup.sh",
        "cleanup": "cleanup.sh",
        "difficulty": "medium",
        "expect": [{
            "answer": "VIOLATING: resource-beta"
        }],
        "script": [{
            "prompt": f"""{description}
    
Look at the resources that have been deployed in namespace {namespace}. Examine them and determine which resource(s) violate the policy.

After your analysis, output a single line starting with "VIOLATING:" followed by the resource name(s) that violate the policy, or "VIOLATING: none" if all resources comply.
IMPORTANT: You must NOT output anything other than the XML block below in your final response.
<answer>VIOLATING: your-answer-here</answer>"""
        }]
    }
    (task_dir / "task.yaml").write_text(yaml.dump(task, default_flow_style=False, sort_keys=False))

    return task_name


def main():
    """Main scraper entry point."""
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    
    # 1. Clone Repo
    clone_repo()
    
    generated = []
    idx = 0

    # CATEGORIES = ["library/general", "library/pod-security-policy"] 
    # mapped to LOCAL_REPO / "library" / "general"
    
    for category_path in CATEGORIES:
        # category_path is like "library/general"
        category = category_path.split("/")[-1]
        print(f"Processing {category}...")
        
        local_category_path = LOCAL_REPO / category_path
        if not local_category_path.exists():
             print(f"  Category path {local_category_path} not found")
             continue

        for policy_dir in sorted(local_category_path.iterdir()):
            if not policy_dir.is_dir() or policy_dir.name.startswith("."):
                continue

            policy_name = policy_dir.name
            
            # SKIP excluded policies
            # Check if policy_name is in EXCLUDED_POLICIES or contains any of them
            should_exclude = any(ex in policy_name for ex in EXCLUDED_POLICIES)
            if should_exclude:
                print(f"  Skipping excluded policy: {policy_name}")
                continue

            print(f"  Policy: {policy_name}")
            
            samples_dir = policy_dir / "samples"
            if not samples_dir.exists():
                print(f"    No samples directory")
                continue

            for sample_dir in samples_dir.iterdir():
                if not sample_dir.is_dir() or sample_dir.name.startswith("."):
                    continue

                # Pass relative path for consistency if needed, or adjust process_sample
                # process_sample expects policy_path relative to repo root
                rel_policy_path = policy_dir.relative_to(LOCAL_REPO)
                
                sample = process_sample(str(rel_policy_path), sample_dir.name, policy_name)
                if sample:
                    task_name = generate_benchmark(policy_name, category, sample, idx)
                    generated.append(task_name)
                    print(f"    Generated: {task_name}")
                    idx += 1

    print(f"\\nGenerated {len(generated)} benchmarks in {OUTPUT_DIR}")


if __name__ == "__main__":
    main()

