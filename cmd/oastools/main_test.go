package main

import (
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
)

// TestJoinFlagsBasic tests basic join flag parsing scenarios
func TestJoinFlagsBasic(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
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
			name: "valid with multiple input files",
			args: []string{"-o", "out.yaml", "f1.yaml", "f2.yaml", "f3.yaml", "f4.yaml"},
			validateFiles: func(t *testing.T, files []string) {
				if len(files) != 4 {
					t.Errorf("expected 4 files, got %d", len(files))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupJoinFlags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			config := joiner.DefaultConfig()
			config.MergeArrays = !flags.noMergeArrays
			config.DeduplicateTags = !flags.noDedupTags

			if flags.pathStrategy != "" {
				config.PathStrategy = joiner.CollisionStrategy(flags.pathStrategy)
			}
			if flags.schemaStrategy != "" {
				config.SchemaStrategy = joiner.CollisionStrategy(flags.schemaStrategy)
			}
			if flags.componentStrategy != "" {
				config.ComponentStrategy = joiner.CollisionStrategy(flags.componentStrategy)
			}

			filePaths := fs.Args()

			if tt.validateConfig != nil {
				tt.validateConfig(t, config)
			}
			if tt.validateFiles != nil {
				tt.validateFiles(t, filePaths)
			}
			if tt.validateOutput != nil {
				tt.validateOutput(t, flags.output)
			}
		})
	}
}

// TestJoinFlagsStrategies tests collision strategy parsing
func TestJoinFlagsStrategies(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		validateConfig func(*testing.T, joiner.JoinerConfig)
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupJoinFlags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			config := joiner.DefaultConfig()
			if flags.pathStrategy != "" {
				config.PathStrategy = joiner.CollisionStrategy(flags.pathStrategy)
			}
			if flags.schemaStrategy != "" {
				config.SchemaStrategy = joiner.CollisionStrategy(flags.schemaStrategy)
			}
			if flags.componentStrategy != "" {
				config.ComponentStrategy = joiner.CollisionStrategy(flags.componentStrategy)
			}

			if tt.validateConfig != nil {
				tt.validateConfig(t, config)
			}
		})
	}
}

// TestJoinFlagsBooleans tests boolean flags
func TestJoinFlagsBooleans(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		validateConfig func(*testing.T, joiner.JoinerConfig)
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupJoinFlags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			config := joiner.DefaultConfig()
			config.MergeArrays = !flags.noMergeArrays
			config.DeduplicateTags = !flags.noDedupTags

			if tt.validateConfig != nil {
				tt.validateConfig(t, config)
			}
		})
	}
}

// TestJoinFlagsErrors tests error cases for join flag parsing
func TestJoinFlagsErrors(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			name:          "error: insufficient input files (none)",
			args:          []string{"-o", "out.yaml"},
			errorContains: "at least 2 input files",
		},
		{
			name:          "error: insufficient input files (only one)",
			args:          []string{"-o", "out.yaml", "f1.yaml"},
			errorContains: "at least 2 input files",
		},
		{
			name:          "error: invalid path strategy",
			args:          []string{"-o", "out.yaml", "--path-strategy", "invalid-strategy", "f1.yaml", "f2.yaml"},
			errorContains: "invalid path-strategy",
		},
		{
			name:          "error: invalid schema strategy",
			args:          []string{"-o", "out.yaml", "--schema-strategy", "bad-value", "f1.yaml", "f2.yaml"},
			errorContains: "invalid schema-strategy",
		},
		{
			name:          "error: invalid component strategy",
			args:          []string{"-o", "out.yaml", "--component-strategy", "unknown", "f1.yaml", "f2.yaml"},
			errorContains: "invalid component-strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupJoinFlags()
			if err := fs.Parse(tt.args); err != nil {
				return // Flag parse error is valid
			}

			// Check validation conditions
			if fs.NArg() < 2 && strings.Contains(tt.errorContains, "at least 2 input files") {
				return
			}
			if flags.pathStrategy != "" && !joiner.IsValidStrategy(flags.pathStrategy) {
				return
			}
			if flags.schemaStrategy != "" && !joiner.IsValidStrategy(flags.schemaStrategy) {
				return
			}
			if flags.componentStrategy != "" && !joiner.IsValidStrategy(flags.componentStrategy) {
				return
			}

			t.Fatal("expected error but got none")
		})
	}
}

// TestParseFlags tests the parse command flag parsing
func TestParseFlags(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectError        bool
		wantResolveRefs    bool
		wantValidateStruct bool
		wantQuiet          bool
	}{
		{
			name:               "no flags",
			args:               []string{"openapi.yaml"},
			wantResolveRefs:    false,
			wantValidateStruct: false,
			wantQuiet:          false,
		},
		{
			name:               "resolve refs only",
			args:               []string{"--resolve-refs", "openapi.yaml"},
			wantResolveRefs:    true,
			wantValidateStruct: false,
			wantQuiet:          false,
		},
		{
			name:               "validate structure only",
			args:               []string{"--validate-structure", "openapi.yaml"},
			wantResolveRefs:    false,
			wantValidateStruct: true,
			wantQuiet:          false,
		},
		{
			name:               "quiet short flag",
			args:               []string{"-q", "openapi.yaml"},
			wantResolveRefs:    false,
			wantValidateStruct: false,
			wantQuiet:          true,
		},
		{
			name:               "quiet long flag",
			args:               []string{"--quiet", "openapi.yaml"},
			wantResolveRefs:    false,
			wantValidateStruct: false,
			wantQuiet:          true,
		},
		{
			name:               "stdin input",
			args:               []string{"-"},
			wantResolveRefs:    false,
			wantValidateStruct: false,
			wantQuiet:          false,
		},
		{
			name:               "stdin with quiet",
			args:               []string{"-q", "-"},
			wantResolveRefs:    false,
			wantValidateStruct: false,
			wantQuiet:          true,
		},
		{
			name:               "both flags",
			args:               []string{"--resolve-refs", "--validate-structure", "openapi.yaml"},
			wantResolveRefs:    true,
			wantValidateStruct: true,
			wantQuiet:          false,
		},
		{
			name:               "all flags",
			args:               []string{"--resolve-refs", "--validate-structure", "--quiet", "openapi.yaml"},
			wantResolveRefs:    true,
			wantValidateStruct: true,
			wantQuiet:          true,
		},
		{
			name:        "no file path",
			args:        []string{"--resolve-refs"},
			expectError: true,
		},
		{
			name:        "too many arguments",
			args:        []string{"file1.yaml", "file2.yaml"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupParseFlags()

			err := fs.Parse(tt.args)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("unexpected parse error: %v", err)
				}
				return
			}

			// Check argument count
			if fs.NArg() != 1 {
				if !tt.expectError {
					t.Errorf("expected exactly 1 file argument, got %d", fs.NArg())
				}
				return
			}

			if tt.expectError {
				t.Fatalf("expected error but got none")
			}

			if flags.resolveRefs != tt.wantResolveRefs {
				t.Errorf("resolveRefs = %v, want %v", flags.resolveRefs, tt.wantResolveRefs)
			}
			if flags.validateStructure != tt.wantValidateStruct {
				t.Errorf("validateStructure = %v, want %v", flags.validateStructure, tt.wantValidateStruct)
			}
			if flags.quiet != tt.wantQuiet {
				t.Errorf("quiet = %v, want %v", flags.quiet, tt.wantQuiet)
			}
		})
	}
}

