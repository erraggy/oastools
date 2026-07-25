package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)

// componentNamePattern is the key charset OAS 3.x requires of every fixed field
// of the Components Object: "All the fixed fields declared above are objects
// that MUST use keys that match the regular expression: ^[a-zA-Z0-9\.\-_]+$".
//
// See: https://spec.openapis.org/oas/v3.0.0.html#components-object
//
// The spec writes "." and "-" escaped inside the character class, where neither
// needs escaping; this is the same expression written plainly.
//
// OAS 2.0 has no counterpart, deliberately. It places no charset constraint on
// the keys of the root-level parameters, definitions, and responses objects, so
// a name containing "/" or "~" is legitimate there and is escaped per RFC 6901
// when referenced: see [pathutil.EscapeRefToken]. Applying this pattern to an
// OAS 2.0 document would reject valid specs.
var componentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

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
// outside [componentNamePattern].
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
func checkComponentNames[V any](
	v *Validator,
	section map[string]V,
	field string,
	result *ValidationResult,
	baseURL string,
) {
	prefix := "components." + field
	specRef := withSpecRef(fmt.Sprintf("%s#components-object", baseURL))

	// if this is schemas, then know it is reported elsewhere
	blankNamesReportedElsewhere := field == componentSchemasName

	for _, name := range maputil.SortedKeys(section) {
		switch {
		case blankNamesReportedElsewhere && strings.TrimSpace(name) == "":
			continue

		case name == "":
			v.addError(result, prefix, "Component name cannot be empty",
				specRef, withField("name"), withValue(""),
			)

		case strings.TrimSpace(name) == "":
			v.addError(result, prefix+"."+name,
				fmt.Sprintf("Component name cannot be whitespace-only: %q", name),
				specRef, withField("name"), withValue(name),
			)

		case !componentNamePattern.MatchString(name):
			v.addError(result, prefix+"."+name,
				fmt.Sprintf("Component name %q must match %s", name, componentNamePattern),
				specRef, withField("name"), withValue(name),
			)
		}
	}
}
