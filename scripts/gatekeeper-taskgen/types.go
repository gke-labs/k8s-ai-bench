package main

import "google.golang.org/genai"

// Config holds generator configuration
type Config struct {
	LibraryRoot  string
	OutputDir    string
	SkipList     []string
	Verbose      bool
	Repair       bool
	GeminiClient *genai.Client
}

// Suite represents a gatekeeper suite.yaml file
type Suite struct {
	Metadata struct{ Name string }
	Tests    []SuiteTest
}

// SuiteTest represents a test within a suite
type SuiteTest struct {
	Name       string
	Template   string
	Constraint string
	Cases      []SuiteCase
}

// SuiteCase represents a test case
type SuiteCase struct {
	Name       string
	Object     string
	Assertions []SuiteAssertion
}

// SuiteAssertion represents a violation assertion
type SuiteAssertion struct {
	Violations any
}

// TaskCase represents a processed test case for task generation
type TaskCase struct {
	Name       string
	Expected   string // "alpha" (compliant) or "beta" (violating)
	ObjectPath string
}

// TaskMetadata holds all info needed to generate a task
type TaskMetadata struct {
	TaskID         string
	SuiteName      string
	TestName       string
	TemplatePath   string
	ConstraintPath string
	Cases          []TaskCase
}

// TaskManifest represents a generated manifest file
type TaskManifest struct {
	Path          string
	RelPath       string
	CaseName      string
	Expected      string
	Kind          string
	Name          string
	Namespace     string
	Doc           map[string]interface{}
}

// TaskArtifacts holds all generated artifacts for a task
type TaskArtifacts struct {
	Manifests  []TaskManifest
	CaseFiles  map[string][]string
	Namespaces []string
}

// PromptContext holds all context needed to generate a prompt
type PromptContext struct {
	TaskID          string
	Title           string
	Description     string
	TemplateYAML    string
	ConstraintYAML  string
	AlphaExamples   []string
	BetaExamples    []string
	Namespace       string
	NamespacedKinds []string
	ClusterKinds    []string
}

// RepairResult tracks what happened during a repair
type RepairResult struct {
	TaskID   string
	Status   string // "repaired", "no_changes", "error"
	FilePath string
	Diff     string
	Error    string
}
