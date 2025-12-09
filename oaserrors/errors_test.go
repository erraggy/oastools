package oaserrors

import (
	"errors"
	"fmt"
	"testing"
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

		msg := err.Error()
		if msg != "parse error in /path/to/file.yaml at line 42, column 10: invalid syntax: underlying error" {
			t.Errorf("unexpected error message: %s", msg)
		}
	})

	t.Run("Error message with minimal fields", func(t *testing.T) {
		err := &ParseError{}
		if err.Error() != "parse error" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with path only", func(t *testing.T) {
		err := &ParseError{Path: "api.yaml"}
		if err.Error() != "parse error in api.yaml" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with line only", func(t *testing.T) {
		err := &ParseError{Line: 10}
		if err.Error() != "parse error at line 10" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("underlying")
		err := &ParseError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		if unwrapped := err.Unwrap(); unwrapped != cause {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("Unwrap returns nil when no cause", func(t *testing.T) {
		err := &ParseError{}
		if err.Unwrap() != nil {
			t.Error("Unwrap should return nil when no cause")
		}
	})

	t.Run("Is matches ErrParse", func(t *testing.T) {
		err := &ParseError{Message: "test"}
		if !errors.Is(err, ErrParse) {
			t.Error("ParseError should match ErrParse")
		}
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ParseError{}
		if errors.Is(err, ErrReference) {
			t.Error("ParseError should not match ErrReference")
		}
		if errors.Is(err, ErrValidation) {
			t.Error("ParseError should not match ErrValidation")
		}
	})

	t.Run("As extracts ParseError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ParseError{Path: "test.yaml", Line: 5})
		var parseErr *ParseError
		if !errors.As(err, &parseErr) {
			t.Fatal("errors.As should succeed")
		}
		if parseErr.Path != "test.yaml" {
			t.Errorf("unexpected path: %s", parseErr.Path)
		}
		if parseErr.Line != 5 {
			t.Errorf("unexpected line: %d", parseErr.Line)
		}
	})
}

func TestReferenceError(t *testing.T) {
	t.Run("Error message for normal reference error", func(t *testing.T) {
		err := &ReferenceError{
			Ref:     "#/components/schemas/Pet",
			RefType: "local",
			Message: "not found",
		}
		expected := "reference error: #/components/schemas/Pet: not found"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message for circular reference", func(t *testing.T) {
		err := &ReferenceError{
			Ref:        "#/components/schemas/Node",
			IsCircular: true,
		}
		expected := "circular reference: #/components/schemas/Node"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message for path traversal", func(t *testing.T) {
		err := &ReferenceError{
			Ref:             "../../../etc/passwd",
			IsPathTraversal: true,
			Message:         "blocked for security",
		}
		expected := "path traversal detected: ../../../etc/passwd: blocked for security"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with cause", func(t *testing.T) {
		cause := errors.New("file not found")
		err := &ReferenceError{
			Ref:     "./models.yaml",
			RefType: "file",
			Cause:   cause,
		}
		if msg := err.Error(); msg != "reference error: ./models.yaml: file not found" {
			t.Errorf("unexpected error message: %s", msg)
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("network error")
		err := &ReferenceError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		if unwrapped := err.Unwrap(); unwrapped != cause {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("Is matches ErrReference", func(t *testing.T) {
		err := &ReferenceError{Ref: "test"}
		if !errors.Is(err, ErrReference) {
			t.Error("ReferenceError should match ErrReference")
		}
	})

	t.Run("Is matches ErrCircularReference when IsCircular", func(t *testing.T) {
		err := &ReferenceError{IsCircular: true}
		if !errors.Is(err, ErrCircularReference) {
			t.Error("ReferenceError with IsCircular should match ErrCircularReference")
		}
		if !errors.Is(err, ErrReference) {
			t.Error("ReferenceError with IsCircular should also match ErrReference")
		}
	})

	t.Run("Is does not match ErrCircularReference when not circular", func(t *testing.T) {
		err := &ReferenceError{IsCircular: false}
		if errors.Is(err, ErrCircularReference) {
			t.Error("ReferenceError without IsCircular should not match ErrCircularReference")
		}
	})

	t.Run("Is matches ErrPathTraversal when IsPathTraversal", func(t *testing.T) {
		err := &ReferenceError{IsPathTraversal: true}
		if !errors.Is(err, ErrPathTraversal) {
			t.Error("ReferenceError with IsPathTraversal should match ErrPathTraversal")
		}
		if !errors.Is(err, ErrReference) {
			t.Error("ReferenceError with IsPathTraversal should also match ErrReference")
		}
	})

	t.Run("Is does not match ErrPathTraversal when not path traversal", func(t *testing.T) {
		err := &ReferenceError{IsPathTraversal: false}
		if errors.Is(err, ErrPathTraversal) {
			t.Error("ReferenceError without IsPathTraversal should not match ErrPathTraversal")
		}
	})

	t.Run("As extracts ReferenceError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ReferenceError{
			Ref:        "#/schemas/X",
			IsCircular: true,
		})
		var refErr *ReferenceError
		if !errors.As(err, &refErr) {
			t.Fatal("errors.As should succeed")
		}
		if !refErr.IsCircular {
			t.Error("IsCircular should be true")
		}
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
		expected := "validation error at paths./pets.get.operationId: must be unique"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with path only", func(t *testing.T) {
		err := &ValidationError{Path: "info.title"}
		if err.Error() != "validation error at info.title" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with cause", func(t *testing.T) {
		cause := errors.New("invalid format")
		err := &ValidationError{
			Path:  "info.version",
			Cause: cause,
		}
		expected := "validation error at info.version: invalid format"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("format error")
		err := &ValidationError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		if unwrapped := err.Unwrap(); unwrapped != cause {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("Is matches ErrValidation", func(t *testing.T) {
		err := &ValidationError{Path: "test"}
		if !errors.Is(err, ErrValidation) {
			t.Error("ValidationError should match ErrValidation")
		}
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ValidationError{}
		if errors.Is(err, ErrParse) {
			t.Error("ValidationError should not match ErrParse")
		}
	})

	t.Run("As extracts ValidationError with Value", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ValidationError{
			Path:  "info.version",
			Value: "invalid",
		})
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Fatal("errors.As should succeed")
		}
		if valErr.Value != "invalid" {
			t.Errorf("unexpected value: %v", valErr.Value)
		}
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
		expected := "resource limit exceeded: ref_depth (limit: 100, actual: 150): too many nested references"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message without actual", func(t *testing.T) {
		err := &ResourceLimitError{
			ResourceType: "file_size",
			Limit:        10485760,
		}
		expected := "resource limit exceeded: file_size (limit: 10485760)"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message minimal", func(t *testing.T) {
		err := &ResourceLimitError{}
		if err.Error() != "resource limit exceeded" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Unwrap returns nil", func(t *testing.T) {
		err := &ResourceLimitError{ResourceType: "test"}
		if err.Unwrap() != nil {
			t.Error("Unwrap should return nil")
		}
	})

	t.Run("Is matches ErrResourceLimit", func(t *testing.T) {
		err := &ResourceLimitError{Limit: 100}
		if !errors.Is(err, ErrResourceLimit) {
			t.Error("ResourceLimitError should match ErrResourceLimit")
		}
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ResourceLimitError{}
		if errors.Is(err, ErrParse) {
			t.Error("ResourceLimitError should not match ErrParse")
		}
	})

	t.Run("As extracts ResourceLimitError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ResourceLimitError{
			ResourceType: "cached_documents",
			Limit:        100,
			Actual:       101,
		})
		var limitErr *ResourceLimitError
		if !errors.As(err, &limitErr) {
			t.Fatal("errors.As should succeed")
		}
		if limitErr.Limit != 100 {
			t.Errorf("unexpected limit: %d", limitErr.Limit)
		}
		if limitErr.Actual != 101 {
			t.Errorf("unexpected actual: %d", limitErr.Actual)
		}
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
		expected := "conversion error (3.1.0 -> 2.0) at paths./pets.get.requestBody: requestBody not supported in OAS 2.0: unsupported feature"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with versions only", func(t *testing.T) {
		err := &ConversionError{
			SourceVersion: "2.0",
			TargetVersion: "3.0.3",
		}
		expected := "conversion error (2.0 -> 3.0.3)"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message minimal", func(t *testing.T) {
		err := &ConversionError{}
		if err.Error() != "conversion error" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("version mismatch")
		err := &ConversionError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		if unwrapped := err.Unwrap(); unwrapped != cause {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("Is matches ErrConversion", func(t *testing.T) {
		err := &ConversionError{SourceVersion: "2.0"}
		if !errors.Is(err, ErrConversion) {
			t.Error("ConversionError should match ErrConversion")
		}
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ConversionError{}
		if errors.Is(err, ErrValidation) {
			t.Error("ConversionError should not match ErrValidation")
		}
	})

	t.Run("As extracts ConversionError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ConversionError{
			SourceVersion: "3.0.0",
			TargetVersion: "3.1.0",
		})
		var convErr *ConversionError
		if !errors.As(err, &convErr) {
			t.Fatal("errors.As should succeed")
		}
		if convErr.SourceVersion != "3.0.0" {
			t.Errorf("unexpected source version: %s", convErr.SourceVersion)
		}
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
		expected := "configuration error for timeout (value: -5): must be positive: invalid value"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with option only", func(t *testing.T) {
		err := &ConfigError{Option: "filePath"}
		expected := "configuration error for filePath"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message minimal", func(t *testing.T) {
		err := &ConfigError{}
		if err.Error() != "configuration error" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Error message with nil value excluded", func(t *testing.T) {
		err := &ConfigError{
			Option:  "input",
			Value:   nil,
			Message: "required",
		}
		expected := "configuration error for input: required"
		if err.Error() != expected {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("missing value")
		err := &ConfigError{Cause: cause}
		//nolint:errorlint // testing pointer identity
		if unwrapped := err.Unwrap(); unwrapped != cause {
			t.Error("Unwrap should return cause")
		}
	})

	t.Run("Is matches ErrConfig", func(t *testing.T) {
		err := &ConfigError{Option: "test"}
		if !errors.Is(err, ErrConfig) {
			t.Error("ConfigError should match ErrConfig")
		}
	})

	t.Run("Is does not match other sentinels", func(t *testing.T) {
		err := &ConfigError{}
		if errors.Is(err, ErrParse) {
			t.Error("ConfigError should not match ErrParse")
		}
	})

	t.Run("As extracts ConfigError", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &ConfigError{
			Option: "maxSize",
			Value:  1000,
		})
		var cfgErr *ConfigError
		if !errors.As(err, &cfgErr) {
			t.Fatal("errors.As should succeed")
		}
		if cfgErr.Option != "maxSize" {
			t.Errorf("unexpected option: %s", cfgErr.Option)
		}
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
			if i != j && errors.Is(s1, s2) {
				t.Errorf("sentinel errors should be distinct: %v should not match %v", s1, s2)
			}
		}
	}
}

func TestErrorChaining(t *testing.T) {
	t.Run("deeply wrapped ParseError", func(t *testing.T) {
		parseErr := &ParseError{Path: "api.yaml", Message: "invalid"}
		wrapped1 := fmt.Errorf("layer 1: %w", parseErr)
		wrapped2 := fmt.Errorf("layer 2: %w", wrapped1)

		if !errors.Is(wrapped2, ErrParse) {
			t.Error("deeply wrapped ParseError should match ErrParse")
		}

		var extracted *ParseError
		if !errors.As(wrapped2, &extracted) {
			t.Fatal("errors.As should work through wrapping")
		}
		if extracted.Path != "api.yaml" {
			t.Errorf("unexpected path: %s", extracted.Path)
		}
	})

	t.Run("error wrapping with Cause", func(t *testing.T) {
		rootCause := errors.New("network timeout")
		refErr := &ReferenceError{
			Ref:   "http://example.com/schema.json",
			Cause: rootCause,
		}
		wrapped := fmt.Errorf("failed to load: %w", refErr)

		// Should be able to check for root cause
		if !errors.Is(wrapped, rootCause) {
			t.Error("should be able to find root cause through Unwrap chain")
		}
	})
}
