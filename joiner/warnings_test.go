package joiner

import (
	"testing"

	"github.com/erraggy/oastools/internal/severity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinWarning_String(t *testing.T) {
	w := &JoinWarning{
		Category: WarnPathCollision,
		Message:  "path '/users' overwritten",
	}

	got := w.String()
	want := "path '/users' overwritten"

	assert.Equal(t, want, got)
}

func TestJoinWarning_HasLocation(t *testing.T) {
	tests := []struct {
		name string
		line int
		want bool
	}{
		{"with line", 10, true},
		{"zero line", 0, false},
		{"negative line", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JoinWarning{Line: tt.line}
			assert.Equal(t, tt.want, w.HasLocation())
		})
	}
}

func TestJoinWarning_Location(t *testing.T) {
	tests := []struct {
		name       string
		sourceFile string
		path       string
		line       int
		column     int
		want       string
	}{
		{
			name:       "file with line and column",
			sourceFile: "api.yaml",
			line:       42,
			column:     5,
			want:       "api.yaml:42:5",
		},
		{
			name:       "file with line only",
			sourceFile: "api.yaml",
			line:       42,
			want:       "api.yaml:42",
		},
		{
			name:   "line and column only",
			line:   42,
			column: 5,
			want:   "42:5",
		},
		{
			name: "line only",
			line: 42,
			want: "42",
		},
		{
			name: "path fallback",
			path: "paths./users.get",
			want: "paths./users.get",
		},
		{
			name:       "file fallback",
			sourceFile: "api.yaml",
			want:       "api.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JoinWarning{
				SourceFile: tt.sourceFile,
				Path:       tt.path,
				Line:       tt.line,
				Column:     tt.column,
			}
			assert.Equal(t, tt.want, w.Location())
		})
	}
}

func TestNewPathCollisionWarning(t *testing.T) {
	w := NewPathCollisionWarning("/users", "overwritten", "base.yaml", "second.yaml", 15, 3)

	assert.Equal(t, WarnPathCollision, w.Category)
	assert.Equal(t, "paths./users", w.Path)
	assert.Equal(t, "second.yaml", w.SourceFile)
	assert.Equal(t, 15, w.Line)
	assert.Equal(t, 3, w.Column)
	assert.Equal(t, severity.SeverityWarning, w.Severity)
	assert.Equal(t, "overwritten", w.Context["resolution"])
}

func TestNewWebhookCollisionWarning(t *testing.T) {
	w := NewWebhookCollisionWarning("orderCreated", "kept from first", "base.yaml", "webhooks.yaml", 25, 0)

	assert.Equal(t, WarnWebhookCollision, w.Category)
	assert.Equal(t, "webhooks.orderCreated", w.Path)
	assert.True(t, w.HasLocation())
}

func TestNewSchemaCollisionWarning(t *testing.T) {
	w := NewSchemaCollisionWarning("Pet", "overwritten", "components.schemas", "base.yaml", "pets.yaml", 100, 10)

	assert.Equal(t, WarnSchemaCollision, w.Category)
	assert.Equal(t, "components.schemas.Pet", w.Path)
	assert.Equal(t, "components.schemas", w.Context["section"])
}

func TestNewSchemaRenamedWarning(t *testing.T) {
	tests := []struct {
		name         string
		keptOriginal bool
		wantContains string
	}{
		{
			name:         "kept original",
			keptOriginal: true,
			wantContains: "kept from first document",
		},
		{
			name:         "renamed source",
			keptOriginal: false,
			wantContains: "renamed to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewSchemaRenamedWarning("Pet", "PetV2", "schemas", "pets.yaml", 50, 0, tt.keptOriginal)

			assert.Equal(t, WarnSchemaRenamed, w.Category)
			assert.Equal(t, severity.SeverityInfo, w.Severity)
			assert.Equal(t, tt.keptOriginal, w.Context["kept_original"])
		})
	}
}

func TestNewSchemaDedupWarning(t *testing.T) {
	w := NewSchemaDedupWarning("User", "definitions", "users.yaml", 30, 5)

	assert.Equal(t, WarnSchemaDeduplicated, w.Category)
	assert.Equal(t, severity.SeverityInfo, w.Severity)
	assert.Equal(t, "users.yaml:30:5", w.Location())
}

