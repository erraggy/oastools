//go:build integration

package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v4"
)

// LoadScenario loads a single scenario from a YAML file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("harness: failed to read scenario file %s: %w", path, err)
	}

	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("harness: failed to parse scenario file %s: %w", path, err)
	}

	scenario.filePath = path

	// Validate the scenario
	if err := ValidateScenario(&scenario); err != nil {
		return nil, fmt.Errorf("harness: invalid scenario %s: %w", path, err)
	}

	return &scenario, nil
}

// LoadAllScenarios loads all scenarios from a directory recursively.
func LoadAllScenarios(dir string) ([]*Scenario, error) {
	var scenarios []*Scenario

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .yaml and .yml files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		scenario, err := LoadScenario(path)
		if err != nil {
			return err
		}

		scenarios = append(scenarios, scenario)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("harness: failed to load scenarios from %s: %w", dir, err)
	}

	return scenarios, nil
}

// ValidateScenario validates a scenario's structure and required fields.
func ValidateScenario(s *Scenario) error {
	if s.Name == "" {
		return fmt.Errorf("scenario must have a name")
	}

	if len(s.Pipeline) == 0 {
		return fmt.Errorf("scenario '%s' must have at least one pipeline step", s.Name)
	}

	// Validate that either base or inputs is specified (unless all steps are special)
	hasParseStep := false
	for _, step := range s.Pipeline {
		if step.Name == "parse" || step.Name == "parse-all" {
			hasParseStep = true
			break
		}
	}

	if hasParseStep {
		if s.Base == "" && len(s.Inputs) == 0 {
			return fmt.Errorf("scenario '%s' has a parse step but no base document or inputs specified", s.Name)
		}
	}

	// Validate each step
	for i, step := range s.Pipeline {
		if err := validateStep(&step, i); err != nil {
			return fmt.Errorf("scenario '%s': %w", s.Name, err)
		}
	}

	return nil
}

// validateStep validates a single pipeline step.
func validateStep(step *Step, index int) error {
	if step.Name == "" {
		return fmt.Errorf("step %d must have a name", index+1)
	}

	// Validate step name is recognized
	validSteps := map[string]bool{
		"parse":       true,
		"parse-all":   true,
		"validate":    true,
		"fix":         true,
		"fix-all":     true,
		"join":        true,
		"convert":     true,
		"convert-all": true,
		"diff":        true,
		"generate":    true,
		"build":       true,
		"overlay":     true,
	}

	if !validSteps[step.Name] {
		return fmt.Errorf("step %d: unknown step type '%s'", index+1, step.Name)
	}

	// Validate expect value if specified
	if step.Expect != "" {
		validExpects := map[string]bool{
			"valid":   true,
			"invalid": true,
			"error":   true,
			"success": true,
		}
		if !validExpects[step.Expect] {
			return fmt.Errorf("step %d (%s): invalid expect value '%s' (must be valid, invalid, error, or success)",
				index+1, step.Name, step.Expect)
		}
	}

	return nil
}

// ScenarioPath returns the relative path of the scenario file for display.
func ScenarioPath(s *Scenario, baseDir string) string {
	if s.filePath == "" {
		return s.Name
	}
	rel, err := filepath.Rel(baseDir, s.filePath)
	if err != nil {
		return s.filePath
	}
	return rel
}

// ScenarioTestName returns a test-friendly name for the scenario.
func ScenarioTestName(s *Scenario, baseDir string) string {
	// Use the relative path without extension as the test name
	path := ScenarioPath(s, baseDir)
	// Remove .yaml/.yml extension
	path = strings.TrimSuffix(path, ".yaml")
	path = strings.TrimSuffix(path, ".yml")
	// Replace path separators with /
	path = strings.ReplaceAll(path, string(filepath.Separator), "/")
	return path
}
