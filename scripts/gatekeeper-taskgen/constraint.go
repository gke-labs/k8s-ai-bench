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
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// rewriteConstraint updates the constraint file to target the task namespace
// when the constraint already specifies a match.namespaces list.
func rewriteConstraint(src, dst, ns string) error {
	doc, err := readConstraintYAML(src)
	if err != nil {
		return err
	}

	changed, msg := rewriteConstraintNamespaces(doc, ns)
	if changed {
		fmt.Printf("Rewriting constraint %s: %s\n", dst, msg)
	} else if msg != "" {
		fmt.Printf("Constraint %s %s\n", dst, msg)
	}

	return writeConstraintYAML(dst, doc)
}

func rewriteConstraintForPrompt(raw []byte, ns string) string {
	doc, err := decodeConstraintYAML(raw)
	if err != nil {
		return string(raw)
	}

	changed, _ := rewriteConstraintNamespaces(doc, ns)
	if !changed {
		return string(raw)
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

func readConstraintYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeConstraintYAML(data)
}

func decodeConstraintYAML(data []byte) (map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return doc, nil
}

func writeConstraintYAML(path string, doc map[string]any) error {
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// rewriteConstraintNamespaces applies the only manual transform we need:
// if spec.match.namespaces is present and non-empty, replace it with the task
// namespace to keep constraints scoped to the isolated test namespace.
func rewriteConstraintNamespaces(doc map[string]any, ns string) (bool, string) {
	spec, ok := getMap(doc, "spec")
	if !ok {
		return false, "spec not found; leaving unchanged"
	}
	match, ok := getMap(spec, "match")
	if !ok {
		return false, "spec.match not found; leaving unchanged"
	}
	rawNamespaces, ok := match["namespaces"]
	if !ok {
		return false, "spec.match.namespaces not found; leaving unchanged"
	}
	namespaces, ok := toStringSlice(rawNamespaces)
	if !ok {
		return false, "spec.match.namespaces is not a string list; leaving unchanged"
	}
	if len(namespaces) == 0 {
		return false, "spec.match.namespaces empty; leaving unchanged"
	}

	match["namespaces"] = []string{ns}

	// Scrub leaking labels if present
	if meta, ok := getMap(doc, "metadata"); ok {
		if labels, ok := getMap(meta, "labels"); ok {
			delete(labels, "k8s-ai-bench/expected")
			delete(labels, "k8s-ai-bench/task")
		}
	}

	return true, fmt.Sprintf("spec.match.namespaces %v -> [%q] (and labels scrubbed)", namespaces, ns)
}

func getMap(doc map[string]any, key string) (map[string]any, bool) {
	child, ok := doc[key].(map[string]any)
	return child, ok
}

func toStringSlice(value any) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		return v, true
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}
