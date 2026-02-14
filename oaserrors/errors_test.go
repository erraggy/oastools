package oaserrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseError(t *testing.T) {
	t.Run("Error message with all fields", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := &ParseError{
			Path:    "/path/to/file.yaml",
			Line:    42,
			Column:  10,
			Message: "invalid syntax",
			Cause:   cause,
		}

		assert.Equal(t, "parse error in /path/to/file.yaml at line 42, column 10: invalid syntax: underlying error", err.Error())
	})

	t.Run("Error message with minimal fields", func(t *testing.T) {
		err := &ParseError{}
		assert.Equal(t, "parse error", err.Error())
	})

	t.Run("Error message with path only", func(t *testing.T) {
		err := &ParseError{Path: "api.yaml"}
		assert.Equal(t, "parse error in api.yaml", err.Error())
	})

	t.Run("Error message with line only", func(t *testing.T) {
		err := &ParseError{Line: 10}
		assert.Equal(t, "parse error at line 10", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("underlying")
		err := &ParseError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		//goland:noinspection GoDirectComparisonOfErrors
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("Unwrap returns nil when no cause", func(t *testing.T) {
		err := &ParseError{}
		assert.Nil(t, err.Unwrap())
	})

	t.Run("Is matches ErrParse", func(t *testing.T) {
		err := &ParseError{Message: "test"}
		assert.True(t, errors.Is(err, ErrParse), "ParseError should match ErrParse")
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ParseError{}
		assert.False(t, errors.Is(err, ErrReference), "ParseError should not match ErrReference")
		assert.False(t, errors.Is(err, ErrValidation), "ParseError should not match ErrValidation")
	})

	t.Run("As extracts ParseError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ParseError{Path: "test.yaml", Line: 5})
		var parseErr *ParseError
		require.True(t, errors.As(err, &parseErr))
		assert.Equal(t, "test.yaml", parseErr.Path)
		assert.Equal(t, 5, parseErr.Line)
	})
}

func TestReferenceError(t *testing.T) {
	t.Run("Error message for normal reference error", func(t *testing.T) {
		err := &ReferenceError{
			Ref:     "#/components/schemas/Pet",
			RefType: "local",
			Message: "not found",
		}
		assert.Equal(t, "reference error: #/components/schemas/Pet: not found", err.Error())
	})

	t.Run("Error message for circular reference", func(t *testing.T) {
		err := &ReferenceError{
			Ref:        "#/components/schemas/Node",
			IsCircular: true,
		}
		assert.Equal(t, "circular reference: #/components/schemas/Node", err.Error())
	})

	t.Run("Error message for path traversal", func(t *testing.T) {
		err := &ReferenceError{
			Ref:             "../../../etc/passwd",
			IsPathTraversal: true,
			Message:         "blocked for security",
		}
		assert.Equal(t, "path traversal detected: ../../../etc/passwd: blocked for security", err.Error())
	})

	t.Run("Error message with cause", func(t *testing.T) {
		cause := errors.New("file not found")
		err := &ReferenceError{
			Ref:     "./models.yaml",
			RefType: "file",
			Cause:   cause,
		}
		assert.Equal(t, "reference error: ./models.yaml: file not found", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("network error")
		err := &ReferenceError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		//goland:noinspection GoDirectComparisonOfErrors
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("Is matches ErrReference", func(t *testing.T) {
		err := &ReferenceError{Ref: "test"}
		assert.True(t, errors.Is(err, ErrReference), "ReferenceError should match ErrReference")
	})

	t.Run("Is matches ErrCircularReference when IsCircular", func(t *testing.T) {
		err := &ReferenceError{IsCircular: true}
		assert.True(t, errors.Is(err, ErrCircularReference), "ReferenceError with IsCircular should match ErrCircularReference")
		assert.True(t, errors.Is(err, ErrReference), "ReferenceError with IsCircular should also match ErrReference")
	})

	t.Run("Is does not match ErrCircularReference when not circular", func(t *testing.T) {
		err := &ReferenceError{IsCircular: false}
		assert.False(t, errors.Is(err, ErrCircularReference), "ReferenceError without IsCircular should not match ErrCircularReference")
	})

	t.Run("Is matches ErrPathTraversal when IsPathTraversal", func(t *testing.T) {
		err := &ReferenceError{IsPathTraversal: true}
		assert.True(t, errors.Is(err, ErrPathTraversal), "ReferenceError with IsPathTraversal should match ErrPathTraversal")
		assert.True(t, errors.Is(err, ErrReference), "ReferenceError with IsPathTraversal should also match ErrReference")
	})

	t.Run("Is does not match ErrPathTraversal when not path traversal", func(t *testing.T) {
		err := &ReferenceError{IsPathTraversal: false}
		assert.False(t, errors.Is(err, ErrPathTraversal), "ReferenceError without IsPathTraversal should not match ErrPathTraversal")
	})

	t.Run("As extracts ReferenceError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ReferenceError{
			Ref:        "#/schemas/X",
			IsCircular: true,
		})
		var refErr *ReferenceError
		require.True(t, errors.As(err, &refErr))
		assert.True(t, refErr.IsCircular)
	})
}

func TestValidationError(t *testing.T) {
	t.Run("Error message with all fields", func(t *testing.T) {
		err := &ValidationError{
			Path:    "paths./pets.get",
			Field:   "operationId",
			Message: "must be unique",
			SpecRef: "https://spec.openapis.org/oas/v3.0.3#operation-object",
		}
		assert.Equal(t, "validation error at paths./pets.get.operationId: must be unique", err.Error())
	})

	t.Run("Error message with path only", func(t *testing.T) {
		err := &ValidationError{Path: "info.title"}
		assert.Equal(t, "validation error at info.title", err.Error())
	})

	t.Run("Error message with cause", func(t *testing.T) {
		cause := errors.New("invalid format")
		err := &ValidationError{
			Path:  "info.version",
			Cause: cause,
		}
		assert.Equal(t, "validation error at info.version: invalid format", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("format error")
		err := &ValidationError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		//goland:noinspection GoDirectComparisonOfErrors
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("Is matches ErrValidation", func(t *testing.T) {
		err := &ValidationError{Path: "test"}
		assert.True(t, errors.Is(err, ErrValidation), "ValidationError should match ErrValidation")
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ValidationError{}
		assert.False(t, errors.Is(err, ErrParse), "ValidationError should not match ErrParse")
	})

	t.Run("As extracts ValidationError with Value", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ValidationError{
			Path:  "info.version",
			Value: "invalid",
		})
		var valErr *ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "invalid", valErr.Value)
	})
}

func TestResourceLimitError(t *testing.T) {
	t.Run("Error message with all fields", func(t *testing.T) {
		err := &ResourceLimitError{
			ResourceType: "ref_depth",
			Limit:        100,
			Actual:       150,
			Message:      "too many nested references",
		}
		assert.Equal(t, "resource limit exceeded: ref_depth (limit: 100, actual: 150): too many nested references", err.Error())
	})

	t.Run("Error message without actual", func(t *testing.T) {
		err := &ResourceLimitError{
			ResourceType: "file_size",
			Limit:        10485760,
		}
		assert.Equal(t, "resource limit exceeded: file_size (limit: 10485760)", err.Error())
	})

	t.Run("Error message minimal", func(t *testing.T) {
		err := &ResourceLimitError{}
		assert.Equal(t, "resource limit exceeded", err.Error())
	})

	t.Run("Unwrap returns nil", func(t *testing.T) {
		err := &ResourceLimitError{ResourceType: "test"}
		assert.Nil(t, err.Unwrap())
	})

	t.Run("Is matches ErrResourceLimit", func(t *testing.T) {
		err := &ResourceLimitError{Limit: 100}
		assert.True(t, errors.Is(err, ErrResourceLimit), "ResourceLimitError should match ErrResourceLimit")
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ResourceLimitError{}
		assert.False(t, errors.Is(err, ErrParse), "ResourceLimitError should not match ErrParse")
	})

	t.Run("As extracts ResourceLimitError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ResourceLimitError{
			ResourceType: "cached_documents",
			Limit:        100,
			Actual:       101,
		})
		var limitErr *ResourceLimitError
		require.True(t, errors.As(err, &limitErr))
		assert.Equal(t, int64(100), limitErr.Limit)
		assert.Equal(t, int64(101), limitErr.Actual)
	})
}

