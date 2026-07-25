// Package paramutil resolves parameter $ref values against a document's
// reusable parameter definitions.
//
// The parser preserves $ref values verbatim so documents round-trip losslessly,
// which means a referenced parameter arrives with empty Name and In fields. Any
// analysis that keys off those fields — path template consistency checks, for
// example — must resolve the reference first or it will conclude the parameter
// declares nothing at all.
package paramutil

import (
	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// maxRefDepth bounds how many $ref hops Resolve will follow. Reusable parameter
// definitions may themselves be references, but a cycle
// (#/parameters/a -> #/parameters/b -> #/parameters/a) must not loop forever.
//
// The value is a sanity limit, not a spec constraint — OAS places no bound on
// alias chain length. It is deliberately independent of the parser's
// MaxRefDepth, which governs loading external documents; this package resolves
// only within one already-parsed document, where chains beyond a couple of hops
// do not occur in practice.
const maxRefDepth = 10

// Resolver maps local parameter $ref values to the definitions they point at.
// The zero value (a nil Resolver) is usable: it resolves inline parameters and
// reports every $ref as unresolvable.
type Resolver map[string]*parser.Parameter

// NewOAS2Resolver builds a Resolver over an OAS 2.0 document's root-level
// parameters, keyed by "#/parameters/{name}".
func NewOAS2Resolver(doc *parser.OAS2Document) Resolver {
	if doc == nil {
		return nil
	}
	return newResolver(doc.Parameters, true)
}

// NewOAS3Resolver builds a Resolver over an OAS 3.x document's
// components.parameters, keyed by "#/components/parameters/{name}".
func NewOAS3Resolver(doc *parser.OAS3Document) Resolver {
	if doc == nil || doc.Components == nil {
		return nil
	}
	return newResolver(doc.Components.Parameters, false)
}

func newResolver(params map[string]*parser.Parameter, oas2 bool) Resolver {
	if len(params) == 0 {
		return nil
	}
	r := make(Resolver, len(params))
	for name, param := range params {
		if param != nil {
			r[pathutil.ParameterRef(name, oas2)] = param
		}
	}
	return r
}

// Defines reports whether ref names a parameter definition in this resolver.
//
// This answers a narrower question than Resolve: it inspects only the ref's
// immediate target, so it stays true for a reference to a parameter whose own
// $ref chain is broken further down. Callers distinguishing "this is not a
// parameter at all" from "this is a parameter that fails to resolve" need that
// distinction — the two are different defects.
func (r Resolver) Defines(ref string) bool {
	_, ok := r[ref]
	return ok
}

// Resolve returns the effective definition for param, following local $ref
// hops until it reaches an inline parameter.
//
// ok is false whenever the chain cannot be followed to an inline parameter:
//
//   - an external $ref — this package never loads other files or URLs, and the
//     parser leaves them unresolved by default, so nothing has expanded them
//   - a local $ref with no matching reusable parameter definition, whether it
//     dangles or names some other kind of component
//   - a reference cycle, or a chain longer than maxRefDepth
//
// Callers must treat a false ok as "unknown" rather than "declares nothing" —
// the whole point of this package is that the two are not the same. Note that
// only the first case is genuinely unknowable; the others are defects, and
// callers that suppress diagnostics on a false ok are responsible for making
// sure something else reports them. Defines distinguishes the second case.
func (r Resolver) Resolve(param *parser.Parameter) (resolved *parser.Parameter, ok bool) {
	for depth := 0; param != nil && param.Ref != ""; depth++ {
		if depth >= maxRefDepth {
			return nil, false
		}
		target, found := r[param.Ref]
		if !found {
			return nil, false
		}
		param = target
	}
	if param == nil {
		return nil, false
	}
	return param, true
}

// DeclaredPathParams returns the names of every path parameter declared across
// the given parameter lists, resolving local $ref values along the way. Lists
// are typically the path item's parameters followed by the operation's. The
// names collapse into a single set rather than the operation's list shadowing
// the path item's, because an operation-level parameter overrides a path-level
// one of the same name and location — both contribute the same name, so a union
// is sufficient here. This is not a general implementation of that override.
//
// complete reports whether every parameter resolved. When it is false the
// returned set is a lower bound — some parameter in the list may declare a name
// that is not present — so callers must not report a path template parameter as
// undeclared based on it.
func (r Resolver) DeclaredPathParams(lists ...[]*parser.Parameter) (declared map[string]bool, complete bool) {
	declared = make(map[string]bool)
	complete = true

	for _, params := range lists {
		for _, param := range params {
			if param == nil {
				continue
			}
			resolved, ok := r.Resolve(param)
			if !ok {
				complete = false
				continue
			}
			if resolved.In == parser.ParamInPath {
				declared[resolved.Name] = true
			}
		}
	}

	return declared, complete
}