// TestValidateFlags tests the validate command flag parsing
func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		wantStrict     bool
		wantNoWarnings bool
		wantQuiet      bool
	}{
		{
			name:           "no flags",
			args:           []string{"openapi.yaml"},
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "strict only",
			args:           []string{"--strict", "openapi.yaml"},
			wantStrict:     true,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "no-warnings only",
			args:           []string{"--no-warnings", "openapi.yaml"},
			wantStrict:     false,
			wantNoWarnings: true,
			wantQuiet:      false,
		},
		{
			name:           "quiet short flag",
			args:           []string{"-q", "openapi.yaml"},
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      true,
		},
		{
			name:           "quiet long flag",
			args:           []string{"--quiet", "openapi.yaml"},
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      true,
		},
		{
			name:           "stdin input",
			args:           []string{"-"},
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "stdin with quiet",
			args:           []string{"-q", "-"},
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      true,
		},
		{
			name:           "both flags",
			args:           []string{"--strict", "--no-warnings", "openapi.yaml"},
			wantStrict:     true,
			wantNoWarnings: true,
			wantQuiet:      false,
		},
		{
			name:           "all flags",
			args:           []string{"--strict", "--no-warnings", "--quiet", "openapi.yaml"},
			wantStrict:     true,
			wantNoWarnings: true,
			wantQuiet:      true,
		},
		{
			name:        "no file path",
			args:        []string{"--strict"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupValidateFlags()

			err := fs.Parse(tt.args)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("unexpected parse error: %v", err)
				}
				return
			}

			if fs.NArg() != 1 {
				if !tt.expectError {
					t.Errorf("expected exactly 1 file argument, got %d", fs.NArg())
				}
				return
			}

			if tt.expectError {
				t.Fatalf("expected error but got none")
			}

			if flags.strict != tt.wantStrict {
				t.Errorf("strict = %v, want %v", flags.strict, tt.wantStrict)
			}
			if flags.noWarnings != tt.wantNoWarnings {
				t.Errorf("noWarnings = %v, want %v", flags.noWarnings, tt.wantNoWarnings)
			}
			if flags.quiet != tt.wantQuiet {
				t.Errorf("quiet = %v, want %v", flags.quiet, tt.wantQuiet)
			}
		})
	}
}

// TestConvertFlags tests the convert command flag parsing
func TestConvertFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		wantTarget     string
		wantOutput     string
		wantStrict     bool
		wantNoWarnings bool
		wantQuiet      bool
	}{
		{
			name:           "minimal flags",
			args:           []string{"-t", "3.0.3", "swagger.yaml"},
			wantTarget:     "3.0.3",
			wantOutput:     "",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "with output",
			args:           []string{"-t", "3.0.3", "-o", "openapi.yaml", "swagger.yaml"},
			wantTarget:     "3.0.3",
			wantOutput:     "openapi.yaml",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "with strict",
			args:           []string{"-t", "3.0.3", "--strict", "swagger.yaml"},
			wantTarget:     "3.0.3",
			wantStrict:     true,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "with no-warnings",
			args:           []string{"-t", "3.0.3", "--no-warnings", "swagger.yaml"},
			wantTarget:     "3.0.3",
			wantStrict:     false,
			wantNoWarnings: true,
			wantQuiet:      false,
		},
		{
			name:           "with quiet short flag",
			args:           []string{"-t", "3.0.3", "-q", "swagger.yaml"},
			wantTarget:     "3.0.3",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      true,
		},
		{
			name:           "with quiet long flag",
			args:           []string{"-t", "3.0.3", "--quiet", "swagger.yaml"},
			wantTarget:     "3.0.3",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      true,
		},
		{
			name:           "stdin input",
			args:           []string{"-t", "3.0.3", "-"},
			wantTarget:     "3.0.3",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:           "stdin with quiet",
			args:           []string{"-t", "3.0.3", "-q", "-"},
			wantTarget:     "3.0.3",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      true,
		},
		{
			name:           "all flags",
			args:           []string{"-t", "3.1.0", "-o", "output.yaml", "--strict", "--no-warnings", "--quiet", "input.yaml"},
			wantTarget:     "3.1.0",
			wantOutput:     "output.yaml",
			wantStrict:     true,
			wantNoWarnings: true,
			wantQuiet:      true,
		},
		{
			name:           "long form flags",
			args:           []string{"--target", "2.0", "--output", "swagger.yaml", "openapi.yaml"},
			wantTarget:     "2.0",
			wantOutput:     "swagger.yaml",
			wantStrict:     false,
			wantNoWarnings: false,
			wantQuiet:      false,
		},
		{
			name:        "no target flag",
			args:        []string{"swagger.yaml"},
			expectError: true,
		},
		{
			name:        "no file path",
			args:        []string{"-t", "3.0.3"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupConvertFlags()

			err := fs.Parse(tt.args)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("unexpected parse error: %v", err)
				}
				return
			}

			// Check for missing target
			if flags.target == "" {
				if !tt.expectError {
					t.Error("target is required but not provided")
				}
				return
			}

			// Check argument count
			if fs.NArg() != 1 {
				if !tt.expectError {
					t.Errorf("expected exactly 1 file argument, got %d", fs.NArg())
				}
				return
			}

			if tt.expectError {
				t.Fatalf("expected error but got none")
			}

			if flags.target != tt.wantTarget {
				t.Errorf("target = %v, want %v", flags.target, tt.wantTarget)
			}
			if flags.output != tt.wantOutput {
				t.Errorf("output = %v, want %v", flags.output, tt.wantOutput)
			}
			if flags.strict != tt.wantStrict {
				t.Errorf("strict = %v, want %v", flags.strict, tt.wantStrict)
			}
			if flags.noWarnings != tt.wantNoWarnings {
				t.Errorf("noWarnings = %v, want %v", flags.noWarnings, tt.wantNoWarnings)
			}
			if flags.quiet != tt.wantQuiet {
				t.Errorf("quiet = %v, want %v", flags.quiet, tt.wantQuiet)
			}
		})
	}
}

