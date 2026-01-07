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

// TestChange_HasLocation tests the HasLocation helper method
func TestChange_HasLocation(t *testing.T) {
	tests := []struct {
		name     string
		change   Change
		expected bool
	}{
		{
			name:     "no location",
			change:   Change{Path: "paths./users.get"},
			expected: false,
		},
		{
			name:     "with line",
			change:   Change{Path: "paths./users.get", Line: 10},
			expected: true,
		},
		{
			name:     "with line and column",
			change:   Change{Path: "paths./users.get", Line: 10, Column: 5},
			expected: true,
		},
		{
			name:     "with file and line",
			change:   Change{Path: "paths./users.get", File: "api.yaml", Line: 10, Column: 5},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.change.HasLocation() != tt.expected {
				t.Errorf("Expected HasLocation() = %v, got %v", tt.expected, tt.change.HasLocation())
			}
		})
	}
}

// TestChange_Location tests the Location helper method
func TestChange_Location(t *testing.T) {
	tests := []struct {
		name     string
		change   Change
		expected string
	}{
		{
			name:     "no location returns path",
			change:   Change{Path: "paths./users.get"},
			expected: "paths./users.get",
		},
		{
			name:     "line and column only",
			change:   Change{Path: "paths./users.get", Line: 10, Column: 5},
			expected: "10:5",
		},
		{
			name:     "file, line and column",
			change:   Change{Path: "paths./users.get", File: "api.yaml", Line: 10, Column: 5},
			expected: "api.yaml:10:5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.change.Location()
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

// TestDiffResult_StatsPopulated tests that document statistics are correctly populated
func TestDiffResult_StatsPopulated(t *testing.T) {
	d := New()
	result, err := d.Diff("../testdata/petstore-v1.yaml", "../testdata/petstore-v2.yaml")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify source stats are populated
	assert.Greater(t, result.SourceStats.PathCount, 0, "Expected source to have paths")
	assert.Greater(t, result.SourceStats.OperationCount, 0, "Expected source to have operations")
	assert.Greater(t, result.SourceStats.SchemaCount, 0, "Expected source to have schemas")
	assert.Greater(t, result.SourceSize, int64(0), "Expected source size to be greater than 0")

	// Verify target stats are populated
	assert.Greater(t, result.TargetStats.PathCount, 0, "Expected target to have paths")
	assert.Greater(t, result.TargetStats.OperationCount, 0, "Expected target to have operations")
	assert.Greater(t, result.TargetStats.SchemaCount, 0, "Expected target to have schemas")
	assert.Greater(t, result.TargetSize, int64(0), "Expected target size to be greater than 0")

	// Verify specific values for petstore files
	assert.Equal(t, 2, result.SourceStats.PathCount, "Expected 2 paths in petstore-v1")
	assert.Equal(t, 3, result.SourceStats.OperationCount, "Expected 3 operations in petstore-v1")
	assert.Equal(t, 2, result.TargetStats.PathCount, "Expected 2 paths in petstore-v2")
	assert.Equal(t, 4, result.TargetStats.OperationCount, "Expected 4 operations in petstore-v2")
}

// TestWithSourceMap_Differ tests the WithSourceMap option function
func TestWithSourceMap_Differ(t *testing.T) {
	sm := parser.NewSourceMap()
	cfg := &diffConfig{}
	opt := WithSourceMap(sm)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, sm, cfg.sourceMap)
}

// TestWithTargetMap_Differ tests the WithTargetMap option function
func TestWithTargetMap_Differ(t *testing.T) {
	sm := parser.NewSourceMap()
	cfg := &diffConfig{}
	opt := WithTargetMap(sm)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, sm, cfg.targetMap)
}

// TestDiffer_SourceMapPassedThrough tests that source maps are passed to the Differ
func TestDiffer_SourceMapPassedThrough(t *testing.T) {
	sourceSM := parser.NewSourceMap()
	targetSM := parser.NewSourceMap()

	// Parse source document
	sourceResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v1.yaml"),
	)
	require.NoError(t, err)

	// Parse target document
	targetResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-v2.yaml"),
	)
	require.NoError(t, err)

	result, err := DiffWithOptions(
		WithSourceParsed(*sourceResult),
		WithTargetParsed(*targetResult),
		WithSourceMap(sourceSM),
		WithTargetMap(targetSM),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Verify diff was performed (result has changes)
	assert.NotEmpty(t, result.Changes)
}

