package parser

import (
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

// SourceLocation represents a position in a source document.
// Line and Column are 1-based (matching editor conventions).
// A zero Line value indicates the location is unknown.
type SourceLocation struct {
	// Line is the 1-based line number (0 if unknown)
	Line int
	// Column is the 1-based column number (0 if unknown)
	Column int
	// File is the source file path (empty for the main document)
	File string
}

// IsKnown returns true if this location has valid line information.
func (s SourceLocation) IsKnown() bool {
	return s.Line > 0
}

// String returns a human-readable location string.
// Format: "file:line:column" or "line:column" if no file, or "<unknown>" if not known.
func (s SourceLocation) String() string {
	if !s.IsKnown() {
		if s.File != "" {
			return s.File
		}
		return "<unknown>"
	}
	if s.File != "" {
		return fmt.Sprintf("%s:%d:%d", s.File, s.Line, s.Column)
	}
	return fmt.Sprintf("%d:%d", s.Line, s.Column)
}

// RefLocation tracks both where a $ref is defined and where it points.
// This enables precise error reporting for reference-related issues.
type RefLocation struct {
	// Origin is where the $ref is written in the source
	Origin SourceLocation
	// Target is where the referenced content is defined
	Target SourceLocation
	// TargetRef is the $ref string value (e.g., "#/components/schemas/Pet")
	TargetRef string
}

// SourceMap provides JSON path to source location mapping.
// It enables looking up the original source position for any element
// in a parsed OpenAPI document.
//
// The SourceMap is built during parsing when WithSourceMap(true) is used.
// It uses JSON path notation for keys (e.g., "$.paths./users.get.responses.200").
type SourceMap struct {
	// locations maps JSON paths to their source locations (value positions)
	locations map[string]SourceLocation
	// keyLocations maps JSON paths to their key positions (for map keys)
	// This is useful for "unknown field" errors that should point at the key
	keyLocations map[string]SourceLocation
	// refs tracks $ref locations: maps from the path containing the $ref
	// to information about both the ref origin and target
	refs map[string]RefLocation
}

// NewSourceMap creates an empty SourceMap.
func NewSourceMap() *SourceMap {
	return &SourceMap{
		locations:    make(map[string]SourceLocation),
		keyLocations: make(map[string]SourceLocation),
		refs:         make(map[string]RefLocation),
	}
}

// Get returns the source location for a JSON path.
// Returns a zero SourceLocation if the path is not found.
func (sm *SourceMap) Get(path string) SourceLocation {
	if sm == nil {
		return SourceLocation{}
	}
	return sm.locations[path]
}

// GetKey returns the source location of a map key at the given path.
// This is useful for errors about the key itself (e.g., "unknown field").
// Returns a zero SourceLocation if the path is not found.
func (sm *SourceMap) GetKey(path string) SourceLocation {
	if sm == nil {
		return SourceLocation{}
	}
	return sm.keyLocations[path]
}

// GetRef returns the reference location information for a path containing a $ref.
// Returns a zero RefLocation if no $ref exists at the path.
func (sm *SourceMap) GetRef(path string) RefLocation {
	if sm == nil {
		return RefLocation{}
	}
	return sm.refs[path]
}

// Has returns true if the path exists in the source map.
func (sm *SourceMap) Has(path string) bool {
	if sm == nil {
		return false
	}
	_, ok := sm.locations[path]
	return ok
}

// Len returns the number of paths in the source map.
func (sm *SourceMap) Len() int {
	if sm == nil {
		return 0
	}
	return len(sm.locations)
}

// Paths returns all JSON paths in the source map, sorted alphabetically.
// Returns nil if the receiver is nil.
func (sm *SourceMap) Paths() []string {
	if sm == nil {
		return nil
	}
	paths := make([]string, 0, len(sm.locations))
	for path := range sm.locations {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// Copy creates a deep copy of the SourceMap.
// Returns nil if the receiver is nil.
func (sm *SourceMap) Copy() *SourceMap {
	if sm == nil {
		return nil
	}
	result := NewSourceMap()
	for path, loc := range sm.locations {
		result.locations[path] = loc
	}
	for path, loc := range sm.keyLocations {
		result.keyLocations[path] = loc
	}
	for path, ref := range sm.refs {
		result.refs[path] = ref
	}
	return result
}

// set adds a location to the source map.
func (sm *SourceMap) set(path string, loc SourceLocation) {
	if sm == nil {
		return
	}
	if sm.locations == nil {
		sm.locations = make(map[string]SourceLocation)
	}
	sm.locations[path] = loc
}

// setKey adds a key location to the source map.
func (sm *SourceMap) setKey(path string, loc SourceLocation) {
	if sm == nil {
		return
	}
	if sm.keyLocations == nil {
		sm.keyLocations = make(map[string]SourceLocation)
	}
	sm.keyLocations[path] = loc
}

// setRef adds a reference location to the source map.
func (sm *SourceMap) setRef(path string, ref RefLocation) {
	if sm == nil {
		return
	}
	if sm.refs == nil {
		sm.refs = make(map[string]RefLocation)
	}
	sm.refs[path] = ref
}

// Merge combines another SourceMap into this one.
// Locations from the other map overwrite existing locations with the same path.
// Does nothing if either receiver or other is nil.
func (sm *SourceMap) Merge(other *SourceMap) {
	if sm == nil || other == nil {
		return
	}
	if sm.locations == nil {
		sm.locations = make(map[string]SourceLocation)
	}
	if sm.keyLocations == nil {
		sm.keyLocations = make(map[string]SourceLocation)
	}
	if sm.refs == nil {
		sm.refs = make(map[string]RefLocation)
	}
	for path, loc := range other.locations {
		sm.locations[path] = loc
	}
	for path, loc := range other.keyLocations {
		sm.keyLocations[path] = loc
	}
	for path, ref := range other.refs {
		sm.refs[path] = ref
	}
}

// buildSourceMap walks a yaml.Node tree and builds a SourceMap
// correlating JSON paths to source locations.
func buildSourceMap(root *yaml.Node, sourcePath string) *SourceMap {
	sm := NewSourceMap()
	if root == nil {
		return sm
	}
	walkNode(root, "$", sm, sourcePath)
	return sm
}

// walkNode recursively walks a yaml.Node tree, recording source locations.
func walkNode(node *yaml.Node, path string, sm *SourceMap, file string) {
	if node == nil {
		return
	}

	// Record this node's location
	sm.set(path, SourceLocation{
		Line:   node.Line,
		Column: node.Column,
		File:   file,
	})

	switch node.Kind {
	case yaml.DocumentNode:
		// Document node wraps the root content
		if len(node.Content) > 0 {
			walkNode(node.Content[0], path, sm, file)
		}

	case yaml.MappingNode:
		// Content alternates: key, value, key, value...
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			key := keyNode.Value
			childPath := buildChildPath(path, key)

			// Record key location (useful for "unknown field" errors)
			sm.setKey(childPath, SourceLocation{
				Line:   keyNode.Line,
				Column: keyNode.Column,
				File:   file,
			})

			// Track $ref specially
			if key == "$ref" && valNode.Kind == yaml.ScalarNode {
				sm.setRef(path, RefLocation{
					Origin: SourceLocation{
						Line:   valNode.Line,
						Column: valNode.Column,
						File:   file,
					},
					TargetRef: valNode.Value,
					// Target will be populated during reference resolution
				})
			}

			walkNode(valNode, childPath, sm, file)
		}

	case yaml.SequenceNode:
		// Array elements
		for i, child := range node.Content {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			walkNode(child, childPath, sm, file)
		}

	case yaml.ScalarNode, yaml.AliasNode:
		// Already recorded above, nothing more to do
	}
}

// buildChildPath constructs a JSON path for a child element.
// Handles special characters in keys by using bracket notation.
func buildChildPath(parent, key string) string {
	// Keys containing special characters need bracket notation
	if needsBracketNotation(key) {
		// Escape single quotes in the key
		escaped := strings.ReplaceAll(key, "'", "\\'")
		return fmt.Sprintf("%s['%s']", parent, escaped)
	}
	return parent + "." + key
}

// needsBracketNotation returns true if the key contains characters
// that require bracket notation in JSON paths.
func needsBracketNotation(key string) bool {
	// Keys starting with a digit, containing dots, brackets, quotes,
	// or whitespace need bracket notation
	if len(key) == 0 {
		return true
	}
	for i, r := range key {
		if i == 0 && r >= '0' && r <= '9' {
			return true
		}
		switch r {
		case '.', '[', ']', '\'', '"', ' ', '\t', '\n', '\r':
			return true
		}
	}
	return false
}

// updateSourceMapFilePath updates all file paths in a SourceMap to use the given path.
// This is used when parsing from readers or bytes where the source path is set after parsing.
func updateSourceMapFilePath(sm *SourceMap, newPath string) {
	if sm == nil {
		return
	}

	// Update locations
	for path, loc := range sm.locations {
		loc.File = newPath
		sm.locations[path] = loc
	}

	// Update key locations
	for path, loc := range sm.keyLocations {
		loc.File = newPath
		sm.keyLocations[path] = loc
	}

	// Update ref origins
	for path, ref := range sm.refs {
		ref.Origin.File = newPath
		sm.refs[path] = ref
	}
}