func TestNewNamespacePrefixWarning(t *testing.T) {
	w := NewNamespacePrefixWarning("Pet", "Auth_Pet", "schemas", "auth.yaml", 0, 0)

	assert.Equal(t, WarnNamespacePrefixed, w.Category)
	assert.Equal(t, "Pet", w.Context["original_name"])
	assert.Equal(t, "Auth_Pet", w.Context["new_name"])
}

func TestNewVersionMismatchWarning(t *testing.T) {
	w := NewVersionMismatchWarning("v1.yaml", "3.0.0", "v2.yaml", "3.0.3", "3.0.3")

	assert.Equal(t, WarnVersionMismatch, w.Category)
	assert.Equal(t, severity.SeverityWarning, w.Severity)
	assert.Equal(t, "3.0.3", w.Context["target_version"])
}

func TestNewMetadataOverrideWarning(t *testing.T) {
	w := NewMetadataOverrideWarning("host", "api.example.com", "other.example.com", "second.yaml")

	assert.Equal(t, WarnMetadataOverride, w.Category)
	assert.Equal(t, "host", w.Path)
	assert.Equal(t, "api.example.com", w.Context["first_value"])
}

func TestNewSemanticDedupSummaryWarning(t *testing.T) {
	w := NewSemanticDedupSummaryWarning(5, "schema")

	assert.Equal(t, WarnSemanticDedup, w.Category)
	assert.Equal(t, 5, w.Context["count"])
	assert.Equal(t, "schema", w.Context["section"])
}

func TestJoinWarnings_Strings(t *testing.T) {
	warnings := JoinWarnings{
		{Message: "warning 1"},
		{Message: "warning 2"},
		{Message: "warning 3"},
	}

	got := warnings.Strings()

	require.Len(t, got, 3)
	assert.Equal(t, "warning 1", got[0])
	assert.Equal(t, "warning 2", got[1])
	assert.Equal(t, "warning 3", got[2])
}

func TestJoinWarnings_ByCategory(t *testing.T) {
	warnings := JoinWarnings{
		{Category: WarnPathCollision, Message: "path 1"},
		{Category: WarnSchemaCollision, Message: "schema 1"},
		{Category: WarnPathCollision, Message: "path 2"},
	}

	pathWarnings := warnings.ByCategory(WarnPathCollision)

	require.Len(t, pathWarnings, 2)
	for _, w := range pathWarnings {
		assert.Equal(t, WarnPathCollision, w.Category, "unexpected category in filtered results")
	}
}

func TestJoinWarnings_BySeverity(t *testing.T) {
	warnings := JoinWarnings{
		{Severity: severity.SeverityWarning, Message: "warning 1"},
		{Severity: severity.SeverityInfo, Message: "info 1"},
		{Severity: severity.SeverityWarning, Message: "warning 2"},
	}

	infoWarnings := warnings.BySeverity(severity.SeverityInfo)

	require.Len(t, infoWarnings, 1)
	assert.Equal(t, "info 1", infoWarnings[0].Message)
}

func TestJoinWarnings_Summary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var warnings JoinWarnings
		assert.Equal(t, "", warnings.Summary())
	})

	t.Run("with warnings", func(t *testing.T) {
		warnings := JoinWarnings{
			{Message: "first warning"},
			{Message: "second warning"},
		}

		got := warnings.Summary()

		assert.NotEqual(t, "", got, "Summary() returned empty string for non-empty warnings")
		// Check it contains expected parts
		assert.Contains(t, got, "2 warning(s)")
		assert.Contains(t, got, "first warning")
		assert.Contains(t, got, "second warning")
	})
}

func TestJoinResult_WarningStrings(t *testing.T) {
	t.Run("uses StructuredWarnings when present", func(t *testing.T) {
		result := &JoinResult{
			StructuredWarnings: JoinWarnings{
				{Message: "structured warning 1"},
				{Message: "structured warning 2"},
			},
			Warnings: []string{"legacy warning"},
		}

		got := result.WarningStrings()

		require.Len(t, got, 2)
		assert.Equal(t, "structured warning 1", got[0])
	})

	t.Run("falls back to Warnings when StructuredWarnings empty", func(t *testing.T) {
		result := &JoinResult{
			Warnings: []string{"legacy warning 1", "legacy warning 2"},
		}

		got := result.WarningStrings()

		require.Len(t, got, 2)
		assert.Equal(t, "legacy warning 1", got[0])
	})
}