// TestDiffResult_ToParseResult tests that ToParseResult returns the target document
func TestDiffResult_ToParseResult(t *testing.T) {
	// Parse source and target documents
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

	// Diff the documents
	result, err := DiffWithOptions(
		WithSourceParsed(*source),
		WithTargetParsed(*target),
		WithMode(ModeBreaking),
	)
	require.NoError(t, err)

	// Convert to ParseResult
	parseResult := result.ToParseResult()
	require.NotNil(t, parseResult)

	// Verify it's the TARGET document, not source
	assert.Equal(t, target.Version, parseResult.Version, "Should return target version")
	assert.Equal(t, target.OASVersion, parseResult.OASVersion, "Should return target OASVersion")
	assert.Equal(t, target.Document, parseResult.Document, "Should return target document")
	assert.Equal(t, target.SourcePath, parseResult.SourcePath, "Should return target source path")
	assert.Equal(t, target.SourceFormat, parseResult.SourceFormat, "Should return target source format")
	assert.Equal(t, target.Stats, parseResult.Stats, "Should return target stats")
	assert.Equal(t, target.SourceSize, parseResult.SourceSize, "Should return target size")
}

// TestDiffResult_ToParseResult_SourcePathFallback tests fallback to "differ" when no source path
func TestDiffResult_ToParseResult_SourcePathFallback(t *testing.T) {
	// Create a DiffResult with no target source path
	result := &DiffResult{
		TargetVersion:      "3.0.0",
		TargetOASVersion:   parser.OASVersion300,
		TargetSourcePath:   "", // Empty path should fall back to "differ"
		TargetSourceFormat: parser.SourceFormatYAML,
	}

	parseResult := result.ToParseResult()
	assert.Equal(t, "differ", parseResult.SourcePath, "Should fall back to 'differ' when source path is empty")
}

// TestDiffResult_ToParseResult_ChangesToWarnings tests that changes are converted to warnings
func TestDiffResult_ToParseResult_ChangesToWarnings(t *testing.T) {
	result := &DiffResult{
		TargetVersion:    "3.0.0",
		TargetOASVersion: parser.OASVersion300,
		TargetSourcePath: "api.yaml",
		TargetDocument:   &parser.OAS3Document{OpenAPI: "3.0.0"},
		Changes: []Change{
			{
				Path:     "paths./users.get",
				Severity: SeverityError,
				Message:  "operation removed",
			},
			{
				Path:     "paths./pets.post",
				Severity: SeverityInfo,
				Message:  "operation added",
			},
			{
				Path:     "paths./orders.put",
				Severity: SeverityWarning,
				Message:  "parameter deprecated",
			},
		},
	}

	parseResult := result.ToParseResult()
	require.Len(t, parseResult.Warnings, 3, "All changes should be converted to warnings")

	// Verify severity prefixes are included
	assert.Contains(t, parseResult.Warnings[0], "[error]", "Should include severity prefix")
	assert.Contains(t, parseResult.Warnings[0], "paths./users.get", "Should include path")
	assert.Contains(t, parseResult.Warnings[0], "operation removed", "Should include message")

	assert.Contains(t, parseResult.Warnings[1], "[info]", "Should include severity prefix")
	assert.Contains(t, parseResult.Warnings[1], "paths./pets.post", "Should include path")

	assert.Contains(t, parseResult.Warnings[2], "[warning]", "Should include severity prefix")
}

// TestDiffResult_ToParseResult_NoChanges tests ToParseResult with no changes
func TestDiffResult_ToParseResult_NoChanges(t *testing.T) {
	result := &DiffResult{
		TargetVersion:    "3.0.0",
		TargetOASVersion: parser.OASVersion300,
		TargetSourcePath: "api.yaml",
		TargetDocument:   &parser.OAS3Document{OpenAPI: "3.0.0"},
		Changes:          []Change{}, // No changes
	}

	parseResult := result.ToParseResult()
	assert.Empty(t, parseResult.Warnings, "Should have empty warnings when no changes")
	assert.Empty(t, parseResult.Errors, "Should always have empty errors")
}

// TestDiffResult_ToParseResult_NilDocument tests that nil TargetDocument produces a warning
func TestDiffResult_ToParseResult_NilDocument(t *testing.T) {
	result := &DiffResult{
		TargetVersion:    "3.0.0",
		TargetOASVersion: parser.OASVersion300,
		TargetSourcePath: "api.yaml",
		TargetDocument:   nil, // Nil document should produce warning
		Changes:          []Change{},
	}

	parseResult := result.ToParseResult()
	require.Len(t, parseResult.Warnings, 1, "Should have one warning for nil document")
	assert.Contains(t, parseResult.Warnings[0], "TargetDocument is nil", "Warning should mention nil document")
	assert.Contains(t, parseResult.Warnings[0], "downstream operations may fail", "Warning should mention downstream impact")
}

