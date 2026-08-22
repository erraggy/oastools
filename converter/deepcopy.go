package converter

import (
	"maps"
	"slices"

	"github.com/erraggy/oastools/parser"
)

// Deep copies for the fields a conversion carries across unchanged. The
// converted document must not share objects with the source it was built from,
// so a field needing no conversion still needs a copy.

// deepCopyTags deep copies a document-level tag list.
func deepCopyTags(tags []*parser.Tag) []*parser.Tag {
	if tags == nil {
		return nil
	}
	cp := make([]*parser.Tag, len(tags))
	for i, tag := range tags {
		cp[i] = tag.DeepCopy()
	}
	return cp
}

// deepCopyStrings copies a plain string slice, so appending to the converted
// document's slice cannot reallocate over the source's backing array.
func deepCopyStrings(v []string) []string {
	if v == nil {
		return nil
	}
	return slices.Clone(v)
}

// deepCopyScopes copies an OAuth flow's scope map. The values are strings, so
// cloning the map is the whole copy. A nil map copies to nil.
func deepCopyScopes(v map[string]string) map[string]string {
	return maps.Clone(v)
}
