package joiner

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/severity"
)

func TestJoinWarning_String(t *testing.T) {
	w := &JoinWarning{
		Category: WarnPathCollision,
		Message:  "path '/users' overwritten",
	}

	got := w.String()
	want := "path '/users' overwritten"

	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
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
			if got := w.HasLocation(); got != tt.want {
				t.Errorf("HasLocation() = %v, want %v", got, tt.want)
			}
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
			if got := w.Location(); got != tt.want {
				t.Errorf("Location() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewPathCollisionWarning(t *testing.T) {
	w := NewPathCollisionWarning("/users", "overwritten", "base.yaml", "second.yaml", 15, 3)

	if w.Category != WarnPathCollision {
		t.Errorf("Category = %v, want %v", w.Category, WarnPathCollision)
	}
	if w.Path != "paths./users" {
		t.Errorf("Path = %q, want %q", w.Path, "paths./users")
	}
	if w.SourceFile != "second.yaml" {
		t.Errorf("SourceFile = %q, want %q", w.SourceFile, "second.yaml")
	}
	if w.Line != 15 || w.Column != 3 {
		t.Errorf("Line:Column = %d:%d, want 15:3", w.Line, w.Column)
	}
	if w.Severity != severity.SeverityWarning {
		t.Errorf("Severity = %v, want %v", w.Severity, severity.SeverityWarning)
	}
	if w.Context["resolution"] != "overwritten" {
		t.Errorf("Context[resolution] = %v, want %q", w.Context["resolution"], "overwritten")
	}
}

func TestNewWebhookCollisionWarning(t *testing.T) {
	w := NewWebhookCollisionWarning("orderCreated", "kept from first", "base.yaml", "webhooks.yaml", 25, 0)

	if w.Category != WarnWebhookCollision {
		t.Errorf("Category = %v, want %v", w.Category, WarnWebhookCollision)
	}
	if w.Path != "webhooks.orderCreated" {
		t.Errorf("Path = %q, want %q", w.Path, "webhooks.orderCreated")
	}
	if !w.HasLocation() {
		t.Error("HasLocation() = false, want true")
	}
}

func TestNewSchemaCollisionWarning(t *testing.T) {
	w := NewSchemaCollisionWarning("Pet", "overwritten", "components.schemas", "base.yaml", "pets.yaml", 100, 10)

	if w.Category != WarnSchemaCollision {
		t.Errorf("Category = %v, want %v", w.Category, WarnSchemaCollision)
	}
	if w.Path != "components.schemas.Pet" {
		t.Errorf("Path = %q, want %q", w.Path, "components.schemas.Pet")
	}
	if w.Context["section"] != "components.schemas" {
		t.Errorf("Context[section] = %v, want %q", w.Context["section"], "components.schemas")
	}
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

			if w.Category != WarnSchemaRenamed {
				t.Errorf("Category = %v, want %v", w.Category, WarnSchemaRenamed)
			}
			if w.Severity != severity.SeverityInfo {
				t.Errorf("Severity = %v, want %v", w.Severity, severity.SeverityInfo)
			}
			if w.Context["kept_original"] != tt.keptOriginal {
				t.Errorf("Context[kept_original] = %v, want %v", w.Context["kept_original"], tt.keptOriginal)
			}
		})
	}
}

func TestNewSchemaDedupWarning(t *testing.T) {
	w := NewSchemaDedupWarning("User", "definitions", "users.yaml", 30, 5)

	if w.Category != WarnSchemaDeduplicated {
		t.Errorf("Category = %v, want %v", w.Category, WarnSchemaDeduplicated)
	}
	if w.Severity != severity.SeverityInfo {
		t.Errorf("Severity = %v, want %v", w.Severity, severity.SeverityInfo)
	}
	if w.Location() != "users.yaml:30:5" {
		t.Errorf("Location() = %q, want %q", w.Location(), "users.yaml:30:5")
	}
}

func TestNewNamespacePrefixWarning(t *testing.T) {
	w := NewNamespacePrefixWarning("Pet", "Auth_Pet", "schemas", "auth.yaml", 0, 0)

	if w.Category != WarnNamespacePrefixed {
		t.Errorf("Category = %v, want %v", w.Category, WarnNamespacePrefixed)
	}
	if w.Context["original_name"] != "Pet" {
		t.Errorf("Context[original_name] = %v, want %q", w.Context["original_name"], "Pet")
	}
	if w.Context["new_name"] != "Auth_Pet" {
		t.Errorf("Context[new_name] = %v, want %q", w.Context["new_name"], "Auth_Pet")
	}
}

func TestNewVersionMismatchWarning(t *testing.T) {
	w := NewVersionMismatchWarning("v1.yaml", "3.0.0", "v2.yaml", "3.0.3", "3.0.3")

	if w.Category != WarnVersionMismatch {
		t.Errorf("Category = %v, want %v", w.Category, WarnVersionMismatch)
	}
	if w.Severity != severity.SeverityWarning {
		t.Errorf("Severity = %v, want %v", w.Severity, severity.SeverityWarning)
	}
	if w.Context["target_version"] != "3.0.3" {
		t.Errorf("Context[target_version] = %v, want %q", w.Context["target_version"], "3.0.3")
	}
}

