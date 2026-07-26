package paramutil_test

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/internal/paramutil"
	"github.com/erraggy/oastools/parser"
)

// maxRefDepthForTest mirrors the package's unexported maxRefDepth. Duplicating
// it is deliberate: the boundary tests assert where the real limit falls, so
// changing the production constant must make them fail rather than silently
// follow along.
const maxRefDepthForTest = 10

func pathParam(name string) *parser.Parameter {
	return &parser.Parameter{Name: name, In: parser.ParamInPath, Required: true}
}

func ref(target string) *parser.Parameter {
	return &parser.Parameter{Ref: target}
}

func TestNewOAS2Resolver(t *testing.T) {
	t.Run("resolves root-level parameters", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Parameters: map[string]*parser.Parameter{"petIdParam": pathParam("petId")},
		}

		resolved, ok := paramutil.NewOAS2Resolver(doc).Resolve(ref("#/parameters/petIdParam"))

		require.True(t, ok)
		assert.Equal(t, "petId", resolved.Name)
		assert.Equal(t, parser.ParamInPath, resolved.In)
	})

	t.Run("does not resolve OAS 3 style refs", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Parameters: map[string]*parser.Parameter{"petIdParam": pathParam("petId")},
		}

		_, ok := paramutil.NewOAS2Resolver(doc).Resolve(ref("#/components/parameters/petIdParam"))

		assert.False(t, ok)
	})

	t.Run("nil document yields nil resolver", func(t *testing.T) {
		assert.Nil(t, paramutil.NewOAS2Resolver(nil))
	})

	t.Run("skips nil definitions", func(t *testing.T) {
		doc := &parser.OAS2Document{
			Parameters: map[string]*parser.Parameter{"broken": nil},
		}

		_, ok := paramutil.NewOAS2Resolver(doc).Resolve(ref("#/parameters/broken"))

		assert.False(t, ok)
	})
}

func TestNewOAS3Resolver(t *testing.T) {
	t.Run("resolves components parameters", func(t *testing.T) {
		doc := &parser.OAS3Document{
			Components: &parser.Components{
				Parameters: map[string]*parser.Parameter{"petIdParam": pathParam("petId")},
			},
		}

		resolved, ok := paramutil.NewOAS3Resolver(doc).Resolve(ref("#/components/parameters/petIdParam"))

		require.True(t, ok)
		assert.Equal(t, "petId", resolved.Name)
	})

	t.Run("nil components yields nil resolver", func(t *testing.T) {
		assert.Nil(t, paramutil.NewOAS3Resolver(&parser.OAS3Document{}))
	})

	t.Run("nil document yields nil resolver", func(t *testing.T) {
		assert.Nil(t, paramutil.NewOAS3Resolver(nil))
	})
}

func TestResolver_Resolve(t *testing.T) {
	resolver := paramutil.NewOAS3Resolver(&parser.OAS3Document{
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"direct":  pathParam("petId"),
				"hop1":    ref("#/components/parameters/hop2"),
				"hop2":    pathParam("ownerId"),
				"loopA":   ref("#/components/parameters/loopB"),
				"loopB":   ref("#/components/parameters/loopA"),
				"selfRef": ref("#/components/parameters/selfRef"),
			},
		},
	})

	tests := []struct {
		name     string
		param    *parser.Parameter
		wantOK   bool
		wantName string
	}{
		{
			name:     "inline parameter returns itself",
			param:    pathParam("inlineId"),
			wantOK:   true,
			wantName: "inlineId",
		},
		{
			name:     "single hop",
			param:    ref("#/components/parameters/direct"),
			wantOK:   true,
			wantName: "petId",
		},
		{
			name:     "chained refs are followed",
			param:    ref("#/components/parameters/hop1"),
			wantOK:   true,
			wantName: "ownerId",
		},
		{
			name:   "dangling local ref is unresolvable",
			param:  ref("#/components/parameters/nope"),
			wantOK: false,
		},
		{
			// Indistinguishable from dangling here by design — Resolve reports
			// only that the chain could not be followed. Defines is what tells
			// the two apart, and callers need that: a dangling ref is reported
			// by reference validation, a wrong-kind one is not.
			name:   "local ref to a component of another kind is unresolvable",
			param:  ref("#/components/schemas/PetId"),
			wantOK: false,
		},
		{
			name:   "external file ref is unresolvable",
			param:  ref("common.yaml#/components/parameters/petId"),
			wantOK: false,
		},
		{
			name:   "remote url ref is unresolvable",
			param:  ref("https://example.com/spec.yaml#/components/parameters/petId"),
			wantOK: false,
		},
		{
			name:   "two-node cycle terminates",
			param:  ref("#/components/parameters/loopA"),
			wantOK: false,
		},
		{
			name:   "self reference terminates",
			param:  ref("#/components/parameters/selfRef"),
			wantOK: false,
		},
		{
			name:   "nil parameter is unresolvable",
			param:  nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, ok := resolver.Resolve(tt.param)

			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.NotNil(t, resolved)
				assert.Equal(t, tt.wantName, resolved.Name)
			} else {
				assert.Nil(t, resolved)
			}
		})
	}
}

