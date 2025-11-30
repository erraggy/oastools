package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// TestDiffWithOptions_FilePaths tests the functional options API with file paths
func TestDiffWithOptions_FilePaths(t *testing.T) {
	result, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		WithMode(ModeSimple),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Changes, "Expected to find changes between v1 and v2")
}

// TestDiffWithOptions_Parsed tests the functional options API with parsed results
func TestDiffWithOptions_Parsed(t *testing.T) {
	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := DiffWithOptions(
		WithSourceParsed(*source),
		WithTargetParsed(*target),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Changes)
}

// TestDiffWithOptions_MixedSources tests using file path for source and parsed for target
func TestDiffWithOptions_MixedSources(t *testing.T) {
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetParsed(*target),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestDiffWithOptions_BreakingMode tests that breaking mode is applied
func TestDiffWithOptions_BreakingMode(t *testing.T) {
	result, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		WithMode(ModeBreaking),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// In breaking mode, changes should have severity assigned
}

// TestDiffWithOptions_DisableInfo tests that info messages can be disabled
func TestDiffWithOptions_DisableInfo(t *testing.T) {
	result, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		WithIncludeInfo(false),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Info count should be 0 when disabled
	assert.Equal(t, 0, result.InfoCount)
}

// TestDiffWithOptions_DefaultValues tests that default values are applied correctly
func TestDiffWithOptions_DefaultValues(t *testing.T) {
	result, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		// Not specifying WithMode or WithIncludeInfo to test defaults
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestDiffWithOptions_NoSource tests error when no source is specified
func TestDiffWithOptions_NoSource(t *testing.T) {
	_, err := DiffWithOptions(
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		WithMode(ModeSimple),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify a source")
}

// TestDiffWithOptions_NoTarget tests error when no target is specified
func TestDiffWithOptions_NoTarget(t *testing.T) {
	_, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithMode(ModeSimple),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify a target")
}

// TestDiffWithOptions_MultipleSources tests error when multiple sources are specified
func TestDiffWithOptions_MultipleSources(t *testing.T) {
	source, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	_, err = DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithSourceParsed(*source),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one source")
}

// TestDiffWithOptions_MultipleTargets tests error when multiple targets are specified
func TestDiffWithOptions_MultipleTargets(t *testing.T) {
	target, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	_, err = DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		WithTargetParsed(*target),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one target")
}

// TestDiffWithOptions_AllOptions tests using all options together
func TestDiffWithOptions_AllOptions(t *testing.T) {
	result, err := DiffWithOptions(
		WithSourceFilePath("../testdata/petstore-v1.yaml"),
		WithTargetFilePath("../testdata/petstore-v2.yaml"),
		WithMode(ModeBreaking),
		WithIncludeInfo(false),
		WithUserAgent("test-differ/1.0"),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.InfoCount)
}

// TestWithSourceFilePath tests the WithSourceFilePath option function
func TestWithSourceFilePath(t *testing.T) {
	cfg := &diffConfig{}
	opt := WithSourceFilePath("source.yaml")
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.sourceFilePath)
	assert.Equal(t, "source.yaml", *cfg.sourceFilePath)
}

// TestWithSourceParsed tests the WithSourceParsed option function
func TestWithSourceParsed(t *testing.T) {
	parseResult := parser.ParseResult{Version: "3.0.0"}
	cfg := &diffConfig{}
	opt := WithSourceParsed(parseResult)
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.sourceParsed)
	assert.Equal(t, "3.0.0", cfg.sourceParsed.Version)
}

// TestWithTargetFilePath tests the WithTargetFilePath option function
func TestWithTargetFilePath(t *testing.T) {
	cfg := &diffConfig{}
	opt := WithTargetFilePath("target.yaml")
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.targetFilePath)
	assert.Equal(t, "target.yaml", *cfg.targetFilePath)
}

// TestWithTargetParsed tests the WithTargetParsed option function
func TestWithTargetParsed(t *testing.T) {
	parseResult := parser.ParseResult{Version: "3.1.0"}
	cfg := &diffConfig{}
	opt := WithTargetParsed(parseResult)
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.targetParsed)
	assert.Equal(t, "3.1.0", cfg.targetParsed.Version)
}

// TestWithMode tests the WithMode option function
func TestWithMode(t *testing.T) {
	tests := []struct {
		name string
		mode DiffMode
	}{
		{"simple", ModeSimple},
		{"breaking", ModeBreaking},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &diffConfig{}
			opt := WithMode(tt.mode)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.mode, cfg.mode)
		})
	}
}

// TestWithIncludeInfo_Differ tests the WithIncludeInfo option function
func TestWithIncludeInfo_Differ(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &diffConfig{}
			opt := WithIncludeInfo(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.includeInfo)
		})
	}
}

// TestWithUserAgent_Differ tests the WithUserAgent option function
func TestWithUserAgent_Differ(t *testing.T) {
	cfg := &diffConfig{}
	opt := WithUserAgent("custom-agent/2.0")
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, "custom-agent/2.0", cfg.userAgent)
}

// TestApplyOptions_Defaults_Differ tests that default values are set correctly
func TestApplyOptions_Defaults_Differ(t *testing.T) {
	cfg, err := applyOptions(
		WithSourceFilePath("source.yaml"),
		WithTargetFilePath("target.yaml"),
	)

	require.NoError(t, err)
	assert.Equal(t, ModeSimple, cfg.mode, "default mode should be ModeSimple")
	assert.True(t, cfg.includeInfo, "default includeInfo should be true")
	assert.Equal(t, "", cfg.userAgent, "default userAgent should be empty")
}

// TestApplyOptions_OverrideDefaults_Differ tests that options override defaults
func TestApplyOptions_OverrideDefaults_Differ(t *testing.T) {
	cfg, err := applyOptions(
		WithSourceFilePath("source.yaml"),
		WithTargetFilePath("target.yaml"),
		WithMode(ModeBreaking),
		WithIncludeInfo(false),
		WithUserAgent("custom/1.0"),
	)

	require.NoError(t, err)
	assert.Equal(t, ModeBreaking, cfg.mode)
	assert.False(t, cfg.includeInfo)
	assert.Equal(t, "custom/1.0", cfg.userAgent)
}
