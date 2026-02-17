//go:build integration

package harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/generator"
)

// executeGenerate generates code from the parsed document.
func executeGenerate(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Generate requires a prior parse result
	if pc.ParseResult == nil {
		return fmt.Errorf("generate step requires a prior parse step")
	}

	// Build generator options from step config
	opts := []generator.Option{
		generator.WithParsed(*pc.ParseResult),
	}

	// Handle package name config
	if packageName, ok := step.Config["package"].(string); ok && packageName != "" {
		opts = append(opts, generator.WithPackageName(packageName))
	} else {
		opts = append(opts, generator.WithPackageName("generated"))
	}

	// Handle client generation config
	if client, ok := step.Config["client"].(bool); ok {
		opts = append(opts, generator.WithClient(client))
	}

	// Handle server generation config
	if server, ok := step.Config["server"].(bool); ok {
		opts = append(opts, generator.WithServer(server))
	}

	// Handle types-only generation config
	if types, ok := step.Config["types"].(bool); ok {
		opts = append(opts, generator.WithTypes(types))
	}

	// Disable README generation for tests (reduces noise)
	opts = append(opts, generator.WithReadme(false))

	// Execute the generator
	genResult, err := generator.GenerateWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("generate failed: %w", err)
	}

	// Check for critical issues
	if genResult.HasCriticalIssues() {
		return fmt.Errorf("generate produced %d critical issues", genResult.CriticalCount)
	}

	// Store the generate result
	pc.GenerateResult = genResult
	result.Output.GenerateResult = genResult

	// Create a temp directory for generated files
	outputDir, ok := step.Config["output-dir"].(string)
	if !ok || outputDir == "" {
		tempDir, err := os.MkdirTemp("", "oastools-generate-*")
		if err != nil {
			return fmt.Errorf("generate: failed to create temp directory: %w", err)
		}
		outputDir = tempDir
		pc.TempDirs = append(pc.TempDirs, tempDir)
	}

	// Write generated files to the output directory
	if err := genResult.WriteFiles(outputDir); err != nil {
		return fmt.Errorf("generate: failed to write files: %w", err)
	}

	// Store the output directory path for subsequent steps (e.g., build)
	pc.StepData["generate-output-dir"] = outputDir

	if pc.Debug {
		t.Logf("  Generated %d files to %s", len(genResult.Files), outputDir)
		for _, f := range genResult.Files {
			t.Logf("    - %s", f.Name)
		}
	}

	return nil
}

// executeBuild runs go build on generated code.
func executeBuild(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Get output directory from generate step
	outputDir, ok := pc.StepData["generate-output-dir"].(string)
	if !ok || outputDir == "" {
		return fmt.Errorf("build step requires a prior generate step")
	}

	// Check that the directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("build: output directory does not exist: %s", outputDir)
	}

	// Initialize go module in temp directory if needed
	goModPath := filepath.Join(outputDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// Get package name from generate result or config
		pkgName := "generated"
		if pc.GenerateResult != nil && pc.GenerateResult.PackageName != "" {
			pkgName = pc.GenerateResult.PackageName
		}

		modCmd := exec.Command("go", "mod", "init", "test/"+pkgName)
		modCmd.Dir = outputDir
		if output, err := modCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build: failed to init go module: %s\n%s", err, output)
		}

		// Run go mod tidy to fetch dependencies
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = outputDir
		if output, err := tidyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build: failed to tidy go module: %s\n%s", err, output)
		}
	}

	// Run go build
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()

	if err != nil {
		// Check if error was expected
		if step.Expect == "error" {
			if pc.Debug {
				t.Logf("  Build failed as expected: %s\n%s", err, output)
			}
			return nil // Expected failure
		}
		return fmt.Errorf("build failed: %s\n%s", err, output)
	}

	// Check if success was not expected
	if step.Expect == "error" {
		return fmt.Errorf("build succeeded but error was expected")
	}

	if pc.Debug {
		t.Logf("  Build succeeded in %s", outputDir)
	}

	return nil
}
