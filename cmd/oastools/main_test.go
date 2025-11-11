package main

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/joiner"
)

func TestParseJoinFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectHelp     bool
		errorContains  string
		validateConfig func(*testing.T, joiner.JoinerConfig)
		validateFiles  func(*testing.T, []string)
		validateOutput func(*testing.T, string)
	}{
		{
			name: "valid basic flags",
			args: []string{"-o", "output.yaml", "file1.yaml", "file2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.MergeArrays != true {
					t.Error("expected MergeArrays to be true by default")
				}
				if config.DeduplicateTags != true {
					t.Error("expected DeduplicateTags to be true by default")
				}
			},
			validateFiles: func(t *testing.T, files []string) {
				if len(files) != 2 {
					t.Errorf("expected 2 files, got %d", len(files))
				}
				if files[0] != "file1.yaml" || files[1] != "file2.yaml" {
					t.Errorf("unexpected file paths: %v", files)
				}
			},
			validateOutput: func(t *testing.T, output string) {
				if output != "output.yaml" {
					t.Errorf("expected output 'output.yaml', got '%s'", output)
				}
			},
		},
		{
			name: "valid with long output flag",
			args: []string{"--output", "result.yaml", "spec1.yaml", "spec2.yaml"},
			validateFiles: func(t *testing.T, files []string) {
				if len(files) != 2 {
					t.Errorf("expected 2 files, got %d", len(files))
				}
			},
			validateOutput: func(t *testing.T, output string) {
				if output != "result.yaml" {
					t.Errorf("expected output 'result.yaml', got '%s'", output)
				}
			},
		},
		{
			name: "valid with path strategy",
			args: []string{"-o", "out.yaml", "--path-strategy", "accept-left", "f1.yaml", "f2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.PathStrategy != joiner.StrategyAcceptLeft {
					t.Errorf("expected PathStrategy to be 'accept-left', got '%s'", config.PathStrategy)
				}
			},
		},
		{
			name: "valid with schema strategy",
			args: []string{"-o", "out.yaml", "--schema-strategy", "accept-right", "f1.yaml", "f2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.SchemaStrategy != joiner.StrategyAcceptRight {
					t.Errorf("expected SchemaStrategy to be 'accept-right', got '%s'", config.SchemaStrategy)
				}
			},
		},
		{
			name: "valid with component strategy",
			args: []string{"-o", "out.yaml", "--component-strategy", "fail", "f1.yaml", "f2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.ComponentStrategy != joiner.StrategyFailOnCollision {
					t.Errorf("expected ComponentStrategy to be 'fail', got '%s'", config.ComponentStrategy)
				}
			},
		},
		{
			name: "valid with all strategies",
			args: []string{
				"-o", "out.yaml",
				"--path-strategy", "fail-on-paths",
				"--schema-strategy", "accept-left",
				"--component-strategy", "accept-right",
				"f1.yaml", "f2.yaml",
			},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.PathStrategy != joiner.StrategyFailOnPaths {
					t.Errorf("expected PathStrategy to be 'fail-on-paths', got '%s'", config.PathStrategy)
				}
				if config.SchemaStrategy != joiner.StrategyAcceptLeft {
					t.Errorf("expected SchemaStrategy to be 'accept-left', got '%s'", config.SchemaStrategy)
				}
				if config.ComponentStrategy != joiner.StrategyAcceptRight {
					t.Errorf("expected ComponentStrategy to be 'accept-right', got '%s'", config.ComponentStrategy)
				}
			},
		},
		{
			name: "valid with no-merge-arrays flag",
			args: []string{"-o", "out.yaml", "--no-merge-arrays", "f1.yaml", "f2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.MergeArrays != false {
					t.Error("expected MergeArrays to be false with --no-merge-arrays")
				}
			},
		},
		{
			name: "valid with no-dedup-tags flag",
			args: []string{"-o", "out.yaml", "--no-dedup-tags", "f1.yaml", "f2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.DeduplicateTags != false {
					t.Error("expected DeduplicateTags to be false with --no-dedup-tags")
				}
			},
		},
		{
			name: "valid with all boolean flags",
			args: []string{"-o", "out.yaml", "--no-merge-arrays", "--no-dedup-tags", "f1.yaml", "f2.yaml"},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.MergeArrays != false {
					t.Error("expected MergeArrays to be false")
				}
				if config.DeduplicateTags != false {
					t.Error("expected DeduplicateTags to be false")
				}
			},
		},
		{
			name: "valid with multiple input files",
			args: []string{"-o", "out.yaml", "f1.yaml", "f2.yaml", "f3.yaml", "f4.yaml"},
			validateFiles: func(t *testing.T, files []string) {
				if len(files) != 4 {
					t.Errorf("expected 4 files, got %d", len(files))
				}
			},
		},
		{
			name:        "help flag short",
			args:        []string{"-h"},
			expectHelp:  true,
			expectError: false,
		},
		{
			name:        "help flag long",
			args:        []string{"--help"},
			expectHelp:  true,
			expectError: false,
		},
		{
			name:          "error: missing output flag",
			args:          []string{"f1.yaml", "f2.yaml"},
			expectError:   true,
			errorContains: "output file is required",
		},
		{
			name:          "error: insufficient input files (none)",
			args:          []string{"-o", "out.yaml"},
			expectError:   true,
			errorContains: "at least 2 input files",
		},
		{
			name:          "error: insufficient input files (only one)",
			args:          []string{"-o", "out.yaml", "f1.yaml"},
			expectError:   true,
			errorContains: "at least 2 input files",
		},
		{
			name:          "error: output flag without argument",
			args:          []string{"-o"},
			expectError:   true,
			errorContains: "requires an argument",
		},
		{
			name:          "error: path-strategy flag without argument",
			args:          []string{"-o", "out.yaml", "--path-strategy"},
			expectError:   true,
			errorContains: "requires an argument",
		},
		{
			name:          "error: schema-strategy flag without argument",
			args:          []string{"-o", "out.yaml", "--schema-strategy"},
			expectError:   true,
			errorContains: "requires an argument",
		},
		{
			name:          "error: component-strategy flag without argument",
			args:          []string{"-o", "out.yaml", "--component-strategy"},
			expectError:   true,
			errorContains: "requires an argument",
		},
		{
			name:          "error: invalid path strategy",
			args:          []string{"-o", "out.yaml", "--path-strategy", "invalid-strategy", "f1.yaml", "f2.yaml"},
			expectError:   true,
			errorContains: "invalid path-strategy 'invalid-strategy'",
		},
		{
			name:          "error: invalid schema strategy",
			args:          []string{"-o", "out.yaml", "--schema-strategy", "bad-value", "f1.yaml", "f2.yaml"},
			expectError:   true,
			errorContains: "invalid schema-strategy 'bad-value'",
		},
		{
			name:          "error: invalid component strategy",
			args:          []string{"-o", "out.yaml", "--component-strategy", "unknown", "f1.yaml", "f2.yaml"},
			expectError:   true,
			errorContains: "invalid component-strategy 'unknown'",
		},
		{
			name: "complex valid case with mixed flag positions",
			args: []string{
				"--path-strategy", "accept-left",
				"-o", "merged.yaml",
				"base.yaml",
				"--schema-strategy", "accept-right",
				"extension.yaml",
				"--no-merge-arrays",
				"addon.yaml",
			},
			validateConfig: func(t *testing.T, config joiner.JoinerConfig) {
				if config.PathStrategy != joiner.StrategyAcceptLeft {
					t.Errorf("expected PathStrategy 'accept-left', got '%s'", config.PathStrategy)
				}
				if config.SchemaStrategy != joiner.StrategyAcceptRight {
					t.Errorf("expected SchemaStrategy 'accept-right', got '%s'", config.SchemaStrategy)
				}
				if config.MergeArrays != false {
					t.Error("expected MergeArrays to be false")
				}
			},
			validateFiles: func(t *testing.T, files []string) {
				expected := []string{"base.yaml", "extension.yaml", "addon.yaml"}
				if len(files) != len(expected) {
					t.Errorf("expected %d files, got %d", len(expected), len(files))
					return
				}
				for i, exp := range expected {
					if files[i] != exp {
						t.Errorf("file %d: expected '%s', got '%s'", i, exp, files[i])
					}
				}
			},
			validateOutput: func(t *testing.T, output string) {
				if output != "merged.yaml" {
					t.Errorf("expected output 'merged.yaml', got '%s'", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runParseJoinFlagsTest(t, tt)
		})
	}
}

// runParseJoinFlagsTest executes a single test case for parseJoinFlags
func runParseJoinFlagsTest(t *testing.T, tt struct {
	name           string
	args           []string
	expectError    bool
	expectHelp     bool
	errorContains  string
	validateConfig func(*testing.T, joiner.JoinerConfig)
	validateFiles  func(*testing.T, []string)
	validateOutput func(*testing.T, string)
}) {
	config, files, output, showHelp, err := parseJoinFlags(tt.args)

	// Check help flag
	if showHelp != tt.expectHelp {
		t.Errorf("expected showHelp=%v, got %v", tt.expectHelp, showHelp)
	}

	// Check error expectations
	if tt.expectError {
		if err == nil {
			t.Fatalf("expected error containing '%s', got nil", tt.errorContains)
		}
		if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
			t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
		}
		return
	}

	// For non-error cases
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Don't validate further if help was requested
	if showHelp {
		return
	}

	// Run validation functions
	if tt.validateConfig != nil {
		tt.validateConfig(t, config)
	}
	if tt.validateFiles != nil {
		tt.validateFiles(t, files)
	}
	if tt.validateOutput != nil {
		tt.validateOutput(t, output)
	}
}
