package overlay

import (
	"fmt"

	"github.com/erraggy/oastools/internal/jsonpath"
)

// SupportedVersion is the overlay specification version supported by this implementation.
const SupportedVersion = "1.0.0"

// Validate checks an overlay document for structural errors.
//
// Returns a slice of validation errors. An empty slice indicates the overlay
// is valid. Validation checks include:
//   - Required fields (overlay version, info.title, info.version, actions)
//   - Supported overlay version (currently only 1.0.0)
//   - Valid JSONPath syntax in action targets
//   - Actions have either update or remove (or both)
func Validate(o *Overlay) []ValidationError {
	var errs []ValidationError

	// Required: overlay version
	if o.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "overlay",
			Message: "version is required",
		})
	} else if o.Version != SupportedVersion {
		errs = append(errs, ValidationError{
			Field:   "overlay",
			Message: fmt.Sprintf("unsupported version %q; only %q is supported", o.Version, SupportedVersion),
		})
	}

	// Required: info.title
	if o.Info.Title == "" {
		errs = append(errs, ValidationError{
			Field:   "info.title",
			Message: "title is required",
		})
	}

	// Required: info.version
	if o.Info.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "info.version",
			Message: "version is required",
		})
	}

	// Required: at least one action
	if len(o.Actions) == 0 {
		errs = append(errs, ValidationError{
			Field:   "actions",
			Message: "at least one action is required",
		})
	}

	// Validate each action
	for i, action := range o.Actions {
		actionErrs := validateAction(action, i)
		errs = append(errs, actionErrs...)
	}

	return errs
}

// validateAction validates a single action.
func validateAction(action Action, index int) []ValidationError {
	var errs []ValidationError
	pathPrefix := fmt.Sprintf("actions[%d]", index)

	// Required: target
	if action.Target == "" {
		errs = append(errs, ValidationError{
			Path:    pathPrefix + ".target",
			Message: "target is required",
		})
	} else {
		// Validate JSONPath syntax
		if _, err := jsonpath.Parse(action.Target); err != nil {
			errs = append(errs, ValidationError{
				Path:    pathPrefix + ".target",
				Message: fmt.Sprintf("invalid JSONPath: %v", err),
			})
		}
	}

	// Must have update or remove (or both - remove takes precedence)
	if action.Update == nil && !action.Remove {
		errs = append(errs, ValidationError{
			Path:    pathPrefix,
			Message: "action must have update or remove",
		})
	}

	return errs
}

// IsValid is a convenience function that returns true if the overlay has no validation errors.
func IsValid(o *Overlay) bool {
	return len(Validate(o)) == 0
}
