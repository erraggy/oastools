package builder

import (
	"reflect"

	"github.com/erraggy/oastools/parser"
)

// schemaCache manages reflection-based schema generation caching.
// It prevents duplicate generation and handles circular references.
type schemaCache struct {
	byType     map[reflect.Type]*parser.Schema // Type → Schema
	byName     map[string]reflect.Type         // Name → Type (for reverse lookup)
	nameByType map[reflect.Type]string         // Type → Name (for O(1) reverse lookup)
	inProgress map[reflect.Type]bool           // Circular reference detection
}

// newSchemaCache creates a new schema cache.
func newSchemaCache() *schemaCache {
	return &schemaCache{
		byType:     make(map[reflect.Type]*parser.Schema),
		byName:     make(map[string]reflect.Type),
		nameByType: make(map[reflect.Type]string),
		inProgress: make(map[reflect.Type]bool),
	}
}

// get returns a cached schema for the given type, or nil if not cached.
func (c *schemaCache) get(t reflect.Type) *parser.Schema {
	return c.byType[t]
}

// set caches a schema for the given type and name.
func (c *schemaCache) set(t reflect.Type, name string, schema *parser.Schema) {
	c.byType[t] = schema
	c.byName[name] = t
	c.nameByType[t] = name
}

// isInProgress returns true if the type is currently being processed.
// This is used for circular reference detection.
func (c *schemaCache) isInProgress(t reflect.Type) bool {
	return c.inProgress[t]
}

// markInProgress marks a type as being processed.
func (c *schemaCache) markInProgress(t reflect.Type) {
	c.inProgress[t] = true
}

// clearInProgress removes the in-progress mark for a type.
func (c *schemaCache) clearInProgress(t reflect.Type) {
	delete(c.inProgress, t)
}

// getNameForType returns the cached name for a type, or empty string if not cached.
// Uses O(1) reverse mapping for performance.
func (c *schemaCache) getNameForType(t reflect.Type) string {
	return c.nameByType[t]
}

// hasName returns true if a name is already registered.
func (c *schemaCache) hasName(name string) bool {
	_, exists := c.byName[name]
	return exists
}

// getTypeForName returns the type registered for a name, or nil if not found.
func (c *schemaCache) getTypeForName(name string) reflect.Type {
	return c.byName[name]
}
