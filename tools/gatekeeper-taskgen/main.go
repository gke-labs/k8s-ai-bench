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
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"google.golang.org/genai"
	"sigs.k8s.io/yaml"
)

var defaultSkipList = []string{
	// requires custom storage types
	"storageclass",
	"storageclass-allowlist",
	// deprecated apis
	"verifydeprecatedapi-1.16",
	"verifydeprecatedapi-1.22",
	"verifydeprecatedapi-1.25",
	"verifydeprecatedapi-1.26",
	"verifydeprecatedapi-1.27",
	"verifydeprecatedapi-1.29",
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.LibraryRoot, "library-root", ".gatekeeper-library/library/general", "Path to gatekeeper-library general directory")
	flag.StringVar(&cfg.OutputDir, "output-dir", "tasks/gatekeeper", "Directory to write tasks")
	flag.Var(&stringSliceFlag{&cfg.SkipList}, "skip", "Patterns to skip (can be repeated)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&cfg.Verify, "verify", false, "Run gator verify on generated tasks")
	flag.BoolVar(&cfg.VerifyOnly, "verify-only", false, "Run gator verify on existing tasks without generation")
	flag.BoolVar(&cfg.Repair, "repair", false, "Run repair on generated tasks using Gemini")
	flag.Parse()

	cfg.SkipList = append(cfg.SkipList, defaultSkipList...)

	// Initialize Gemini client if API key is available
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize Gemini client: %v\n", err)
		} else {
			cfg.GeminiClient = client
			fmt.Println("Gemini client initialized - will generate prompts using AI")
		}
	} else if !cfg.VerifyOnly {
		fmt.Fprintln(os.Stderr, "GEMINI_API_KEY not set - Gemini is required for prompt generation")
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(cfg Config) error {
	if cfg.VerifyOnly {
		return verifyTasks(cfg.OutputDir)
	}

	taskMap, err := ParseSuites(cfg.LibraryRoot)
	if err != nil {
		return err
	}
	if len(taskMap) == 0 {
		return fmt.Errorf("no suite.yaml files found under %s", cfg.LibraryRoot)
	}

	os.MkdirAll(cfg.OutputDir, 0755)

	var generated, skipped int
	for _, id := range sortedKeys(taskMap) {
		task := taskMap[id]
		if skip, reason := shouldSkip(cfg, task); skip {
			fmt.Printf("Skipped %s: %s\n", id, reason)
			skipped++
			continue
		}
		if err := generateTask(cfg, task); err != nil {
			fmt.Printf("Skipped %s: %v\n", id, err)
			skipped++
		} else {
			if cfg.Verbose {
				fmt.Printf("Generated task %s\n", id)
			}
			generated++
		}
	}
	fmt.Printf("Generated tasks: %d (skipped %d)\n", generated, skipped)

	if cfg.Verify {
		if err := verifyTasks(cfg.OutputDir); err != nil {
			// Don't fail the entire run if verification fails, just report it
			fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
		}
	}

	if cfg.Repair {
		return runRepair(cfg)
	}

	return nil
}

func runRepair(cfg Config) error {
	taskMap, err := ParseSuites(cfg.LibraryRoot)
	if err != nil {
		return err
	}

	var allResults []RepairResult
	var repaired, errorsCount int
	var mu sync.Mutex

	fmt.Printf("Starting repair on %s...\n", cfg.OutputDir)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit concurrency to 10

	for _, id := range sortedKeys(taskMap) {
		outDir := filepath.Join(cfg.OutputDir, id)
		if _, err := os.Stat(outDir); os.IsNotExist(err) {
			continue
		}

		wg.Add(1)
		go func(id, outDir string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, err := repairTask(cfg, outDir, id)

			mu.Lock()
			defer mu.Unlock()

			allResults = append(allResults, results...)

			for _, r := range results {
				switch r.Status {
				case "repaired":
					repaired++
					fmt.Printf("Repaired %s: %s\n", id, filepath.Base(r.FilePath))
				case "error":
					errorsCount++
					fmt.Printf("Error repairing %s: %s\n", id, r.Error)
				}
			}
			if err != nil && cfg.Verbose {
				fmt.Printf("Repair warning for %s: %v\n", id, err)
			}
		}(id, outDir)
	}

	wg.Wait()

	fmt.Printf("Repair complete. Repaired: %d, Errors: %d\n", repaired, errorsCount)

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].TaskID < allResults[j].TaskID
	})

	return writeRepairReport(cfg.OutputDir, allResults)
}

func shouldSkip(cfg Config, task TaskMetadata) (bool, string) {
	if slices.ContainsFunc(cfg.SkipList, func(skip string) bool {
		return skip == task.TestName || skip == task.SuiteName || strings.Contains(task.TestName, skip)
	}) {
		return true, "skip list"
	}
	alpha, beta := 0, 0
	for _, c := range task.Cases {
		if c.Expected == "alpha" {
			alpha++
		} else {
			beta++
		}
	}
	if alpha == 0 || beta == 0 {
		return true, fmt.Sprintf("missing alpha or beta cases (alpha=%d beta=%d)", alpha, beta)
	}
	return false, ""
}

