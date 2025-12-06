package main

import (
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/joiner"
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
			name:          "error: missing output flag",
			args:          []string{"f1.yaml", "f2.yaml"},
			errorContains: "output file is required",
		},
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
			if flags.output == "" && strings.Contains(tt.errorContains, "output file is required") {
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