// TestDiffFlags tests the diff command flag parsing
func TestDiffFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectError  bool
		wantBreaking bool
		wantNoInfo   bool
	}{
		{
			name:         "no flags",
			args:         []string{"api-v1.yaml", "api-v2.yaml"},
			wantBreaking: false,
			wantNoInfo:   false,
		},
		{
			name:         "breaking only",
			args:         []string{"--breaking", "api-v1.yaml", "api-v2.yaml"},
			wantBreaking: true,
			wantNoInfo:   false,
		},
		{
			name:         "no-info only",
			args:         []string{"--no-info", "api-v1.yaml", "api-v2.yaml"},
			wantBreaking: false,
			wantNoInfo:   true,
		},
		{
			name:         "both flags",
			args:         []string{"--breaking", "--no-info", "api-v1.yaml", "api-v2.yaml"},
			wantBreaking: true,
			wantNoInfo:   true,
		},
		{
			name:        "missing target file",
			args:        []string{"api-v1.yaml"},
			expectError: true,
		},
		{
			name:        "no file paths",
			args:        []string{"--breaking"},
			expectError: true,
		},
		{
			name:        "too many file paths",
			args:        []string{"file1.yaml", "file2.yaml", "file3.yaml"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupDiffFlags()

			err := fs.Parse(tt.args)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("unexpected parse error: %v", err)
				}
				return
			}

			// Check argument count
			if fs.NArg() != 2 {
				if !tt.expectError {
					t.Errorf("expected exactly 2 file arguments, got %d", fs.NArg())
				}
				return
			}

			if tt.expectError {
				t.Fatalf("expected error but got none")
			}

			if flags.breaking != tt.wantBreaking {
				t.Errorf("breaking = %v, want %v", flags.breaking, tt.wantBreaking)
			}
			if flags.noInfo != tt.wantNoInfo {
				t.Errorf("noInfo = %v, want %v", flags.noInfo, tt.wantNoInfo)
			}
		})
	}
}

// TestGenerateFlags tests the generate command flag parsing
func TestGenerateFlags(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectError      bool
		wantClient       bool
		wantServer       bool
		wantTypes        bool
		wantPackage      string
		wantOutput       string
		wantNoPointers   bool
		wantNoValidation bool
		wantStrict       bool
		wantNoWarnings   bool
	}{
		{
			name:        "client only",
			args:        []string{"--client", "-o", "output", "openapi.yaml"},
			wantClient:  true,
			wantServer:  false,
			wantTypes:   true,
			wantPackage: "api",
			wantOutput:  "output",
		},
		{
			name:        "server only",
			args:        []string{"--server", "-o", "output", "openapi.yaml"},
			wantClient:  false,
			wantServer:  true,
			wantTypes:   true,
			wantPackage: "api",
			wantOutput:  "output",
		},
		{
			name:        "types only",
			args:        []string{"--types", "-o", "output", "openapi.yaml"},
			wantClient:  false,
			wantServer:  false,
			wantTypes:   true,
			wantPackage: "api",
			wantOutput:  "output",
		},
		{
			name:        "client and server",
			args:        []string{"--client", "--server", "-o", "output", "openapi.yaml"},
			wantClient:  true,
			wantServer:  true,
			wantTypes:   true,
			wantPackage: "api",
			wantOutput:  "output",
		},
		{
			name:        "custom package name",
			args:        []string{"--client", "-o", "output", "-p", "petstore", "openapi.yaml"},
			wantClient:  true,
			wantPackage: "petstore",
			wantOutput:  "output",
			wantTypes:   true,
		},
		{
			name:        "long package flag",
			args:        []string{"--client", "--output", "output", "--package", "myapi", "openapi.yaml"},
			wantClient:  true,
			wantPackage: "myapi",
			wantOutput:  "output",
			wantTypes:   true,
		},
		{
			name:             "all options",
			args:             []string{"--client", "--server", "--no-pointers", "--no-validation", "--strict", "--no-warnings", "-o", "out", "-p", "pkg", "api.yaml"},
			wantClient:       true,
			wantServer:       true,
			wantTypes:        true,
			wantPackage:      "pkg",
			wantOutput:       "out",
			wantNoPointers:   true,
			wantNoValidation: true,
			wantStrict:       true,
			wantNoWarnings:   true,
		},
		{
			name:        "missing output",
			args:        []string{"--client", "openapi.yaml"},
			expectError: true,
		},
		{
			name:        "no file path",
			args:        []string{"--client", "-o", "output"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := setupGenerateFlags()

			err := fs.Parse(tt.args)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("unexpected parse error: %v", err)
				}
				return
			}

			// Check for missing output
			if flags.output == "" {
				if !tt.expectError {
					t.Error("output is required but not provided")
				}
				return
			}

			// Check argument count
			if fs.NArg() != 1 {
				if !tt.expectError {
					t.Errorf("expected exactly 1 file argument, got %d", fs.NArg())
				}
				return
			}

			if tt.expectError {
				t.Fatalf("expected error but got none")
			}

			if flags.client != tt.wantClient {
				t.Errorf("client = %v, want %v", flags.client, tt.wantClient)
			}
			if flags.server != tt.wantServer {
				t.Errorf("server = %v, want %v", flags.server, tt.wantServer)
			}
			if flags.types != tt.wantTypes {
				t.Errorf("types = %v, want %v", flags.types, tt.wantTypes)
			}
			if flags.packageName != tt.wantPackage {
				t.Errorf("packageName = %v, want %v", flags.packageName, tt.wantPackage)
			}
			if flags.output != tt.wantOutput {
				t.Errorf("output = %v, want %v", flags.output, tt.wantOutput)
			}
			if flags.noPointers != tt.wantNoPointers {
				t.Errorf("noPointers = %v, want %v", flags.noPointers, tt.wantNoPointers)
			}
			if flags.noValidation != tt.wantNoValidation {
				t.Errorf("noValidation = %v, want %v", flags.noValidation, tt.wantNoValidation)
			}
			if flags.strict != tt.wantStrict {
				t.Errorf("strict = %v, want %v", flags.strict, tt.wantStrict)
			}
			if flags.noWarnings != tt.wantNoWarnings {
				t.Errorf("noWarnings = %v, want %v", flags.noWarnings, tt.wantNoWarnings)
			}
		})
	}
}

// TestValidateOutputFormat tests the validateOutputFormat helper function
func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		expectError bool
	}{
		{
			name:        "valid format text",
			format:      FormatText,
			expectError: false,
		},
		{
			name:        "valid format json",
			format:      FormatJSON,
			expectError: false,
		},
		{
			name:        "valid format yaml",
			format:      FormatYAML,
			expectError: false,
		},
		{
			name:        "invalid format xml",
			format:      "xml",
			expectError: true,
		},
		{
			name:        "invalid format csv",
			format:      "csv",
			expectError: true,
		},
		{
			name:        "invalid format empty string",
			format:      "",
			expectError: true,
		},
		{
			name:        "invalid format random string",
			format:      "invalid-format",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputFormat(tt.format)
			if tt.expectError && err == nil {
				t.Errorf("expected error for format '%s', but got none", tt.format)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for format '%s': %v", tt.format, err)
			}
		})
	}
}

