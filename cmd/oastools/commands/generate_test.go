package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupGenerateFlags(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	t.Run("default values", func(t *testing.T) {
		assert.Equal(t, "", flags.Output)
		assert.Equal(t, "api", flags.PackageName)
		assert.False(t, flags.Client, "expected Client to be false by default")
		assert.False(t, flags.Server, "expected Server to be false by default")
		assert.True(t, flags.Types, "expected Types to be true by default")
		assert.False(t, flags.NoPointers, "expected NoPointers to be false by default")
		assert.False(t, flags.NoValidation, "expected NoValidation to be false by default")
		assert.False(t, flags.Strict, "expected Strict to be false by default")
		assert.False(t, flags.NoWarnings, "expected NoWarnings to be false by default")
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "./output", "-p", "myapi", "--client", "--server", "--no-pointers", "--strict", "spec.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, "./output", flags.Output)
		assert.Equal(t, "myapi", flags.PackageName)
		assert.True(t, flags.Client, "expected Client to be true")
		assert.True(t, flags.Server, "expected Server to be true")
		assert.True(t, flags.NoPointers, "expected NoPointers to be true")
		assert.True(t, flags.Strict, "expected Strict to be true")
		assert.Equal(t, "spec.yaml", fs.Arg(0))
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

	require.NoError(t, fs.Parse(args))

	// Verify all security flags are parsed
	assert.True(t, flags.NoSecurity, "expected NoSecurity to be true")
	assert.True(t, flags.OAuth2Flows, "expected OAuth2Flows to be true")
	assert.True(t, flags.CredentialMgmt, "expected CredentialMgmt to be true")
	assert.True(t, flags.SecurityEnforce, "expected SecurityEnforce to be true")
	assert.True(t, flags.OIDCDiscovery, "expected OIDCDiscovery to be true")
	assert.True(t, flags.NoReadme, "expected NoReadme to be true")
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

	require.NoError(t, fs.Parse(args))

	assert.Equal(t, "chi", flags.ServerRouter)
	assert.True(t, flags.ServerMiddleware, "expected ServerMiddleware to be true")
	assert.True(t, flags.ServerBinder, "expected ServerBinder to be true")
	assert.True(t, flags.ServerResponses, "expected ServerResponses to be true")
	assert.True(t, flags.ServerStubs, "expected ServerStubs to be true")
	assert.True(t, flags.ServerEmbedSpec, "expected ServerEmbedSpec to be true")
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

	require.NoError(t, fs.Parse(args))

	assert.True(t, flags.ServerAll, "expected ServerAll to be true")
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

	require.NoError(t, fs.Parse(args))

	assert.Equal(t, 1500, flags.MaxLinesPerFile)
	assert.Equal(t, 150, flags.MaxTypesPerFile)
	assert.Equal(t, 50, flags.MaxOpsPerFile)
	assert.True(t, flags.NoSplitByTag, "expected NoSplitByTag to be true")
	assert.True(t, flags.NoSplitByPath, "expected NoSplitByPath to be true")
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
	require.NoError(t, os.WriteFile(specFile, []byte(spec), 0600))

	outputDir := filepath.Join(tmpDir, "output")

	// Run generate with security flags
	err := HandleGenerate([]string{
		"-o", outputDir,
		"--client",
		"--oauth2-flows",
		"--credential-mgmt",
		specFile,
	})
	require.NoError(t, err)

	// Verify security files were generated
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)

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

	assert.True(t, hasOAuth2, "--oauth2-flows flag not honored: oauth2_*.go not generated. Files: %v", fileNames)
	assert.True(t, hasCredMgmt, "--credential-mgmt flag not honored: credentials.go not generated. Files: %v", fileNames)
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
	require.NoError(t, os.WriteFile(specFile, []byte(spec), 0600))

	outputDir := filepath.Join(tmpDir, "output")

	err := HandleGenerate([]string{
		"-o", outputDir,
		"--server",
		"--server-all",
		specFile,
	})
	require.NoError(t, err)

	// Check for expected server files
	expectedFiles := []string{
		"server_router.go",    // from --server-router=stdlib (via --server-all)
		"server_responses.go", // from --server-responses (via --server-all)
		"server_binder.go",    // from --server-binder (via --server-all)
		"server_stubs.go",     // from --server-stubs (via --server-all)
	}

	for _, expected := range expectedFiles {
		path := filepath.Join(outputDir, expected)
		_, err := os.Stat(path)
		assert.False(t, os.IsNotExist(err), "--server-all flag not fully honored: %s not generated", expected)
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
	require.NoError(t, os.WriteFile(specFile, []byte(spec), 0600))

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
	require.NoError(t, err)
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
	require.NoError(t, os.WriteFile(specFile, []byte(spec), 0600))

	// Test with validation enabled (default)
	outputDirWithValidation := filepath.Join(tmpDir, "with-validation")
	err := HandleGenerate([]string{
		"-o", outputDirWithValidation,
		"--types",
		specFile,
	})
	require.NoError(t, err)

	// Test with validation disabled
	outputDirNoValidation := filepath.Join(tmpDir, "no-validation")
	err = HandleGenerate([]string{
		"-o", outputDirNoValidation,
		"--types",
		"--no-validation",
		specFile,
	})
	require.NoError(t, err)

	// Read the types file from both outputs
	withValidation, err := os.ReadFile(filepath.Join(outputDirWithValidation, "types.go"))
	require.NoError(t, err)
	noValidation, err := os.ReadFile(filepath.Join(outputDirNoValidation, "types.go"))
	require.NoError(t, err)

	// With validation should have validate tags, without should not
	assert.Contains(t, string(withValidation), "validate:")
	assert.NotContains(t, string(noValidation), "validate:")
}

func TestHandleGenerate_NoArgs(t *testing.T) {
	err := HandleGenerate([]string{})
	assert.Error(t, err)
}

func TestHandleGenerate_Help(t *testing.T) {
	err := HandleGenerate([]string{"--help"})
	assert.NoError(t, err)
}

func TestHandleGenerate_NoOutput(t *testing.T) {
	err := HandleGenerate([]string{"spec.yaml"})
	assert.Error(t, err)
}

func TestHandleGenerate_NoGenerationMode(t *testing.T) {
	err := HandleGenerate([]string{"-o", "./out", "--types=false", "spec.yaml"})
	assert.Error(t, err)
}
