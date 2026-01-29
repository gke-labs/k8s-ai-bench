// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"slices"
	"strings"

	"sigs.k8s.io/yaml"
)

type manifestRewriteContext struct {
	name     string
	ns       string
	taskID   string
	expected string
}

func rewriteManifest(doc map[string]any, name, ns, taskID, expected, constraintYAML string) {
	res := NewResource(doc)
	ctx := manifestRewriteContext{
		name:     name,
		ns:       ns,
		taskID:   taskID,
		expected: expected,
	}

	applyIdentity(res, ctx)
	applyDeployabilityFixes(res)

	if shouldNormalizeResources(constraintYAML) {
		enforceMinimalResources(res)
	}
}

func applyIdentity(res *Resource, ctx manifestRewriteContext) {
	res.SetName(ctx.name)
	if res.IsClusterScoped() {
		meta := res.NestedMap("metadata")
		delete(meta, "namespace")
	} else {
		res.SetNamespace(ctx.ns)
	}
	res.SetLabel("k8s-ai-bench/task", ctx.taskID)
}

func applyDeployabilityFixes(res *Resource) {
	fixInitContainers(res)
	fixBadImages(res)
}

// fixInitContainers adds exit command to init containers that would run forever
func fixInitContainers(res *Resource) {
	podSpec := podSpecForWorkload(res)
	if podSpec == nil {
		return
	}

	initContainers, ok := podSpec["initContainers"].([]any)
	if !ok {
		return
	}

	for _, c := range initContainers {
		container, ok := c.(map[string]any)
		if !ok {
			continue
		}
		image, _ := container["image"].(string)
		if strings.Contains(image, "nginx") {
			container["command"] = []any{"sh", "-c", "exit 0"}
			delete(container, "args")
		} else if strings.Contains(image, "opa") {
			// OPA image doesn't have sh, use built-in eval that exits
			container["command"] = []any{"opa", "eval", "true"}
			delete(container, "args")
		}
	}
}

func fixBadImages(res *Resource) {
	podSpec := podSpecForWorkload(res)
	if podSpec == nil {
		return
	}

	replacements := map[string]string{
		"tomcat":      "nginx",      // required-probes: policy checks probes, not image
		"nginx:1.7.9": "nginx:1.25", // old nginx tag doesn't exist
	}

	for _, key := range []string{"containers", "initContainers"} {
		containers, ok := podSpec[key].([]any)
		if !ok {
			continue
		}
		for _, c := range containers {
			container, ok := c.(map[string]any)
			if !ok {
				continue
			}
			image, ok := container["image"].(string)
			if !ok {
				continue
			}
			for bad, good := range replacements {
				if image == bad {
					container["image"] = good
				}
			}
		}
	}
}

func podSpecForWorkload(res *Resource) map[string]any {
	if res.Kind() != "Pod" {
		return nil
	}
	return res.Spec()
}

func shouldNormalizeResources(constraintYAML string) bool {
	var normalizationSkipKinds = []string{
		"K8sContainerLimits",
		"K8sContainerRequests",
		"K8sContainerRatios",
		"K8sRequiredResources",
		"K8sContainerEphemeralStorageLimit",
	}

	kind := constraintKind(constraintYAML)
	return !slices.Contains(normalizationSkipKinds, kind)
}

func constraintKind(raw string) string {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
		return ""
	}
	if v, ok := obj["kind"].(string); ok {
		return v
	}
	return ""
}

func normalizeYAML(raw, constraintYAML string) (string, error) {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
		return raw, err
	}
	res := NewResource(obj)

	if shouldNormalizeResources(constraintYAML) {
		enforceMinimalResources(res)
	}
	if shouldNormalizeReplicas(constraintYAML) {
		enforceMinimalReplicas(res)
	}

	out, err := yaml.Marshal(res.Object)
	if err != nil {
		return raw, err
	}
	return strings.TrimSpace(string(out)), nil
}

func enforceMinimalResources(res *Resource) {
	podSpec := podSpecForWorkload(res)
	if podSpec == nil {
		return
	}

	for _, field := range []string{"containers", "initContainers"} {
		if list, ok := podSpec[field].([]any); ok {
			for i, item := range list {
				container, ok := item.(map[string]any)
				if !ok {
					continue
				}

				if resources, ok := container["resources"].(map[string]any); ok {
					if limits, ok := resources["limits"].(map[string]any); ok {
						if _, ok := limits["cpu"]; ok {
							limits["cpu"] = "1m"
						}
						if _, ok := limits["memory"]; ok {
							limits["memory"] = "1Mi"
						}
					}
					if requests, ok := resources["requests"].(map[string]any); ok {
						if _, ok := requests["cpu"]; ok {
							requests["cpu"] = "1m"
						}
						if _, ok := requests["memory"]; ok {
							requests["memory"] = "1Mi"
						}
					}
				}

				list[i] = container
			}
		}
	}
}

func enforceMinimalReplicas(res *Resource) {
	var replicaWorkloads = []string{
		"Deployment",
		"StatefulSet",
		"ReplicaSet",
		"ReplicationController",
	}

	if slices.Contains(replicaWorkloads, res.Kind()) {
		spec := res.Spec()
		if spec == nil {
			return
		}
		if _, ok := spec["replicas"]; ok {
			spec["replicas"] = 1
		}
	}
}

func shouldNormalizeReplicas(constraintYAML string) bool {
	if constraintYAML == "" {
		return true
	}
	kind := constraintKind(constraintYAML)
	return !strings.Contains(strings.ToLower(kind), "replica")
}
