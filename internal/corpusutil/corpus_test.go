package corpusutil

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorpus_Count(t *testing.T) {
	assert.Equal(t, 10, len(Corpus), "Corpus should contain exactly 10 specifications")
}

func TestCorpus_UniqueNames(t *testing.T) {
	names := make(map[string]bool)
	for _, spec := range Corpus {
		assert.False(t, names[spec.Name], "Duplicate name found: %s", spec.Name)
		names[spec.Name] = true
	}
}

func TestCorpus_UniqueFilenames(t *testing.T) {
	filenames := make(map[string]bool)
	for _, spec := range Corpus {
		assert.False(t, filenames[spec.Filename], "Duplicate filename found: %s", spec.Filename)
		filenames[spec.Filename] = true
	}
}

func TestCorpus_ValidURLs(t *testing.T) {
	for _, spec := range Corpus {
		t.Run(spec.Name, func(t *testing.T) {
			assert.True(t, strings.HasPrefix(spec.URL, "https://"),
				"%s URL should start with https://", spec.Name)
		})
	}
}

func TestCorpus_ValidFormats(t *testing.T) {
	for _, spec := range Corpus {
		t.Run(spec.Name, func(t *testing.T) {
			assert.Contains(t, []string{"json", "yaml"}, spec.Format,
				"%s format should be json or yaml", spec.Name)

			// Verify filename matches format
			ext := filepath.Ext(spec.Filename)
			if spec.Format == "json" {
				assert.Equal(t, ".json", ext)
			} else {
				assert.Contains(t, []string{".yaml", ".yml"}, ext)
			}
		})
	}
}

func TestCorpus_ValidOASVersions(t *testing.T) {
	validVersions := []string{"2.0", "3.0.0", "3.0.3", "3.0.4", "3.1.0"}
	for _, spec := range Corpus {
		t.Run(spec.Name, func(t *testing.T) {
			assert.Contains(t, validVersions, spec.OASVersion,
				"%s should have a valid OAS version", spec.Name)
		})
	}
}

func TestCorpus_LargeSpecsMarked(t *testing.T) {
	for _, spec := range Corpus {
		t.Run(spec.Name, func(t *testing.T) {
			if spec.SizeBytes > 5_000_000 {
				assert.True(t, spec.IsLarge,
					"%s is >5MB (%d bytes) but not marked as large",
					spec.Name, spec.SizeBytes)
			}
		})
	}
}

func TestGetSpecs_ExcludesLarge(t *testing.T) {
	specs := GetSpecs(false)
	for _, spec := range specs {
		assert.False(t, spec.IsLarge, "GetSpecs(false) should exclude large specs")
	}
}

func TestGetSpecs_IncludesLarge(t *testing.T) {
	specsWithLarge := GetSpecs(true)
	specsWithoutLarge := GetSpecs(false)
	assert.Greater(t, len(specsWithLarge), len(specsWithoutLarge),
		"GetSpecs(true) should include more specs than GetSpecs(false)")
}

func TestGetValidSpecs(t *testing.T) {
	validSpecs := GetValidSpecs(false)
	for _, spec := range validSpecs {
		assert.True(t, spec.ExpectedValid, "GetValidSpecs should only return valid specs")
		assert.Equal(t, 0, spec.ExpectedErrors, "Valid specs should have 0 expected errors")
	}
}

func TestGetInvalidSpecs(t *testing.T) {
	invalidSpecs := GetInvalidSpecs(false)
	for _, spec := range invalidSpecs {
		assert.False(t, spec.ExpectedValid, "GetInvalidSpecs should only return invalid specs")
		assert.Greater(t, spec.ExpectedErrors, 0, "Invalid specs should have >0 expected errors")
	}
}

func TestGetOAS2Specs(t *testing.T) {
	oas2Specs := GetOAS2Specs()
	require.Len(t, oas2Specs, 1, "Should have exactly 1 OAS 2.0 spec (Petstore)")
	assert.Equal(t, "Petstore", oas2Specs[0].Name)
	assert.Equal(t, "2.0", oas2Specs[0].OASVersion)
}

func TestGetOAS3Specs(t *testing.T) {
	oas3Specs := GetOAS3Specs(false)
	for _, spec := range oas3Specs {
		assert.NotEqual(t, "2.0", spec.OASVersion, "GetOAS3Specs should not include OAS 2.0")
		assert.True(t, strings.HasPrefix(spec.OASVersion, "3."),
			"OAS version should start with 3.")
	}
}

func TestGetLargeSpecs(t *testing.T) {
	largeSpecs := GetLargeSpecs()
	assert.Equal(t, 2, len(largeSpecs), "Should have exactly 2 large specs")
	for _, spec := range largeSpecs {
		assert.True(t, spec.IsLarge)
		assert.Greater(t, spec.SizeBytes, int64(5_000_000))
	}
}

func TestGetByName(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		spec := GetByName("Stripe")
		assert.NotNil(t, spec)
		assert.Equal(t, "Stripe", spec.Name)
	})

	t.Run("not_found", func(t *testing.T) {
		spec := GetByName("NonExistent")
		assert.Nil(t, spec)
	})
}

func TestSpecInfo_GetLocalPath(t *testing.T) {
	spec := Corpus[0]
	path := spec.GetLocalPath()
	assert.True(t, strings.HasSuffix(path, filepath.Join("testdata", "corpus", spec.Filename)),
		"Path should end with testdata/corpus/<filename>")
}

func TestCorpusDir(t *testing.T) {
	dir := CorpusDir()
	assert.True(t, strings.HasSuffix(dir, filepath.Join("testdata", "corpus")),
		"CorpusDir should end with testdata/corpus")
}

func TestVersionCoverage(t *testing.T) {
	// Verify we have good version coverage
	versions := make(map[string]int)
	for _, spec := range Corpus {
		versions[spec.OASVersion]++
	}

	assert.GreaterOrEqual(t, versions["2.0"], 1, "Should have at least 1 OAS 2.0 spec")
	assert.GreaterOrEqual(t, versions["3.0.0"], 1, "Should have at least 1 OAS 3.0.0 spec")
	assert.GreaterOrEqual(t, versions["3.1.0"], 1, "Should have at least 1 OAS 3.1.0 spec")
}

func TestFormatCoverage(t *testing.T) {
	// Verify we have both JSON and YAML coverage
	formats := make(map[string]int)
	for _, spec := range Corpus {
		formats[spec.Format]++
	}

	assert.GreaterOrEqual(t, formats["json"], 1, "Should have at least 1 JSON spec")
	assert.GreaterOrEqual(t, formats["yaml"], 1, "Should have at least 1 YAML spec")
}
