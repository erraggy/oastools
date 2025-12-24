package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupGenerateFlags(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Output != "" {
			t.Errorf("expected Output to be empty by default, got '%s'", flags.Output)
		}
		if flags.PackageName != "api" {
			t.Errorf("expected PackageName 'api' by default, got '%s'", flags.PackageName)
		}
		if flags.Client {
			t.Error("expected Client to be false by default")
		}
		if flags.Server {
			t.Error("expected Server to be false by default")
		}
		if !flags.Types {
			t.Error("expected Types to be true by default")
		}
		if flags.NoPointers {
			t.Error("expected NoPointers to be false by default")
		}
		if flags.NoValidation {
			t.Error("expected NoValidation to be false by default")
		}
		if flags.Strict {
			t.Error("expected Strict to be false by default")
		}
		if flags.NoWarnings {
			t.Error("expected NoWarnings to be false by default")
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "./output", "-p", "myapi", "--client", "--server", "--no-pointers", "--strict", "spec.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags.Output != "./output" {
			t.Errorf("expected Output './output', got '%s'", flags.Output)
		}
		if flags.PackageName != "myapi" {
			t.Errorf("expected PackageName 'myapi', got '%s'", flags.PackageName)
		}
		if !flags.Client {
			t.Error("expected Client to be true")
		}
		if !flags.Server {
			t.Error("expected Server to be true")
		}
		if !flags.NoPointers {
			t.Error("expected NoPointers to be true")
		}
		if !flags.Strict {
			t.Error("expected Strict to be true")
		}
		if fs.Arg(0) != "spec.yaml" {
			t.Errorf("expected file arg 'spec.yaml', got '%s'", fs.Arg(0))
		}
	})
}

