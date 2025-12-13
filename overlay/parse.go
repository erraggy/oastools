package overlay

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// ParseOverlay parses an overlay document from YAML or JSON bytes.
//
// The function automatically detects the format (JSON or YAML) and parses
// accordingly. Returns the parsed Overlay or an error if parsing fails.
func ParseOverlay(data []byte) (*Overlay, error) {
	var o Overlay

	// yaml.Unmarshal handles both YAML and JSON
	if err := yaml.Unmarshal(data, &o); err != nil {
		return nil, &ParseError{Cause: err}
	}

	return &o, nil
}

// ParseOverlayFile parses an overlay document from a file path.
//
// The function reads the file and parses it as an overlay document.
// Supports both YAML (.yaml, .yml) and JSON (.json) files.
func ParseOverlayFile(path string) (*Overlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ParseError{Path: path, Cause: err}
	}

	o, err := ParseOverlay(data)
	if err != nil {
		var pe *ParseError
		if errors.As(err, &pe) {
			pe.Path = path
			return nil, pe
		}
		return nil, &ParseError{Path: path, Cause: err}
	}

	return o, nil
}

// IsOverlayDocument checks if the given bytes appear to be an overlay document.
//
// This is a heuristic check that looks for the "overlay" version field.
// Returns true if the document looks like an overlay, false otherwise.
func IsOverlayDocument(data []byte) bool {
	// Quick check for the overlay version field
	// This handles both YAML and JSON formats
	return bytes.Contains(data, []byte("overlay:")) ||
		bytes.Contains(data, []byte(`"overlay":`))
}

// MarshalOverlay serializes an overlay to YAML bytes.
func MarshalOverlay(o *Overlay) ([]byte, error) {
	data, err := yaml.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("overlay: failed to marshal: %w", err)
	}
	return data, nil
}
