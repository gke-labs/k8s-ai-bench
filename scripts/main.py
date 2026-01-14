#!/usr/bin/env python3
import os
import yaml
import requests
from pathlib import Path
import subprocess
import shutil
import time

GITHUB_API = "https://api.github.com/repos/open-policy-agent/gatekeeper-library/contents"
RAW_BASE = "https://raw.githubusercontent.com/open-policy-agent/gatekeeper-library/master"
CATEGORIES = ["library/general"]
OUTPUT_DIR = Path(__file__).parent.parent / "tasks" / "gatekeeper"
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY", "")

EXCLUDED_POLICIES = [
    "verifydeprecatedapi",
    "ephemeralstoragelimit",
    "forbidden-sysctls",  # Excluded: requires complex sysctl values that are hard to patch safely
    "flexvolume-drivers", # Excluded: test drivers don't exist on standard clusters
    "proc-mount",         # Excluded: requires Kubelet Featuregate explicitly enabled
    "read-only-root-filesystem", # Excluded: requires specific image or complex patching
    "containerresourceratios",   # Excluded: Can produce invalid K8s manifests (requests > limits) which K8s catches natively
    "allowedrepos",              # Excluded: Users preferred v2
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

def fix_manifest_with_gemini(manifest: str, policy_desc: str, policy_name: str, is_allowed: bool) -> str:
    """Use Gemini to intelligently fix the manifest to be deployable while preserving test logic."""
    if not GEMINI_API_KEY:
        return manifest  # Fallback to original if no key

    test_type = "ALLOWED (Should be admitted)" if is_allowed else "DISALLOWED (Should be blocked)"
    preserve_instr = "Ensure the resource remains COMPLIANT with the policy." if is_allowed else "Ensure the resource remains NON-COMPLIANT (violating) the policy."

    prompt = f"""You are an expert Kubernetes engineer.
I have a Kubernetes manifest that is used as a test case for a Gatekeeper policy.

Policy Name: {policy_name}
Policy Description: {policy_desc}
Test Type: {test_type}

Manifest:
```yaml
{manifest}
```

Your Task:
1. **PRIMARY GOAL**: {preserve_instr}
   - You MUST ensure the final manifest yields the expected result (Allowed or Disallowed) when validated against the policy.
   - If the policy checks for specific image tags (e.g., "latest"), ensures your image choice reflects that (e.g., use 'nginx:latest' to violate, 'nginx:1.25' to comply).

2. **SECONDARY GOAL**: Fix "noise" to make it deployable on Kind.
   - **ENSURE KUBERNETES VALIDITY**: The manifest must pass basic API validation.
     - `resources.requests` MUST NOT be greater than `resources.limits` (unless the policy *specifically* tests this invalid state).
     - Required fields (like `image`) must be present.
   - Replace obscure or placeholder images (like 'openpolicyagent/opa', 'foo', 'ubuntu') with 'nginx' or 'busybox', **UNLESS** the policy specifically requires the original image name.
   - When replacing images, **carefully select the tag** to satisfy the Primary Goal.
   - Remove invalid arguments or commands that would cause 'nginx' to crash.
   - Ensure 'securityContext' is valid (remove 'Localhost' seccomp profiles).
   - If 'readOnlyRootFilesystem: true' is required/present, ADD an 'emptyDir' volume at '/tmp' (or appropriate path) so the pod can start.

3. Return ONLY the cleaned, valid YAML block.
"""
    try:
        time.sleep(1) # Rate limit safety
        resp = requests.post(
            f"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key={GEMINI_API_KEY}",
            json={
                "contents": [{"parts": [{"text": prompt}]}]
            },
            timeout=60,
        )
        if resp.ok:
            content = resp.json()["candidates"][0]["content"]["parts"][0]["text"].strip()
            # Strip markdown code blocks if present
            if content.startswith("```yaml"):
                content = content[7:]
            if content.startswith("```"):
                content = content[3:]
            if content.endswith("```"):
                content = content[:-3]
            return content.strip()
        else:
            print(f"Gemini API Error: {resp.status_code} - {resp.text}")
    except Exception as e:
        print(f"Gemini API Exception: {e}")
    
    return manifest # Fallback


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

def neutralize_manifest_with_index(manifest: str, suffix: str, index: int) -> str:
    """Neutralize manifest with index to ensure uniqueness."""
    docs = list(yaml.safe_load_all(manifest))
    for doc in docs:
        if not doc or "metadata" not in doc:
            continue
        doc["metadata"]["name"] = f"resource-{suffix}-{index}"
        doc["metadata"].pop("namespace", None)
        
        new_app_label = f"app-{suffix}"

        def update_labels(obj):
            if not isinstance(obj, dict): return
            if "labels" in obj and "app" in obj["labels"]:
                obj["labels"]["app"] = new_app_label
            if "selector" in obj:
                selector = obj["selector"]
                if "matchLabels" in selector and "app" in selector["matchLabels"]:
                    selector["matchLabels"]["app"] = new_app_label
            if "template" in obj:
                update_labels(obj["template"]["metadata"])

        if "labels" in doc["metadata"]:
             if "app" in doc["metadata"]["labels"]:
                 doc["metadata"]["labels"]["app"] = new_app_label

        if "spec" in doc:
            update_labels(doc["spec"])

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
                new_name = f"{prefix}-{suffix}-{index}-{i}" if i else f"{prefix}-{suffix}-{index}"
                c["name"] = new_name
                if old_name:
                    renames[old_name] = new_name
        
        # Generic Fix: Update annotations that reference container names
        annotations = doc["metadata"].get("annotations", {})
        if annotations and renames:
            new_annotations = {}
            for k, v in annotations.items():
                updated_k = k
                for old, new in renames.items():
                    if k.endswith("/" + old):
                         updated_k = k.replace("/" + old, "/" + new)
                         break
                new_annotations[updated_k] = v
            doc["metadata"]["annotations"] = new_annotations

    return yaml.dump_all(docs, default_flow_style=False)

# patch_manifest is no longer used, replaced by fix_manifest_with_gemini

def process_sample(policy_path: str, sample_name: str, policy_name: str) -> dict | None:
    """Process a single sample directory."""
    
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
    description = generate_description(constraint)

    # Concatenate all allowed files
    allowed_contents = []
    for i, fname in enumerate(sorted(allowed_files)):
        content = read_file(files[fname])
        neutralized_content = neutralize_manifest_with_index(content, "alpha", i)
        allowed_contents.append(neutralized_content)
    
    allowed = "\n---\n".join(allowed_contents)

    # Concatenate all disallowed files
    disallowed_contents = []
    for i, fname in enumerate(sorted(disallowed_files)):
        content = read_file(files[fname])
        neutralized_content = neutralize_manifest_with_index(content, "beta", i)
        disallowed_contents.append(neutralized_content)

    disallowed = "\n---\n".join(disallowed_contents)

    # Apply Gemini fixing
    allowed = fix_manifest_with_gemini(allowed, description, policy_name, is_allowed=True)
    disallowed = fix_manifest_with_gemini(disallowed, description, policy_name, is_allowed=False)

    return {
        "constraint": constraint,
        "description": description, # Pass description to generate_benchmark
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

    (artifacts_dir / "resource-alpha.yaml").write_text(sample["allowed"])
    (artifacts_dir / "resource-beta.yaml").write_text(sample["disallowed"])

    # Prepare wait commands
    # Skipped waiting for readiness to simplify setup - we only care about admission (creation)
    wait_cmds = []
    # wait_cmds.extend(get_wait_command(sample["allowed"], namespace))
    # wait_cmds.extend(get_wait_command(sample["disallowed"], namespace))

    if wait_cmds:
        wait_block = 'echo "Waiting for resources to be ready..."\n' + "\n".join(wait_cmds)
    else:
        wait_block = 'echo "Resources deployed. Waiting for stability..."\nsleep 5'

    # Add privileged label for PSP tasks
    extra_setup = ""

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
    description = sample.get("description", "")
    if not description:
        description = "A compliance policy is in effect for this cluster."

    # task.yaml
    task = {
        "setup": "setup.sh",
        "cleanup": "cleanup.sh",
        "difficulty": "medium",
        "expect": [{
            "contains": "VIOLATING:.*resource-beta.*",
            "notContains": "resource-alpha"
        }],
        "script": [{
            "prompt": f"""{description}
    
Look at the resources that have been deployed in namespace {namespace}. Examine them and determine which resource(s) violate the policy.

After your analysis, output a single line starting with "VIOLATING:" followed by the resource name(s) that violate the policy, or "VIOLATING: none" if all resources comply.
The output format must be EXACTLY as follows, with no leading whitespace or extra text:
VIOLATING: resource-name-1, resource-name-2

IMPORTANT: You must NOT output anything other than the XML block below in your final response.
<answer>VIOLATING: your-answer-here</answer>"""
        }]
    }
    (task_dir / "task.yaml").write_text(yaml.dump(task, default_flow_style=False, sort_keys=False))

    return task_name


def main():
    """Main scraper entry point."""
    # Clean up old tasks first
    if OUTPUT_DIR.exists():
        shutil.rmtree(OUTPUT_DIR)
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    
    clone_repo()
    
    generated = []
    idx = 0

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

