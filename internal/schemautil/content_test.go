package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

func TestSortedContentTypes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content map[string]*parser.MediaType
		want    []string
	}{
		{
			name: "json outranks the rest, and ties go by name",
			content: map[string]*parser.MediaType{
				"text/plain":           {},
				"application/ld+json":  {},
				"application/json":     {},
				"application/geo+json": {},
				"application/xml":      {},
			},
			want: []string{
				"application/json",     // exact
				"application/geo+json", // suffix, then by name
				"application/ld+json",
				"application/xml", // other, then by name
				"text/plain",
			},
		},
		{
			name:    "no json at all still orders by name",
			content: map[string]*parser.MediaType{"text/plain": {}, "application/xml": {}},
			want:    []string{"application/xml", "text/plain"},
		},
		{
			name:    "empty",
			content: map[string]*parser.MediaType{},
			want:    []string{},
		},
		{
			name:    "nil",
			content: nil,
			want:    []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, SortedContentTypes(tc.content))
		})
	}
}

// TestSortedContentTypesIsRepeatable is the property the callers depend on. Go
// randomizes map order per range, so an ordering still driven by it shows up
// over repeated calls rather than in a single comparison.
func TestSortedContentTypesIsRepeatable(t *testing.T) {
	content := map[string]*parser.MediaType{
		"application/xml": {}, "application/json": {}, "text/plain": {},
		"application/ld+json": {}, "application/geo+json": {}, "text/csv": {},
	}
	first := SortedContentTypes(content)
	for range 100 {
		assert.Equal(t, first, SortedContentTypes(content))
	}
}
