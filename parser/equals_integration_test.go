package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: ptr, intPtr, and boolPtr helper functions are defined in schema_test_helpers.go

// =============================================================================
// Integration Tests - ParseResult Equality with Real Parsed Files
// =============================================================================

func TestEquals_ParsedFiles(t *testing.T) {
	tests := []struct {
		name      string
		path1     string
		path2     string
		wantEqual bool
	}{
		{
			name:      "same OAS 3.0 file parsed twice",
			path1:     "../testdata/petstore-3.0.yaml",
			path2:     "../testdata/petstore-3.0.yaml",
			wantEqual: true,
		},
		{
			name:      "same OAS 3.1 file parsed twice",
			path1:     "../testdata/petstore-3.1.yaml",
			path2:     "../testdata/petstore-3.1.yaml",
			wantEqual: true,
		},
		{
			name:      "same OAS 2.0 file parsed twice",
			path1:     "../testdata/petstore-2.0.yaml",
			path2:     "../testdata/petstore-2.0.yaml",
			wantEqual: true,
		},
		{
			name:      "different petstore versions v1 vs v2",
			path1:     "../testdata/petstore-v1.yaml",
			path2:     "../testdata/petstore-v2.yaml",
			wantEqual: false,
		},
		{
			name:      "different OAS major versions 2.0 vs 3.0",
			path1:     "../testdata/petstore-2.0.yaml",
			path2:     "../testdata/petstore-3.0.yaml",
			wantEqual: false,
		},
		{
			name:      "different OAS minor versions 3.0 vs 3.1",
			path1:     "../testdata/petstore-3.0.yaml",
			path2:     "../testdata/petstore-3.1.yaml",
			wantEqual: false,
		},
		{
			name:      "minimal OAS 2.0 parsed twice",
			path1:     "../testdata/minimal-oas2.yaml",
			path2:     "../testdata/minimal-oas2.yaml",
			wantEqual: true,
		},
		{
			name:      "minimal OAS 3.0 parsed twice",
			path1:     "../testdata/minimal-oas3.yaml",
			path2:     "../testdata/minimal-oas3.yaml",
			wantEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result1, err := ParseWithOptions(WithFilePath(tt.path1))
			require.NoError(t, err)

			result2, err := ParseWithOptions(WithFilePath(tt.path2))
			require.NoError(t, err)

			got := result1.Equals(result2)
			assert.Equal(t, tt.wantEqual, got, "Equals() mismatch")

			// Also test symmetry: result2.Equals(result1) should give same result
			gotReverse := result2.Equals(result1)
			assert.Equal(t, tt.wantEqual, gotReverse, "Equals() symmetry failed")
		})
	}
}

func TestEquals_MetadataIgnored(t *testing.T) {
	// Parse the same file
	result1, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.0.yaml"))
	require.NoError(t, err)

	result2, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.0.yaml"))
	require.NoError(t, err)

	// Verify they start equal
	require.True(t, result1.Equals(result2), "initial parsed results should be equal")

	t.Run("different SourcePath", func(t *testing.T) {
		result2.SourcePath = "/different/path/api.yaml"
		assert.True(t, result1.Equals(result2), "Equals should ignore SourcePath differences")
	})

	t.Run("different SourceFormat", func(t *testing.T) {
		result2.SourceFormat = SourceFormatJSON
		assert.True(t, result1.Equals(result2), "Equals should ignore SourceFormat differences")
	})

	t.Run("different LoadTime", func(t *testing.T) {
		result2.LoadTime = result1.LoadTime + 1000000000 // 1 second difference
		assert.True(t, result1.Equals(result2), "Equals should ignore LoadTime differences")
	})

	t.Run("different SourceSize", func(t *testing.T) {
		result2.SourceSize = result1.SourceSize + 1000
		assert.True(t, result1.Equals(result2), "Equals should ignore SourceSize differences")
	})

	t.Run("different Errors", func(t *testing.T) {
		result2.Errors = append(result2.Errors, fmt.Errorf("test error"))
		assert.True(t, result1.Equals(result2), "Equals should ignore Errors differences")
	})

	t.Run("different Warnings", func(t *testing.T) {
		result2.Warnings = append(result2.Warnings, "test warning")
		assert.True(t, result1.Equals(result2), "Equals should ignore Warnings differences")
	})

	t.Run("different Stats", func(t *testing.T) {
		result2.Stats = DocumentStats{
			PathCount:      999,
			OperationCount: 999,
			SchemaCount:    999,
		}
		assert.True(t, result1.Equals(result2), "Equals should ignore Stats differences")
	})
}

