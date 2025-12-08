package fixer

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// deepCopyOAS2Document creates a deep copy of an OAS 2.0 document.
// Uses generated DeepCopy methods for type-safe, efficient copying.
func deepCopyOAS2Document(doc *parser.OAS2Document) (*parser.OAS2Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("cannot copy nil document")
	}

	cp := doc.DeepCopy()
	// OASVersion is copied by DeepCopyInto (shallow copy of struct)
	return cp, nil
}

// deepCopyOAS3Document creates a deep copy of an OAS 3.x document.
// Uses generated DeepCopy methods for type-safe, efficient copying.
func deepCopyOAS3Document(doc *parser.OAS3Document) (*parser.OAS3Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("cannot copy nil document")
	}

	cp := doc.DeepCopy()
	// OASVersion is copied by DeepCopyInto (shallow copy of struct)
	return cp, nil
}
