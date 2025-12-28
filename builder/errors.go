package builder

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/oaserrors"
)

// ComponentType identifies the type of component where an error occurred.
type ComponentType string

const (
	// ComponentOperation indicates an error in an operation definition.
	ComponentOperation ComponentType = "operation"
	// ComponentWebhook indicates an error in a webhook definition.
	ComponentWebhook ComponentType = "webhook"
	// ComponentParameter indicates an error in a parameter definition.
	ComponentParameter ComponentType = "parameter"
	// ComponentSchema indicates an error in a schema definition.
	ComponentSchema ComponentType = "schema"
	// ComponentRequestBody indicates an error in a request body definition.
	ComponentRequestBody ComponentType = "request_body"
	// ComponentResponse indicates an error in a response definition.
	ComponentResponse ComponentType = "response"
	// ComponentSecurityScheme indicates an error in a security scheme.
	ComponentSecurityScheme ComponentType = "security_scheme"
	// ComponentServer indicates an error in a server definition.
	ComponentServer ComponentType = "server"
)

// operationLocation tracks where an operationID was first defined.
type operationLocation struct {
	Method    string
	Path      string
	IsWebhook bool
}

// String returns a human-readable location description.
func (ol operationLocation) String() string {
	if ol.IsWebhook {
		return fmt.Sprintf("webhook %s (%s)", ol.Path, ol.Method)
	}
	return fmt.Sprintf("%s %s", ol.Method, ol.Path)
}

// BuilderError represents a structured error from the builder package.
// It provides detailed context about where and why an error occurred during
// the fluent API building process.
type BuilderError struct {
	// Component is the type of component where the error occurred.
	Component ComponentType
	// Method is the HTTP method (for operation/webhook errors).
	Method string
	// Path is the API path (for operation errors) or webhook name.
	Path string
	// OperationID is the operation identifier (if applicable).
	OperationID string
	// Field is the specific field with the error (e.g., "minimum").
	Field string
	// Message describes the error.
	Message string
	// Context provides additional details (e.g., conflicting values).
	Context map[string]any
	// FirstOccurrence tracks where a duplicate was first defined.
	FirstOccurrence *operationLocation
	// Cause is the underlying error, if any.
	Cause error
}

// Error implements the error interface with a detailed, formatted message.
func (e *BuilderError) Error() string {
	var sb strings.Builder
	sb.WriteString("builder")

	// Add component context
	if e.Component != "" {
		sb.WriteString(": ")
		sb.WriteString(string(e.Component))
	}

	// Add method and path for operations/webhooks
	if e.Method != "" && e.Path != "" {
		sb.WriteString(" ")
		sb.WriteString(e.Method)
		sb.WriteString(" ")
		sb.WriteString(e.Path)
	} else if e.Path != "" {
		sb.WriteString(" ")
		sb.WriteString(e.Path)
	}

	// Add operationID if present
	if e.OperationID != "" {
		sb.WriteString(" [operationId: ")
		sb.WriteString(e.OperationID)
		sb.WriteString("]")
	}

	// Add field if present
	if e.Field != "" {
		sb.WriteString(" field ")
		sb.WriteString(e.Field)
	}

	// Add the message
	if e.Message != "" {
		sb.WriteString(": ")
		sb.WriteString(e.Message)
	}

	// Add first occurrence context for duplicates
	if e.FirstOccurrence != nil {
		sb.WriteString(" (first defined at ")
		sb.WriteString(e.FirstOccurrence.String())
		sb.WriteString(")")
	}

	// Add underlying cause if present
	if e.Cause != nil {
		sb.WriteString(": ")
		sb.WriteString(e.Cause.Error())
	}

	return sb.String()
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *BuilderError) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error type.
// All BuilderErrors are classified as ErrConfig errors, enabling callers to use
// errors.Is(err, oaserrors.ErrConfig) to detect builder configuration issues.
// This includes validation errors, duplicate detection, and unsupported features.
func (e *BuilderError) Is(target error) bool {
	return target == oaserrors.ErrConfig
}

