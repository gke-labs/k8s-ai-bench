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
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

type templateMeta struct {
	Title         string
	Description   string
	TemplateYAML  string
	NameSensitive bool
}

type exampleCollector struct {
	Alpha []string
	Beta  []string
}

type docInfo struct {
	doc     *Resource
	newName string
}

type manifestGenConfig struct {
	outDir         string
	taskID         string
	defaultNS      string
	constraintYAML string
	nameSensitive  bool
}

type manifestGenState struct {
	nameRegistry *nameRegistry
	resourceIdx  int
	alphaFileIdx int
	betaFileIdx  int
	nsSet        map[string]bool
	artifacts    *TaskArtifacts
	examples     *exampleCollector
}

// GenerateManifests processes task cases and generates artifact files
func GenerateManifests(task TaskMetadata, outDir string) (TaskArtifacts, PromptContext, error) {
	os.MkdirAll(filepath.Join(outDir, "artifacts"), 0755)

	defaultNS := "gk-" + task.TaskID
	artifacts := TaskArtifacts{
		CaseFiles:      map[string][]string{},
		InventoryFiles: map[string][]string{},
	}
	meta := loadTemplateMeta(task.TemplatePath)
	constraintYAML := loadConstraintYAML(task.ConstraintPath, defaultNS)
	examples := exampleCollector{}
	cfg := manifestGenConfig{
		outDir:         outDir,
		taskID:         task.TaskID,
		defaultNS:      defaultNS,
		constraintYAML: constraintYAML,
		nameSensitive:  meta.NameSensitive,
	}
	state := newManifestGenState(&artifacts, &examples)

	for _, c := range task.Cases {
		caseDocs, _ := readYAMLDocs(c.ObjectPath)
		if len(caseDocs) == 0 {
			continue
		}

		docInfos := buildDocInfos(cfg, &state, caseDocs, c.Expected)
		if len(docInfos) == 0 {
			continue
		}

		writeCaseArtifacts(cfg, &state, c, docInfos)
		writeInventoryArtifacts(cfg, &state, c)
	}

	artifacts.Namespaces = sortedKeys(state.nsSet)
	promptCtx := buildPromptContext(task.TaskID, meta, constraintYAML, defaultNS, artifacts, examples)
	return artifacts, promptCtx, nil
}

func newManifestGenState(artifacts *TaskArtifacts, examples *exampleCollector) manifestGenState {
	return manifestGenState{
		nameRegistry: newNameRegistry(),
		resourceIdx:  1,
		alphaFileIdx: 1,
		betaFileIdx:  1,
		nsSet:        map[string]bool{},
		artifacts:    artifacts,
		examples:     examples,
	}
}

func buildPromptContext(taskID string, meta templateMeta, constraintYAML, defaultNS string, artifacts TaskArtifacts, examples exampleCollector) PromptContext {
	namespacedKindsSet := map[string]bool{}
	clusterKindsSet := map[string]bool{}
	for _, manifest := range artifacts.Manifests {
		if isClusterScopedKind(manifest.Kind) {
			clusterKindsSet[manifest.Kind] = true
		} else {
			namespacedKindsSet[manifest.Kind] = true
		}
	}

	namespaceForPrompt := defaultNS
	if len(namespacedKindsSet) == 0 {
		namespaceForPrompt = ""
	}

	return PromptContext{
		TaskID:          taskID,
		Title:           meta.Title,
		Description:     meta.Description,
		TemplateYAML:    meta.TemplateYAML,
		ConstraintYAML:  constraintYAML,
		AlphaExamples:   examples.Alpha,
		BetaExamples:    examples.Beta,
		Namespace:       namespaceForPrompt,
		NamespacedKinds: sortedKeys(namespacedKindsSet),
		ClusterKinds:    sortedKeys(clusterKindsSet),
	}
}

func loadTemplateMeta(path string) templateMeta {
	meta := templateMeta{}
	if docs, _ := readYAMLDocs(path); len(docs) > 0 {
		if data, err := os.ReadFile(path); err == nil {
			meta.TemplateYAML = string(data)
			meta.NameSensitive = templateRequiresLiteralName(meta.TemplateYAML)
		}
		res := NewResource(docs[0])
		if m := res.NestedMap("metadata"); m != nil {
			if ann := ensureMap(m, "annotations"); ann != nil {
				if v, ok := ann["metadata.gatekeeper.sh/title"].(string); ok {
					meta.Title = v
				}
				if v, ok := ann["description"].(string); ok {
					meta.Description = strings.TrimSpace(v)
				}
			}
		}
	}
	return meta
}

func loadConstraintYAML(path, defaultNS string) string {
	if data, err := os.ReadFile(path); err == nil {
		// Mirror the on-disk namespace rewrite so the prompt reflects the isolated namespace.
		return rewriteConstraintForPrompt(data, defaultNS)
	}
	return ""
}