func TestConversionError(t *testing.T) {
	t.Run("Error message with all fields", func(t *testing.T) {
		cause := errors.New("unsupported feature")
		err := &ConversionError{
			SourceVersion: "3.1.0",
			TargetVersion: "2.0",
			Path:          "paths./pets.get.requestBody",
			Message:       "requestBody not supported in OAS 2.0",
			Cause:         cause,
		}
		assert.Equal(t, "conversion error (3.1.0 -> 2.0) at paths./pets.get.requestBody: requestBody not supported in OAS 2.0: unsupported feature", err.Error())
	})

	t.Run("Error message with versions only", func(t *testing.T) {
		err := &ConversionError{
			SourceVersion: "2.0",
			TargetVersion: "3.0.3",
		}
		assert.Equal(t, "conversion error (2.0 -> 3.0.3)", err.Error())
	})

	t.Run("Error message minimal", func(t *testing.T) {
		err := &ConversionError{}
		assert.Equal(t, "conversion error", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("version mismatch")
		err := &ConversionError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		//goland:noinspection GoDirectComparisonOfErrors
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("Is matches ErrConversion", func(t *testing.T) {
		err := &ConversionError{SourceVersion: "2.0"}
		assert.True(t, errors.Is(err, ErrConversion), "ConversionError should match ErrConversion")
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ConversionError{}
		assert.False(t, errors.Is(err, ErrValidation), "ConversionError should not match ErrValidation")
	})

	t.Run("As extracts ConversionError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ConversionError{
			SourceVersion: "3.0.0",
			TargetVersion: "3.1.0",
		})
		var convErr *ConversionError
		require.True(t, errors.As(err, &convErr))
		assert.Equal(t, "3.0.0", convErr.SourceVersion)
	})
}

