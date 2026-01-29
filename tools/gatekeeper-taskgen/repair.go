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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/genai"
)

const repairModel = "gemini-2.5-flash"

func repairTask(cfg Config, outDir, taskID string) ([]RepairResult, error) {
	if cfg.GeminiClient == nil {
		return []RepairResult{{TaskID: taskID, Status: "error", Error: "GEMINI_API_KEY not set"}}, fmt.Errorf("GEMINI_API_KEY not set")
	}

	alphaPaths, betaPaths, err := findArtifacts(outDir)
	if err != nil {
		return []RepairResult{{TaskID: taskID, Status: "error", Error: err.Error()}}, err
	}

	constraintPath := filepath.Join(outDir, "constraint.yaml")
	templatePath := filepath.Join(outDir, "template.yaml")
	constraintYAML, err := os.ReadFile(constraintPath)
	if err != nil {
		return []RepairResult{{TaskID: taskID, Status: "error", Error: err.Error()}}, err
	}
	var templateYAML []byte
	if data, err := os.ReadFile(templatePath); err == nil {
		templateYAML = data
	} else if !os.IsNotExist(err) {
		return []RepairResult{{TaskID: taskID, Status: "error", Error: err.Error()}}, err
	}

	var results []RepairResult
	var firstErr error
	for _, alphaPath := range alphaPaths {
		r := repairManifest(cfg, taskID, alphaPath, "alpha (must be compliant)", string(constraintYAML), string(templateYAML))
		results = append(results, r)
		if r.Status == "error" && firstErr == nil {
			firstErr = fmt.Errorf("%s", r.Error)
		}
	}
	for _, betaPath := range betaPaths {
		r := repairManifest(cfg, taskID, betaPath, "beta (must violate)", string(constraintYAML), string(templateYAML))
		results = append(results, r)
		if r.Status == "error" && firstErr == nil {
			firstErr = fmt.Errorf("%s", r.Error)
		}
	}

	return results, firstErr
}

func findArtifacts(outDir string) ([]string, []string, error) {
	artifactsDir := filepath.Join(outDir, "artifacts")
	alphaMatches, _ := filepath.Glob(filepath.Join(artifactsDir, "alpha-*.yaml"))
	betaMatches, _ := filepath.Glob(filepath.Join(artifactsDir, "beta-*.yaml"))
	sort.Strings(alphaMatches)
	sort.Strings(betaMatches)
	if len(alphaMatches) == 0 || len(betaMatches) == 0 {
		return nil, nil, fmt.Errorf("missing alpha or beta artifacts")
	}
	return alphaMatches, betaMatches, nil
}

func buildRepairPrompt(targetPath, targetRole, constraintYAML, templateYAML, targetYAML string) string {
	var b strings.Builder

	b.WriteString("# Context\n")
	b.WriteString("You are a Kubernetes Expert optimizing a benchmarking suite.\n")
	b.WriteString("We have a set of manifests used to test Gatekeeper constraints.\n")
	b.WriteString("Your task is to repair a specific manifest to either satisfy or violate a constraint, as requested.\n\n")

	b.WriteString("# Goal\n")
	fmt.Fprintf(&b, "Target Role: %s\n", targetRole)
	b.WriteString("1. Ensure the manifest fulfills the Target Role.\n")
	b.WriteString("2. Maintain validity: Ensure the manifest remains a valid Kubernetes object.\n\n")

	b.WriteString("# Instructions\n")
	b.WriteString("1. Edit ONLY the target manifest. Do not modify any other files.\n")
	fmt.Fprintf(&b, "Target path (for reference): %s\n", targetPath)
	b.WriteString("Keep metadata.name, metadata.namespace, and all labels unchanged.\n")
	b.WriteString("Do not change kind, apiVersion, or container names.\n")
	b.WriteString("Prefer the smallest possible resource values (cpu: 1m, memory: 1Mi, ephemeral-storage: 1Mi) while satisfying the constraint.\n")
	b.WriteString("If resource values must exceed a max to violate, set them just above the limit (never exactly equal).\n")
	b.WriteString("If the constraint enforces required resources, alpha must include all required keys; beta must omit at least one required key.\n")
	b.WriteString("If the constraint enforces ratios, alpha should have limits == requests; beta should have limits > requests so the ratio exceeds the max.\n")
	b.WriteString("Do not add or remove containers unless required to satisfy the policy.\n")
	b.WriteString("Return ONLY the full updated YAML for the target manifest. Do not return a diff.\n")
	fmt.Fprintf(&b, "Target path (for reference) %s\n", targetPath)
	b.WriteString("If the target already satisfies the role with minimal values, respond with NO_CHANGES.\n\n")

	appendYAMLSection(&b, "Constraint", constraintYAML, 2000)
	if strings.TrimSpace(templateYAML) != "" {
		appendYAMLSection(&b, "Template", templateYAML, 2000)
	}
	appendYAMLSection(&b, "Target manifest", targetYAML, 2000)

	return strings.TrimSpace(b.String())
}

