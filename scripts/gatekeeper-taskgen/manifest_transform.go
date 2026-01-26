package main

import "strings"

type manifestRewriteContext struct {
	name     string
	ns       string
	taskID   string
	expected string
}

func rewriteManifest(doc map[string]any, name, ns, taskID, expected string) {
	res := NewResource(doc)
	ctx := manifestRewriteContext{
		name:     name,
		ns:       ns,
		taskID:   taskID,
		expected: expected,
	}

	applyIdentity(res, ctx)
	applyDeployabilityFixes(res)
}

func applyIdentity(res *Resource, ctx manifestRewriteContext) {
	res.SetName(ctx.name)
	res.SetNamespace(ctx.ns)
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
