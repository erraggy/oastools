package converter

import (
	"slices"

	"github.com/erraggy/oastools/parser"
)

// The converters build a new document out of an existing one. Fields that need
// no conversion were assigned straight across, which left the two documents
// sharing the object: writing to the converted document reached back into the
// caller's source, and for Info that included the title, not just extensions.
//
// These clone the fields that have no per-version conversion of their own, so
// "no conversion needed" stops meaning "same object".

// cloneTags deep copies a document-level tag list.
func cloneTags(tags []*parser.Tag) []*parser.Tag {
	if tags == nil {
		return nil
	}
	cp := make([]*parser.Tag, len(tags))
	for i, tag := range tags {
		cp[i] = tag.DeepCopy()
	}
	return cp
}

// cloneHeaders deep copies a response's header map.
func cloneHeaders(headers map[string]*parser.Header) map[string]*parser.Header {
	if headers == nil {
		return nil
	}
	cp := make(map[string]*parser.Header, len(headers))
	for name, header := range headers {
		cp[name] = header.DeepCopy()
	}
	return cp
}

// cloneStrings copies a plain string slice, so appending to the converted
// document's slice cannot reallocate over the source's backing array.
func cloneStrings(v []string) []string {
	if v == nil {
		return nil
	}
	return slices.Clone(v)
}