func generateTask(cfg Config, task TaskMetadata) error {
	outDir := filepath.Join(cfg.OutputDir, task.TaskID)

	// Generate manifests and collect prompt context
	artifacts, promptCtx, err := GenerateManifests(task, outDir)
	if err != nil {
		return err
	}
	// Generate prompt
	prompt, err := BuildPrompt(cfg, promptCtx)
	if err != nil {
		return err
	}

	contains, notContains := buildExpectations(artifacts)
	var expectLines []string
	for _, name := range contains {
		expectLines = append(expectLines, fmt.Sprintf(`- contains: "VIOLATING: %s"`, regexp.QuoteMeta(name)))
	}
	for _, name := range notContains {
		expectLines = append(expectLines, fmt.Sprintf(`- notContains: "VIOLATING: %s"`, regexp.QuoteMeta(name)))
	}
	if len(expectLines) == 0 {
		expectLines = append(expectLines, `- contains: "VIOLATING: resource-beta-\\d+"`)
	}

	// Write task.yaml
	taskYAML := fmt.Sprintf(`script:
- prompt: |
%s
setup: setup.sh
cleanup: cleanup.sh
expect:
%s
isolation: cluster
timeout: 5m
`, indent(prompt, "    "), strings.Join(expectLines, "\n"))
	os.WriteFile(filepath.Join(outDir, "task.yaml"), []byte(taskYAML), 0644)

	// Write suite.yaml
	writeSuite(outDir, task, artifacts)

	// Rewrite constraint
	rewriteConstraint(task.ConstraintPath, filepath.Join(outDir, "constraint.yaml"), "gk-"+task.TaskID)
	copyFile(task.TemplatePath, filepath.Join(outDir, "template.yaml"))

	// Write setup/cleanup scripts
	writeScripts(outDir, task.TaskID, artifacts)

	return nil
}

func caseKinds(artifacts TaskArtifacts) []string {
	kinds := map[string]bool{}
	for _, manifest := range artifacts.Manifests {
		if manifest.Kind != "" {
			kinds[manifest.Kind] = true
		}
	}
	return sortedKeys(kinds)
}

func isPodOnly(kinds []string) bool {
	return len(kinds) == 1 && kinds[0] == "Pod"
}

func buildExpectations(artifacts TaskArtifacts) (contains []string, notContains []string) {
	containsSet := map[string]bool{}
	notContainsSet := map[string]bool{}

	for _, manifest := range artifacts.Manifests {
		if manifest.Name == "" {
			continue
		}
		switch manifest.Expected {
		case "beta":
			containsSet[manifest.Name] = true
		case "alpha":
			notContainsSet[manifest.Name] = true
		}
	}

	return sortedKeys(containsSet), sortedKeys(notContainsSet)
}

func writeSuite(outDir string, task TaskMetadata, artifacts TaskArtifacts) {
	var cases []map[string]interface{}
	for _, c := range task.Cases {
		for _, cf := range artifacts.CaseFiles[c.Name] {
			violations := "no"
			if c.Expected == "beta" {
				violations = "yes"
			}
			entry := map[string]interface{}{
				"name":       c.Name,
				"object":     cf,
				"assertions": []map[string]interface{}{{"violations": violations}},
			}
			if inv := artifacts.InventoryFiles[c.Name]; len(inv) > 0 {
				entry["inventory"] = inv
			}
			cases = append(cases, entry)
		}
	}
	suite := map[string]interface{}{
		"kind":       "Suite",
		"apiVersion": "test.gatekeeper.sh/v1alpha1",
		"metadata":   map[string]interface{}{"name": task.TaskID},
		"tests": []map[string]interface{}{{
			"name":       task.TestName,
			"template":   "template.yaml",
			"constraint": "constraint.yaml",
			"cases":      cases,
		}},
	}
	data, _ := yaml.Marshal(suite)
	os.WriteFile(filepath.Join(outDir, "suite.yaml"), data, 0644)
}

func writeScripts(outDir, taskID string, artifacts TaskArtifacts) {
	ns := "gk-" + taskID
	var nsSetup, nsCleanup strings.Builder
	for _, n := range artifacts.Namespaces {
		if n == "default" || n == "kube-system" {
			continue
		}
		fmt.Fprintf(&nsSetup, "kubectl delete namespace %q --ignore-not-found\n", n)
		fmt.Fprintf(&nsSetup, "kubectl create namespace %q\n", n)
		fmt.Fprintf(&nsSetup, "kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=120s namespace %q\n", n)
		fmt.Fprintf(&nsCleanup, "kubectl delete namespace %q --ignore-not-found\n", n)
	}

	hasPods := false
	for _, manifest := range artifacts.Manifests {
		if manifest.Kind == "Pod" {
			hasPods = true
			break
		}
	}
	podSummary := ""
	if hasPods {
		podSummary = "kubectl get pods -n \"$TASK_NAMESPACE\" 2>/dev/null || true"
	}

	setup := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
TASK_NAMESPACE=%q
%s
ARTIFACTS_DIR="$(dirname "$0")/artifacts"
# Apply inventory
for file in "$ARTIFACTS_DIR"/inventory-*.yaml; do
  kubectl apply -f "$file"
done
# Apply alpha/beta resources
for file in "$ARTIFACTS_DIR"/alpha-*.yaml; do
  kubectl apply -f "$file"
done
for file in "$ARTIFACTS_DIR"/beta-*.yaml; do
  kubectl apply -f "$file"
done
%s
`, ns, strings.TrimSpace(nsSetup.String()), podSummary)
	os.WriteFile(filepath.Join(outDir, "setup.sh"), []byte(setup), 0755)

	cleanup := fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\n%s", nsCleanup.String())

	os.WriteFile(filepath.Join(outDir, "cleanup.sh"), []byte(cleanup), 0755)
}

// Helpers

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

type stringSliceFlag struct {
	values *[]string
}

func (f *stringSliceFlag) String() string {
	if f.values == nil {
		return ""
	}
	return strings.Join(*f.values, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	*f.values = append(*f.values, value)
	return nil
}
