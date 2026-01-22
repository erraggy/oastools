// Package issues provides a unified issue type for validation and conversion problems.
package issues

import "fmt"

// OperationContext provides API operation context for validation issues.
// For issues under paths.*, it identifies the specific operation.
// For issues outside paths.*, it shows which operations reference the component.
type OperationContext struct {
	// Method is the HTTP method (GET, POST, etc.) - empty for path-level issues
	Method string
	// Path is the API path pattern (e.g., "/users/{id}") or webhook name
	Path string
	// OperationID is the operationId if defined (may be empty)
	OperationID string
	// IsReusableComponent is true when the issue is in components/definitions
	IsReusableComponent bool
	// IsWebhook is true when the issue is in a webhook operation
	IsWebhook bool
	// AdditionalRefs is the count of other operations referencing this component.
	// Only relevant when IsReusableComponent is true.
	// -1 indicates the component is unused (not referenced by any operation).
	AdditionalRefs int
}

// String returns a formatted string representation of the operation context.
// Returns empty string if the context is empty.
func (c OperationContext) String() string {
	if c.IsEmpty() {
		return ""
	}

	// Handle unused component
	if c.IsReusableComponent && c.AdditionalRefs == -1 {
		return "(unused component)"
	}

	// Handle webhook
	if c.IsWebhook {
		return fmt.Sprintf("(webhook: %s)", c.Path)
	}

	// Build the primary identifier
	var primary string
	if c.OperationID != "" {
		primary = fmt.Sprintf("operationId: %s", c.OperationID)
	} else if c.Method != "" {
		primary = fmt.Sprintf("%s %s", c.Method, c.Path)
	} else if c.Path != "" {
		// Path-level (no method)
		return fmt.Sprintf("(path: %s)", c.Path)
	}

	// Add additional refs count for reusable components
	if c.IsReusableComponent && c.AdditionalRefs > 0 {
		return fmt.Sprintf("(%s, +%d operations)", primary, c.AdditionalRefs)
	}

	return fmt.Sprintf("(%s)", primary)
}

// IsEmpty returns true if the context has no meaningful information.
func (c OperationContext) IsEmpty() bool {
	// Unused component is not empty - it's valid context
	if c.IsReusableComponent && c.AdditionalRefs == -1 {
		return false
	}
	return c.Method == "" && c.Path == "" && c.OperationID == ""
}
