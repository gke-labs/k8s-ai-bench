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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// GenerateManifests processes task cases and generates artifact files
func GenerateManifests(task TaskMetadata, outDir string) (TaskArtifacts, PromptContext, error) {
	os.MkdirAll(filepath.Join(outDir, "artifacts"), 0755)

	defaultNS := "gk-" + task.TaskID
	artifacts := TaskArtifacts{
		CaseFiles: map[string][]string{},
	}
	resourceIdx := 1
	alphaFileIdx, betaFileIdx := 1, 1
	nsSet := map[string]bool{defaultNS: true}
	nameRegistry := newNameRegistry()

	var templateTitle, templateDesc, templateYAML, constraintYAML string
	var alphaExamples, betaExamples []string

	// Read template metadata
	if docs, _ := readYAMLDocs(task.TemplatePath); len(docs) > 0 {
		if data, err := os.ReadFile(task.TemplatePath); err == nil {
			templateYAML = string(data)
		}
		res := NewResource(docs[0])
		if meta := res.NestedMap("metadata"); meta != nil {
			if ann := ensureMap(meta, "annotations"); ann != nil {
				if v, ok := ann["metadata.gatekeeper.sh/title"].(string); ok {
					templateTitle = v
				}
				if v, ok := ann["description"].(string); ok {
					templateDesc = strings.TrimSpace(v)
				}
			}
		}
	}

	// Read constraint
	if data, err := os.ReadFile(task.ConstraintPath); err == nil {
		// Mirror the on-disk namespace rewrite so the prompt reflects the isolated namespace.
		constraintYAML = rewriteConstraintForPrompt(data, defaultNS)
	}

	for _, c := range task.Cases {
		caseDocs, _ := readYAMLDocs(c.ObjectPath)
		if len(caseDocs) == 0 {
			continue
		}

		firstRes := NewResource(caseDocs[0])
		if isAdmissionReview(firstRes) || !isDeployable(firstRes) {
			continue
		}

		// Build docs
		type docInfo struct {
			doc     *Resource
			newName string
		}
		var allDocs []docInfo

		for _, doc := range caseDocs[:1] {
			res := NewResource(doc)
			kind := res.Kind()
			namespace := defaultNS

			// obfuscate resources names to the model
			baseName := fmt.Sprintf("resource-%03d", resourceIdx)
			resourceIdx++

			name, _ := nameRegistry.allocate(kind, namespace, baseName)
			allDocs = append(allDocs, docInfo{res, name})
		}

		// Rewrite and save
		for _, d := range allDocs {
			rewriteManifest(d.doc.Object, d.newName, defaultNS, task.TaskID, c.Expected, constraintYAML)
			kind := d.doc.Kind()
			ns := d.doc.Namespace()
			if ns != "" {
				nsSet[ns] = true
			}

			var fileName string
			if c.Expected == "alpha" {
				fileName = fmt.Sprintf("alpha-%02d.yaml", alphaFileIdx)
				alphaFileIdx++
			} else {
				fileName = fmt.Sprintf("beta-%02d.yaml", betaFileIdx)
				betaFileIdx++
			}
			relPath := "artifacts/" + fileName

			data, _ := yaml.Marshal(d.doc.Object)
			os.WriteFile(filepath.Join(outDir, relPath), data, 0644)

			if c.Expected == "alpha" && len(alphaExamples) < 2 {
				alphaExamples = append(alphaExamples, string(data))
			} else if c.Expected == "beta" && len(betaExamples) < 2 {
				betaExamples = append(betaExamples, string(data))
			}

			artifacts.Manifests = append(artifacts.Manifests, TaskManifest{
				Path:      filepath.Join(outDir, relPath),
				RelPath:   relPath,
				Doc:       d.doc.Object,
				CaseName:  c.Name,
				Expected:  c.Expected,
				Kind:      kind,
				Name:      d.newName,
				Namespace: ns,
			})

			artifacts.CaseFiles[c.Name] = append(artifacts.CaseFiles[c.Name], relPath)
		}

		// Process Inventory
		for i, invPath := range c.Inventory {
			invDocs, _ := readYAMLDocs(invPath)
			for j, doc := range invDocs {
				res := NewResource(doc)
				// KEEP ORIGINAL NAME for inventory to support references
				name := res.Name()
				if name == "" {
					name = fmt.Sprintf("inventory-%s-%d-%d", c.Name, i, j)
				}

				// Rewrite with original name
				rewriteManifest(doc, name, defaultNS, task.TaskID, "inventory", constraintYAML)

				// Save
				fileName := fmt.Sprintf("inventory-%s-%d-%d.yaml", c.Name, i, j)
				relPath := "artifacts/" + fileName
				data, _ := yaml.Marshal(doc)
				os.WriteFile(filepath.Join(outDir, relPath), data, 0644)

				artifacts.Manifests = append(artifacts.Manifests, TaskManifest{
					Path:      filepath.Join(outDir, relPath),
					RelPath:   relPath,
					Doc:       doc,
					CaseName:  c.Name,
					Expected:  "inventory",
					Kind:      res.Kind(),
					Name:      name,
					Namespace: defaultNS,
				})
				artifacts.CaseFiles[c.Name] = append(artifacts.CaseFiles[c.Name], relPath)
			}
		}
	}

	artifacts.Namespaces = sortedKeys(nsSet)

	namespacedKindsSet := map[string]bool{}
	for _, manifest := range artifacts.Manifests {
		namespacedKindsSet[manifest.Kind] = true
	}

	promptCtx := PromptContext{
		TaskID:          task.TaskID,
		Title:           templateTitle,
		Description:     templateDesc,
		TemplateYAML:    templateYAML,
		ConstraintYAML:  constraintYAML,
		AlphaExamples:   alphaExamples,
		BetaExamples:    betaExamples,
		Namespace:       defaultNS,
		NamespacedKinds: sortedKeys(namespacedKindsSet),
		ClusterKinds:    []string{},
	}

	return artifacts, promptCtx, nil
}

func isAdmissionReview(res *Resource) bool {
	return res.Kind() == "AdmissionReview"
}

func isDeployable(res *Resource) bool {
	if res.Kind() != "Pod" {
		return true
	}
	spec := res.Spec()
	if _, hasEphemeral := spec["ephemeralContainers"]; hasEphemeral {
		return false
	}
	names := map[string]bool{}
	for _, key := range []string{"containers", "initContainers"} {
		if containers, ok := spec[key].([]interface{}); ok {
			for _, c := range containers {
				if cm, ok := c.(map[string]interface{}); ok {
					if name, ok := cm["name"].(string); ok {
						if names[name] {
							return false
						}
						names[name] = true
					}
				}
			}
		}
	}
	return true
}

// YAML helpers

func readYAMLDocs(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var results []map[string]any
	for doc := range bytes.SplitSeq(data, []byte("---")) {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		var obj map[string]any
		if yaml.Unmarshal(doc, &obj) == nil {
			results = append(results, obj)
		}
	}
	return results, nil
}