// TestOutputStructured tests the outputStructured helper function
func TestOutputStructured(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		format      string
		expectError bool
	}{
		{
			name: "json format with simple struct",
			data: struct {
				Name  string
				Value int
			}{Name: "test", Value: 42},
			format:      FormatJSON,
			expectError: false,
		},
		{
			name: "yaml format with simple struct",
			data: struct {
				Name  string
				Value int
			}{Name: "test", Value: 42},
			format:      FormatYAML,
			expectError: false,
		},
		{
			name: "json format with map",
			data: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			format:      FormatJSON,
			expectError: false,
		},
		{
			name: "yaml format with map",
			data: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			format:      FormatYAML,
			expectError: false,
		},
		{
			name:        "json format with nil data",
			data:        nil,
			format:      FormatJSON,
			expectError: false,
		},
		{
			name:        "yaml format with nil data",
			data:        nil,
			format:      FormatYAML,
			expectError: false,
		},
		{
			name:        "invalid format text",
			data:        struct{ Name string }{Name: "test"},
			format:      FormatText,
			expectError: true,
		},
		{
			name:        "invalid format random",
			data:        struct{ Name string }{Name: "test"},
			format:      "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := outputStructured(tt.data, tt.format)
			if tt.expectError && err == nil {
				t.Errorf("expected error for format '%s', but got none", tt.format)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for format '%s': %v", tt.format, err)
			}
		})
	}
}

// TestHandleValidateWithStdin tests the validate command with stdin input
func TestHandleValidateWithStdin(t *testing.T) {
	// Create a temporary test file
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	tmpFile := "/tmp/test-validate-stdin.yaml"
	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	// Test validate with file input (baseline)
	err = handleValidate([]string{tmpFile})
	if err != nil {
		t.Errorf("handleValidate with file failed: %v", err)
	}
}

// TestHandleValidateWithQuietMode tests the validate command with quiet flag
func TestHandleValidateWithQuietMode(t *testing.T) {
	tmpFile := "/tmp/test-validate-quiet.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	// Test with quiet mode
	err = handleValidate([]string{"-q", tmpFile})
	if err != nil {
		t.Errorf("handleValidate with quiet mode failed: %v", err)
	}
}

// TestHandleValidateWithFormat tests the validate command with different output formats
func TestHandleValidateWithFormat(t *testing.T) {
	tmpFile := "/tmp/test-validate-format.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	tests := []struct {
		name   string
		format string
	}{
		{"json format", FormatJSON},
		{"yaml format", FormatYAML},
		{"text format", FormatText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleValidate([]string{"--format", tt.format, tmpFile})
			if err != nil {
				t.Errorf("handleValidate with format %s failed: %v", tt.format, err)
			}
		})
	}
}

// TestHandleConvertWithQuietMode tests the convert command with quiet flag
func TestHandleConvertWithQuietMode(t *testing.T) {
	tmpFile := "/tmp/test-convert-quiet.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	// Test with quiet mode
	err = handleConvert([]string{"-q", "-t", "3.0.3", tmpFile})
	if err != nil {
		t.Errorf("handleConvert with quiet mode failed: %v", err)
	}
}

// TestHandleConvertWithOutput tests the convert command with output file
func TestHandleConvertWithOutput(t *testing.T) {
	tmpFile := "/tmp/test-convert-input.yaml"
	outFile := "/tmp/test-convert-output.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleConvert([]string{"-t", "3.0.3", "-o", outFile, tmpFile})
	if err != nil {
		t.Errorf("handleConvert with output file failed: %v", err)
	}

	// Check that output file was created
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Errorf("output file was not created")
	}
}

// TestHandleParseWithQuietMode tests the parse command with quiet flag
func TestHandleParseWithQuietMode(t *testing.T) {
	tmpFile := "/tmp/test-parse-quiet.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{"-q", tmpFile})
	if err != nil {
		t.Errorf("handleParse with quiet mode failed: %v", err)
	}
}

// TestHandleDiffWithFormat tests the diff command with different output formats
func TestHandleDiffWithFormat(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-1.yaml"
	tmpFile2 := "/tmp/test-diff-2.yaml"
	content1 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`
	content2 := `openapi: 3.0.0
info:
  title: Test API
  version: 2.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	tests := []struct {
		name   string
		format string
	}{
		{"json format", FormatJSON},
		{"yaml format", FormatYAML},
		{"text format", FormatText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleDiff([]string{"--format", tt.format, tmpFile1, tmpFile2})
			if err != nil {
				t.Errorf("handleDiff with format %s failed: %v", tt.format, err)
			}
		})
	}
}