// refChain builds a chain of reusable parameter definitions requiring exactly
// hops resolution steps: the returned parameter references the first link, each
// link references the next, and the final link is an inline path parameter.
func refChain(prefix string, hops int) (start *parser.Parameter, defs map[string]*parser.Parameter) {
	defs = make(map[string]*parser.Parameter, hops)
	for i := 1; i < hops; i++ {
		defs[fmt.Sprintf("%s%d", prefix, i)] = ref(fmt.Sprintf("#/components/parameters/%s%d", prefix, i+1))
	}
	defs[fmt.Sprintf("%s%d", prefix, hops)] = pathParam(prefix + "Id")
	return ref(fmt.Sprintf("#/components/parameters/%s1", prefix)), defs
}

// TestResolver_Resolve_RefDepthBoundary exercises the maxRefDepth cutoff
// directly. The cycle cases in TestResolver_Resolve terminate via the same
// mechanism, but only a chain of known length pins down where the boundary
// actually falls.
func TestResolver_Resolve_RefDepthBoundary(t *testing.T) {
	atLimit, atLimitDefs := refChain("atLimit", maxRefDepthForTest)
	overLimit, overLimitDefs := refChain("overLimit", maxRefDepthForTest+1)

	params := make(map[string]*parser.Parameter, len(atLimitDefs)+len(overLimitDefs))
	maps.Copy(params, atLimitDefs)
	maps.Copy(params, overLimitDefs)

	resolver := paramutil.NewOAS3Resolver(&parser.OAS3Document{
		Components: &parser.Components{Parameters: params},
	})

	t.Run("chain of exactly maxRefDepth hops resolves", func(t *testing.T) {
		resolved, ok := resolver.Resolve(atLimit)

		require.True(t, ok)
		assert.Equal(t, "atLimitId", resolved.Name)
	})

	t.Run("chain exceeding maxRefDepth hops is unresolvable", func(t *testing.T) {
		resolved, ok := resolver.Resolve(overLimit)

		assert.False(t, ok)
		assert.Nil(t, resolved)
	})
}

// TestResolver_Defines covers the narrower question Defines answers: whether a
// ref names a parameter definition at all, independent of whether that
// definition resolves. Callers use the gap between the two to tell "not a
// parameter" apart from "a parameter with a broken chain".
func TestResolver_Defines(t *testing.T) {
	resolver := paramutil.NewOAS3Resolver(&parser.OAS3Document{
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"petIdParam": pathParam("petId"),
				"brokenLink": ref("#/components/parameters/goneParam"),
			},
		},
	})

	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "names a resolvable parameter", ref: "#/components/parameters/petIdParam", want: true},
		{
			// The point of the method: still true, even though Resolve fails.
			name: "names a parameter whose own chain is broken",
			ref:  "#/components/parameters/brokenLink",
			want: true,
		},
		{name: "names a component of another kind", ref: "#/components/schemas/PetId", want: false},
		{name: "names nothing", ref: "#/components/parameters/absent", want: false},
		{name: "external ref", ref: "shared.yaml#/components/parameters/petIdParam", want: false},
		{name: "empty ref", ref: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolver.Defines(tt.ref))
		})
	}

	t.Run("Defines and Resolve disagree on a broken chain", func(t *testing.T) {
		const brokenRef = "#/components/parameters/brokenLink"

		require.True(t, resolver.Defines(brokenRef), "it is a parameter")
		_, ok := resolver.Resolve(ref(brokenRef))
		assert.False(t, ok, "but it does not resolve")
	})

	t.Run("nil resolver defines nothing", func(t *testing.T) {
		var nilResolver paramutil.Resolver

		assert.False(t, nilResolver.Defines("#/components/parameters/petIdParam"))
	})
}