func TestEquals_CopyConsistency(t *testing.T) {
	testFiles := []string{
		"../testdata/petstore-3.0.yaml",
		"../testdata/petstore-3.1.yaml",
		"../testdata/petstore-2.0.yaml",
		"../testdata/minimal-oas3.yaml",
		"../testdata/minimal-oas2.yaml",
	}

	for _, path := range testFiles {
		t.Run(path, func(t *testing.T) {
			original, err := ParseWithOptions(WithFilePath(path))
			require.NoError(t, err)

			copied := original.Copy()

			// Original should equal copy
			assert.True(t, original.Equals(copied), "original.Equals(copy) should be true")

			// Copy should equal original (symmetry)
			assert.True(t, copied.Equals(original), "copy.Equals(original) should be true")

			// Reflexivity: copy should equal itself
			assert.True(t, copied.Equals(copied), "copy.Equals(copy) should be true")
		})
	}
}

func TestEquals_ModifiedDocument(t *testing.T) {
	t.Run("OAS3 modified Info.Title", func(t *testing.T) {
		original, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.0.yaml"))
		require.NoError(t, err)

		copied := original.Copy()

		// Modify the copy's document title
		doc, ok := copied.OAS3Document()
		require.True(t, ok, "expected OAS3 document")
		doc.Info.Title = "Modified Title"

		assert.False(t, original.Equals(copied), "original should NOT equal copy after modifying Info.Title")
	})

	t.Run("OAS3 add path", func(t *testing.T) {
		original, err := ParseWithOptions(WithFilePath("../testdata/minimal-oas3.yaml"))
		require.NoError(t, err)

		copied := original.Copy()

		// Add a new path to the copy
		doc, ok := copied.OAS3Document()
		require.True(t, ok, "expected OAS3 document")
		if doc.Paths == nil {
			doc.Paths = make(map[string]*PathItem)
		}
		doc.Paths["/new-endpoint"] = &PathItem{
			Get: &Operation{
				Summary: "New endpoint",
			},
		}

		assert.False(t, original.Equals(copied), "original should NOT equal copy after adding path")
	})

	t.Run("OAS3 modify schema property", func(t *testing.T) {
		original, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.0.yaml"))
		require.NoError(t, err)

		copied := original.Copy()

		// Modify a schema in the copy
		doc, ok := copied.OAS3Document()
		require.True(t, ok, "expected OAS3 document")
		if doc.Components != nil && doc.Components.Schemas != nil {
			for _, schema := range doc.Components.Schemas {
				schema.Description = "Modified description"
				break // Just modify the first one
			}
		}

		assert.False(t, original.Equals(copied), "original should NOT equal copy after modifying schema")
	})

	t.Run("OAS3 add extension field", func(t *testing.T) {
		original, err := ParseWithOptions(WithFilePath("../testdata/petstore-3.0.yaml"))
		require.NoError(t, err)

		copied := original.Copy()

		// Add an extension to the copy (Extra field holds x- extensions)
		doc, ok := copied.OAS3Document()
		require.True(t, ok, "expected OAS3 document")
		if doc.Extra == nil {
			doc.Extra = make(map[string]any)
		}
		doc.Extra["x-custom-field"] = "custom value"

		assert.False(t, original.Equals(copied), "original should NOT equal copy after adding extension")
	})

	t.Run("OAS2 modified Info.Title", func(t *testing.T) {
		original, err := ParseWithOptions(WithFilePath("../testdata/petstore-2.0.yaml"))
		require.NoError(t, err)

		copied := original.Copy()

		// Modify the copy's document title
		doc, ok := copied.OAS2Document()
		require.True(t, ok, "expected OAS2 document")
		doc.Info.Title = "Modified Title"

		assert.False(t, original.Equals(copied), "original should NOT equal copy after modifying Info.Title")
	})

	t.Run("OAS2 add path", func(t *testing.T) {
		original, err := ParseWithOptions(WithFilePath("../testdata/minimal-oas2.yaml"))
		require.NoError(t, err)

		copied := original.Copy()

		// Add a new path to the copy
		doc, ok := copied.OAS2Document()
		require.True(t, ok, "expected OAS2 document")
		if doc.Paths == nil {
			doc.Paths = make(map[string]*PathItem)
		}
		doc.Paths["/new-endpoint"] = &PathItem{
			Get: &Operation{
				Summary: "New endpoint",
			},
		}

		assert.False(t, original.Equals(copied), "original should NOT equal copy after adding path")
	})
}