// TestHandleDiffWithBreaking tests the diff command with breaking change detection
func TestHandleDiffWithBreaking(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-breaking-1.yaml"
	tmpFile2 := "/tmp/test-diff-breaking-2.yaml"
	content1 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`
	content2 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--breaking", tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff with breaking mode failed: %v", err)
	}
}

// TestHandleParseWithResolveRefs tests the parse command with resolve-refs flag
func TestHandleParseWithResolveRefs(t *testing.T) {
	tmpFile := "/tmp/test-parse-refs.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{"--resolve-refs", tmpFile})
	if err != nil {
		t.Errorf("handleParse with resolve-refs failed: %v", err)
	}
}

// TestHandleParseWithValidateStructure tests the parse command with validate-structure flag
func TestHandleParseWithValidateStructure(t *testing.T) {
	tmpFile := "/tmp/test-parse-validate.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{"--validate-structure", tmpFile})
	if err != nil {
		t.Errorf("handleParse with validate-structure failed: %v", err)
	}
}

// TestHandleValidateWithStrict tests the validate command with strict mode
func TestHandleValidateWithStrict(t *testing.T) {
	tmpFile := "/tmp/test-validate-strict.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleValidate([]string{"--strict", tmpFile})
	if err != nil {
		t.Errorf("handleValidate with strict mode failed: %v", err)
	}
}

// TestHandleValidateWithNoWarnings tests the validate command with no-warnings flag
func TestHandleValidateWithNoWarnings(t *testing.T) {
	tmpFile := "/tmp/test-validate-nowarn.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleValidate([]string{"--no-warnings", tmpFile})
	if err != nil {
		t.Errorf("handleValidate with no-warnings failed: %v", err)
	}
}

// TestHandleConvertWithStrict tests the convert command with strict mode
func TestHandleConvertWithStrict(t *testing.T) {
	tmpFile := "/tmp/test-convert-strict.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleConvert([]string{"-t", "3.0.3", "--strict", tmpFile})
	if err != nil {
		t.Errorf("handleConvert with strict mode failed: %v", err)
	}
}

// TestHandleConvertWithNoWarnings tests the convert command with no-warnings flag
func TestHandleConvertWithNoWarnings(t *testing.T) {
	tmpFile := "/tmp/test-convert-nowarn.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleConvert([]string{"-t", "3.0.3", "--no-warnings", tmpFile})
	if err != nil {
		t.Errorf("handleConvert with no-warnings failed: %v", err)
	}
}

// TestHandleDiffWithNoInfo tests the diff command with no-info flag
func TestHandleDiffWithNoInfo(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-noinfo-1.yaml"
	tmpFile2 := "/tmp/test-diff-noinfo-2.yaml"
	content1 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`
	content2 := `openapi: 3.0.0
info:
  title: Test API
  version: 2.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--no-info", tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff with no-info failed: %v", err)
	}
}

// TestHandleValidateInvalidFormat tests the validate command with invalid format
func TestHandleValidateInvalidFormat(t *testing.T) {
	tmpFile := "/tmp/test-validate-invalid.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleValidate([]string{"--format", "invalid", tmpFile})
	if err == nil {
		t.Error("handleValidate with invalid format should return error")
	}
}

// TestHandleDiffInvalidFormat tests the diff command with invalid format
func TestHandleDiffInvalidFormat(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-invalid-1.yaml"
	tmpFile2 := "/tmp/test-diff-invalid-2.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--format", "xml", tmpFile1, tmpFile2})
	if err == nil {
		t.Error("handleDiff with invalid format should return error")
	}
}

// TestHandleParseWithAllFlags tests parse with multiple flags combined
func TestHandleParseWithAllFlags(t *testing.T) {
	tmpFile := "/tmp/test-parse-all.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{"-q", "--resolve-refs", "--validate-structure", tmpFile})
	if err != nil {
		t.Errorf("handleParse with all flags failed: %v", err)
	}
}

// TestHandleValidateWithAllFlags tests validate with multiple flags combined
func TestHandleValidateWithAllFlags(t *testing.T) {
	tmpFile := "/tmp/test-validate-all.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleValidate([]string{"-q", "--strict", "--no-warnings", tmpFile})
	if err != nil {
		t.Errorf("handleValidate with all flags failed: %v", err)
	}
}

// TestHandleConvertWithAllFlags tests convert with multiple flags combined
func TestHandleConvertWithAllFlags(t *testing.T) {
	tmpFile := "/tmp/test-convert-all.yaml"
	outFile := "/tmp/test-convert-all-out.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleConvert([]string{"-q", "-t", "3.0.3", "-o", outFile, "--strict", "--no-warnings", tmpFile})
	if err != nil {
		t.Errorf("handleConvert with all flags failed: %v", err)
	}
}

// TestHandleDiffWithAllFlags tests diff with multiple flags combined
func TestHandleDiffWithAllFlags(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-all-1.yaml"
	tmpFile2 := "/tmp/test-diff-all-2.yaml"
	content1 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`
	content2 := `openapi: 3.0.0
info:
  title: Test API
  version: 2.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--breaking", "--no-info", "--format", FormatJSON, tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff with all flags failed: %v", err)
	}
}

// TestSetupParseFlagsWithQuiet tests that setupParseFlags includes the quiet flag
func TestSetupParseFlagsWithQuiet(t *testing.T) {
	fs, flags := setupParseFlags()

	// Test parsing quiet flag short form
	err := fs.Parse([]string{"-q", "test.yaml"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !flags.quiet {
		t.Error("expected quiet flag to be true")
	}
}

// TestSetupParseFlagsWithQuietLong tests that setupParseFlags includes the quiet long flag
func TestSetupParseFlagsWithQuietLong(t *testing.T) {
	fs, flags := setupParseFlags()

	// Test parsing quiet flag long form
	err := fs.Parse([]string{"--quiet", "test.yaml"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !flags.quiet {
		t.Error("expected quiet flag to be true")
	}
}

// TestSetupValidateFlagsWithQuiet tests that setupValidateFlags includes the quiet flag
func TestSetupValidateFlagsWithQuiet(t *testing.T) {
	fs, flags := setupValidateFlags()

	// Test parsing quiet flag short form
	err := fs.Parse([]string{"-q", "test.yaml"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !flags.quiet {
		t.Error("expected quiet flag to be true")
	}
}

// TestSetupValidateFlagsWithFormat tests that setupValidateFlags includes the format flag
func TestSetupValidateFlagsWithFormat(t *testing.T) {
	fs, flags := setupValidateFlags()

	tests := []struct {
		name           string
		args           []string
		expectedFormat string
	}{
		{"default format", []string{"test.yaml"}, FormatText},
		{"json format", []string{"--format", "json", "test.yaml"}, FormatJSON},
		{"yaml format", []string{"--format", "yaml", "test.yaml"}, FormatYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			fs, flags = setupValidateFlags()
			err := fs.Parse(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if flags.format != tt.expectedFormat {
				t.Errorf("expected format %s, got %s", tt.expectedFormat, flags.format)
			}
		})
	}
}

// TestSetupConvertFlagsWithQuiet tests that setupConvertFlags includes the quiet flag
func TestSetupConvertFlagsWithQuiet(t *testing.T) {
	fs, flags := setupConvertFlags()

	// Test parsing quiet flag short form
	err := fs.Parse([]string{"-q", "-t", "3.0.3", "test.yaml"})
	if err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !flags.quiet {
		t.Error("expected quiet flag to be true")
	}
}

// TestSetupDiffFlagsWithFormat tests that setupDiffFlags includes the format flag
func TestSetupDiffFlagsWithFormat(t *testing.T) {
	fs, flags := setupDiffFlags()

	tests := []struct {
		name           string
		args           []string
		expectedFormat string
	}{
		{"default format", []string{"file1.yaml", "file2.yaml"}, FormatText},
		{"json format", []string{"--format", "json", "file1.yaml", "file2.yaml"}, FormatJSON},
		{"yaml format", []string{"--format", "yaml", "file1.yaml", "file2.yaml"}, FormatYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			fs, flags = setupDiffFlags()
			err := fs.Parse(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if flags.format != tt.expectedFormat {
				t.Errorf("expected format %s, got %s", tt.expectedFormat, flags.format)
			}
		})
	}
}

// TestSetupParseFlagsUsage tests that setupParseFlags usage includes pipeline info
func TestSetupParseFlagsUsage(t *testing.T) {
	fs, _ := setupParseFlags()

	// Redirect output to discard
	var buf strings.Builder
	fs.SetOutput(&buf)

	// Trigger usage
	fs.Usage()

	// Check that usage was generated (not empty)
	if buf.Len() == 0 {
		t.Error("expected usage output, got empty string")
	}
}

// TestSetupValidateFlagsUsage tests that setupValidateFlags usage can be called
func TestSetupValidateFlagsUsage(t *testing.T) {
	fs, _ := setupValidateFlags()

	var buf strings.Builder
	fs.SetOutput(&buf)
	fs.Usage()

	if buf.Len() == 0 {
		t.Error("expected usage output, got empty string")
	}
}

// TestSetupConvertFlagsUsage tests that setupConvertFlags usage includes pipeline info
func TestSetupConvertFlagsUsage(t *testing.T) {
	fs, _ := setupConvertFlags()

	var buf strings.Builder
	fs.SetOutput(&buf)
	fs.Usage()

	if buf.Len() == 0 {
		t.Error("expected usage output, got empty string")
	}
}

// TestSetupDiffFlagsUsage tests that setupDiffFlags usage includes format info
func TestSetupDiffFlagsUsage(t *testing.T) {
	fs, _ := setupDiffFlags()

	var buf strings.Builder
	fs.SetOutput(&buf)
	fs.Usage()

	if buf.Len() == 0 {
		t.Error("expected usage output, got empty string")
	}
}

// TestHandleJoinBasic tests the join command with basic inputs
func TestHandleJoinBasic(t *testing.T) {
	tmpFile1 := "/tmp/test-join-1.yaml"
	tmpFile2 := "/tmp/test-join-2.yaml"
	outFile := "/tmp/test-join-out.yaml"
	content1 := `openapi: 3.0.0
info:
  title: API 1
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
      responses:
        '200':
          description: Success`
	content2 := `openapi: 3.0.0
info:
  title: API 2
  version: 1.0.0
paths:
  /posts:
    get:
      summary: Get posts
      responses:
        '200':
          description: Success`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleJoin([]string{"-o", outFile, tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleJoin failed: %v", err)
	}

	// Check that output file was created
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

// TestHandleJoinWithStrategies tests the join command with collision strategies
func TestHandleJoinWithStrategies(t *testing.T) {
	tmpFile1 := "/tmp/test-join-strat-1.yaml"
	tmpFile2 := "/tmp/test-join-strat-2.yaml"
	outFile := "/tmp/test-join-strat-out.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleJoin([]string{
		"-o", outFile,
		"--path-strategy", "accept-left",
		"--schema-strategy", "accept-right",
		tmpFile1, tmpFile2,
	})
	if err != nil {
		t.Errorf("handleJoin with strategies failed: %v", err)
	}
}

// TestHandleJoinWithBooleanFlags tests the join command with boolean flags
func TestHandleJoinWithBooleanFlags(t *testing.T) {
	tmpFile1 := "/tmp/test-join-bool-1.yaml"
	tmpFile2 := "/tmp/test-join-bool-2.yaml"
	outFile := "/tmp/test-join-bool-out.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleJoin([]string{
		"-o", outFile,
		"--no-merge-arrays",
		"--no-dedup-tags",
		tmpFile1, tmpFile2,
	})
	if err != nil {
		t.Errorf("handleJoin with boolean flags failed: %v", err)
	}
}

// TestHandleJoinMissingOutput tests the join command with missing output flag
func TestHandleJoinMissingOutput(t *testing.T) {
	err := handleJoin([]string{"file1.yaml", "file2.yaml"})
	if err == nil {
		t.Error("handleJoin should fail with missing output flag")
	}
}

// TestHandleJoinInsufficientFiles tests the join command with insufficient input files
func TestHandleJoinInsufficientFiles(t *testing.T) {
	err := handleJoin([]string{"-o", "out.yaml", "file1.yaml"})
	if err == nil {
		t.Error("handleJoin should fail with only one input file")
	}
}

// TestHandleJoinInvalidStrategy tests the join command with invalid collision strategy
func TestHandleJoinInvalidStrategy(t *testing.T) {
	tmpFile1 := "/tmp/test-join-invalid-1.yaml"
	tmpFile2 := "/tmp/test-join-invalid-2.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleJoin([]string{
		"-o", "out.yaml",
		"--path-strategy", "invalid-strategy",
		tmpFile1, tmpFile2,
	})
	if err == nil {
		t.Error("handleJoin should fail with invalid strategy")
	}
}

// TestHandleGenerateBasic tests the generate command with basic inputs
func TestHandleGenerateBasic(t *testing.T) {
	tmpFile := "/tmp/test-generate-basic.yaml"
	outDir := "/tmp/test-generate-basic-out"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
      responses:
        '200':
          description: Success
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.RemoveAll(outDir) }()

	err = handleGenerate([]string{"--client", "-o", outDir, tmpFile})
	if err != nil {
		t.Errorf("handleGenerate failed: %v", err)
	}

	// Check that output directory was created
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		t.Error("output directory was not created")
	}
}

// TestHandleGenerateWithServer tests the generate command with server flag
func TestHandleGenerateWithServer(t *testing.T) {
	tmpFile := "/tmp/test-generate-server.yaml"
	outDir := "/tmp/test-generate-server-out"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: string`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.RemoveAll(outDir) }()

	err = handleGenerate([]string{"--server", "-o", outDir, tmpFile})
	if err != nil {
		t.Errorf("handleGenerate with server flag failed: %v", err)
	}
}