// HasLocation returns true if this error has location context.
// BuilderError uses component/method/path instead of line/column.
func (e *BuilderError) HasLocation() bool {
	return e.Path != "" || e.Component != ""
}

// Location returns a descriptive location string.
func (e *BuilderError) Location() string {
	if e.Method != "" && e.Path != "" {
		return fmt.Sprintf("%s %s", e.Method, e.Path)
	}
	if e.Path != "" {
		return e.Path
	}
	if e.Component != "" {
		return string(e.Component)
	}
	return "unknown"
}

// NewDuplicateOperationIDError creates an error for duplicate operation IDs.
func NewDuplicateOperationIDError(operationID, method, path string, first *operationLocation) *BuilderError {
	return &BuilderError{
		Component:       ComponentOperation,
		Method:          method,
		Path:            path,
		OperationID:     operationID,
		Message:         fmt.Sprintf("duplicate operationId %q", operationID),
		FirstOccurrence: first,
	}
}

// NewDuplicateWebhookOperationIDError creates an error for duplicate operation IDs in webhooks.
func NewDuplicateWebhookOperationIDError(operationID, webhookName, method string, first *operationLocation) *BuilderError {
	return &BuilderError{
		Component:       ComponentWebhook,
		Method:          method,
		Path:            webhookName,
		OperationID:     operationID,
		Message:         fmt.Sprintf("duplicate operationId %q", operationID),
		FirstOccurrence: first,
	}
}

// NewUnsupportedMethodError creates an error for unsupported HTTP methods.
func NewUnsupportedMethodError(method, path, minVersion string) *BuilderError {
	return &BuilderError{
		Component: ComponentOperation,
		Method:    method,
		Path:      path,
		Message:   fmt.Sprintf("HTTP method %s requires OAS version %s or later", method, minVersion),
		Context: map[string]any{
			"min_version": minVersion,
		},
	}
}

// NewInvalidMethodError creates an error for invalid/unknown HTTP methods.
func NewInvalidMethodError(method, path string) *BuilderError {
	return &BuilderError{
		Component: ComponentOperation,
		Method:    method,
		Path:      path,
		Message:   fmt.Sprintf("unsupported HTTP method: %s", method),
	}
}

// NewParameterConstraintError creates an error for parameter constraint violations.
func NewParameterConstraintError(paramName, operationContext, field, message string) *BuilderError {
	return &BuilderError{
		Component: ComponentParameter,
		Path:      operationContext,
		Field:     field,
		Message:   fmt.Sprintf("parameter %q: %s", paramName, message),
	}
}

// NewSchemaError creates an error for schema-related issues.
func NewSchemaError(schemaName, message string, cause error) *BuilderError {
	return &BuilderError{
		Component: ComponentSchema,
		Path:      schemaName,
		Message:   message,
		Cause:     cause,
	}
}

// BuilderErrors is a collection of BuilderError with formatting support.
type BuilderErrors []*BuilderError

// Error implements the error interface with a formatted multi-error message.
func (errs BuilderErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	if len(errs) == 1 {
		if errs[0] == nil {
			return ""
		}
		return errs[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("builder: %d error(s):\n", len(errs)))
	for _, e := range errs {
		if e == nil {
			continue
		}
		sb.WriteString("  - ")
		// Strip the "builder: " prefix for nested errors to avoid repetition
		errMsg := strings.TrimPrefix(e.Error(), "builder: ")
		sb.WriteString(errMsg)
		sb.WriteString("\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// Unwrap returns the errors for Go 1.20+ error wrapping semantics,
// enabling errors.Is and errors.As to work with multiple wrapped errors.
func (errs BuilderErrors) Unwrap() []error {
	result := make([]error, 0, len(errs))
	for _, e := range errs {
		if e == nil {
			continue
		}
		result = append(result, e)
	}
	return result
}
