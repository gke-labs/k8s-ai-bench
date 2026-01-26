package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/genai"
	"sigs.k8s.io/yaml"
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
			firstErr = fmt.Errorf(r.Error)
		}
	}
	for _, betaPath := range betaPaths {
		r := repairManifest(cfg, taskID, betaPath, "beta (must violate)", string(constraintYAML), string(templateYAML))
		results = append(results, r)
		if r.Status == "error" && firstErr == nil {
			firstErr = fmt.Errorf(r.Error)
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

	b.WriteString("Edit ONLY the target manifest. Do not modify any other files.\n")
	b.WriteString(fmt.Sprintf("Target role: %s\n", targetRole))
	b.WriteString("Keep metadata.name, metadata.namespace, and all labels unchanged.\n")
	b.WriteString("Do not change kind, apiVersion, or container names.\n")
	b.WriteString("Prefer the smallest possible resource values (cpu: 1m, memory: 1Mi, ephemeral-storage: 1Mi) while satisfying the role.\n")
	b.WriteString("If resource values must exceed a max to violate, set them just above the limit (never exactly equal).\n")
	b.WriteString("If the constraint enforces required resources, alpha must include all required keys; beta must omit at least one required key.\n")
	b.WriteString("If the constraint enforces ratios, alpha should have limits == requests; beta should have limits > requests so the ratio exceeds the max.\n")
	b.WriteString("Do not add or remove containers unless required to satisfy the policy.\n")
	b.WriteString("Return ONLY the full updated YAML for the target manifest. Do not return a diff.\n")
	b.WriteString(fmt.Sprintf("Target path (for reference): %s\n", targetPath))
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

	cleaned := stripCodeFences(text)
	if strings.Contains(strings.ToUpper(cleaned), "NO_CHANGES") {
		normalized := strings.TrimSpace(string(targetYAML))
		if shouldNormalizeResources(constraintYAML) {
			if out, err := normalizeResourceValues(normalized); err == nil {
				normalized = out
			}
		}
		if normalized != strings.TrimSpace(string(targetYAML)) {
			if err := os.WriteFile(targetPath, []byte(normalized+"\n"), 0644); err != nil {
				return RepairResult{TaskID: taskID, Status: "error", FilePath: targetPath, Error: err.Error()}
			}
			return RepairResult{TaskID: taskID, Status: "repaired", FilePath: targetPath}
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
	if shouldNormalizeResources(constraintYAML) {
		if normalized, err := normalizeResourceValues(trimmed); err == nil {
			trimmed = normalized
		}
	}
	if err := os.WriteFile(targetPath, []byte(trimmed+"\n"), 0644); err != nil {
		return RepairResult{TaskID: taskID, Status: "error", FilePath: targetPath, Error: err.Error()}
	}
	return RepairResult{TaskID: taskID, Status: "repaired", FilePath: targetPath}
}

func shouldNormalizeResources(constraintYAML string) bool {
	kind := constraintKind(constraintYAML)
	switch kind {
	case "K8sContainerLimits", "K8sContainerRequests", "K8sContainerRatios":
		return false
	default:
		return true
	}
}

func constraintKind(raw string) string {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
		return ""
	}
	if v, ok := obj["kind"].(string); ok {
		return v
	}
	return ""
}

func normalizeResourceValues(raw string) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
		return raw, err
	}

	spec, _ := obj["spec"].(map[string]interface{})
	for _, field := range []string{"containers", "initContainers"} {
		if list, ok := spec[field].([]interface{}); ok {
			for _, item := range list {
				container, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				resources, _ := container["resources"].(map[string]interface{})
				if len(resources) == 0 {
					continue
				}
				if limits, ok := resources["limits"].(map[string]interface{}); ok {
					if _, ok := limits["cpu"]; ok {
						limits["cpu"] = "1m"
					}
					if _, ok := limits["memory"]; ok {
						limits["memory"] = "1Mi"
					}
				}
				if requests, ok := resources["requests"].(map[string]interface{}); ok {
					if _, ok := requests["cpu"]; ok {
						requests["cpu"] = "1m"
					}
					if _, ok := requests["memory"]; ok {
						requests["memory"] = "1Mi"
					}
				}
			}
		}
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		return raw, err
	}
	return strings.TrimSpace(string(out)), nil
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
