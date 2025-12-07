package fixer

import (
	"encoding/json"
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// deepCopyOAS2Document creates a deep copy of an OAS 2.0 document.
// This uses JSON marshaling/unmarshaling to ensure all nested structures
// and maps are properly copied.
func deepCopyOAS2Document(doc *parser.OAS2Document) (*parser.OAS2Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("cannot copy nil document")
	}

	// Marshal to JSON
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	// Unmarshal into a new document
	var copy parser.OAS2Document
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Preserve the OASVersion field which is not marshaled
	copy.OASVersion = doc.OASVersion

	return &copy, nil
}

// deepCopyOAS3Document creates a deep copy of an OAS 3.x document.
// This uses JSON marshaling/unmarshaling to ensure all nested structures
// and maps are properly copied.
func deepCopyOAS3Document(doc *parser.OAS3Document) (*parser.OAS3Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("cannot copy nil document")
	}

	// Marshal to JSON
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	// Unmarshal into a new document
	var copy parser.OAS3Document
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Preserve the OASVersion field which is not marshaled
	copy.OASVersion = doc.OASVersion

	return &copy, nil
}
