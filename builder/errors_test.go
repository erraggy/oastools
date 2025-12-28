package builder

import (
	"errors"
	"testing"

	"github.com/erraggy/oastools/oaserrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *BuilderError
		contains []string
	}{
		{
			name: "duplicate operationID with first occurrence",
			err: NewDuplicateOperationIDError("getUser", "POST", "/users/{id}",
				&operationLocation{Method: "GET", Path: "/users/{id}"}),
			contains: []string{"builder", "operation", "POST /users/{id}", "duplicate", "getUser", "GET /users/{id}"},
		},
		{
			name:     "duplicate operationID without first occurrence",
			err:      NewDuplicateOperationIDError("getUser", "POST", "/users/{id}", nil),
			contains: []string{"builder", "operation", "POST /users/{id}", "duplicate", "getUser"},
		},
		{
			name:     "unsupported method TRACE",
			err:      NewUnsupportedMethodError("TRACE", "/debug", "3.0.0"),
			contains: []string{"builder", "operation", "TRACE", "/debug", "3.0.0"},
		},
		{
			name:     "unsupported method QUERY",
			err:      NewUnsupportedMethodError("QUERY", "/search", "3.2.0"),
			contains: []string{"builder", "operation", "QUERY", "/search", "3.2.0"},
		},
		{
			name:     "invalid method",
			err:      NewInvalidMethodError("INVALID", "/path"),
			contains: []string{"builder", "operation", "INVALID", "/path", "unsupported"},
		},
		{
			name: "webhook duplicate operationID",
			err: NewDuplicateWebhookOperationIDError("createUser", "myWebhook", "POST",
				&operationLocation{Method: "POST", Path: "/users", IsWebhook: false}),
			contains: []string{"builder", "webhook", "myWebhook", "duplicate", "createUser", "POST /users"},
		},
		{
			name:     "schema error with cause",
			err:      NewSchemaError("UserSchema", "deduplication failed", errors.New("hash collision")),
			contains: []string{"builder", "schema", "UserSchema", "deduplication failed", "hash collision"},
		},
		{
			name:     "parameter constraint error",
			err:      NewParameterConstraintError("age", "POST /users", "minimum", "minimum (100) cannot exceed maximum (1)"),
			contains: []string{"builder", "parameter", "POST /users", "age", "minimum"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, s := range tt.contains {
				assert.Contains(t, msg, s, "error message should contain %q", s)
			}
		})
	}
}

func TestBuilderError_Is(t *testing.T) {
	err := NewDuplicateOperationIDError("test", "GET", "/", nil)
	assert.True(t, errors.Is(err, oaserrors.ErrConfig), "BuilderError should match oaserrors.ErrConfig")
}

func TestBuilderError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewSchemaError("Test", "failed", cause)

	assert.Equal(t, cause, errors.Unwrap(err), "Unwrap should return the cause")
}

func TestBuilderError_HasLocation(t *testing.T) {
	tests := []struct {
		name        string
		err         *BuilderError
		hasLocation bool
	}{
		{
			name:        "with path",
			err:         &BuilderError{Path: "/users"},
			hasLocation: true,
		},
		{
			name:        "with component only",
			err:         &BuilderError{Component: ComponentParameter},
			hasLocation: true,
		},
		{
			name:        "without location",
			err:         &BuilderError{Message: "something went wrong"},
			hasLocation: false,
		},
		{
			name:        "with method and path",
			err:         &BuilderError{Method: "GET", Path: "/users"},
			hasLocation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasLocation, tt.err.HasLocation())
		})
	}
}

func TestBuilderError_Location(t *testing.T) {
	tests := []struct {
		name     string
		err      *BuilderError
		expected string
	}{
		{
			name:     "method and path",
			err:      &BuilderError{Method: "GET", Path: "/users"},
			expected: "GET /users",
		},
		{
			name:     "path only",
			err:      &BuilderError{Path: "/users"},
			expected: "/users",
		},
		{
			name:     "component only",
			err:      &BuilderError{Component: ComponentParameter},
			expected: "parameter",
		},
		{
			name:     "no location",
			err:      &BuilderError{Message: "error"},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Location())
		})
	}
}

