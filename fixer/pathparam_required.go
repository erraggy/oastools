// pathparam_required.go sets required: true on path parameters that omit it,
// repairing what the validator reports as "Path parameters must have
// required: true".

package fixer

import (
	"fmt"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/internal/paramutil"
	"github.com/erraggy/oastools/parser"
)

// fixPathParamsRequiredOAS2 sets required: true on OAS 2.0 path parameters that
// omit it, covering both the root-level reusable parameters and those declared
// on path items and operations.
func (f *Fixer) fixPathParamsRequiredOAS2(doc *parser.OAS2Document, result *FixResult) {
	f.fixPathParamsRequiredInDefinitions(doc.Parameters, "parameters", result)
	f.fixPathParamsRequiredInPaths(doc.Paths, parser.OASVersion20, result)
}

// fixPathParamsRequiredOAS3 sets required: true on OAS 3.x path parameters that
// omit it, covering both components.parameters and those declared on path items
// and operations.
func (f *Fixer) fixPathParamsRequiredOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc.Components != nil {
		f.fixPathParamsRequiredInDefinitions(doc.Components.Parameters, "components.parameters", result)
	}
	f.fixPathParamsRequiredInPaths(doc.Paths, doc.OASVersion, result)
}

// fixPathParamsRequiredInDefinitions repairs a document's reusable parameter
// definitions. Fixing them here is what lets the use-site passes skip $refs:
// a referenced parameter is repaired once, in the definition that owns it.
//
// Definitions are visited in name order so the recorded fixes are deterministic.
func (f *Fixer) fixPathParamsRequiredInDefinitions(defs map[string]*parser.Parameter, prefix string, result *FixResult) {
	if len(defs) == 0 {
		return
	}

	names := maputil.SortedKeys(defs)

	for _, name := range names {
		param := defs[name]
		if !paramutil.NeedsRequiredTrue(param) {
			continue
		}
		f.setPathParamRequired(param, prefix+"."+name, result)
	}
}

// fixPathParamsRequiredInPaths repairs the parameters declared on path items and
// their operations, in path then method order for deterministic output.
func (f *Fixer) fixPathParamsRequiredInPaths(paths parser.Paths, version parser.OASVersion, result *FixResult) {
	if len(paths) == 0 {
		return
	}

	pathPatterns := maputil.SortedKeys(paths)

	for _, pathPattern := range pathPatterns {
		pathItem := paths[pathPattern]
		if pathItem == nil {
			continue
		}
		prefix := "paths." + pathPattern

		// A path item's parameters apply to every operation it holds, so they are
		// visited once here rather than inside the operation loop below — which
		// would otherwise record the same fix once per operation.
		f.setPathParamsRequired(pathItem.Parameters, prefix, result)

		operations := parser.GetOperations(pathItem, version)
		methods := maputil.SortedKeys(operations)

		for _, method := range methods {
			op := operations[method]
			if op == nil {
				continue
			}
			f.setPathParamsRequired(op.Parameters, prefix+"."+operationPathSegment(pathItem, method), result)
		}
	}
}

// setPathParamsRequired repairs one parameter list, reporting each fix by the
// parameter's position so the path matches the one the validator reports.
func (f *Fixer) setPathParamsRequired(params []*parser.Parameter, prefix string, result *FixResult) {
	for i, param := range params {
		if !paramutil.NeedsRequiredTrue(param) {
			continue
		}
		f.setPathParamRequired(param, fmt.Sprintf("%s.parameters[%d]", prefix, i), result)
	}
}

// setPathParamRequired sets required: true on one path parameter and records the
// fix. The spec permits no other value, so there is nothing to infer, nothing to
// configure, and no information to lose.
func (f *Fixer) setPathParamRequired(param *parser.Parameter, path string, result *FixResult) {
	if !f.DryRun {
		param.Required = true
	}

	fix := Fix{
		Type:        FixTypePathParameterNotRequired,
		Path:        path,
		Description: buildPathParamRequiredDescription(param.Name),
		Before:      false,
		After:       true,
	}
	f.populateFixLocation(&fix)
	result.Fixes = append(result.Fixes, fix)
}

// buildPathParamRequiredDescription describes a required: true fix, naming the
// parameter when it has a name. A path parameter without one is invalid for a
// separate reason the validator reports on its own; the description just avoids
// rendering an empty quoted name.
func buildPathParamRequiredDescription(paramName string) string {
	if paramName == "" {
		return "Set required: true on path parameter"
	}
	return fmt.Sprintf("Set required: true on path parameter '%s'", paramName)
}
