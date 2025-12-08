package generator

import (
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_Generate tests code generation for all non-large corpus specifications.
// This test ensures generated code is valid Go syntax by parsing it.
func TestCorpus_Generate(t *testing.T) {
	specs := corpusutil.GetParseableSpecs(false) // Exclude large specs and specs with parsing issues

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			g := New()
			g.PackageName = "testpkg"
			g.GenerateClient = true
			g.GenerateServer = true
			g.GenerateTypes = true
			g.UsePointers = true

			result, err := g.Generate(spec.GetLocalPath())
			require.NoError(t, err, "Generation should complete for %s", spec.Name)
			require.NotNil(t, result)

			// Verify we generated files
			assert.NotEmpty(t, result.Files, "%s should generate files", spec.Name)

			// Verify generated code is valid Go syntax
			fset := token.NewFileSet()
			for _, file := range result.Files {
				_, err := goparser.ParseFile(fset, file.Name, file.Content, goparser.AllErrors)
				assert.NoError(t, err, "%s: generated %s should be valid Go syntax", spec.Name, file.Name)
			}

			// Check for critical issues
			assert.False(t, result.HasCriticalIssues(),
				"%s should not have critical generation issues", spec.Name)

			t.Logf("%s: Types=%d, Operations=%d, Files=%d, Issues=%d",
				spec.Name, result.GeneratedTypes, result.GeneratedOperations,
				len(result.Files), len(result.Issues))
		})
	}
}

// TestCorpus_GenerateTypesOnly tests type generation for corpus specs.
func TestCorpus_GenerateTypesOnly(t *testing.T) {
	specs := corpusutil.GetParseableSpecs(false)

	for _, spec := range specs {
		t.Run(spec.Name+"_TypesOnly", func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			result, err := GenerateWithOptions(
				WithFilePath(spec.GetLocalPath()),
				WithPackageName("testpkg"),
				WithTypes(true),
				WithClient(false),
				WithServer(false),
				WithPointers(true),
			)
			require.NoError(t, err)

			// Should have types.go
			typesFile := result.GetFile("types.go")
			require.NotNil(t, typesFile, "%s should generate types.go", spec.Name)

			// Verify valid Go syntax
			fset := token.NewFileSet()
			_, err = goparser.ParseFile(fset, "types.go", typesFile.Content, goparser.AllErrors)
			assert.NoError(t, err, "%s: types.go should be valid Go syntax", spec.Name)

			t.Logf("%s: Generated %d types", spec.Name, result.GeneratedTypes)
		})
	}
}

// TestCorpus_GenerateClient tests client generation for corpus specs.
func TestCorpus_GenerateClient(t *testing.T) {
	// Test a selection of OAS 3.x specs for client generation
	specNames := []string{"Discord", "GoogleMaps", "GitHub", "Asana"}

	for _, name := range specNames {
		spec := corpusutil.GetByName(name)
		require.NotNil(t, spec, "Spec %s should exist in corpus", name)

		t.Run(spec.Name+"_Client", func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, *spec)
			corpusutil.SkipIfHasParsingIssues(t, *spec)

			result, err := GenerateWithOptions(
				WithFilePath(spec.GetLocalPath()),
				WithPackageName("testpkg"),
				WithClient(true),
				WithTypes(true),
				WithPointers(true),
			)
			require.NoError(t, err)

			// Should have client.go and types.go
			clientFile := result.GetFile("client.go")
			require.NotNil(t, clientFile, "%s should generate client.go", spec.Name)

			// Verify valid Go syntax
			fset := token.NewFileSet()
			_, err = goparser.ParseFile(fset, "client.go", clientFile.Content, goparser.AllErrors)
			assert.NoError(t, err, "%s: client.go should be valid Go syntax", spec.Name)

			t.Logf("%s: Generated %d operations", spec.Name, result.GeneratedOperations)
		})
	}
}

// TestCorpus_OAS2Generation tests generation for OAS 2.0 (Swagger) specs.
func TestCorpus_OAS2Generation(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec, "Petstore spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := GenerateWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithPackageName("petstore"),
		WithClient(true),
		WithServer(true),
		WithTypes(true),
	)
	require.NoError(t, err)

	// Verify all generated files are valid Go
	fset := token.NewFileSet()
	for _, file := range result.Files {
		_, err := goparser.ParseFile(fset, file.Name, file.Content, goparser.AllErrors)
		assert.NoError(t, err, "Petstore: %s should be valid Go syntax", file.Name)
	}

	assert.False(t, result.HasCriticalIssues(),
		"Petstore should not have critical generation issues")

	t.Logf("Petstore OAS 2.0: Types=%d, Operations=%d",
		result.GeneratedTypes, result.GeneratedOperations)
}

// TestCorpus_OAS31Generation tests generation for OAS 3.1.0 specs.
func TestCorpus_OAS31Generation(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec, "Discord spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := GenerateWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithPackageName("discord"),
		WithClient(true),
		WithServer(true),
		WithTypes(true),
		WithPointers(true),
	)
	require.NoError(t, err)

	// Verify all generated files are valid Go
	fset := token.NewFileSet()
	for _, file := range result.Files {
		_, err := goparser.ParseFile(fset, file.Name, file.Content, goparser.AllErrors)
		assert.NoError(t, err, "Discord: %s should be valid Go syntax", file.Name)
	}

	assert.False(t, result.HasCriticalIssues(),
		"Discord should not have critical generation issues")

	t.Logf("Discord OAS 3.1.0: Types=%d, Operations=%d",
		result.GeneratedTypes, result.GeneratedOperations)
}