// TestHandleGenerateWithTypes tests the generate command with types flag
func TestHandleGenerateWithTypes(t *testing.T) {
	tmpFile := "/tmp/test-generate-types.yaml"
	outDir := "/tmp/test-generate-types-out"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Product:
      type: object
      properties:
        name:
          type: string`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.RemoveAll(outDir) }()

	err = handleGenerate([]string{"--types", "-o", outDir, tmpFile})
	if err != nil {
		t.Errorf("handleGenerate with types flag failed: %v", err)
	}
}

// TestHandleGenerateWithCustomPackage tests the generate command with custom package name
func TestHandleGenerateWithCustomPackage(t *testing.T) {
	tmpFile := "/tmp/test-generate-pkg.yaml"
	outDir := "/tmp/test-generate-pkg-out"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Data:
      type: object`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.RemoveAll(outDir) }()

	err = handleGenerate([]string{"--client", "-o", outDir, "-p", "myapi", tmpFile})
	if err != nil {
		t.Errorf("handleGenerate with custom package failed: %v", err)
	}
}

// TestHandleGenerateWithAllOptions tests the generate command with all options
func TestHandleGenerateWithAllOptions(t *testing.T) {
	tmpFile := "/tmp/test-generate-all.yaml"
	outDir := "/tmp/test-generate-all-out"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Entity:
      type: object`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.RemoveAll(outDir) }()

	err = handleGenerate([]string{
		"--client",
		"--server",
		"--no-pointers",
		"--no-validation",
		"--strict",
		"--no-warnings",
		"-o", outDir,
		"-p", "testpkg",
		tmpFile,
	})
	if err != nil {
		t.Errorf("handleGenerate with all options failed: %v", err)
	}
}

// TestHandleGenerateMissingOutput tests the generate command with missing output flag
func TestHandleGenerateMissingOutput(t *testing.T) {
	err := handleGenerate([]string{"--client", "openapi.yaml"})
	if err == nil {
		t.Error("handleGenerate should fail with missing output flag")
	}
}

// TestHandleGenerateNoFileArg tests the generate command with missing file argument
func TestHandleGenerateNoFileArg(t *testing.T) {
	err := handleGenerate([]string{"--client", "-o", "/tmp/out"})
	if err == nil {
		t.Error("handleGenerate should fail with missing file argument")
	}
}

// TestValidateOutputPath tests the validateOutputPath helper function
func TestValidateOutputPath(t *testing.T) {
	tests := []struct {
		name        string
		outputPath  string
		inputPaths  []string
		expectError bool
	}{
		{
			name:        "valid output path",
			outputPath:  "/tmp/output.yaml",
			inputPaths:  []string{"/tmp/input1.yaml", "/tmp/input2.yaml"},
			expectError: false,
		},
		{
			name:        "output would overwrite input",
			outputPath:  "/tmp/input.yaml",
			inputPaths:  []string{"/tmp/input.yaml", "/tmp/other.yaml"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputPath(tt.outputPath, tt.inputPaths)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestMarshalDocument tests the marshalDocument helper function
func TestMarshalDocument(t *testing.T) {
	type testDoc struct {
		Name  string
		Value int
	}

	doc := testDoc{Name: "test", Value: 42}

	tests := []struct {
		name        string
		format      parser.SourceFormat
		expectError bool
	}{
		{
			name:        "JSON format",
			format:      parser.SourceFormatJSON,
			expectError: false,
		},
		{
			name:        "YAML format",
			format:      parser.SourceFormatYAML,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := marshalDocument(doc, tt.format)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && len(data) == 0 {
				t.Error("expected non-empty output")
			}
		})
	}
}

// TestValidateCollisionStrategy tests the validateCollisionStrategy helper function
func TestValidateCollisionStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		value        string
		expectError  bool
	}{
		{
			name:         "valid strategy accept-left",
			strategyName: "path-strategy",
			value:        "accept-left",
			expectError:  false,
		},
		{
			name:         "valid strategy accept-right",
			strategyName: "schema-strategy",
			value:        "accept-right",
			expectError:  false,
		},
		{
			name:         "valid strategy fail",
			strategyName: "component-strategy",
			value:        "fail",
			expectError:  false,
		},
		{
			name:         "valid strategy fail-on-paths",
			strategyName: "path-strategy",
			value:        "fail-on-paths",
			expectError:  false,
		},
		{
			name:         "empty strategy (allowed)",
			strategyName: "path-strategy",
			value:        "",
			expectError:  false,
		},
		{
			name:         "invalid strategy",
			strategyName: "path-strategy",
			value:        "invalid-strategy",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollisionStrategy(tt.strategyName, tt.value)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestSetupJoinFlagsUsage tests that setupJoinFlags usage can be called
func TestSetupJoinFlagsUsage(t *testing.T) {
	fs, _ := setupJoinFlags()

	var buf strings.Builder
	fs.SetOutput(&buf)
	fs.Usage()

	if buf.Len() == 0 {
		t.Error("expected usage output, got empty string")
	}
}

// TestSetupGenerateFlagsUsage tests that setupGenerateFlags usage can be called
func TestSetupGenerateFlagsUsage(t *testing.T) {
	fs, _ := setupGenerateFlags()

	var buf strings.Builder
	fs.SetOutput(&buf)
	fs.Usage()

	if buf.Len() == 0 {
		t.Error("expected usage output, got empty string")
	}
}

// TestHandleParseWithOAS2Document tests parse with OAS 2.0 document
func TestHandleParseWithOAS2Document(t *testing.T) {
	tmpFile := "/tmp/test-parse-oas2.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{tmpFile})
	if err != nil {
		t.Errorf("handleParse with OAS 2.0 failed: %v", err)
	}
}

// TestHandleParseWithOAS3Summary tests parse with OAS 3.0 summary field
func TestHandleParseWithOAS3Summary(t *testing.T) {
	tmpFile := "/tmp/test-parse-oas3-summary.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  summary: A test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{tmpFile})
	if err != nil {
		t.Errorf("handleParse with summary field failed: %v", err)
	}
}

// TestHandleParseWithWebhooks tests parse with OAS 3.1+ webhooks
func TestHandleParseWithWebhooks(t *testing.T) {
	tmpFile := "/tmp/test-parse-webhooks.yaml"
	content := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
webhooks:
  newPost:
    post:
      summary: New post webhook
      responses:
        '200':
          description: Success`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleParse([]string{tmpFile})
	if err != nil {
		t.Errorf("handleParse with webhooks failed: %v", err)
	}
}

