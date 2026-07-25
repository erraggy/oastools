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
	"strings"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// localRefPrefix marks a $ref that points within the current document.
const localRefPrefix = "#/"

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

// Reason explains the outcome of following a parameter's $ref chain.
//
// Only ReasonExternal is genuinely unknowable. The rest are defects in the
// document, so a caller that suppresses a diagnostic because resolution failed
// must make sure something reports the reason instead — silence is only
// warranted for ReasonExternal.
type Reason int

const (
	// ReasonResolved means the chain reached an inline parameter.
	ReasonResolved Reason = iota
	// ReasonExternal means the $ref points outside this document. This package
	// never loads other files or URLs, and the parser leaves such references
	// unexpanded by default, so nothing has resolved them.
	ReasonExternal
	// ReasonNotAParameter means a local $ref that names no parameter definition
	// in this document — it either dangles or names a different kind of
	// component. Distinguishing those two needs the document's full set of
	// valid references, which this package does not have.
	ReasonNotAParameter
	// ReasonCycle means the chain revisits a reference it already followed.
	ReasonCycle
	// ReasonTooDeep means the chain exceeded maxRefDepth without repeating,
	// so it is not a cycle — just longer than this package will follow.
	ReasonTooDeep
)

// Classify follows param's $ref chain and reports what happened. The returned
// parameter is non-nil only for ReasonResolved.
//
// Callers that only need "did it resolve" should use Resolve. Classify exists
// for callers that suppress diagnostics on failure and therefore owe the user
// an explanation of why.
func (r Resolver) Classify(param *parser.Parameter) (*parser.Parameter, Reason) {
	// Tracks refs already followed so a cycle is reported as a cycle rather
	// than surfacing as depth exhaustion, which is a different defect.
	var visited map[string]bool

	for param != nil && param.Ref != "" {
		if !strings.HasPrefix(param.Ref, localRefPrefix) {
			return nil, ReasonExternal
		}
		if visited[param.Ref] {
			return nil, ReasonCycle
		}
		if len(visited) >= maxRefDepth {
			return nil, ReasonTooDeep
		}
		if visited == nil {
			visited = make(map[string]bool, maxRefDepth)
		}
		visited[param.Ref] = true

		target, found := r[param.Ref]
		if !found {
			return nil, ReasonNotAParameter
		}
		param = target
	}
	if param == nil {
		return nil, ReasonNotAParameter
	}
	return param, ReasonResolved
}

// Resolve returns the effective definition for param, following local $ref
// hops until it reaches an inline parameter.
//
// Returns false whenever the chain cannot be followed. Callers must treat that as
// "unknown" rather than "declares nothing" — the whole point of this package is
// that the two are not the same. Use Classify when the distinction between the
// ways it can fail matters, which it does for anything that reports diagnostics.
func (r Resolver) Resolve(param *parser.Parameter) (*parser.Parameter, bool) {
	resolved, reason := r.Classify(param)
	return resolved, reason == ReasonResolved
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