func buildDocInfos(cfg manifestGenConfig, state *manifestGenState, docs []map[string]any, expected string) []docInfo {
	var allDocs []docInfo
	for _, doc := range docs {
		res := NewResource(doc)
		if isAdmissionReview(res) || !isDeployable(res) {
			continue
		}
		kind := res.Kind()
		namespace := cfg.defaultNS
		if res.IsClusterScoped() {
			namespace = ""
		}

		// obfuscate resources names to the model
		baseName := fmt.Sprintf("resource-%03d", state.resourceIdx)
		if cfg.nameSensitive && expected == "beta" {
			if orig := res.Name(); orig != "" {
				baseName = orig
			}
		}
		state.resourceIdx++

		name, _ := state.nameRegistry.allocate(kind, namespace, baseName)
		allDocs = append(allDocs, docInfo{doc: res, newName: name})
	}
	return allDocs
}

func writeCaseArtifacts(cfg manifestGenConfig, state *manifestGenState, taskCase TaskCase, docs []docInfo) {
	for _, d := range docs {
		rewriteManifest(d.doc.Object, d.newName, cfg.defaultNS, cfg.taskID, taskCase.Expected, cfg.constraintYAML)
		kind := d.doc.Kind()
		ns := d.doc.Namespace()
		if ns != "" {
			state.nsSet[ns] = true
		}

		var fileName string
		if taskCase.Expected == "alpha" {
			fileName = fmt.Sprintf("alpha-%02d.yaml", state.alphaFileIdx)
			state.alphaFileIdx++
		} else {
			fileName = fmt.Sprintf("beta-%02d.yaml", state.betaFileIdx)
			state.betaFileIdx++
		}
		relPath := "artifacts/" + fileName

		data, _ := yaml.Marshal(d.doc.Object)
		os.WriteFile(filepath.Join(cfg.outDir, relPath), data, 0644)

		if taskCase.Expected == "alpha" && len(state.examples.Alpha) < 2 {
			state.examples.Alpha = append(state.examples.Alpha, string(data))
		} else if taskCase.Expected == "beta" && len(state.examples.Beta) < 2 {
			state.examples.Beta = append(state.examples.Beta, string(data))
		}

		state.artifacts.Manifests = append(state.artifacts.Manifests, TaskManifest{
			Path:      filepath.Join(cfg.outDir, relPath),
			RelPath:   relPath,
			Doc:       d.doc.Object,
			CaseName:  taskCase.Name,
			Expected:  taskCase.Expected,
			Kind:      kind,
			Name:      d.newName,
			Namespace: ns,
		})

		state.artifacts.CaseFiles[taskCase.Name] = append(state.artifacts.CaseFiles[taskCase.Name], relPath)
	}
}

func writeInventoryArtifacts(cfg manifestGenConfig, state *manifestGenState, taskCase TaskCase) {
	for i, invPath := range taskCase.Inventory {
		invDocs, _ := readYAMLDocs(invPath)
		for j, doc := range invDocs {
			res := NewResource(doc)
			name := res.Name()
			if name == "" {
				name = fmt.Sprintf("inventory-%s-%d-%d", taskCase.Name, i, j)
			}

			// Rewrite with original name
			rewriteManifest(doc, name, cfg.defaultNS, cfg.taskID, "inventory", cfg.constraintYAML)
			ns := res.Namespace()
			if ns != "" {
				state.nsSet[ns] = true
			}

			// Save
			fileName := fmt.Sprintf("inventory-%s-%d-%d.yaml", taskCase.Name, i, j)
			relPath := "artifacts/" + fileName
			data, _ := yaml.Marshal(doc)
			os.WriteFile(filepath.Join(cfg.outDir, relPath), data, 0644)

			state.artifacts.Manifests = append(state.artifacts.Manifests, TaskManifest{
				Path:      filepath.Join(cfg.outDir, relPath),
				RelPath:   relPath,
				Doc:       doc,
				CaseName:  taskCase.Name,
				Expected:  "inventory",
				Kind:      res.Kind(),
				Name:      name,
				Namespace: res.Namespace(),
			})
			state.artifacts.InventoryFiles[taskCase.Name] = append(state.artifacts.InventoryFiles[taskCase.Name], relPath)
		}
	}
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
		if containers, ok := spec[key].([]any); ok {
			for _, c := range containers {
				if cm, ok := c.(map[string]any); ok {
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

var nameLiteralPattern = regexp.MustCompile(`metadata\.name\s*[=!]=\s*"`)

func templateRequiresLiteralName(templateYAML string) bool {
	return nameLiteralPattern.MatchString(templateYAML)
}
