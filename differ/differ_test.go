package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestDifferConvenience(t *testing.T) {
	// Test the convenience function Diff
	result, err := Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Changes) == 0 {
		t.Error("Expected changes between v1 and v2")
	}
}

func TestDifferParsedConvenience(t *testing.T) {
	// Test the convenience function DiffParsed
	source, err := parser.Parse("../testdata/petstore-v1.yaml", false, true)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	target, err := parser.Parse("../testdata/petstore-v2.yaml", false, true)
	if err != nil {
		t.Fatalf("Failed to parse target: %v", err)
	}

	result, err := DiffParsed(*source, *target)
	if err != nil {
		t.Fatalf("DiffParsed failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Changes) == 0 {
		t.Error("Expected changes between v1 and v2")
	}
}

func TestDifferNew(t *testing.T) {
	d := New()
	if d == nil {
		t.Fatal("Expected non-nil Differ")
	}

	if d.Mode != ModeSimple {
		t.Errorf("Expected default mode to be ModeSimple, got %d", d.Mode)
	}

	if !d.IncludeInfo {
		t.Error("Expected IncludeInfo to be true by default")
	}
}

func TestDifferDiff(t *testing.T) {
	d := New()
	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if result.SourceVersion != "3.0.3" {
		t.Errorf("Expected source version 3.0.3, got %s", result.SourceVersion)
	}

	if result.TargetVersion != "3.0.3" {
		t.Errorf("Expected target version 3.0.3, got %s", result.TargetVersion)
	}

	if len(result.Changes) == 0 {
		t.Error("Expected changes between v1 and v2")
	}
}

func TestDifferDiffInvalidSource(t *testing.T) {
	d := New()
	_, err := d.Diff("nonexistent.yaml", "../testdata/petstore-v2.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent source file")
	}
}

func TestDifferDiffInvalidTarget(t *testing.T) {
	d := New()
	_, err := d.Diff("../testdata/petstore-v1.yaml", "nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent target file")
	}
}

func TestDifferSimpleMode(t *testing.T) {
	d := New()
	d.Mode = ModeSimple

	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// In simple mode, changes don't have severity set meaningfully
	// Just verify we get changes
	if len(result.Changes) == 0 {
		t.Error("Expected changes in simple mode")
	}
}

func TestDifferBreakingMode(t *testing.T) {
	d := New()
	d.Mode = ModeBreaking

	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Verify we categorized changes by severity
	hasInfo := false

	for _, change := range result.Changes {
		switch change.Severity {
		case SeverityInfo:
			hasInfo = true
		case SeverityWarning:
			// Warning changes may exist
		case SeverityError, SeverityCritical:
			// Breaking changes may exist
		}
	}

	if !hasInfo {
		t.Error("Expected at least one info-level change")
	}

	// Counts should match
	if result.InfoCount == 0 {
		t.Error("Expected InfoCount > 0")
	}

	if result.BreakingCount+result.WarningCount+result.InfoCount != len(result.Changes) {
		t.Error("Change counts don't add up")
	}
}

func TestDifferIncludeInfo(t *testing.T) {
	d := New()
	d.Mode = ModeBreaking
	d.IncludeInfo = false

	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Verify no info-level changes in result
	for _, change := range result.Changes {
		if change.Severity == SeverityInfo {
			t.Error("Expected no info-level changes when IncludeInfo=false")
		}
	}
}

func TestDifferIdenticalSpecs(t *testing.T) {
	d := New()
	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v1.yaml")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Changes) != 0 {
		t.Errorf("Expected no changes for identical specs, got %d", len(result.Changes))
	}

	if result.HasBreakingChanges {
		t.Error("Expected no breaking changes for identical specs")
	}
}

func TestChangeString(t *testing.T) {
	tests := []struct {
		name     string
		change   Change
		expected string
	}{
		{
			name: "Critical change",
			change: Change{
				Path:     "paths./pets.get",
				Type:     ChangeTypeRemoved,
				Category: CategoryOperation,
				Severity: SeverityCritical,
				Message:  "operation removed",
			},
			expected: "✗ paths./pets.get [removed] operation: operation removed",
		},
		{
			name: "Warning change",
			change: Change{
				Path:     "paths./pets.get.deprecated",
				Type:     ChangeTypeModified,
				Category: CategoryOperation,
				Severity: SeverityWarning,
				Message:  "operation marked as deprecated",
			},
			expected: "⚠ paths./pets.get.deprecated [modified] operation: operation marked as deprecated",
		},
		{
			name: "Info change",
			change: Change{
				Path:     "paths./pets.post",
				Type:     ChangeTypeAdded,
				Category: CategoryOperation,
				Severity: SeverityInfo,
				Message:  "operation added",
			},
			expected: "ℹ paths./pets.post [added] operation: operation added",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.change.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDiffResultHasBreakingChanges(t *testing.T) {
	result := &DiffResult{
		Changes: []Change{
			{Severity: SeverityInfo},
			{Severity: SeverityWarning},
		},
		BreakingCount: 0,
	}

	if result.HasBreakingChanges {
		t.Error("Expected HasBreakingChanges=false when no breaking changes")
	}

	result.BreakingCount = 1
	result.HasBreakingChanges = true

	if !result.HasBreakingChanges {
		t.Error("Expected HasBreakingChanges=true when breaking changes exist")
	}
}

func TestDifferModes(t *testing.T) {
	tests := []struct {
		name string
		mode DiffMode
	}{
		{"Simple mode", ModeSimple},
		{"Breaking mode", ModeBreaking},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode

			result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
			if err != nil {
				t.Fatalf("Diff failed: %v", err)
			}

			if len(result.Changes) == 0 {
				t.Error("Expected changes")
			}
		})
	}
}

func TestDifferUserAgent(t *testing.T) {
	d := New()
	d.UserAgent = "test-agent/1.0"

	// This should work even with custom UserAgent
	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	if err != nil {
		t.Fatalf("Diff failed with custom UserAgent: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}
