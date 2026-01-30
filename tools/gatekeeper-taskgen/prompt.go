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
	"strings"

	"google.golang.org/genai"
)

// BuildPrompt generates a task prompt using Gemini.
func BuildPrompt(cfg Config, ctx PromptContext) (string, error) {
	if cfg.GeminiClient == nil {
		return "", fmt.Errorf("Gemini client not configured; set GEMINI_API_KEY")
	}
	return generatePromptWithGemini(cfg.GeminiClient, ctx)
}

func generatePromptWithGemini(client *genai.Client, ctx PromptContext) (string, error) {
	geminiCtx := context.Background()

	var metaPrompt strings.Builder
	metaPrompt.WriteString(`You are helping generate a task prompt for a Kubernetes policy compliance benchmark.

The benchmark tests whether an AI can identify policy-violating Kubernetes resources by inspecting a live cluster.
The AI being tested will use kubectl to examine deployed resources and determine which ones violate the policy.

Generate a clear, concise task prompt that:
1. Sounds like a real human request (first-person or direct ask), not a role-play instruction
2. Explains what policy is being enforced in natural language, keep it concise
3. Asks the AI to identify the violating resource names
4. Mentions which resource kinds to check (keep it brief and natural)

IMPORTANT: Do NOT mention resource naming conventions, alpha/beta patterns, or any hints about which resources are violating.
The AI must determine violations by examining the actual resource configurations, not by name patterns.

Required Output Format:
For each violating resource, the AI must print exactly one line:
VIOLATING: <resource name>

If there are namespaced resource kinds, the prompt MUST specify that the resources to be checked are in the provided Namespace.
If there are cluster-scoped kinds, the prompt MUST say to check those cluster-wide and must not imply everything is inside the namespace.
Do NOT use 'default' namespace unless the provided Namespace is 'default'.
Replace 'default' in the description with the provided Namespace if necessary.



Policy Information:
`)

	if ctx.Title != "" {
		fmt.Fprintf(&metaPrompt, "Title: %s\n", ctx.Title)
	}
	if ctx.Description != "" {
		fmt.Fprintf(&metaPrompt, "Description: %s\n", ctx.Description)
	}
	if ctx.Namespace != "" {
		fmt.Fprintf(&metaPrompt, "Namespace: %s\n", ctx.Namespace)
	}
	if len(ctx.NamespacedKinds) > 0 {
		fmt.Fprintf(&metaPrompt, "NamespacedKinds: %s\n", strings.Join(ctx.NamespacedKinds, ", "))
	}
	if len(ctx.ClusterKinds) > 0 {
		fmt.Fprintf(&metaPrompt, "ClusterKinds: %s\n", strings.Join(ctx.ClusterKinds, ", "))
	}

	if ctx.ConstraintYAML != "" {
		constraint := ctx.ConstraintYAML
		if len(constraint) > 2000 {
			constraint = constraint[:2000] + "\n... (truncated)"
		}
		fmt.Fprintf(&metaPrompt, "\nConstraint Definition:\n```yaml\n%s\n```\n", constraint)
	}

	if len(ctx.AlphaExamples) > 0 {
		example := ctx.AlphaExamples[0]
		if len(example) > 1500 {
			example = example[:1500] + "\n... (truncated)"
		}
		fmt.Fprintf(&metaPrompt, "\nExample COMPLIANT resource:\n```yaml\n%s\n```\n", example)
	}
	if len(ctx.BetaExamples) > 0 {
		example := ctx.BetaExamples[0]
		if len(example) > 1500 {
			example = example[:1500] + "\n... (truncated)"
		}
		fmt.Fprintf(&metaPrompt, "\nExample VIOLATING resource:\n```yaml\n%s\n```\n", example)
	}

	metaPrompt.WriteString(`
Generate only the task prompt text, nothing else. Do not include markdown formatting.
Do not mention anything about resource naming patterns or conventions.
	The prompt should end with strict instructions to use the "VIOLATING: <resource name>" format for every violation found.`)

	result, err := client.Models.GenerateContent(geminiCtx, "gemini-2.0-flash", genai.Text(metaPrompt.String()), nil)
	if err != nil {
		return "", fmt.Errorf("gemini API error: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	text := result.Candidates[0].Content.Parts[0].Text
	if text == "" {
		return "", fmt.Errorf("empty text in Gemini response")
	}

	return strings.TrimSpace(text), nil
}
