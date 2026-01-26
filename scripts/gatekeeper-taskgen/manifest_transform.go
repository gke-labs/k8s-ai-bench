package main

import "strings"

// Manifest rewrites are grouped to make intent explicit:
// 1) identity: isolate the task with stable names/namespaces/labels
// 2) references: keep object references consistent with renamed resources
// 3) deployability: safe tweaks that avoid stuck pods or image pull failures
type manifestRewriteContext struct {
	name     string
	ns       string
	taskID   string
	expected string
}

func rewriteManifest(doc map[string]interface{}, name, ns, taskID, expected string) {
	ctx := manifestRewriteContext{
		name:     name,
		ns:       ns,
		taskID:   taskID,
		expected: expected,
	}

	applyIdentity(doc, ctx)
	applyDeployabilityFixes(doc)
}

func applyIdentity(doc map[string]interface{}, ctx manifestRewriteContext) {
	meta := ensureMap(doc, "metadata")
	meta["name"] = ctx.name
	meta["namespace"] = ctx.ns

	labels := ensureMap(meta, "labels")
	labels["k8s-ai-bench/task"] = ctx.taskID
	labels["k8s-ai-bench/expected"] = ctx.expected
}

func applyDeployabilityFixes(doc map[string]interface{}) {
	fixInitContainers(doc)
	fixBadImages(doc)
}

// fixInitContainers adds exit command to init containers that would run forever
func fixInitContainers(doc map[string]interface{}) {
	podSpec := podSpecForWorkload(doc)
	if podSpec == nil {
		return
	}

	initContainers, ok := podSpec["initContainers"].([]interface{})
	if !ok {
		return
	}

	for _, c := range initContainers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		// Init containers need to exit for the pod to start.
		// Override command/args with a simple exit for images that run servers.
		image, _ := container["image"].(string)
		if strings.Contains(image, "nginx") {
			container["command"] = []interface{}{"sh", "-c", "exit 0"}
			delete(container, "args")
		} else if strings.Contains(image, "opa") {
			// OPA image doesn't have sh, use built-in eval that exits
			container["command"] = []interface{}{"opa", "eval", "true"}
			delete(container, "args")
		}
	}
}

// fixBadImages replaces images that fail to pull with working alternatives
// Only for images where the replacement doesn't affect the policy test
func fixBadImages(doc map[string]interface{}) {
	podSpec := podSpecForWorkload(doc)
	if podSpec == nil {
		return
	}

	// Only fix specific images where replacement doesn't break test semantics
	replacements := map[string]string{
		"tomcat":      "nginx",      // required-probes: policy checks probes, not image
		"nginx:1.7.9": "nginx:1.25", // old nginx tag doesn't exist
	}

	for _, key := range []string{"containers", "initContainers"} {
		containers, ok := podSpec[key].([]interface{})
		if !ok {
			continue
		}
		for _, c := range containers {
			container, ok := c.(map[string]interface{})
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

func podSpecForWorkload(doc map[string]interface{}) map[string]interface{} {
	kind := getStr(doc, "kind")
	if kind != "Pod" {
		return nil
	}
	podSpec, _ := doc["spec"].(map[string]interface{})
	return podSpec
}
