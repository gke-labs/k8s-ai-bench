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
	HostContext string
}

func New(hostContext string) cluster.Provider {
	return &Provider{
		HostContext: hostContext,
	}
}

func (p *Provider) Exists(name string) (bool, error) {
	args := []string{"list", "--output", "json"}
	if p.HostContext != "" {
		args = append(args, "--context", p.HostContext)
	}

	cmd := exec.Command("vcluster", args...)
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
		
		args := []string{"create", name, "--connect=false"}
		if p.HostContext != "" {
			args = append(args, "--context", p.HostContext)
		}

		createCmd := exec.Command("vcluster", args...)
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
	
	return exec.Command("vcluster", args...).Output()
}
