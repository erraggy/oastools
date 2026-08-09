package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenameScopeRedirect covers the three shapes a withdrawal takes.
func TestRenameScopeRedirect(t *testing.T) {
	t.Run("the document's own spelling survives, so the rename goes", func(t *testing.T) {
		scope := newRenameScope(2, parser.OASVersion303)
		scope.registerRight(1, "Pet", "Pet_v1")

		scope.redirect(1, "Pet_v1", "Pet")

		assert.Empty(t, scope.byDoc[1], "with nothing left to rewrite, the document is skipped entirely")
		assert.True(t, scope.empty())
	})

	t.Run("another name survives, so the rename follows it", func(t *testing.T) {
		scope := newRenameScope(2, parser.OASVersion303)
		scope.registerRight(1, "Pet", "Pet_v1")

		scope.redirect(1, "Pet_v1", "Pet_v0")

		assert.Equal(t, map[string]string{"Pet": "Pet_v0"}, scope.byDoc[1])
	})

	t.Run("a name no rename produced gains one", func(t *testing.T) {
		scope := newRenameScope(2, parser.OASVersion303)

		scope.redirect(0, "Pet", "Common")

		assert.Equal(t, map[string]string{"Pet": "Common"}, scope.byDoc[0])
	})

	t.Run("out of range and self-directed calls do nothing", func(t *testing.T) {
		scope := newRenameScope(1, parser.OASVersion303)

		scope.redirect(-1, "Pet", "Common")
		scope.redirect(9, "Pet", "Common")
		scope.redirect(0, "Pet", "Pet")

		assert.True(t, scope.empty())
	})
}

// TestRefView covers both spellings a schema name arrives in, and the identity
// a nil view stands for.
func TestRefView(t *testing.T) {
	scope := newRenameScope(1, parser.OASVersion303)
	scope.registerRight(0, "Pet", "Pet_v1")
	view := scope.view(0)
	require.NotNil(t, view)

	assert.Equal(t, "#/components/schemas/Pet_v1", view.ref("#/components/schemas/Pet"))
	assert.Equal(t, "#/components/schemas/Other", view.ref("#/components/schemas/Other"),
		"a name with no rename reads as itself")

	// A discriminator may name a schema either way.
	assert.Equal(t, "#/components/schemas/Pet_v1", view.name("#/components/schemas/Pet"))
	assert.Equal(t, "Pet_v1", view.name("Pet"))

	var identity *refView
	assert.Equal(t, "#/components/schemas/Pet", identity.ref("#/components/schemas/Pet"))
	assert.Equal(t, "Pet", identity.name("Pet"))
	assert.Nil(t, scope.view(0+1), "an out of range document has no view")

	empty := newRenameScope(1, parser.OASVersion303)
	assert.Nil(t, empty.view(0), "a document with no renames needs no view")
	assert.Nil(t, empty.mergedView())
}

// TestRefViewIsNotFollowedByRedirect pins the copy in newRefView: a view built
// before a collapse must keep comparing against what it was built from.
func TestRefViewIsNotFollowedByRedirect(t *testing.T) {
	scope := newRenameScope(1, parser.OASVersion303)
	scope.registerRight(0, "Pet", "Pet_v1")
	view := scope.view(0)

	scope.redirect(0, "Pet_v1", "Pet")

	assert.Equal(t, "Pet_v1", view.name("Pet"))
}

// TestCanonicalName covers the naming rule and its tie-break.
func TestCanonicalName(t *testing.T) {
	tests := []struct {
		name      string
		class     []string
		generated []string
		want      string
	}{
		{
			name:      "a name no rename produced beats one that sorts earlier",
			class:     []string{"Api_Common", "Common"},
			generated: []string{"Api_Common"},
			want:      "Common",
		},
		{
			name:  "equally original names are ordered alphabetically",
			class: []string{"Location", "Address"},
			want:  "Address",
		},
		{
			name:      "a class of only generated names still picks one",
			class:     []string{"Pet_v2", "Pet_v1"},
			generated: []string{"Pet_v1", "Pet_v2"},
			want:      "Pet_v1",
		},
		{
			name:  "a single name is its own canonical",
			class: []string{"Pet"},
			want:  "Pet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generated := make(map[string]bool, len(tt.generated))
			for _, name := range tt.generated {
				generated[name] = true
			}
			assert.Equal(t, tt.want, canonicalName(tt.class, generated))
		})
	}
}

// TestCollapseGroupsConnectsChains covers the grouping: a name that collides
// once per document leaves a chain of pairs that is one group, and a name
// whose entry no document contributed takes its group with it.
func TestCollapseGroupsConnectsChains(t *testing.T) {
	schemas := map[string]*parser.Schema{
		"Pet": {}, "Pet_v1": {}, "Pet_v2": {},
		"Tag": {}, "Tag_v1": {},
		"Custom": {}, "Custom_v1": {},
	}
	owner := map[any]int{
		schemas["Pet"]: 0, schemas["Pet_v1"]: 1, schemas["Pet_v2"]: 2,
		schemas["Tag"]: 0, schemas["Tag_v1"]: 1,
		// Custom is deliberately unowned, as a collision handler's custom value
		// would be.
		schemas["Custom_v1"]: 1,
	}
	views := newViewIndex(newRenameScope(3, parser.OASVersion303), owner, schemas)

	groups := collapseGroups([]deferredRename{
		{name: "Pet", newName: "Pet_v1"},
		{name: "Tag", newName: "Tag_v1"},
		{name: "Pet", newName: "Pet_v2"},
		{name: "Custom", newName: "Custom_v1"},
		{name: "Gone", newName: "Gone_v1"},
	}, schemas, views)

	assert.Equal(t, [][]string{{"Pet", "Pet_v1", "Pet_v2"}, {"Tag", "Tag_v1"}}, groups,
		"three documents spelling Pet are one group; the unowned and the missing are dropped")
}
