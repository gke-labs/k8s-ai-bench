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

import "google.golang.org/genai"

// Config holds generator configuration
type Config struct {
	LibraryRoot  string
	OutputDir    string
	SkipList     []string
	Verbose      bool
	Verify       bool
	VerifyOnly   bool
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
	Inventory  []string
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
	Inventory  []string
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
	Manifests      []TaskManifest
	CaseFiles      map[string][]string
	InventoryFiles map[string][]string
	Namespaces     []string
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

func (r *Resource) IsClusterScoped() bool {
	return isClusterScopedKind(r.Kind())
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