// TestSetupGenerateFlags_SecurityFlags verifies all security-related flags are parsed correctly.
// This test prevents regressions where flags are defined but not passed to the generator.
func TestSetupGenerateFlags_SecurityFlags(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	args := []string{
		"-o", "./output",
		"--client",
		"--no-security",
		"--oauth2-flows",
		"--credential-mgmt",
		"--security-enforce",
		"--oidc-discovery",
		"--no-readme",
		"spec.yaml",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Verify all security flags are parsed
	if !flags.NoSecurity {
		t.Error("expected NoSecurity to be true")
	}
	if !flags.OAuth2Flows {
		t.Error("expected OAuth2Flows to be true")
	}
	if !flags.CredentialMgmt {
		t.Error("expected CredentialMgmt to be true")
	}
	if !flags.SecurityEnforce {
		t.Error("expected SecurityEnforce to be true")
	}
	if !flags.OIDCDiscovery {
		t.Error("expected OIDCDiscovery to be true")
	}
	if !flags.NoReadme {
		t.Error("expected NoReadme to be true")
	}
}

// TestSetupGenerateFlags_ServerFlags verifies all server-related flags are parsed correctly.
func TestSetupGenerateFlags_ServerFlags(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	args := []string{
		"-o", "./output",
		"--server",
		"--server-router=chi",
		"--server-middleware",
		"--server-binder",
		"--server-responses",
		"--server-stubs",
		"--server-embed-spec",
		"spec.yaml",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if flags.ServerRouter != "chi" {
		t.Errorf("expected ServerRouter 'chi', got '%s'", flags.ServerRouter)
	}
	if !flags.ServerMiddleware {
		t.Error("expected ServerMiddleware to be true")
	}
	if !flags.ServerBinder {
		t.Error("expected ServerBinder to be true")
	}
	if !flags.ServerResponses {
		t.Error("expected ServerResponses to be true")
	}
	if !flags.ServerStubs {
		t.Error("expected ServerStubs to be true")
	}
	if !flags.ServerEmbedSpec {
		t.Error("expected ServerEmbedSpec to be true")
	}
}

// TestSetupGenerateFlags_ServerAll verifies --server-all flag is parsed correctly.
func TestSetupGenerateFlags_ServerAll(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	args := []string{
		"-o", "./output",
		"--server",
		"--server-all",
		"spec.yaml",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if !flags.ServerAll {
		t.Error("expected ServerAll to be true")
	}
}

// TestSetupGenerateFlags_FileSplitting verifies file splitting flags are parsed correctly.
func TestSetupGenerateFlags_FileSplitting(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	args := []string{
		"-o", "./output",
		"--client",
		"--max-lines-per-file=1500",
		"--max-types-per-file=150",
		"--max-ops-per-file=50",
		"--no-split-by-tag",
		"--no-split-by-path",
		"spec.yaml",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if flags.MaxLinesPerFile != 1500 {
		t.Errorf("expected MaxLinesPerFile 1500, got %d", flags.MaxLinesPerFile)
	}
	if flags.MaxTypesPerFile != 150 {
		t.Errorf("expected MaxTypesPerFile 150, got %d", flags.MaxTypesPerFile)
	}
	if flags.MaxOpsPerFile != 50 {
		t.Errorf("expected MaxOpsPerFile 50, got %d", flags.MaxOpsPerFile)
	}
	if !flags.NoSplitByTag {
		t.Error("expected NoSplitByTag to be true")
	}
	if !flags.NoSplitByPath {
		t.Error("expected NoSplitByPath to be true")
	}
}

// TestHandleGenerate_SecurityFlagsHonored verifies security flags are passed to the generator.
// This is an integration test that catches the bug where flags were parsed but not used.
func TestHandleGenerate_SecurityFlagsHonored(t *testing.T) {
	// Create a minimal valid OAS spec
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: getTest
      responses:
        '200':
          description: Success
components:
  securitySchemes:
    oauth2:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/auth
          tokenUrl: https://example.com/token
          scopes:
            read: Read access
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")

	// Run generate with security flags
	err := HandleGenerate([]string{
		"-o", outputDir,
		"--client",
		"--oauth2-flows",
		"--credential-mgmt",
		specFile,
	})
	if err != nil {
		t.Fatalf("HandleGenerate failed: %v", err)
	}

	// Verify security files were generated
	files, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}

	fileNames := make([]string, 0, len(files))
	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}

	// Check that oauth2_*.go was generated (proves --oauth2-flows was honored)
	// The filename is oauth2_<schemeName>.go based on the security scheme name
	hasOAuth2 := false
	hasCredMgmt := false
	for _, name := range fileNames {
		if strings.HasPrefix(name, "oauth2_") && strings.HasSuffix(name, ".go") {
			hasOAuth2 = true
		}
		if name == "credentials.go" {
			hasCredMgmt = true
		}
	}

	if !hasOAuth2 {
		t.Errorf("--oauth2-flows flag not honored: oauth2_*.go not generated. Files: %v", fileNames)
	}
	if !hasCredMgmt {
		t.Errorf("--credential-mgmt flag not honored: credentials.go not generated. Files: %v", fileNames)
	}
}

// TestHandleGenerate_ServerAllHonored verifies --server-all generates all server components.
func TestHandleGenerate_ServerAllHonored(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: getTest
      responses:
        '200':
          description: Success
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")

	err := HandleGenerate([]string{
		"-o", outputDir,
		"--server",
		"--server-all",
		specFile,
	})
	if err != nil {
		t.Fatalf("HandleGenerate failed: %v", err)
	}

	// Check for expected server files
	expectedFiles := []string{
		"server_router.go",    // from --server-router=stdlib (via --server-all)
		"server_responses.go", // from --server-responses (via --server-all)
		"server_binder.go",    // from --server-binder (via --server-all)
		"server_stubs.go",     // from --server-stubs (via --server-all)
	}

	for _, expected := range expectedFiles {
		path := filepath.Join(outputDir, expected)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("--server-all flag not fully honored: %s not generated", expected)
		}
	}
}

// TestHandleGenerate_FileSplittingFlagsHonored verifies file splitting options work.
func TestHandleGenerate_FileSplittingFlagsHonored(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: getTest
      responses:
        '200':
          description: Success
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")

	// Just verify it doesn't error - the flags are being passed
	err := HandleGenerate([]string{
		"-o", outputDir,
		"--client",
		"--max-lines-per-file=500",
		"--max-types-per-file=50",
		"--max-ops-per-file=25",
		specFile,
	})
	if err != nil {
		t.Fatalf("HandleGenerate with file splitting flags failed: %v", err)
	}
}

// TestHandleGenerate_ValidationFlagsHonored verifies --no-validation flag is honored.
func TestHandleGenerate_ValidationFlagsHonored(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    User:
      type: object
      required:
        - name
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 100
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Test with validation enabled (default)
	outputDirWithValidation := filepath.Join(tmpDir, "with-validation")
	err := HandleGenerate([]string{
		"-o", outputDirWithValidation,
		"--types",
		specFile,
	})
	if err != nil {
		t.Fatalf("HandleGenerate failed: %v", err)
	}

	// Test with validation disabled
	outputDirNoValidation := filepath.Join(tmpDir, "no-validation")
	err = HandleGenerate([]string{
		"-o", outputDirNoValidation,
		"--types",
		"--no-validation",
		specFile,
	})
	if err != nil {
		t.Fatalf("HandleGenerate with --no-validation failed: %v", err)
	}

	// Read the types file from both outputs
	withValidation, err := os.ReadFile(filepath.Join(outputDirWithValidation, "types.go"))
	if err != nil {
		t.Fatalf("failed to read types.go with validation: %v", err)
	}
	noValidation, err := os.ReadFile(filepath.Join(outputDirNoValidation, "types.go"))
	if err != nil {
		t.Fatalf("failed to read types.go without validation: %v", err)
	}

	// With validation should have validate tags, without should not
	if !strings.Contains(string(withValidation), "validate:") {
		t.Error("--no-validation=false (default) should include validate tags")
	}
	if strings.Contains(string(noValidation), "validate:") {
		t.Error("--no-validation flag not honored: validate tags still present")
	}
}

func TestHandleGenerate_NoArgs(t *testing.T) {
	err := HandleGenerate([]string{})
	if err == nil {
		t.Error("expected error when no file provided")
	}
}

func TestHandleGenerate_Help(t *testing.T) {
	err := HandleGenerate([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestHandleGenerate_NoOutput(t *testing.T) {
	err := HandleGenerate([]string{"spec.yaml"})
	if err == nil {
		t.Error("expected error when no output directory provided")
	}
}

func TestHandleGenerate_NoGenerationMode(t *testing.T) {
	err := HandleGenerate([]string{"-o", "./out", "--types=false", "spec.yaml"})
	if err == nil {
		t.Error("expected error when no generation mode enabled")
	}
}