func repairManifest(cfg Config, taskID, targetPath, targetRole, constraintYAML, templateYAML string) RepairResult {
	targetYAML, err := os.ReadFile(targetPath)
	if err != nil {
		return RepairResult{TaskID: taskID, Status: "error", Error: err.Error()}
	}

	prompt := buildRepairPrompt(targetPath, targetRole, constraintYAML, templateYAML, string(targetYAML))
	ctx := context.Background()
	result, err := cfg.GeminiClient.Models.GenerateContent(ctx, repairModel, genai.Text(prompt), nil)
	if err != nil {
		return RepairResult{TaskID: taskID, Status: "error", Error: fmt.Sprintf("gemini API error: %v", err)}
	}
	text, err := extractGeminiText(result)
	if err != nil {
		return RepairResult{TaskID: taskID, Status: "error", Error: err.Error()}
	}

	normalizedOriginal := strings.TrimSpace(string(targetYAML))
	if shouldNormalizeResources(constraintYAML) || shouldNormalizeReplicas(constraintYAML) {
		if out, err := normalizeYAML(normalizedOriginal, constraintYAML); err == nil {
			normalizedOriginal = out
		}
	}

	cleaned := stripCodeFences(text)
	if strings.Contains(strings.ToUpper(cleaned), "NO_CHANGES") {
		if normalizedOriginal != strings.TrimSpace(string(targetYAML)) {
			if err := os.WriteFile(targetPath, []byte(normalizedOriginal+"\n"), 0644); err != nil {
				return RepairResult{TaskID: taskID, Status: "error", FilePath: targetPath, Error: err.Error()}
			}
			return RepairResult{TaskID: taskID, Status: "no_changes", FilePath: targetPath}
		}
		if cfg.Verbose {
			fmt.Printf("Repair %s: NO_CHANGES (%s)\n", taskID, targetPath)
		}
		return RepairResult{TaskID: taskID, Status: "no_changes", FilePath: targetPath}
	}

	trimmed := strings.TrimSpace(cleaned)
	if !strings.Contains(trimmed, "apiVersion:") && !strings.Contains(trimmed, "kind:") {
		return RepairResult{TaskID: taskID, Status: "error", FilePath: targetPath, Error: "repair output missing manifest YAML"}
	}

	finalContent := trimmed
	if shouldNormalizeResources(constraintYAML) || shouldNormalizeReplicas(constraintYAML) {
		if normalized, err := normalizeYAML(trimmed, constraintYAML); err == nil {
			finalContent = normalized
		}
	}

	if finalContent == normalizedOriginal {
		if normalizedOriginal != strings.TrimSpace(string(targetYAML)) {
			if err := os.WriteFile(targetPath, []byte(normalizedOriginal+"\n"), 0644); err != nil {
				return RepairResult{TaskID: taskID, Status: "error", FilePath: targetPath, Error: err.Error()}
			}
		}
		return RepairResult{TaskID: taskID, Status: "no_changes", FilePath: targetPath}
	}

	diff := computeDiff(normalizedOriginal, finalContent)
	if err := os.WriteFile(targetPath, []byte(finalContent+"\n"), 0644); err != nil {
		return RepairResult{TaskID: taskID, Status: "error", FilePath: targetPath, Error: err.Error()}
	}
	return RepairResult{TaskID: taskID, Status: "repaired", FilePath: targetPath, Diff: diff}
}