// TestResolver_Classify covers each Reason directly. Resolve collapses them all
// to a single false, but callers that suppress diagnostics on failure owe the
// user an explanation, and only ReasonExternal warrants silence — so the
// distinctions have to hold.
func TestResolver_Classify(t *testing.T) {
	deep, deepDefs := refChain("deep", maxRefDepthForTest+1)

	params := map[string]*parser.Parameter{
		"petIdParam": pathParam("petId"),
		"aliasParam": ref("#/components/parameters/petIdParam"),
		"brokenLink": ref("#/components/parameters/goneParam"),
		"loopA":      ref("#/components/parameters/loopB"),
		"loopB":      ref("#/components/parameters/loopA"),
		"selfRef":    ref("#/components/parameters/selfRef"),
	}
	maps.Copy(params, deepDefs)

	resolver := paramutil.NewOAS3Resolver(&parser.OAS3Document{
		Components: &parser.Components{Parameters: params},
	})

	tests := []struct {
		name       string
		param      *parser.Parameter
		wantReason paramutil.Reason
	}{
		{name: "inline parameter", param: pathParam("petId"), wantReason: paramutil.ReasonResolved},
		{name: "single hop", param: ref("#/components/parameters/petIdParam"), wantReason: paramutil.ReasonResolved},
		{name: "chained hops", param: ref("#/components/parameters/aliasParam"), wantReason: paramutil.ReasonResolved},
		{
			name:       "external file ref",
			param:      ref("shared.yaml#/components/parameters/petIdParam"),
			wantReason: paramutil.ReasonExternal,
		},
		{
			name:       "remote url ref",
			param:      ref("https://example.com/spec.yaml#/components/parameters/petIdParam"),
			wantReason: paramutil.ReasonExternal,
		},
		{
			name:       "dangling local ref",
			param:      ref("#/components/parameters/absent"),
			wantReason: paramutil.ReasonNotAParameter,
		},
		{
			name:       "local ref to another component kind",
			param:      ref("#/components/schemas/PetId"),
			wantReason: paramutil.ReasonNotAParameter,
		},
		{
			// The break is downstream, but Classify reports the outcome of the
			// whole chain — callers use Defines to locate where it broke.
			name:       "chain breaking at a later hop",
			param:      ref("#/components/parameters/brokenLink"),
			wantReason: paramutil.ReasonNotAParameter,
		},
		{
			name:       "two-node cycle is a cycle, not depth exhaustion",
			param:      ref("#/components/parameters/loopA"),
			wantReason: paramutil.ReasonCycle,
		},
		{
			name:       "self reference is a cycle",
			param:      ref("#/components/parameters/selfRef"),
			wantReason: paramutil.ReasonCycle,
		},
		{
			// Nothing repeats here, so it must NOT be classified as a cycle.
			name:       "too-long chain is depth exhaustion, not a cycle",
			param:      deep,
			wantReason: paramutil.ReasonTooDeep,
		},
		{name: "nil parameter", param: nil, wantReason: paramutil.ReasonNotAParameter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, reason := resolver.Classify(tt.param)

			assert.Equal(t, tt.wantReason, reason)
			if tt.wantReason == paramutil.ReasonResolved {
				assert.NotNil(t, resolved)
			} else {
				assert.Nil(t, resolved, "only a resolved chain yields a parameter")
			}
		})
	}
}

func TestResolver_Resolve_NilResolver(t *testing.T) {
	var resolver paramutil.Resolver

	t.Run("still resolves inline parameters", func(t *testing.T) {
		resolved, ok := resolver.Resolve(pathParam("petId"))

		require.True(t, ok)
		assert.Equal(t, "petId", resolved.Name)
	})

	t.Run("reports every ref as unresolvable", func(t *testing.T) {
		_, ok := resolver.Resolve(ref("#/components/parameters/petIdParam"))

		assert.False(t, ok)
	})
}

func TestResolver_DeclaredPathParams(t *testing.T) {
	resolver := paramutil.NewOAS3Resolver(&parser.OAS3Document{
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"petIdParam":   pathParam("petId"),
				"ownerIdParam": pathParam("ownerId"),
				"limitParam":   {Name: "limit", In: parser.ParamInQuery},
			},
		},
	})

	tests := []struct {
		name         string
		lists        [][]*parser.Parameter
		wantDeclared []string
		wantComplete bool
	}{
		{
			name:         "no parameters",
			lists:        nil,
			wantDeclared: nil,
			wantComplete: true,
		},
		{
			name:         "inline path parameter",
			lists:        [][]*parser.Parameter{{pathParam("petId")}},
			wantDeclared: []string{"petId"},
			wantComplete: true,
		},
		{
			name:         "referenced path parameter",
			lists:        [][]*parser.Parameter{{ref("#/components/parameters/petIdParam")}},
			wantDeclared: []string{"petId"},
			wantComplete: true,
		},
		{
			name: "path-item and operation lists merge",
			lists: [][]*parser.Parameter{
				{ref("#/components/parameters/petIdParam")},
				{ref("#/components/parameters/ownerIdParam")},
			},
			wantDeclared: []string{"ownerId", "petId"},
			wantComplete: true,
		},
		{
			name:         "non-path parameters are excluded",
			lists:        [][]*parser.Parameter{{ref("#/components/parameters/limitParam")}},
			wantDeclared: nil,
			wantComplete: true,
		},
		{
			name:         "nil entries are skipped",
			lists:        [][]*parser.Parameter{{nil, pathParam("petId"), nil}},
			wantDeclared: []string{"petId"},
			wantComplete: true,
		},
		{
			name: "unresolvable ref marks the set incomplete",
			lists: [][]*parser.Parameter{
				{pathParam("petId"), ref("shared.yaml#/parameters/ownerId")},
			},
			wantDeclared: []string{"petId"},
			wantComplete: false,
		},
		{
			name:         "dangling local ref marks the set incomplete",
			lists:        [][]*parser.Parameter{{ref("#/components/parameters/missing")}},
			wantDeclared: nil,
			wantComplete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			declared, complete := resolver.DeclaredPathParams(tt.lists...)

			assert.Equal(t, tt.wantComplete, complete)
			if tt.wantDeclared == nil {
				assert.Nil(t, declared, "no path parameter declared, so no map should be allocated")
			}
			assert.Len(t, declared, len(tt.wantDeclared))
			for _, name := range tt.wantDeclared {
				assert.True(t, declared[name], "expected %q to be declared", name)
			}
		})
	}
}
