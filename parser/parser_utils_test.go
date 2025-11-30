package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectVersion(t *testing.T) {
	parser := New()

	tests := []struct {
		name     string
		data     map[string]any
		expected string
		wantErr  bool
	}{
		{
			name:     "OAS 2.0",
			data:     map[string]any{"swagger": "2.0"},
			expected: "2.0",
			wantErr:  false,
		},
		{
			name:     "OAS 3.0.0",
			data:     map[string]any{"openapi": "3.0.0"},
			expected: "3.0.0",
			wantErr:  false,
		},
		{
			name:     "OAS 3.1.0",
			data:     map[string]any{"openapi": "3.1.0"},
			expected: "3.1.0",
			wantErr:  false,
		},
		{
			name:     "Missing version",
			data:     map[string]any{"info": "test"},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := parser.detectVersion(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.expected {
				t.Errorf("detectVersion() = %v, want %v", version, tt.expected)
			}
		})
	}
}

// TestVersionInRange tests the semantic version range checking
// This test would have caught the bug where string comparison was used
func TestVersionInRange(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		minVersion string
		maxVersion string
		expected   bool
	}{
		// Exclusive upper bound tests [min, max)
		{
			name:       "3.0.0 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.0.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.1.0 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.1.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.10.0 in range [3.0.0, 4.0.0) exclusive - would fail with string comparison",
			version:    "3.10.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.2.0 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.2.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.99.99 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.99.99",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "4.0.0 not in range [3.0.0, 4.0.0) - exclusive upper bound",
			version:    "4.0.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   false,
		},
		{
			name:       "2.0 not in range [3.0.0, 4.0.0) exclusive",
			version:    "2.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   false,
		},
		{
			name:       "3.0.0 in range [3.0.0, 3.1.0) exclusive",
			version:    "3.0.0",
			minVersion: "3.0.0",
			maxVersion: "3.1.0",
			expected:   true,
		},
		{
			name:       "3.0.9 in range [3.0.0, 3.1.0) exclusive",
			version:    "3.0.9",
			minVersion: "3.0.0",
			maxVersion: "3.1.0",
			expected:   true,
		},
		{
			name:       "3.1.0 not in range [3.0.0, 3.1.0) - exclusive upper bound",
			version:    "3.1.0",
			minVersion: "3.0.0",
			maxVersion: "3.1.0",
			expected:   false,
		},

		// No upper bound tests (empty maxVersion) - equivalent to v >= minVersion
		{
			name:       "3.1.0 >= 3.1.0 (no upper bound)",
			version:    "3.1.0",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   true,
		},
		{
			name:       "3.2.0 >= 3.1.0 (no upper bound)",
			version:    "3.2.0",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   true,
		},
		{
			name:       "3.10.0 >= 3.1.0 (no upper bound) - would fail with string comparison",
			version:    "3.10.0",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   true,
		},
		{
			name:       "3.0.9 not >= 3.1.0 (no upper bound)",
			version:    "3.0.9",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   false,
		},

		// Less than tests (min="0.0.0", exclusive max) - equivalent to v < maxVersion
		{
			name:       "3.0.0 < 3.1.0 (lower bound 0.0.0)",
			version:    "3.0.0",
			minVersion: "0.0.0",
			maxVersion: "3.1.0",
			expected:   true,
		},
		{
			name:       "3.1.0 not < 3.1.0 (lower bound 0.0.0)",
			version:    "3.1.0",
			minVersion: "0.0.0",
			maxVersion: "3.1.0",
			expected:   false,
		},
		{
			name:       "3.2.0 < 3.10.0 (lower bound 0.0.0) - would be wrong with string comparison",
			version:    "3.2.0",
			minVersion: "0.0.0",
			maxVersion: "3.10.0",
			expected:   true,
		},
		{
			name:       "3.10.0 not < 3.2.0 (lower bound 0.0.0) - would be wrong with string comparison",
			version:    "3.10.0",
			minVersion: "0.0.0",
			maxVersion: "3.2.0",
			expected:   false,
		},

		// Invalid version string
		{
			name:       "invalid version string",
			version:    "invalid",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionInRangeExclusive(tt.version, tt.minVersion, tt.maxVersion)
			if result != tt.expected {
				t.Errorf("versionInRangeExclusive(%s, %s, %s) = %v, want %v",
					tt.version, tt.minVersion, tt.maxVersion, result, tt.expected)
			}
		})
	}
}

// TestFormatBytes tests the FormatBytes helper function with various byte sizes
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024, "1.0 KiB"},
		{"kilobytes decimal", 1536, "1.5 KiB"},
		{"megabytes", 1048576, "1.0 MiB"},
		{"megabytes decimal", 5242880, "5.0 MiB"},
		{"gigabytes", 1073741824, "1.0 GiB"},
		{"gigabytes decimal", 2147483648, "2.0 GiB"},
		{"terabytes", 1099511627776, "1.0 TiB"},
		{"petabytes", 1125899906842624, "1.0 PiB"},
		{"exabytes", 1152921504606846976, "1.0 EiB"},
		{"large", 5368709120, "5.0 GiB"},
		{"negative bytes", -1024, "-1024 B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

// TestFormatDetection tests format detection for various inputs
func TestFormatDetection(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedFormat SourceFormat
	}{
		{
			name:           "JSON object",
			input:          []byte(`{"openapi": "3.0.0"}`),
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "JSON array",
			input:          []byte(`[{"test": "value"}]`),
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "JSON with leading whitespace",
			input:          []byte("  \n\t  {\"openapi\": \"3.0.0\"}"),
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "YAML content",
			input:          []byte("openapi: 3.0.0\ninfo:\n  title: Test"),
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "YAML with leading whitespace",
			input:          []byte("  \n  openapi: 3.0.0"),
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "empty content",
			input:          []byte(""),
			expectedFormat: SourceFormatUnknown,
		},
		{
			name:           "only whitespace",
			input:          []byte("   \n\t  \r\n  "),
			expectedFormat: SourceFormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := detectFormatFromContent(tt.input)
			assert.Equal(t, tt.expectedFormat, format)
		})
	}
}

// TestParseFileFormatDetection tests format detection from file extension
func TestParseFileFormatDetection(t *testing.T) {
	tests := []struct {
		name           string
		filepath       string
		expectedFormat SourceFormat
	}{
		{
			name:           "JSON file extension",
			filepath:       "../testdata/minimal-oas2.json",
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "YAML file extension",
			filepath:       "../testdata/minimal-oas2.yaml",
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "YML file extension",
			filepath:       "../testdata/petstore-2.0.yaml",
			expectedFormat: SourceFormatYAML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse(tt.filepath)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedFormat, result.SourceFormat)
		})
	}
}
