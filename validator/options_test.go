package validator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithFilePath_Validator tests the WithFilePath option function
func TestWithFilePath_Validator(t *testing.T) {
	cfg := &validateConfig{}
	opt := WithFilePath("test.yaml")
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.filePath)
	assert.Equal(t, "test.yaml", *cfg.filePath)
}

// TestWithParsed tests the WithParsed option function
func TestWithParsed(t *testing.T) {
	parseResult := parser.ParseResult{Version: "3.0.0"}
	cfg := &validateConfig{}
	opt := WithParsed(parseResult)
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.parsed)
	assert.Equal(t, "3.0.0", cfg.parsed.Version)
}

// TestWithIncludeWarnings tests the WithIncludeWarnings option function
func TestWithIncludeWarnings(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &validateConfig{}
			opt := WithIncludeWarnings(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.includeWarnings)
		})
	}
}

// TestWithStrictMode tests the WithStrictMode option function
func TestWithStrictMode(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &validateConfig{}
			opt := WithStrictMode(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.strictMode)
		})
	}
}

// TestWithUserAgent_Validator tests the WithUserAgent option function
func TestWithUserAgent_Validator(t *testing.T) {
	cfg := &validateConfig{}
	opt := WithUserAgent("custom-agent/2.0")
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, "custom-agent/2.0", cfg.userAgent)
}

// TestApplyOptions_Defaults_Validator tests that default values are set correctly
func TestApplyOptions_Defaults_Validator(t *testing.T) {
	cfg, err := applyOptions(WithFilePath("test.yaml"))

	require.NoError(t, err)
	assert.True(t, cfg.includeWarnings, "default includeWarnings should be true")
	assert.False(t, cfg.strictMode, "default strictMode should be false")
	assert.True(t, cfg.validateStructure, "default validateStructure should be true")
	assert.Equal(t, "", cfg.userAgent, "default userAgent should be empty")
}

// TestApplyOptions_OverrideDefaults_Validator tests that options override defaults
func TestApplyOptions_OverrideDefaults_Validator(t *testing.T) {
	cfg, err := applyOptions(
		WithFilePath("test.yaml"),
		WithIncludeWarnings(false),
		WithStrictMode(true),
		WithValidateStructure(false),
		WithUserAgent("custom/1.0"),
	)

	require.NoError(t, err)
	assert.False(t, cfg.includeWarnings)
	assert.True(t, cfg.strictMode)
	assert.False(t, cfg.validateStructure)
	assert.Equal(t, "custom/1.0", cfg.userAgent)
}

// TestWithSourceMap tests the WithSourceMap option function
func TestWithSourceMap(t *testing.T) {
	sm := parser.NewSourceMap()
	cfg := &validateConfig{}
	opt := WithSourceMap(sm)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, sm, cfg.sourceMap)
}

// TestWithSourceMap_Nil tests WithSourceMap with nil value
func TestWithSourceMap_Nil(t *testing.T) {
	cfg := &validateConfig{}
	opt := WithSourceMap(nil)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Nil(t, cfg.sourceMap)
}

// TestValidateWithOptions_FilePath tests the functional options API with file path
func TestValidateWithOptions_FilePath(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithIncludeWarnings(true),
		WithStrictMode(false),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "3.0.3", result.Version)
}

// TestValidateWithOptions_Parsed tests the functional options API with parsed result
func TestValidateWithOptions_Parsed(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := ValidateWithOptions(
		WithParsed(*parseResult),
		WithIncludeWarnings(true),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "3.0.3", result.Version)
}

// TestValidateWithOptions_StrictMode tests that strict mode is applied
func TestValidateWithOptions_StrictMode(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithStrictMode(true),
		WithIncludeWarnings(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Strict mode may generate additional warnings
}

// TestValidateWithOptions_ValidateStructure tests that structure validation can be controlled
func TestValidateWithOptions_ValidateStructure(t *testing.T) {
	t.Run("default enabled", func(t *testing.T) {
		result, err := ValidateWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
			// Not specifying WithValidateStructure to test default (true)
		)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("explicitly enabled", func(t *testing.T) {
		result, err := ValidateWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
			WithValidateStructure(true),
		)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		result, err := ValidateWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
			WithValidateStructure(false),
		)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		// With a valid file, both enabled and disabled should pass
	})
}

// TestValidateWithOptions_DisableWarnings tests that warnings can be disabled
func TestValidateWithOptions_DisableWarnings(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithIncludeWarnings(false),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Warnings, "warnings should be filtered out when IncludeWarnings=false")
	assert.Equal(t, 0, result.WarningCount)
}

// TestValidateWithOptions_DefaultValues tests that default values are applied correctly
func TestValidateWithOptions_DefaultValues(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		// Not specifying WithIncludeWarnings or WithStrictMode to test defaults
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	// Default: IncludeWarnings = true, so warnings may be present
	// (though petstore might not have warnings)
}

// TestValidateWithOptions_NoInputSource tests error when no input source is specified
func TestValidateWithOptions_NoInputSource(t *testing.T) {
	_, err := ValidateWithOptions(
		WithIncludeWarnings(true),
		WithStrictMode(false),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify an input source")
}

// TestValidateWithOptions_MultipleInputSources tests error when multiple input sources are specified
func TestValidateWithOptions_MultipleInputSources(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	_, err = ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithParsed(*parseResult),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one input source")
}

// TestValidateWithOptions_AllOptions tests using all options together
func TestValidateWithOptions_AllOptions(t *testing.T) {
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithIncludeWarnings(false),
		WithStrictMode(true),
		WithUserAgent("test-validator/1.0"),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Warnings)
}

// TestValidateWithSourceMap tests validation with source map integration
func TestValidateWithSourceMap(t *testing.T) {
	// Parse with source map
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/invalid-oas3.yaml"),
		parser.WithSourceMap(true),
	)
	require.NoError(t, err)
	require.NotNil(t, parseResult.SourceMap)

	// Validate with source map
	result, err := ValidateWithOptions(
		WithParsed(*parseResult),
		WithSourceMap(parseResult.SourceMap),
	)
	require.NoError(t, err)

	// Check that errors have line numbers
	require.NotEmpty(t, result.Errors)
	hasLocationCount := 0
	for _, e := range result.Errors {
		if e.HasLocation() {
			hasLocationCount++
			assert.Greater(t, e.Line, 0)
		}
	}
	// Some errors should have locations (may not be all depending on path matching)
	t.Logf("Validation errors with location: %d out of %d", hasLocationCount, len(result.Errors))
}

// TestValidateWithoutSourceMap tests validation without source map (default behavior)
func TestValidateWithoutSourceMap(t *testing.T) {
	// Validate without source map (default behavior)
	result, err := ValidateWithOptions(
		WithFilePath("../testdata/invalid-oas3.yaml"),
	)
	require.NoError(t, err)

	// Errors should NOT have line numbers
	require.NotEmpty(t, result.Errors)
	for _, e := range result.Errors {
		assert.False(t, e.HasLocation(), "Error should not have location when SourceMap not provided: %s", e.Path)
	}
}

// TestSourceMapIntegrationWithValidDocument tests source map with a valid document
func TestSourceMapIntegrationWithValidDocument(t *testing.T) {
	// Parse a valid document with source map
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithSourceMap(true),
	)
	require.NoError(t, err)
	require.NotNil(t, parseResult.SourceMap)

	// Validate with source map
	result, err := ValidateWithOptions(
		WithParsed(*parseResult),
		WithSourceMap(parseResult.SourceMap),
	)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}
