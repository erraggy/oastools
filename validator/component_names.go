package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/internal/naming"
	"github.com/erraggy/oastools/parser"
)

// validateOAS3ComponentNames reports component names that violate the charset
// OAS 3.x requires.
//
// The rule covers every fixed field of the Components Object, not just schemas.
// Checking the names in one pass here rather than inside each section's
// value-validation loop keeps the rule in a single readable place, and means a
// nil value cannot cause its name to go unchecked.
func (v *Validator) validateOAS3ComponentNames(components *parser.Components, result *ValidationResult, baseURL string) {
	if components == nil {
		return
	}

	checkComponentNames(v, components.Schemas, componentSchemasName, result, baseURL)
	checkComponentNames(v, components.Responses, "responses", result, baseURL)
	checkComponentNames(v, components.Parameters, "parameters", result, baseURL)
	checkComponentNames(v, components.Examples, "examples", result, baseURL)
	checkComponentNames(v, components.RequestBodies, "requestBodies", result, baseURL)
	checkComponentNames(v, components.Headers, "headers", result, baseURL)
	checkComponentNames(v, components.SecuritySchemes, "securitySchemes", result, baseURL)
	checkComponentNames(v, components.Links, "links", result, baseURL)
	checkComponentNames(v, components.Callbacks, "callbacks", result, baseURL)
	checkComponentNames(v, components.PathItems, "pathItems", result, baseURL)
	checkComponentNames(v, components.MediaTypes, "mediaTypes", result, baseURL)
}

// componentSchemasName is a special case
const componentSchemasName = "schemas"

// checkComponentNames reports every key of one Components section that falls
// outside [naming.ComponentNamePattern].
//
// A free function rather than a method because Go does not permit type
// parameters on methods, and the sections hold eleven different value types.
//
// An empty or whitespace-only name is reported as a missing name rather than a
// charset violation, because that is the more accurate description and the more
// actionable message. Schemas are the exception: validateSchemaName already says
// exactly that for them, so blankNamesReportedElsewhere suppresses the duplicate.
//
// Keys are visited in sorted order so a document with several offending names
// reports them the same way on every run.
//
// The section is scanned for a defect before any reporting machinery is built.
// Building the issue path prefix and the spec-reference option cost more than
// every other part of this check combined — they are eleven allocations per
// document that the overwhelmingly common case, a document whose names are all
// legal, then discards unused.
func checkComponentNames[V any](
	v *Validator,
	section map[string]V,
	field string,
	result *ValidationResult,
	baseURL string,
) {
	if len(section) == 0 {
		return
	}

	// if this is schemas, then know it is reported elsewhere
	blankNamesReportedElsewhere := field == componentSchemasName

	if !hasComponentNameDefect(section, blankNamesReportedElsewhere) {
		return
	}

	prefix := "components." + field
	specRef := withSpecRef(fmt.Sprintf("%s#components-object", baseURL))

	for _, name := range maputil.SortedKeys(section) {
		switch classifyComponentName(name, blankNamesReportedElsewhere) {
		case componentNameOK:
			continue

		case componentNameEmpty:
			v.addError(result, prefix, "Component name cannot be empty",
				specRef, withField("name"), withValue(""),
			)

		case componentNameWhitespace:
			v.addError(result, prefix+"."+name,
				fmt.Sprintf("Component name cannot be whitespace-only: %q", name),
				specRef, withField("name"), withValue(name),
			)

		case componentNameCharset:
			v.addError(result, prefix+"."+name,
				fmt.Sprintf("Component name %q must match %s", name, naming.ComponentNamePattern),
				specRef, withField("name"), withValue(name),
			)
		}
	}
}

// componentNameDefect describes what, if anything, is wrong with a component name.
type componentNameDefect int

const (
	componentNameOK componentNameDefect = iota
	componentNameEmpty
	componentNameWhitespace
	componentNameCharset
)

// classifyComponentName is the single source of truth for what counts as a
// defective component name, shared by the scan and the reporting loop so the
// two cannot disagree about which names are worth reporting.
func classifyComponentName(name string, blankNamesReportedElsewhere bool) componentNameDefect {
	switch {
	case blankNamesReportedElsewhere && strings.TrimSpace(name) == "":
		return componentNameOK
	case name == "":
		return componentNameEmpty
	case strings.TrimSpace(name) == "":
		return componentNameWhitespace
	case !naming.IsValidComponentName(name):
		return componentNameCharset
	default:
		return componentNameOK
	}
}

// hasComponentNameDefect reports whether any key of the section is defective.
//
// It ranges the map directly rather than over sorted keys: order does not matter
// to a yes-or-no answer, and sorting would allocate for the common case this
// exists to keep cheap.
func hasComponentNameDefect[V any](section map[string]V, blankNamesReportedElsewhere bool) bool {
	for name := range section {
		if classifyComponentName(name, blankNamesReportedElsewhere) != componentNameOK {
			return true
		}
	}
	return false
}