// TestHandleValidateWithJSONFormat tests validate with JSON output and success result
func TestHandleValidateWithJSONFormatSuccess(t *testing.T) {
	tmpFile := "/tmp/test-validate-json-success.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleValidate([]string{"--format", "json", tmpFile})
	if err != nil {
		t.Errorf("handleValidate with JSON format failed: %v", err)
	}
}

// TestHandleValidateWithYAMLFormat tests validate with YAML output and success result
func TestHandleValidateWithYAMLFormatSuccess(t *testing.T) {
	tmpFile := "/tmp/test-validate-yaml-success.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err = handleValidate([]string{"--format", "yaml", tmpFile})
	if err != nil {
		t.Errorf("handleValidate with YAML format failed: %v", err)
	}
}

// TestHandleDiffSimpleMode tests diff in simple mode
func TestHandleDiffSimpleMode(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-simple-1.yaml"
	tmpFile2 := "/tmp/test-diff-simple-2.yaml"
	content1 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`
	content2 := `openapi: 3.0.0
info:
  title: Test API v2
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff in simple mode failed: %v", err)
	}
}

// TestHandleDiffBreakingModeWithChanges tests diff in breaking mode with changes
func TestHandleDiffBreakingModeWithChanges(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-breaking-changes-1.yaml"
	tmpFile2 := "/tmp/test-diff-breaking-changes-2.yaml"
	content1 := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
      responses:
        '200':
          description: Success`
	content2 := `openapi: 3.0.0
info:
  title: Test API
  version: 2.0.0
paths:
  /users:
    get:
      summary: Get all users
      responses:
        '200':
          description: Success`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--breaking", tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff in breaking mode with changes failed: %v", err)
	}
}

// TestHandleDiffJSONFormat tests diff with JSON output
func TestHandleDiffJSONFormat(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-json-1.yaml"
	tmpFile2 := "/tmp/test-diff-json-2.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--format", "json", tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff with JSON format failed: %v", err)
	}
}

// TestHandleDiffYAMLFormat tests diff with YAML output
func TestHandleDiffYAMLFormat(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-yaml-1.yaml"
	tmpFile2 := "/tmp/test-diff-yaml-2.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	err = handleDiff([]string{"--format", "yaml", tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleDiff with YAML format failed: %v", err)
	}
}

// TestHandleJoinAllStrategies tests join with all strategy flags
func TestHandleJoinAllStrategies(t *testing.T) {
	tmpFile1 := "/tmp/test-join-allstrat-1.yaml"
	tmpFile2 := "/tmp/test-join-allstrat-2.yaml"
	outFile := "/tmp/test-join-allstrat-out.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleJoin([]string{
		"-o", outFile,
		"--path-strategy", "accept-left",
		"--schema-strategy", "accept-right",
		"--component-strategy", "fail-on-paths",
		tmpFile1, tmpFile2,
	})
	if err != nil {
		t.Errorf("handleJoin with all strategies failed: %v", err)
	}
}

// TestHandleParseWithWarnings tests parse with warnings
func TestHandleParseWithWarnings(t *testing.T) {
	tmpFile := "/tmp/test-parse-warnings.yaml"
	// Create a spec with duplicate operation IDs (triggers warnings in some parsers)
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: Success
  /posts:
    get:
      operationId: getUsers
      responses:
        '200':
          description: Success`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	// This test just ensures we can handle parsing with potential warnings
	err = handleParse([]string{tmpFile})
	if err != nil {
		t.Errorf("handleParse with warnings failed: %v", err)
	}
}

