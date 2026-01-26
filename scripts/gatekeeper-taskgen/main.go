package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/genai"
	"sigs.k8s.io/yaml"
)

var defaultSkipList = []string{}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.LibraryRoot, "library-root", ".gatekeeper-library/library/general", "Path to gatekeeper-library general directory")
	flag.StringVar(&cfg.OutputDir, "output-dir", "tasks/gatekeeper", "Directory to write tasks")
	flag.Var(&stringSliceFlag{&cfg.SkipList}, "skip", "Patterns to skip (can be repeated)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&cfg.Verify, "verify", false, "Run gator verify on generated tasks")
	flag.BoolVar(&cfg.VerifyOnly, "verify-only", false, "Run gator verify on existing tasks without generation")
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

	return nil
}

func shouldSkip(cfg Config, task TaskMetadata) (bool, string) {
	for _, skip := range cfg.SkipList {
		if skip == task.TestName || skip == task.SuiteName || strings.Contains(task.TestName, skip) {
			return true, "skip list"
		}
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
	caseKinds := caseKinds(artifacts)
	if !isPodOnly(caseKinds) {
		_ = os.RemoveAll(outDir)
		return fmt.Errorf("non-pod task (kinds: %s)", strings.Join(caseKinds, ","))
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
			cases = append(cases, map[string]interface{}{
				"name":       c.Name,
				"object":     cf,
				"assertions": []map[string]interface{}{{"violations": violations}},
			})
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

	setup := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
TASK_NAMESPACE=%q
%s
ARTIFACTS_DIR="$(dirname "$0")/artifacts"
# Apply alpha/beta pod resources
for file in "$ARTIFACTS_DIR"/alpha-*.yaml; do
  kubectl apply -f "$file"
done
for file in "$ARTIFACTS_DIR"/beta-*.yaml; do
  kubectl apply -f "$file"
done
kubectl get pods -n "$TASK_NAMESPACE" 2>/dev/null || true
`, ns, strings.TrimSpace(nsSetup.String()))
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