func TestBuilderErrors_Error(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errs := BuilderErrors{
			NewDuplicateOperationIDError("a", "GET", "/a", nil),
		}

		msg := errs.Error()
		assert.Contains(t, msg, "GET /a")
		assert.NotContains(t, msg, "error(s)") // Single error format
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := BuilderErrors{
			NewDuplicateOperationIDError("a", "GET", "/a", nil),
			NewUnsupportedMethodError("TRACE", "/debug", "3.0.0"),
		}

		msg := errs.Error()
		assert.Contains(t, msg, "2 error(s)")
		assert.Contains(t, msg, "GET /a")
		assert.Contains(t, msg, "TRACE /debug")
	})

	t.Run("empty errors", func(t *testing.T) {
		errs := BuilderErrors{}
		assert.Empty(t, errs.Error())
	})
}

func TestBuilderErrors_Unwrap(t *testing.T) {
	errs := BuilderErrors{
		NewDuplicateOperationIDError("a", "GET", "/a", nil),
		NewDuplicateOperationIDError("b", "POST", "/b", nil),
	}

	unwrapped := errs.Unwrap()
	require.Len(t, unwrapped, 2)
	assert.Contains(t, unwrapped[0].Error(), "GET /a")
	assert.Contains(t, unwrapped[1].Error(), "POST /b")
}

func TestOperationLocation_String(t *testing.T) {
	tests := []struct {
		name     string
		loc      operationLocation
		expected string
	}{
		{
			name:     "operation",
			loc:      operationLocation{Method: "GET", Path: "/users"},
			expected: "GET /users",
		},
		{
			name:     "webhook",
			loc:      operationLocation{Method: "POST", Path: "userCreated", IsWebhook: true},
			expected: "webhook userCreated (POST)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.loc.String())
		})
	}
}

func TestConstraintError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConstraintError
		contains []string
	}{
		{
			name: "basic constraint error",
			err: &ConstraintError{
				Field:   "minimum",
				Message: "minimum (100) cannot exceed maximum (1)",
			},
			contains: []string{"constraint error", "minimum", "cannot exceed maximum"},
		},
		{
			name: "constraint error with param name",
			err: &ConstraintError{
				Field:     "minimum",
				Message:   "value must be positive",
				ParamName: "age",
			},
			contains: []string{"constraint error", "parameter", "age", "minimum"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, s := range tt.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}

func TestConstraintError_HasLocation(t *testing.T) {
	tests := []struct {
		name        string
		err         *ConstraintError
		hasLocation bool
	}{
		{
			name:        "with param name",
			err:         &ConstraintError{Field: "min", Message: "test", ParamName: "age"},
			hasLocation: true,
		},
		{
			name:        "with operation context",
			err:         &ConstraintError{Field: "min", Message: "test", OperationContext: "POST /users"},
			hasLocation: true,
		},
		{
			name:        "without context",
			err:         &ConstraintError{Field: "min", Message: "test"},
			hasLocation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasLocation, tt.err.HasLocation())
		})
	}
}

func TestConstraintError_Location(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConstraintError
		expected string
	}{
		{
			name: "operation context and param name",
			err: &ConstraintError{
				Field:            "min",
				Message:          "test",
				ParamName:        "age",
				OperationContext: "POST /users",
			},
			expected: "POST /users parameter \"age\"",
		},
		{
			name:     "param name only",
			err:      &ConstraintError{Field: "min", Message: "test", ParamName: "age"},
			expected: "parameter \"age\"",
		},
		{
			name:     "operation context only",
			err:      &ConstraintError{Field: "min", Message: "test", OperationContext: "POST /users"},
			expected: "POST /users",
		},
		{
			name:     "field fallback",
			err:      &ConstraintError{Field: "minimum", Message: "test"},
			expected: "minimum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Location())
		})
	}
}