func TestConfigError(t *testing.T) {
	t.Run("Error message with all fields", func(t *testing.T) {
		cause := errors.New("invalid value")
		err := &ConfigError{
			Option:  "timeout",
			Value:   -5,
			Message: "must be positive",
			Cause:   cause,
		}
		assert.Equal(t, "configuration error for timeout (value: -5): must be positive: invalid value", err.Error())
	})

	t.Run("Error message with option only", func(t *testing.T) {
		err := &ConfigError{Option: "filePath"}
		assert.Equal(t, "configuration error for filePath", err.Error())
	})

	t.Run("Error message minimal", func(t *testing.T) {
		err := &ConfigError{}
		assert.Equal(t, "configuration error", err.Error())
	})

	t.Run("Error message with nil value excluded", func(t *testing.T) {
		err := &ConfigError{
			Option:  "input",
			Value:   nil,
			Message: "required",
		}
		assert.Equal(t, "configuration error for input: required", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("missing value")
		err := &ConfigError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		//goland:noinspection GoDirectComparisonOfErrors
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("Is matches ErrConfig", func(t *testing.T) {
		err := &ConfigError{Option: "test"}
		assert.True(t, errors.Is(err, ErrConfig), "ConfigError should match ErrConfig")
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ConfigError{}
		assert.False(t, errors.Is(err, ErrParse), "ConfigError should not match ErrParse")
	})

	t.Run("As extracts ConfigError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ConfigError{
			Option: "maxSize",
			Value:  1000,
		})
		var cfgErr *ConfigError
		require.True(t, errors.As(err, &cfgErr))
		assert.Equal(t, "maxSize", cfgErr.Option)
	})
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are distinct
	sentinels := []error{
		ErrParse,
		ErrReference,
		ErrCircularReference,
		ErrPathTraversal,
		ErrValidation,
		ErrResourceLimit,
		ErrConversion,
		ErrConfig,
	}

	for i, s1 := range sentinels {
		for j, s2 := range sentinels {
			if i != j {
				assert.False(t, errors.Is(s1, s2), "sentinel errors should be distinct: %v should not match %v", s1, s2)
			}
		}
	}
}

func TestErrorChaining(t *testing.T) {
	t.Run("deeply wrapped ParseError", func(t *testing.T) {
		parseErr := &ParseError{Path: "api.yaml", Message: "invalid"}
		wrapped1 := fmt.Errorf("layer 1: %w", parseErr)
		wrapped2 := fmt.Errorf("layer 2: %w", wrapped1)

		assert.True(t, errors.Is(wrapped2, ErrParse), "deeply wrapped ParseError should match ErrParse")

		var extracted *ParseError
		require.True(t, errors.As(wrapped2, &extracted))
		assert.Equal(t, "api.yaml", extracted.Path)
	})

	t.Run("error wrapping with Cause", func(t *testing.T) {
		rootCause := errors.New("network timeout")
		refErr := &ReferenceError{
			Ref:   "http://example.com/schema.json",
			Cause: rootCause,
		}
		wrapped := fmt.Errorf("failed to load: %w", refErr)

		// Should be able to check for root cause
		assert.True(t, errors.Is(wrapped, rootCause), "should be able to find root cause through Unwrap chain")
	})
}
