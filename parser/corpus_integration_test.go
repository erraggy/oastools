package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_Parse tests parsing all non-large corpus specifications.
func TestCorpus_Parse(t *testing.T) {
	specs := corpusutil.GetParseableSpecs(false) // Exclude large specs and specs with parsing issues

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			result, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Parser should not return error for %s", spec.Name)
			require.NotNil(t, result)

			// Verify version detection
			if spec.OASVersion == "2.0" {
				assert.Equal(t, "2.0", result.Version,
					"Version should be 2.0 for %s", spec.Name)
			} else {
				assert.True(t, strings.HasPrefix(result.Version, "3."),
					"Version should start with 3. for %s (got %s)", spec.Name, result.Version)
			}

			// Verify format detection
			expectedFormat := SourceFormatJSON
			if spec.Format == "yaml" {
				expectedFormat = SourceFormatYAML
			}
			assert.Equal(t, expectedFormat, result.SourceFormat,
				"Format should be detected correctly for %s", spec.Name)

			// Verify document type based on version
			if spec.OASVersion == "2.0" {
				_, ok := result.Document.(*OAS2Document)
				assert.True(t, ok, "Should parse as OAS2Document for %s", spec.Name)
			} else {
				_, ok := result.Document.(*OAS3Document)
				assert.True(t, ok, "Should parse as OAS3Document for %s", spec.Name)
			}

			// Log stats for visibility
			t.Logf("%s: Version=%s, Format=%s, Size=%d bytes",
				spec.Name, result.Version, result.SourceFormat, result.SourceSize)
		})
	}
}

// TestCorpus_ParseBytes tests parsing from byte slices for smaller specs.
func TestCorpus_ParseBytes(t *testing.T) {
	for _, spec := range corpusutil.GetParseableSpecs(false) {
		if spec.SizeBytes > 2_000_000 { // Skip specs > 2MB
			continue
		}

		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			data, err := os.ReadFile(spec.GetLocalPath())
			require.NoError(t, err)

			p := New()
			result, err := p.ParseBytes(data)
			require.NoError(t, err)
			require.NotNil(t, result)

			if spec.OASVersion == "2.0" {
				assert.Equal(t, "2.0", result.Version)
			} else {
				assert.True(t, strings.HasPrefix(result.Version, "3."))
			}
		})
	}
}

// TestCorpus_OAS31Features tests OAS 3.1.0 specific features using Discord spec.
func TestCorpus_OAS31Features(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec, "Discord spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	p := New()
	result, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok, "Discord should parse as OAS3Document")

	assert.Equal(t, "3.1.0", doc.OpenAPI, "Discord should be OAS 3.1.0")
	assert.Equal(t, OASVersion310, doc.OASVersion, "OASVersion should be 3.1.0")

	t.Logf("Discord OAS 3.1.0 parsed successfully with %d paths", len(doc.Paths))
}

// TestCorpus_OAS2Features tests OAS 2.0 specific features using Petstore spec.
func TestCorpus_OAS2Features(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec, "Petstore spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	p := New()
	result, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	doc, ok := result.Document.(*OAS2Document)
	require.True(t, ok, "Petstore should parse as OAS2Document")

	assert.Equal(t, "2.0", doc.Swagger, "Petstore should be Swagger 2.0")
	assert.NotEmpty(t, doc.Host, "Petstore should have host defined")
	assert.NotEmpty(t, doc.BasePath, "Petstore should have basePath defined")
	assert.NotEmpty(t, doc.Definitions, "Petstore should have definitions")

	t.Logf("Petstore OAS 2.0 parsed: %d paths, %d definitions",
		len(doc.Paths), len(doc.Definitions))
}

// TestCorpus_VersionDetection verifies correct version detection across all specs.
func TestCorpus_VersionDetection(t *testing.T) {
	versionMap := map[string]string{
		"Petstore": "2.0",
		"Discord":  "3.1.0",
		"Asana":    "3.0.0",
	}

	for name, expectedVersion := range versionMap {
		t.Run(name, func(t *testing.T) {
			spec := corpusutil.GetByName(name)
			require.NotNil(t, spec)
			corpusutil.SkipIfNotCached(t, *spec)

			p := New()
			result, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err)

			assert.Equal(t, expectedVersion, result.Version,
				"%s should have version %s", name, expectedVersion)
		})
	}
}

// TestCorpus_FormatDetection verifies correct format detection.
func TestCorpus_FormatDetection(t *testing.T) {
	yamlSpecs := []string{"Asana", "Plaid"}
	jsonSpecs := []string{"Petstore", "Discord", "GoogleMaps", "USNWS"}

	for _, name := range yamlSpecs {
		t.Run(name+"_YAML", func(t *testing.T) {
			spec := corpusutil.GetByName(name)
			require.NotNil(t, spec)
			corpusutil.SkipIfNotCached(t, *spec)

			p := New()
			result, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err)
			assert.Equal(t, SourceFormatYAML, result.SourceFormat)
		})
	}

	for _, name := range jsonSpecs {
		t.Run(name+"_JSON", func(t *testing.T) {
			spec := corpusutil.GetByName(name)
			require.NotNil(t, spec)
			corpusutil.SkipIfNotCached(t, *spec)

			p := New()
			result, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err)
			assert.Equal(t, SourceFormatJSON, result.SourceFormat)
		})
	}
}

// TestCorpus_ValidateStructure tests structural validation during parsing.
func TestCorpus_ValidateStructure(t *testing.T) {
	for _, spec := range corpusutil.GetParseableSpecs(false) {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			p := New()
			p.ValidateStructure = true

			result, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Structural validation should pass for %s", spec.Name)
			require.NotNil(t, result)
		})
	}
}
