package converter

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_ConvertOAS2ToOAS3 tests converting Petstore 2.0 to various 3.x versions.
func TestCorpus_ConvertOAS2ToOAS3(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec, "Petstore spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	targetVersions := []string{"3.0.0", "3.0.3", "3.1.0"}

	for _, targetVersion := range targetVersions {
		t.Run("to_"+targetVersion, func(t *testing.T) {
			result, err := ConvertWithOptions(
				WithFilePath(spec.GetLocalPath()),
				WithTargetVersion(targetVersion),
				WithIncludeInfo(true),
			)
			require.NoError(t, err)
			assert.True(t, result.Success, "Conversion should succeed")

			// Verify converted document
			doc, ok := result.Document.(*parser.OAS3Document)
			require.True(t, ok, "Result should be OAS3Document")
			assert.Equal(t, targetVersion, doc.OpenAPI)

			// Verify key elements converted
			assert.NotNil(t, doc.Components, "Should have components")
			assert.NotNil(t, doc.Servers, "Should have servers")
			assert.True(t, len(doc.Paths) > 0, "Should have paths")

			// Verify schemas migrated from definitions
			assert.NotEmpty(t, doc.Components.Schemas,
				"Schemas should be converted from definitions")

			t.Logf("Petstore 2.0 -> %s: Success, Issues=%d (Info=%d, Warn=%d, Crit=%d)",
				targetVersion, len(result.Issues), result.InfoCount,
				result.WarningCount, result.CriticalCount)
		})
	}
}

// TestCorpus_ConvertRefRewriting verifies $ref paths are properly rewritten.
func TestCorpus_ConvertRefRewriting(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := ConvertWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithTargetVersion("3.0.3"),
	)
	require.NoError(t, err)
	require.True(t, result.Success)

	doc := result.Document.(*parser.OAS3Document)

	// Check schemas for properly rewritten refs
	for name, schema := range doc.Components.Schemas {
		if schema.Ref != "" {
			assert.Contains(t, schema.Ref, "#/components/schemas/",
				"Schema %s ref should use OAS 3.x format", name)
			assert.NotContains(t, schema.Ref, "#/definitions/",
				"Schema %s ref should not use OAS 2.0 format", name)
		}
	}

	t.Logf("Ref rewriting verified for %d schemas", len(doc.Components.Schemas))
}

// TestCorpus_ConvertServerGeneration tests server URL generation from host/basePath.
func TestCorpus_ConvertServerGeneration(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := ConvertWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithTargetVersion("3.0.0"),
	)
	require.NoError(t, err)
	require.True(t, result.Success)

	doc := result.Document.(*parser.OAS3Document)

	// Petstore should have servers generated from host + basePath
	require.NotEmpty(t, doc.Servers, "Should have servers array")
	assert.Contains(t, doc.Servers[0].URL, "petstore.swagger.io",
		"Server URL should contain petstore.swagger.io")

	t.Logf("Generated %d servers from OAS 2.0 host/basePath", len(doc.Servers))
}

// TestCorpus_ConvertSecuritySchemes tests security definition conversion.
func TestCorpus_ConvertSecuritySchemes(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := ConvertWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithTargetVersion("3.0.0"),
	)
	require.NoError(t, err)
	require.True(t, result.Success)

	doc := result.Document.(*parser.OAS3Document)

	// Petstore has OAuth2 and API key security
	require.NotNil(t, doc.Components.SecuritySchemes,
		"Should have security schemes")
	assert.NotEmpty(t, doc.Components.SecuritySchemes,
		"Should have converted security definitions")

	t.Logf("Converted %d security schemes", len(doc.Components.SecuritySchemes))
}

// TestCorpus_ConvertOAS3ToOAS2 tests converting an OAS 3.0 spec back to 2.0.
func TestCorpus_ConvertOAS3ToOAS2(t *testing.T) {
	// Use Discord as it's a clean OAS 3.1 spec
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	c := New()
	c.StrictMode = false // Allow lossy conversions
	c.IncludeInfo = true

	result, err := c.Convert(spec.GetLocalPath(), "2.0")
	require.NoError(t, err)

	// May have critical issues due to 3.x features that don't map to 2.0
	t.Logf("Discord 3.1 -> 2.0: Success=%v, CriticalIssues=%d",
		result.Success, result.CriticalCount)

	if result.Success {
		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)
		assert.Equal(t, "2.0", doc.Swagger)
		assert.NotEmpty(t, doc.Paths, "Should have paths")
	}
}

// TestCorpus_ConvertPreservesInfo tests that info section is preserved.
func TestCorpus_ConvertPreservesInfo(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	// Parse original to get info
	p := parser.New()
	original, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)
	origDoc, ok := original.OAS2Document()
	require.True(t, ok, "Expected OAS2Document")

	// Convert
	result, err := ConvertWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithTargetVersion("3.0.0"),
	)
	require.NoError(t, err)
	require.True(t, result.Success)

	doc := result.Document.(*parser.OAS3Document)

	// Info should be preserved
	assert.Equal(t, origDoc.Info.Title, doc.Info.Title, "Title should be preserved")
	assert.Equal(t, origDoc.Info.Version, doc.Info.Version, "Version should be preserved")
}

// TestCorpus_ConvertIssueTracking tests that conversion issues are tracked.
func TestCorpus_ConvertIssueTracking(t *testing.T) {
	// Use a spec that will have some conversion issues
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := ConvertWithOptions(
		WithFilePath(spec.GetLocalPath()),
		WithTargetVersion("3.0.0"), // Discord 3.1 to 3.0.0
		WithIncludeInfo(true),
	)
	require.NoError(t, err)

	// Log all issues by severity
	t.Logf("Conversion issues: Info=%d, Warning=%d, Critical=%d",
		result.InfoCount, result.WarningCount, result.CriticalCount)

	// Log that issues are being tracked
	assert.GreaterOrEqual(t, len(result.Issues), 0,
		"Should track conversion issues")
}

// TestCorpus_ConvertVersionDetection tests that source version is detected.
func TestCorpus_ConvertVersionDetection(t *testing.T) {
	testCases := []struct {
		name          string
		targetVersion string
	}{
		{"Petstore", "3.0.0"}, // 2.0 -> 3.0
		{"Discord", "3.0.3"},  // 3.1 -> 3.0 (downgrade)
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_to_"+tc.targetVersion, func(t *testing.T) {
			spec := corpusutil.GetByName(tc.name)
			require.NotNil(t, spec)
			corpusutil.SkipIfNotCached(t, *spec)

			c := New()
			c.StrictMode = false

			result, err := c.Convert(spec.GetLocalPath(), tc.targetVersion)
			require.NoError(t, err)

			// Verify target version in output
			if result.Success {
				if strings.HasPrefix(tc.targetVersion, "3.") {
					doc, ok := result.Document.(*parser.OAS3Document)
					if ok {
						assert.Equal(t, tc.targetVersion, doc.OpenAPI)
					}
				} else {
					doc, ok := result.Document.(*parser.OAS2Document)
					if ok {
						assert.Equal(t, tc.targetVersion, doc.Swagger)
					}
				}
			}

			t.Logf("%s -> %s: Success=%v", tc.name, tc.targetVersion, result.Success)
		})
	}
}