// TestHandleConvertWithStdinInput tests convert command with stdin
func TestHandleConvertWithStdinInput(t *testing.T) {
	// We can't easily test stdin in this context, but we can test the error path
	// when parsing fails
	err := handleConvert([]string{"-t", "3.0.3", "nonexistent-file.yaml"})
	if err == nil {
		t.Error("handleConvert should fail with nonexistent file")
	}
}

// TestHandleValidateTextModeWithWarnings tests validate in text mode with warnings
func TestHandleValidateTextModeWithWarnings(t *testing.T) {
	tmpFile := "/tmp/test-validate-text-warnings.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	// Test validate in text mode (default format) - this covers the text output path
	err = handleValidate([]string{tmpFile})
	if err != nil {
		t.Errorf("handleValidate in text mode failed: %v", err)
	}
}

// TestHandleJoinWithWarnings tests join command with warnings
func TestHandleJoinWithWarnings(t *testing.T) {
	tmpFile1 := "/tmp/test-join-warn-1.yaml"
	tmpFile2 := "/tmp/test-join-warn-2.yaml"
	outFile := "/tmp/test-join-warn-out.yaml"
	// Create specs with potential warnings (different versions)
	content1 := `openapi: 3.0.0
info:
  title: API 1
  version: 1.0.0
paths: {}`
	content2 := `openapi: 3.0.1
info:
  title: API 2
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()
	defer func() { _ = os.Remove(outFile) }()

	// This will trigger a warning about version mismatch
	err = handleJoin([]string{"-o", outFile, tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleJoin with version mismatch warning failed: %v", err)
	}
}

// TestHandleConvertLongFlags tests convert with long flag names
func TestHandleConvertLongFlags(t *testing.T) {
	tmpFile := "/tmp/test-convert-long.yaml"
	outFile := "/tmp/test-convert-long-out.yaml"
	content := `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleConvert([]string{"--target", "3.0.3", "--output", outFile, tmpFile})
	if err != nil {
		t.Errorf("handleConvert with long flags failed: %v", err)
	}
}

// TestHandleJoinLongOutputFlag tests join with long output flag
func TestHandleJoinLongOutputFlag(t *testing.T) {
	tmpFile1 := "/tmp/test-join-longflag-1.yaml"
	tmpFile2 := "/tmp/test-join-longflag-2.yaml"
	outFile := "/tmp/test-join-longflag-out.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()
	defer func() { _ = os.Remove(outFile) }()

	err = handleJoin([]string{"--output", outFile, tmpFile1, tmpFile2})
	if err != nil {
		t.Errorf("handleJoin with --output flag failed: %v", err)
	}
}

// TestHandleGenerateLongPackageFlag tests generate with long package flag
func TestHandleGenerateLongPackageFlag(t *testing.T) {
	tmpFile := "/tmp/test-generate-longpkg.yaml"
	outDir := "/tmp/test-generate-longpkg-out"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      type: object`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()
	defer func() { _ = os.RemoveAll(outDir) }()

	err = handleGenerate([]string{"--client", "--output", outDir, "--package", "testapi", tmpFile})
	if err != nil {
		t.Errorf("handleGenerate with long package flag failed: %v", err)
	}
}

// TestHandleParseWithEmptyStdin tests the parse command with empty stdin in quiet mode
func TestHandleParseWithEmptyStdin(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe with empty content
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	_ = w.Close() // Close write end immediately to simulate empty stdin

	err = handleParse([]string{"-q", "-"})
	if err == nil {
		t.Error("handleParse with empty stdin should return error")
	}
}

// TestHandleValidateWithEmptyStdin tests the validate command with empty stdin in quiet mode
func TestHandleValidateWithEmptyStdin(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe with empty content
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	_ = w.Close() // Close write end immediately to simulate empty stdin

	err = handleValidate([]string{"-q", "-"})
	if err == nil {
		t.Error("handleValidate with empty stdin should return error")
	}
}

// TestHandleConvertWithEmptyStdin tests the convert command with empty stdin in quiet mode
func TestHandleConvertWithEmptyStdin(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe with empty content
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	_ = w.Close() // Close write end immediately to simulate empty stdin

	err = handleConvert([]string{"-q", "-t", "3.0.3", "-"})
	if err == nil {
		t.Error("handleConvert with empty stdin should return error")
	}
}

// TestValidateOutputFormatOrder tests that format validation happens before expensive operations
func TestValidateOutputFormatOrder(t *testing.T) {
	// This test verifies that an invalid format flag fails fast without performing validation
	tmpFile := "/tmp/test-format-order.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	// Test that invalid format fails immediately
	err = handleValidate([]string{"--format", "invalid-format", tmpFile})
	if err == nil {
		t.Error("handleValidate should fail with invalid format")
	}

	// Verify the error message is about format, not about validation
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("Expected format validation error, got: %v", err)
	}
}

// TestDiffOutputFormatOrder tests that format validation happens before expensive operations
func TestDiffOutputFormatOrder(t *testing.T) {
	tmpFile1 := "/tmp/test-diff-format-1.yaml"
	tmpFile2 := "/tmp/test-diff-format-2.yaml"
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(tmpFile1, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile1) }()

	err = os.WriteFile(tmpFile2, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile2) }()

	// Test that invalid format fails immediately
	err = handleDiff([]string{"--format", "invalid-format", tmpFile1, tmpFile2})
	if err == nil {
		t.Error("handleDiff should fail with invalid format")
	}

	// Verify the error message is about format, not about diff
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("Expected format validation error, got: %v", err)
	}
}
