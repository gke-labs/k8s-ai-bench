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

package kind

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gke-labs/k8s-ai-bench/pkg/cluster"
)

type Provider struct{}

func New() cluster.Provider {
	return &Provider{}
}

func (p *Provider) Exists(name string) (bool, error) {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to run 'kind get clusters': %w", err)
	}
	clusters := strings.Split(string(output), "\n")
	for _, cluster := range clusters {
		if cluster == name {
			return true, nil
		}
	}
	return false, nil
}

func (p *Provider) Create(name string) error {
	var createErr error
	for retry := range 3 {
		if retry > 0 {
			fmt.Printf("Retrying cluster creation, attempt %d\n", retry+1)
			time.Sleep(5 * time.Second)
		}
		createCmd := exec.Command("kind", "create", "cluster", "--name", name, "--wait", "5m")
		fmt.Printf("Creating kind cluster %q\n", name)
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr
		createErr = createCmd.Run()
		if createErr == nil {
			return nil
		}
		fmt.Printf("failed to create kind cluster, retrying...: %v\n", createErr)
	}
	return fmt.Errorf("failed to create kind cluster after multiple retries: %w", createErr)
}

func (p *Provider) Delete(name string) error {
	deleteCmd := exec.Command("kind", "delete", "cluster", "--name", name)
	fmt.Printf("Deleting kind cluster %q\n", name)
	deleteCmd.Stdout = os.Stdout
	deleteCmd.Stderr = os.Stderr
	return deleteCmd.Run()
}

func (p *Provider) GetKubeconfig(name string) ([]byte, error) {
	return exec.Command("kind", "get", "kubeconfig", "--name", name).Output()
}