func TestJoinWarnings_Strings_WithNil(t *testing.T) {
	warnings := JoinWarnings{
		{Message: "first"},
		nil,
		{Message: "third"},
	}

	got := warnings.Strings()

	require.Len(t, got, 3)
	assert.Equal(t, "first", got[0])
	assert.Equal(t, "", got[1], "Expected empty string for nil")
	assert.Equal(t, "third", got[2])
}

func TestIsGenericSourceName(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath string
		want       bool
	}{
		// Generic names (should return true)
		{"empty string", "", true},
		{"ParseBytes.yaml", "ParseBytes.yaml", true},
		{"ParseBytes.json", "ParseBytes.json", true},
		{"ParseReader.yaml", "ParseReader.yaml", true},
		{"ParseReader.json", "ParseReader.json", true},

		// Meaningful names (should return false)
		{"real file path", "api.yaml", false},
		{"service name", "users-api", false},
		{"path with directory", "/path/to/spec.yaml", false},
		{"URL-like", "https://example.com/api.yaml", false},
		{"service identifier", "billing-service-v2", false},
		{"contains ParseBytes but not prefix", "my-ParseBytes.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsGenericSourceName(tt.sourcePath))
		})
	}
}

func TestNewGenericSourceNameWarning(t *testing.T) {
	t.Run("empty source path", func(t *testing.T) {
		w := NewGenericSourceNameWarning("", 0)

		assert.Equal(t, WarnGenericSourceName, w.Category)
		assert.Equal(t, severity.SeverityInfo, w.Severity)
		assert.Contains(t, w.Message, "empty source name")
		assert.Contains(t, w.Message, "ParseResult.SourcePath")
		assert.Equal(t, 0, w.Context["doc_index"])
	})

	t.Run("generic source path", func(t *testing.T) {
		w := NewGenericSourceNameWarning("ParseBytes.yaml", 5)

		assert.Equal(t, WarnGenericSourceName, w.Category)
		assert.Contains(t, w.Message, "generic source name")
		assert.Contains(t, w.Message, "ParseBytes.yaml")
		assert.Equal(t, 5, w.Context["doc_index"])
		assert.Equal(t, "ParseBytes.yaml", w.Context["source_path"])
		assert.Equal(t, "ParseBytes.yaml", w.SourceFile)
	})
}

func TestHandlerWarningCategories(t *testing.T) {
	// Test WarnHandlerError
	assert.Equal(t, WarningCategory("handler_error"), WarnHandlerError)

	// Test WarnHandlerResolution
	assert.Equal(t, WarningCategory("handler_resolution"), WarnHandlerResolution)
}

func TestNewHandlerErrorWarning(t *testing.T) {
	warn := NewHandlerErrorWarning("$.components.schemas.User", "handler failed: timeout", "overlay.yaml", 42, 5)

	assert.Equal(t, WarnHandlerError, warn.Category)
	assert.Equal(t, "$.components.schemas.User", warn.Path)
	assert.Equal(t, "handler failed: timeout", warn.Message)
	assert.Equal(t, "overlay.yaml", warn.SourceFile)
	assert.Equal(t, 42, warn.Line)
	assert.Equal(t, 5, warn.Column)
	assert.Equal(t, severity.SeverityWarning, warn.Severity)
}

func TestNewHandlerResolutionWarning(t *testing.T) {
	warn := NewHandlerResolutionWarning("$.components.schemas.User", "custom merge applied", "overlay.yaml", 42, 5)

	assert.Equal(t, WarnHandlerResolution, warn.Category)
	assert.Equal(t, "$.components.schemas.User", warn.Path)
	assert.Equal(t, "custom merge applied", warn.Message)
	assert.Equal(t, "overlay.yaml", warn.SourceFile)
	assert.Equal(t, 42, warn.Line)
	assert.Equal(t, 5, warn.Column)
	assert.Equal(t, severity.SeverityInfo, warn.Severity)
}
