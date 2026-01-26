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