func TestComponentType_Values(t *testing.T) {
	// Verify component types are defined as expected strings
	assert.Equal(t, ComponentType("operation"), ComponentOperation)
	assert.Equal(t, ComponentType("webhook"), ComponentWebhook)
	assert.Equal(t, ComponentType("parameter"), ComponentParameter)
	assert.Equal(t, ComponentType("schema"), ComponentSchema)
	assert.Equal(t, ComponentType("request_body"), ComponentRequestBody)
	assert.Equal(t, ComponentType("response"), ComponentResponse)
	assert.Equal(t, ComponentType("security_scheme"), ComponentSecurityScheme)
	assert.Equal(t, ComponentType("server"), ComponentServer)
}

func TestConstraintError_Is(t *testing.T) {
	err := &ConstraintError{Field: "minimum", Message: "invalid"}
	assert.True(t, errors.Is(err, oaserrors.ErrConfig), "ConstraintError should match oaserrors.ErrConfig")
}

func TestConstraintError_Unwrap(t *testing.T) {
	err := &ConstraintError{Field: "minimum", Message: "invalid"}
	assert.Nil(t, errors.Unwrap(err), "ConstraintError.Unwrap should return nil")
}

func TestBuilderErrors_ErrorsAs(t *testing.T) {
	t.Run("errors.As with single BuilderError", func(t *testing.T) {
		errs := BuilderErrors{
			NewDuplicateOperationIDError("getUser", "GET", "/users", nil),
		}

		var be *BuilderError
		if !errors.As(errs, &be) {
			t.Fatal("errors.As should find BuilderError in BuilderErrors")
		}
		assert.Equal(t, "getUser", be.OperationID)
	})

	t.Run("errors.As with multiple BuilderErrors", func(t *testing.T) {
		errs := BuilderErrors{
			NewUnsupportedMethodError("TRACE", "/debug", "3.0.0"),
			NewDuplicateOperationIDError("getUser", "GET", "/users", nil),
		}

		var be *BuilderError
		if !errors.As(errs, &be) {
			t.Fatal("errors.As should find BuilderError in BuilderErrors")
		}
		// Should find the first one
		assert.Equal(t, "TRACE", be.Method)
	})

	t.Run("errors.Is with ErrConfig", func(t *testing.T) {
		errs := BuilderErrors{
			NewDuplicateOperationIDError("getUser", "GET", "/users", nil),
		}

		// errors.Is should traverse through the unwrapped errors
		assert.True(t, errors.Is(errs, oaserrors.ErrConfig),
			"errors.Is should find ErrConfig through BuilderErrors")
	})
}

func TestBuilderErrors_NilHandling(t *testing.T) {
	t.Run("Error with nil element", func(t *testing.T) {
		errs := BuilderErrors{
			NewDuplicateOperationIDError("a", "GET", "/a", nil),
			nil,
			NewDuplicateOperationIDError("b", "POST", "/b", nil),
		}

		// Should not panic
		msg := errs.Error()
		assert.Contains(t, msg, "GET /a")
		assert.Contains(t, msg, "POST /b")
	})

	t.Run("Unwrap skips nil elements", func(t *testing.T) {
		errs := BuilderErrors{
			NewDuplicateOperationIDError("a", "GET", "/a", nil),
			nil,
			NewDuplicateOperationIDError("b", "POST", "/b", nil),
		}

		unwrapped := errs.Unwrap()
		assert.Len(t, unwrapped, 2, "Unwrap should skip nil elements")
	})

	t.Run("single nil element", func(t *testing.T) {
		errs := BuilderErrors{nil}
		assert.Empty(t, errs.Error())
	})
}
