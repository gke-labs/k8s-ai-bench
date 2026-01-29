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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// verifyTasks runs `gator verify` on all generated tasks that have a suite.yaml
func verifyTasks(outDir string) error {
	fmt.Println("\nVerifying tasks with gator...")
	fmt.Println("=============================")

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("reading output directory: %w", err)
	}

	var verified, failed, skipped int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		taskDir := filepath.Join(outDir, taskID)
		suitePath := filepath.Join(taskDir, "suite.yaml")

		// Check if suite.yaml exists
		if _, err := os.Stat(suitePath); os.IsNotExist(err) {
			skipped++
			continue
		}

		fmt.Printf("Verifying %s... ", taskID)

		// Run gator verify
		cmd := exec.Command("gator", "verify", suitePath)
		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Println("FAILED")
			fmt.Printf("  Error: %v\n", err)
			fmt.Println("  Output:")
			fmt.Println(indent(string(output), "    "))
			failed++
		} else {
			fmt.Println("OK")
			verified++
		}
	}

	fmt.Printf("\nVerification Complete: %d passed, %d failed, %d skipped\n", verified, failed, skipped)
	if failed > 0 {
		return fmt.Errorf("%d tasks failed verification", failed)
	}

	return nil
}
