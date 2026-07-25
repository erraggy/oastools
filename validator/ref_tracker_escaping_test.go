package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRefLookupMatchesIssuePath pins the invariant the reference tracker depends
// on: the key it stores, derived from a $ref, must be findable from the issue
// path the validator reports for that same component.
//
// The two come from different inputs — a $ref carries RFC 6901 escaping and the
// issue path carries the component's real name, which may itself contain dots —
// so nothing but a test holds them together. When they disagreed,
// getComponentOperationContext found no operations for a component that is in
// fact referenced and annotated it "unused component".
//
// Covers issues #380 (escaping, mostly OAS 2.0 where such names are legitimate)
// and #383 (dots, legal and common in OAS 3.x).
func TestRefLookupMatchesIssuePath(t *testing.T) {
	tests := []struct {
		name string
		// ref is what a document carries; issuePath is where the validator
		// reports a problem with the component that ref names.
		ref       string
		issuePath string
	}{
		{
			name:      "OAS 2.0 definition",
			ref:       "#/definitions/User",
			issuePath: "definitions.User",
		},
		{
			name:      "OAS 2.0 definition with escaped slash",
			ref:       "#/definitions/pet~1summary",
			issuePath: "definitions.pet/summary",
		},
		{
			name:      "OAS 2.0 definition with escaped tilde",
			ref:       "#/definitions/pet~0summary",
			issuePath: "definitions.pet~summary",
		},
		{
			name:      "OAS 2.0 parameter with escaped slash",
			ref:       "#/parameters/pet~1id",
			issuePath: "parameters.pet/id",
		},
		{
			name:      "OAS 2.0 response with escaped slash",
			ref:       "#/responses/not~1found",
			issuePath: "responses.not/found",
		},
		{
			name:      "OAS 3.x schema",
			ref:       "#/components/schemas/User",
			issuePath: "components.schemas.User",
		},
		{
			name:      "OAS 3.x schema with escaped slash",
			ref:       "#/components/schemas/pet~1summary",
			issuePath: "components.schemas.pet/summary",
		},
		{
			name:      "OAS 3.x schema with dots in the name",
			ref:       "#/components/schemas/microsoft.graph.user",
			issuePath: "components.schemas.microsoft.graph.user",
		},
		{
			// The two defects together: a name that both contains dots and needs
			// escaping must survive the round trip through either mechanism.
			name:      "OAS 3.x schema with dots and an escaped slash",
			ref:       "#/components/schemas/microsoft.graph~1user",
			issuePath: "components.schemas.microsoft.graph/user",
		},
		{
			// The issue is nested inside the component rather than on the
			// component itself, which is the case getComponentRoot existed for.
			name:      "issue nested inside a dotted component",
			ref:       "#/components/schemas/microsoft.graph.user",
			issuePath: "components.schemas.microsoft.graph.user.properties.id",
		},
	}

	op := operationRef{Method: "GET", Path: "/pets", OperationID: "listPets"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := newRefTracker()
			rt.addRef(tt.ref, op)

			assert.Equal(t, []operationRef{op}, rt.getOperationsForComponent(tt.issuePath),
				"a component referenced by %q must be found from issue path %q", tt.ref, tt.issuePath)
		})
	}
}

// TestRefLookupPrefersLongestMatch guards the one way prefix matching can go
// wrong: a component whose name is a prefix of another's must not claim the
// other's issue paths.
func TestRefLookupPrefersLongestMatch(t *testing.T) {
	user := operationRef{Method: "GET", Path: "/users", OperationID: "getUser"}
	profile := operationRef{Method: "GET", Path: "/profiles", OperationID: "getProfile"}

	rt := newRefTracker()
	rt.addRef("#/components/schemas/User", user)
	rt.addRef("#/components/schemas/User.Profile", profile)

	assert.Equal(t, []operationRef{user}, rt.getOperationsForComponent("components.schemas.User"))
	assert.Equal(t, []operationRef{user}, rt.getOperationsForComponent("components.schemas.User.properties.id"))
	assert.Equal(t, []operationRef{profile}, rt.getOperationsForComponent("components.schemas.User.Profile"))
	assert.Equal(t, []operationRef{profile}, rt.getOperationsForComponent("components.schemas.User.Profile.properties.id"))
}

// TestRefLookupUnreferencedComponent keeps "nothing matched" meaning what it
// should. componentToOps holds exactly the components something references, so
// finding no match is the correct answer for an orphan rather than a lookup
// failure — the distinction the annotation depends on.
func TestRefLookupUnreferencedComponent(t *testing.T) {
	rt := newRefTracker()
	rt.addRef("#/components/schemas/User", operationRef{Method: "GET", Path: "/users"})

	assert.Empty(t, rt.getOperationsForComponent("components.schemas.Orphan"))
	assert.Empty(t, rt.getOperationsForComponent("components.schemas.Orphan.properties.id"))
	assert.Empty(t, rt.getOperationsForComponent(""))
}

// TestNormalizeRefUnescapesPerToken guards the ordering the fix depends on.
//
// Unescaping the whole ref before joining tokens with "." would turn the "/"
// recovered from "~1" into a ".", silently renaming the component. Only a name
// containing an escape sequence can catch this, which is why it is asserted
// directly rather than left to the pairing test above.
func TestNormalizeRefUnescapesPerToken(t *testing.T) {
	assert.Equal(t, "definitions.pet/summary", normalizeRef("#/definitions/pet~1summary"))
	assert.NotEqual(t, "definitions.pet.summary", normalizeRef("#/definitions/pet~1summary"),
		"the recovered slash must not be rewritten as a path separator")
}

// TestNormalizeRefIgnoresExternalRefs keeps the early return intact: an external
// ref names nothing in this document and is deliberately untracked.
func TestNormalizeRefIgnoresExternalRefs(t *testing.T) {
	assert.Empty(t, normalizeRef("other.yaml#/definitions/User"))
	assert.Empty(t, normalizeRef("https://example.com/spec.yaml#/definitions/User"))
}
