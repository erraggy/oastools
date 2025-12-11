package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	g := New()

	require.NotNil(t, g, "New() should not return nil")
	assert.Equal(t, "api", g.PackageName)
	assert.False(t, g.GenerateClient, "GenerateClient should be false by default")
	assert.False(t, g.GenerateServer, "GenerateServer should be false by default")
	assert.True(t, g.GenerateTypes, "GenerateTypes should be true by default")
	assert.True(t, g.UsePointers, "UsePointers should be true by default")
	assert.True(t, g.IncludeValidation, "IncludeValidation should be true by default")
}

func TestGenerateWithOptions_RequiresInputSource(t *testing.T) {
	_, err := GenerateWithOptions(
		WithPackageName("test"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify an input source")
}

func TestGenerateWithOptions_OnlyOneInputSource(t *testing.T) {
	parsed := parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
	}

	_, err := GenerateWithOptions(
		WithFilePath("test.yaml"),
		WithParsed(parsed),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one input source")
}

func TestWithPackageName_Empty(t *testing.T) {
	_, err := GenerateWithOptions(
		WithFilePath("test.yaml"),
		WithPackageName(""),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package name cannot be empty")
}

func TestWithOptions(t *testing.T) {
	t.Run("WithFilePath", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithFilePath("test.yaml")(cfg)
		require.NoError(t, err)
		require.NotNil(t, cfg.filePath)
		assert.Equal(t, "test.yaml", *cfg.filePath)
	})

	t.Run("WithClient", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithClient(true)(cfg)
		require.NoError(t, err)
		assert.True(t, cfg.generateClient)
	})

	t.Run("WithServer", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithServer(true)(cfg)
		require.NoError(t, err)
		assert.True(t, cfg.generateServer)
	})

	t.Run("WithTypes", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithTypes(false)(cfg)
		require.NoError(t, err)
		assert.False(t, cfg.generateTypes)
	})

	t.Run("WithPointers", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithPointers(false)(cfg)
		require.NoError(t, err)
		assert.False(t, cfg.usePointers)
	})

	t.Run("WithValidation", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithValidation(false)(cfg)
		require.NoError(t, err)
		assert.False(t, cfg.includeValidation)
	})

	t.Run("WithStrictMode", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithStrictMode(true)(cfg)
		require.NoError(t, err)
		assert.True(t, cfg.strictMode)
	})

	t.Run("WithIncludeInfo", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithIncludeInfo(false)(cfg)
		require.NoError(t, err)
		assert.False(t, cfg.includeInfo)
	})

	t.Run("WithUserAgent", func(t *testing.T) {
		cfg := &generateConfig{}
		err := WithUserAgent("test-agent")(cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-agent", cfg.userAgent)
	})
}

func TestGenerateResult_WriteFiles(t *testing.T) {
	result := &GenerateResult{
		Files: []GeneratedFile{
			{Name: "types.go", Content: []byte("package test\n\ntype Foo struct{}\n")},
			{Name: "client.go", Content: []byte("package test\n\nfunc NewClient() {}\n")},
		},
	}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	err := result.WriteFiles(outputDir)
	require.NoError(t, err)

	for _, file := range result.Files {
		filePath := filepath.Join(outputDir, file.Name)
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "should read %s", file.Name)
		assert.Equal(t, string(file.Content), string(content))
	}
}

func TestGenerateResult_GetFile(t *testing.T) {
	result := &GenerateResult{
		Files: []GeneratedFile{
			{Name: "types.go", Content: []byte("package test")},
			{Name: "client.go", Content: []byte("package test")},
		},
	}

	assert.NotNil(t, result.GetFile("types.go"), "should find types.go")
	assert.Nil(t, result.GetFile("nonexistent.go"), "should return nil for non-existing file")
}

func TestGenerateResult_HasCriticalIssues(t *testing.T) {
	result := &GenerateResult{CriticalCount: 0}
	assert.False(t, result.HasCriticalIssues())

	result.CriticalCount = 1
	assert.True(t, result.HasCriticalIssues())
}

func TestGenerateResult_HasWarnings(t *testing.T) {
	result := &GenerateResult{WarningCount: 0}
	assert.False(t, result.HasWarnings())

	result.WarningCount = 1
	assert.True(t, result.HasWarnings())
}

func TestGeneratedFile_WriteFile(t *testing.T) {
	file := &GeneratedFile{
		Name:    "test.go",
		Content: []byte("package test\n"),
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "subdir", "test.go")

	err := file.WriteFile(filePath)
	require.NoError(t, err)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "package test\n", string(content))
}

func TestGeneratorStruct_Generate(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.PackageName = "testapi"
	g.GenerateTypes = true

	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "testapi", result.PackageName)
	assert.Equal(t, 1, result.GeneratedTypes)
}

func TestGeneratorStruct_GenerateParsed(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	require.NoError(t, err)

	g := New()
	g.PackageName = "testapi"
	g.GenerateClient = true
	g.GenerateServer = true

	result, err := g.GenerateParsed(*parseResult)
	require.NoError(t, err)

	assert.NotNil(t, result.GetFile("types.go"))
	assert.NotNil(t, result.GetFile("client.go"))
	assert.NotNil(t, result.GetFile("server.go"))
}

func TestGenerateWithParsedDocument(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithParsed(*parseResult),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	assert.Equal(t, 1, result.GeneratedTypes)
}

func TestGenerateEmptyPackageName(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.PackageName = ""
	g.GenerateTypes = true

	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "api", result.PackageName, "should default to 'api'")
}

func TestGenerateFileNotFound(t *testing.T) {
	_, err := GenerateWithOptions(
		WithFilePath("nonexistent.yaml"),
		WithPackageName("testapi"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generator: failed to parse specification")
}

func TestGenerateInvalidSpec(t *testing.T) {
	spec := `not valid yaml: [[[`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	_, err = GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.Error(t, err)
}

func TestGenerateWithStrictMode(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
    Cat:
      type: object
      properties:
        meow:
          type: boolean
    Dog:
      type: object
      properties:
        bark:
          type: boolean
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithStrictMode(false),
		WithIncludeInfo(true),
	)
	require.NoError(t, err)

	assert.Greater(t, result.InfoCount, 0, "should have info messages about oneOf")
}

func TestGenerateWithoutInfo(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - type: string
        - type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithIncludeInfo(false),
	)
	require.NoError(t, err)

	for _, issue := range result.Issues {
		assert.NotEqual(t, SeverityInfo, issue.Severity, "info messages should be filtered out")
	}
}
