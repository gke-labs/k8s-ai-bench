// Copyright 2025 Google LLC
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

package vcluster

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gke-labs/k8s-ai-bench/pkg/cluster"
)

type Provider struct {
	HostContext    string
	HostKubeConfig string
	ValuesPath     string
}

func New(hostContext, hostKubeConfig string) (cluster.Provider, func(), error) {
	// Create a temporary file for vcluster values
	valuesContent := `sync:
  toHost:
    persistentVolumeClaims:
      enabled: true
    persistentVolumes:
      enabled: true
    storageClasses:
      enabled: true
`
	tmpFile, err := os.CreateTemp("", "vcluster-values-*.yaml")
	fmt.Printf("create a temp vcluster values file: %s\n", tmpFile.Name())
	if err != nil {
		fmt.Printf("failed to create temp values file: %v\n", err)
		return nil, func() {}, err
	}

	if _, err := tmpFile.Write([]byte(valuesContent)); err != nil {
		fmt.Printf("failed to write to temp values file: %v\n", err)
		os.Remove(tmpFile.Name())
		return nil, func() {}, err
	}
	if err := tmpFile.Close(); err != nil {
		fmt.Printf("failed to close temp values file: %v\n", err)
		os.Remove(tmpFile.Name())
		return nil, func() {}, err
	}

	p := &Provider{
		HostContext:    hostContext,
		HostKubeConfig: hostKubeConfig,
		ValuesPath:     tmpFile.Name(),
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return p, cleanup, nil
}

func (p *Provider) Exists(name string) (bool, error) {
	args := []string{"list", "--output", "json"}
	if p.HostContext != "" {
		args = append(args, "--context", p.HostContext)
	}

	cmd := exec.Command("vcluster", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to run 'vcluster list': %w", err)
	}

	var clusters []struct {
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(output, &clusters); err != nil {
		// Fallback to text parsing if JSON fails (older vcluster versions might behave differently)
		return false, fmt.Errorf("failed to parse vcluster list json: %w", err)
	}

	for _, c := range clusters {
		if c.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (p *Provider) Create(name string) error {
	var createErr error
	for retry := range 3 {
		if retry > 0 {
			fmt.Printf("Retrying vcluster creation, attempt %d\n", retry+1)
			time.Sleep(5 * time.Second)
		}

		args := []string{"create", name, "--connect=false", "--values", p.ValuesPath}
		if p.HostContext != "" {
			args = append(args, "--context", p.HostContext)
		}

		createCmd := exec.Command("vcluster", args...)
		createCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
		fmt.Printf("Creating vcluster %q\n", name)
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr
		createErr = createCmd.Run()
		if createErr == nil {
			return nil
		}
		fmt.Printf("failed to create vcluster, retrying...: %v\n", createErr)
	}
	return fmt.Errorf("failed to create vcluster after multiple retries: %w", createErr)
}

func (p *Provider) Delete(name string) error {
	args := []string{"delete", name}
	if p.HostContext != "" {
		args = append(args, "--context", p.HostContext)
	}

	deleteCmd := exec.Command("vcluster", args...)
	deleteCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
	fmt.Printf("Deleting vcluster %q\n", name)
	deleteCmd.Stdout = os.Stdout
	deleteCmd.Stderr = os.Stderr
	return deleteCmd.Run()
}

func (p *Provider) GetKubeconfig(name string) ([]byte, error) {
	// vcluster connect <name> --print
	args := []string{"connect", name, "--print"}
	if p.HostContext != "" {
		args = append(args, "--context", p.HostContext)
	}

	cmd := exec.Command("vcluster", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
	config, err := cmd.Output()

	// Wait 60 secs for the local background proxy on docker to be running.
	exec.Command("sleep", "60").Run()

	return config, err
}
