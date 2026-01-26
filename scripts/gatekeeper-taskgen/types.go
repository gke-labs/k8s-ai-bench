package main

import "google.golang.org/genai"

// Config holds generator configuration
type Config struct {
	LibraryRoot  string
	OutputDir    string
	SkipList     []string
	Verbose      bool
	Verify       bool
	VerifyOnly   bool
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
	Path      string
	RelPath   string
	CaseName  string
	Expected  string
	Kind      string
	Name      string
	Namespace string
	Doc       map[string]any
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

// Resource is a strongly-typed wrapper around map[string]any for K8s resources
type Resource struct {
	Object map[string]any
}

func NewResource(obj map[string]any) *Resource {
	if obj == nil {
		obj = make(map[string]any)
	}
	return &Resource{Object: obj}
}

func (r *Resource) Kind() string {
	return getStr(r.Object, "kind")
}

func (r *Resource) SetKind(kind string) {
	r.Object["kind"] = kind
}

func (r *Resource) Name() string {
	return getStr(r.Object, "metadata", "name")
}

func (r *Resource) SetName(name string) {
	ensureMap(r.Object, "metadata")["name"] = name
}

func (r *Resource) Namespace() string {
	return getStr(r.Object, "metadata", "namespace")
}

func (r *Resource) SetNamespace(ns string) {
	ensureMap(r.Object, "metadata")["namespace"] = ns
}

func (r *Resource) Labels() map[string]any {
	meta := ensureMap(r.Object, "metadata")
	return ensureMap(meta, "labels")
}

func (r *Resource) SetLabel(key, value string) {
	r.Labels()[key] = value
}

// Spec returns the spec map, creating it if it doesn't exist
func (r *Resource) Spec() map[string]any {
	return ensureMap(r.Object, "spec")
}

// NestedMap returns a nested map, creating it if it doesn't exist
func (r *Resource) NestedMap(keys ...string) map[string]any {
	m := r.Object
	for _, k := range keys {
		m = ensureMap(m, k)
	}
	return m
}

// GetString returns a string value from a nested path
func (r *Resource) GetString(keys ...string) string {
	return getStr(r.Object, keys...)
}