func computeDiff(a, b string) string {
	f1, _ := os.CreateTemp("", "diff-a-")
	f2, _ := os.CreateTemp("", "diff-b-")
	defer os.Remove(f1.Name())
	defer os.Remove(f2.Name())

	f1.WriteString(a)
	f1.Close()
	f2.WriteString(b)
	f2.Close()

	cmd := exec.Command("diff", "-u", f1.Name(), f2.Name())
	out, _ := cmd.CombinedOutput()

	return string(out)
}

func appendYAMLSection(b *strings.Builder, title, raw string, limit int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	fmt.Fprintf(b, "%s:\n```yaml\n%s\n```\n\n", title, truncateString(raw, limit))
}

func truncateString(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "\n... (truncated)"
}

func stripCodeFences(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func extractGeminiText(result *genai.GenerateContentResponse) (string, error) {
	if result == nil || len(result.Candidates) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}
	content := result.Candidates[0].Content
	if content == nil || len(content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}
	text := content.Parts[0].Text
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("empty response from Gemini")
	}
	return strings.TrimSpace(text), nil
}

func writeRepairReport(outputDir string, results []RepairResult) error {
	var b strings.Builder

	// Header
	b.WriteString("# Gatekeeper Task Repair Report\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Count stats
	var repaired, noChanges, errors int
	for _, r := range results {
		switch r.Status {
		case "repaired":
			repaired++
		case "no_changes":
			noChanges++
		case "error":
			errors++
		}
	}

	// Summary table
	b.WriteString("## Summary\n\n")
	b.WriteString("| Status | Count |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Repaired | %d |\n", repaired))
	b.WriteString(fmt.Sprintf("| No Changes | %d |\n", noChanges))
	b.WriteString(fmt.Sprintf("| Errors | %d |\n", errors))
	b.WriteString("\n---\n\n")

	// Repaired tasks with diffs
	if repaired > 0 {
		b.WriteString("## Repaired Tasks\n\n")
		for _, r := range results {
			if r.Status == "repaired" {
				b.WriteString(fmt.Sprintf("### %s\n\n", r.TaskID))
				b.WriteString(fmt.Sprintf("**File:** `%s`\n\n", r.FilePath))
				b.WriteString("```diff\n")
				b.WriteString(r.Diff)
				if !strings.HasSuffix(r.Diff, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("```\n\n")
			}
		}
		b.WriteString("---\n\n")
	}

	// No changes list
	if noChanges > 0 {
		b.WriteString("## No Changes Needed\n\n")
		for _, r := range results {
			if r.Status == "no_changes" {
				b.WriteString(fmt.Sprintf("- %s\n", r.TaskID))
			}
		}
		b.WriteString("\n---\n\n")
	}

	// Errors
	if errors > 0 {
		b.WriteString("## Errors\n\n")
		for _, r := range results {
			if r.Status == "error" {
				b.WriteString(fmt.Sprintf("### %s\n\n", r.TaskID))
				b.WriteString("```\n")
				b.WriteString(r.Error)
				if !strings.HasSuffix(r.Error, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("```\n\n")
			}
		}
	}

	reportPath := filepath.Join(outputDir, "repair-report.md")
	return os.WriteFile(reportPath, []byte(b.String()), 0644)
}
