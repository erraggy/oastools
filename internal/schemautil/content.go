// content.go orders the media types of a Content Object.

package schemautil

import (
	"slices"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)

// SortedContentTypes returns a Content Object's media types in preference
// order: rank first, then name.
//
// A caller picking one of several media types needs an order that does not come
// from ranging the map, since that order is randomized and the choice reaches
// the output. The ordering lives here so the packages making that choice agree
// on it; each applies its own predicate to the result.
func SortedContentTypes(content map[string]*parser.MediaType) []string {
	names := maputil.SortedKeys(content)
	slices.SortStableFunc(names, func(a, b string) int {
		return httputil.MediaTypeRank(a) - httputil.MediaTypeRank(b)
	})
	return names
}