func TestNewMetadataOverrideWarning(t *testing.T) {
	w := NewMetadataOverrideWarning("host", "api.example.com", "other.example.com", "second.yaml")

	if w.Category != WarnMetadataOverride {
		t.Errorf("Category = %v, want %v", w.Category, WarnMetadataOverride)
	}
	if w.Path != "host" {
		t.Errorf("Path = %q, want %q", w.Path, "host")
	}
	if w.Context["first_value"] != "api.example.com" {
		t.Errorf("Context[first_value] = %v, want %q", w.Context["first_value"], "api.example.com")
	}
}

func TestNewSemanticDedupSummaryWarning(t *testing.T) {
	w := NewSemanticDedupSummaryWarning(5, "schema")

	if w.Category != WarnSemanticDedup {
		t.Errorf("Category = %v, want %v", w.Category, WarnSemanticDedup)
	}
	if w.Context["count"] != 5 {
		t.Errorf("Context[count] = %v, want 5", w.Context["count"])
	}
	if w.Context["section"] != "schema" {
		t.Errorf("Context[section] = %v, want %q", w.Context["section"], "schema")
	}
}

func TestJoinWarnings_Strings(t *testing.T) {
	warnings := JoinWarnings{
		{Message: "warning 1"},
		{Message: "warning 2"},
		{Message: "warning 3"},
	}

	got := warnings.Strings()

	if len(got) != 3 {
		t.Fatalf("len(Strings()) = %d, want 3", len(got))
	}
	if got[0] != "warning 1" || got[1] != "warning 2" || got[2] != "warning 3" {
		t.Errorf("Strings() = %v, want [warning 1, warning 2, warning 3]", got)
	}
}

func TestJoinWarnings_ByCategory(t *testing.T) {
	warnings := JoinWarnings{
		{Category: WarnPathCollision, Message: "path 1"},
		{Category: WarnSchemaCollision, Message: "schema 1"},
		{Category: WarnPathCollision, Message: "path 2"},
	}

	pathWarnings := warnings.ByCategory(WarnPathCollision)

	if len(pathWarnings) != 2 {
		t.Fatalf("ByCategory(WarnPathCollision) len = %d, want 2", len(pathWarnings))
	}
	for _, w := range pathWarnings {
		if w.Category != WarnPathCollision {
			t.Errorf("unexpected category %v in filtered results", w.Category)
		}
	}
}

func TestJoinWarnings_BySeverity(t *testing.T) {
	warnings := JoinWarnings{
		{Severity: severity.SeverityWarning, Message: "warning 1"},
		{Severity: severity.SeverityInfo, Message: "info 1"},
		{Severity: severity.SeverityWarning, Message: "warning 2"},
	}

	infoWarnings := warnings.BySeverity(severity.SeverityInfo)

	if len(infoWarnings) != 1 {
		t.Fatalf("BySeverity(SeverityInfo) len = %d, want 1", len(infoWarnings))
	}
	if infoWarnings[0].Message != "info 1" {
		t.Errorf("wrong warning filtered: %q", infoWarnings[0].Message)
	}
}

func TestJoinWarnings_Summary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var warnings JoinWarnings
		if got := warnings.Summary(); got != "" {
			t.Errorf("Summary() = %q, want empty string", got)
		}
	})

	t.Run("with warnings", func(t *testing.T) {
		warnings := JoinWarnings{
			{Message: "first warning"},
			{Message: "second warning"},
		}

		got := warnings.Summary()

		if got == "" {
			t.Error("Summary() returned empty string for non-empty warnings")
		}
		// Check it contains expected parts
		if !strings.Contains(got, "2 warning(s)") {
			t.Errorf("Summary() missing count: %q", got)
		}
		if !strings.Contains(got, "first warning") || !strings.Contains(got, "second warning") {
			t.Errorf("Summary() missing warning messages: %q", got)
		}
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

		if len(got) != 2 {
			t.Fatalf("WarningStrings() len = %d, want 2", len(got))
		}
		if got[0] != "structured warning 1" {
			t.Errorf("WarningStrings()[0] = %q, want %q", got[0], "structured warning 1")
		}
	})

	t.Run("falls back to Warnings when StructuredWarnings empty", func(t *testing.T) {
		result := &JoinResult{
			Warnings: []string{"legacy warning 1", "legacy warning 2"},
		}

		got := result.WarningStrings()

		if len(got) != 2 {
			t.Fatalf("WarningStrings() len = %d, want 2", len(got))
		}
		if got[0] != "legacy warning 1" {
			t.Errorf("WarningStrings()[0] = %q, want %q", got[0], "legacy warning 1")
		}
	})
}

func TestJoinWarnings_Strings_WithNil(t *testing.T) {
	warnings := JoinWarnings{
		{Message: "first"},
		nil,
		{Message: "third"},
	}

	got := warnings.Strings()

	if len(got) != 3 {
		t.Fatalf("Strings() len = %d, want 3", len(got))
	}
	if got[0] != "first" {
		t.Errorf("Strings()[0] = %q, want %q", got[0], "first")
	}
	if got[1] != "" {
		t.Errorf("Strings()[1] = %q, want empty string for nil", got[1])
	}
	if got[2] != "third" {
		t.Errorf("Strings()[2] = %q, want %q", got[2], "third")
	}
}
