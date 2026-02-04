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
	"strings"

	"github.com/gke-labs/k8s-ai-bench/pkg/cluster"
)

type Provider struct {
	HostContext       string
	HostKubeConfig    string
	IngressExternalIP string
}

func (p *Provider) UseIngress() bool {
	return p.IngressExternalIP != ""
}

func New(hostContext, hostKubeConfig, ingressExternalIP string) cluster.Provider {
	p := &Provider{
		HostContext:       hostContext,
		HostKubeConfig:    hostKubeConfig,
		IngressExternalIP: ingressExternalIP,
	}

	return p
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
	if err := p.prepareEnv(name); err != nil {
		return fmt.Errorf("failed to prepare env: %w", err)
	}

	valuesFile, err := p.createValuesFile(name)
	if err != nil {
		return fmt.Errorf("failed to create values file: %w", err)
	}
	defer os.Remove(valuesFile)

	args := []string{"create", name, "--connect=false", "--context", p.HostContext, "--values", valuesFile}

	createCmd := exec.Command("vcluster", args...)
	createCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
	fmt.Printf("Creating vcluster %q\n", name)
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	return createCmd.Run()
}

func (p *Provider) Delete(name string) error {
	args := []string{"delete", name, "--context", p.HostContext, "--delete-namespace"}

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
	if p.UseIngress() {
		serverURL := fmt.Sprintf("https://%s.%s.nip.io", name, p.IngressExternalIP)
		args = append(args, "--server", serverURL)
	}

	cmd := exec.Command("vcluster", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
	config, err := cmd.Output()

	if !p.UseIngress() {
		// Wait 60 secs for the local background proxy on docker to be running.
		exec.Command("sleep", "60").Run()
	}

	return config, err
}

func (p *Provider) createValuesFile(name string) (string, error) {
	valuesContent := `sync:
  toHost:
    persistentVolumeClaims:
      enabled: true
    persistentVolumes:
      enabled: true
    storageClasses:
      enabled: true
`
	if p.UseIngress() {
		ingressHost := fmt.Sprintf("%s.%s.nip.io", name, p.IngressExternalIP)
		valuesContent += fmt.Sprintf(`controlPlane:
  proxy:
    extraSANs:
    - %s
`, ingressHost)
	}

	tmpFile, err := os.CreateTemp("", "vcluster-values-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp values file: %w", err)
	}

	if _, err := tmpFile.Write([]byte(valuesContent)); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write to temp values file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp values file: %w", err)
	}

	return tmpFile.Name(), nil
}

func (p *Provider) prepareEnv(name string) error {
	namespace := fmt.Sprintf("vcluster-%s", name)
	// Create namespace if it doesn't exist
	// kubectl create namespace <ns> --dry-run=client -o yaml | kubectl apply -f -
	// simpler: just run create and ignore "already exists" error, or check first.
	// explicit check is better or "create ns x" and check err.
	// "kubectl create ns x" fails if exists.

	// Better approach: apply a namespace manifest.
	nsManifest := fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, namespace)

	if err := p.applyManifest(nsManifest); err != nil {
		return fmt.Errorf("failed to ensure namespace %s: %w", namespace, err)
	}

	if p.UseIngress() {
		ingressManifest := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: HTTPS
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  name: %s
  namespace: %s
spec:
  ingressClassName: nginx
  rules:
  - host: %s.%s.nip.io
    http:
      paths:
      - backend:
          service:
            name: %s
            port:
              number: 443
        path: /
        pathType: ImplementationSpecific
`, name, namespace, name, p.IngressExternalIP, name)

		if err := p.applyManifest(ingressManifest); err != nil {
			return fmt.Errorf("failed to apply ingress: %w", err)
		}
	}
	return nil
}

func (p *Provider) applyManifest(manifest string) error {
	args := []string{"apply", "-f", "-"}
	if p.HostContext != "" {
		args = append(args, "--context", p.HostContext)
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", p.HostKubeConfig))
	cmd.Stdin = strings.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply failed: %s: %w", string(out), err)
	}
	return nil
}
