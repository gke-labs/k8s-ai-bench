package main

import (
	"io/fs"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// ParseSuites finds and parses all suite.yaml files under libraryRoot
func ParseSuites(libraryRoot string) (map[string]TaskMetadata, error) {
	var suitePaths []string
	filepath.WalkDir(libraryRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "suite.yaml" {
			suitePaths = append(suitePaths, path)
		}
		return nil
	})

	taskMap := map[string]TaskMetadata{}
	testCounts := map[string]int{}

	for _, suitePath := range suitePaths {
		data, err := os.ReadFile(suitePath)
		if err != nil {
			continue
		}
		var suite Suite
		if err := yaml.Unmarshal(data, &suite); err != nil {
			continue
		}
		suiteDir := filepath.Dir(suitePath)

		for _, test := range suite.Tests {
			meta := TaskMetadata{
				SuiteName:      suite.Metadata.Name,
				TestName:       test.Name,
				TemplatePath:   filepath.Join(suiteDir, test.Template),
				ConstraintPath: filepath.Join(suiteDir, test.Constraint),
			}

			for _, c := range test.Cases {
				if c.Object == "" {
					continue
				}
				expected := classifyCase(c.Assertions)
				if expected == "" {
					continue
				}
				meta.Cases = append(meta.Cases, TaskCase{
					Name:       c.Name,
					Expected:   expected,
					ObjectPath: filepath.Join(suiteDir, c.Object),
				})
			}

			if len(meta.Cases) > 0 {
				meta.TaskID = test.Name
				testCounts[test.Name]++
				taskMap[meta.TaskID] = meta
			}
		}
	}

	// Handle duplicate test names by prefixing with suite name
	for id, meta := range taskMap {
		if testCounts[meta.TestName] > 1 {
			delete(taskMap, id)
			meta.TaskID = meta.SuiteName + "-" + meta.TestName
			taskMap[meta.TaskID] = meta
		}
	}

	return taskMap, nil
}

// classifyCase determines if a case is alpha (compliant) or beta (violating)
func classifyCase(assertions []SuiteAssertion) string {
	for _, a := range assertions {
		if hasViolations(a.Violations) {
			return "beta"
		}
	}
	if len(assertions) > 0 {
		return "alpha"
	}
	return ""
}

// hasViolations checks if a violations value indicates violations exist
func hasViolations(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "yes"
	case int:
		return val > 0
	case float64:
		return val > 0
	}
	return false
}
