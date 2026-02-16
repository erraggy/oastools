package parser

import "fmt"

// decodeDocumentFromMap creates a version-specific document struct directly
// from a map[string]any, bypassing the marshal->unmarshal roundtrip.
//
// This is used when ResolveRefs is enabled: the resolver modifies the map
// in-place, and decodeDocumentFromMap converts the resolved map directly
// to a typed struct without the intermediate []byte allocation.
func decodeDocumentFromMap(data map[string]any, version string) (any, OASVersion, error) {
	v, ok := ParseVersion(version)
	if !ok {
		return nil, 0, fmt.Errorf("parser: invalid OAS version: %s", version)
	}
	switch v {
	case OASVersion20:
		var doc OAS2Document
		doc.decodeFromMap(data)
		doc.OASVersion = v
		return &doc, v, nil

	case OASVersion300, OASVersion301, OASVersion302, OASVersion303, OASVersion304,
		OASVersion310, OASVersion311, OASVersion312, OASVersion320:
		var doc OAS3Document
		doc.decodeFromMap(data)
		doc.OASVersion = v
		return &doc, v, nil

	default:
		return nil, 0, fmt.Errorf("parser: unsupported OpenAPI version: %s (only 2.0 and 3.x versions are supported)", version)
	}
}