// TestDiffResult_ToParseResult_VersionAndStats tests version and stats population
func TestDiffResult_ToParseResult_VersionAndStats(t *testing.T) {
	expectedStats := parser.DocumentStats{
		PathCount:      5,
		OperationCount: 10,
		SchemaCount:    15,
	}

	result := &DiffResult{
		TargetVersion:      "3.1.0",
		TargetOASVersion:   parser.OASVersion310,
		TargetStats:        expectedStats,
		TargetSize:         12345,
		TargetSourcePath:   "api-v2.yaml",
		TargetSourceFormat: parser.SourceFormatJSON,
	}

	parseResult := result.ToParseResult()
	assert.Equal(t, "3.1.0", parseResult.Version)
	assert.Equal(t, parser.OASVersion310, parseResult.OASVersion)
	assert.Equal(t, expectedStats, parseResult.Stats)
	assert.Equal(t, int64(12345), parseResult.SourceSize)
	assert.Equal(t, "api-v2.yaml", parseResult.SourcePath)
	assert.Equal(t, parser.SourceFormatJSON, parseResult.SourceFormat)
}

// TestDiffResult_ToParseResult_CriticalSeverity tests that critical severity is formatted correctly
func TestDiffResult_ToParseResult_CriticalSeverity(t *testing.T) {
	result := &DiffResult{
		TargetVersion:    "3.0.0",
		TargetOASVersion: parser.OASVersion300,
		TargetSourcePath: "api.yaml",
		TargetDocument:   &parser.OAS3Document{OpenAPI: "3.0.0"},
		Changes: []Change{
			{
				Path:     "paths./critical.delete",
				Severity: SeverityCritical,
				Message:  "endpoint removed",
			},
		},
	}

	parseResult := result.ToParseResult()
	require.Len(t, parseResult.Warnings, 1)
	assert.Contains(t, parseResult.Warnings[0], "[critical]", "Should include critical severity prefix")
}

// TestDiffResult_ToParseResult_OAS2 tests ToParseResult with an OAS2 document
func TestDiffResult_ToParseResult_OAS2(t *testing.T) {
	result := &DiffResult{
		TargetVersion:      "2.0",
		TargetOASVersion:   parser.OASVersion20,
		TargetSourcePath:   "swagger.json",
		TargetSourceFormat: parser.SourceFormatJSON,
		TargetDocument:     &parser.OAS2Document{Swagger: "2.0", Info: &parser.Info{Title: "Swagger API"}},
	}

	parseResult := result.ToParseResult()

	assert.Equal(t, "2.0", parseResult.Version)
	assert.Equal(t, parser.OASVersion20, parseResult.OASVersion)
	assert.Equal(t, "swagger.json", parseResult.SourcePath)
	assert.Equal(t, parser.SourceFormatJSON, parseResult.SourceFormat)
	doc, ok := parseResult.Document.(*parser.OAS2Document)
	assert.True(t, ok, "Document should be an OAS2Document")
	assert.Equal(t, "Swagger API", doc.Info.Title)
}

// TestDiffResult_ToParseResult_Pipeline tests a realistic pipeline workflow
func TestDiffResult_ToParseResult_Pipeline(t *testing.T) {
	// Parse source and target documents
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

	// Diff the documents
	diffResult, err := DiffWithOptions(
		WithSourceParsed(*source),
		WithTargetParsed(*target),
		WithMode(ModeBreaking),
	)
	require.NoError(t, err)

	// Convert to ParseResult for pipeline continuation
	parseResult := diffResult.ToParseResult()
	require.NotNil(t, parseResult)

	// The result should be usable in downstream operations
	// For example, it should have a valid document that can be validated
	assert.NotNil(t, parseResult.Document, "Document should be available for downstream processing")
	assert.NotEmpty(t, parseResult.Version, "Version should be set")
	assert.NotEqual(t, parser.Unknown, parseResult.OASVersion, "OASVersion should be valid")

	// Verify warnings contain the changes from the diff
	if len(diffResult.Changes) > 0 {
		assert.NotEmpty(t, parseResult.Warnings, "Warnings should contain changes from diff")
	}
}
