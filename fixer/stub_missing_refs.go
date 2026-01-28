// This file implements stubbing for missing local $ref targets.
// When a document references schemas or responses that don't exist,
// this fixer creates empty stub definitions to make the document structurally valid.

package fixer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// StubConfig configures how missing reference stubs are created.
type StubConfig struct {
	// ResponseDescription is the description text for stub responses.
	// If empty, defaults to DefaultStubConfig().ResponseDescription.
	ResponseDescription string
}

// DefaultStubConfig returns the default configuration for stub creation.
func DefaultStubConfig() StubConfig {
	return StubConfig{
		ResponseDescription: "Auto-generated stub for missing reference",
	}
}

// isLocalRef returns true if the reference is a local JSON pointer (starts with "#/").
// External references (file paths or URLs) are not stubbed.
func isLocalRef(ref string) bool {
	return strings.HasPrefix(ref, "#/")
}

// extractResponseNameFromRef extracts the response name from a reference path.
// Returns empty string if the reference is not a response reference.
func extractResponseNameFromRef(ref string, version parser.OASVersion) string {
	var prefix string
	if version == parser.OASVersion20 {
		prefix = pathutil.RefPrefixResponses
	} else {
		prefix = pathutil.RefPrefixResponses3
	}

	if name, found := strings.CutPrefix(ref, prefix); found {
		return name
	}
	return ""
}

// stubMissingRefsOAS2 creates stub definitions for unresolved local references
// in an OAS 2.0 document.
func (f *Fixer) stubMissingRefsOAS2(doc *parser.OAS2Document, result *FixResult) {
	if doc == nil {
		return
	}

	// Collect all refs in the document
	collector := NewRefCollector()
	collector.CollectOAS2(doc)

	// Check schema refs and stub missing ones
	// Sort refs for deterministic output order
	schemaRefs := make([]string, 0, len(collector.RefsByType[RefTypeSchema]))
	for ref := range collector.RefsByType[RefTypeSchema] {
		schemaRefs = append(schemaRefs, ref)
	}
	sort.Strings(schemaRefs)
	for _, ref := range schemaRefs {
		if !isLocalRef(ref) {
			continue
		}
		name := ExtractSchemaNameFromRef(ref, parser.OASVersion20)
		if name == "" {
			continue
		}
		// Check if schema exists
		if doc.Definitions == nil || doc.Definitions[name] == nil {
			f.stubSchemaOAS2(doc, name, result)
		}
	}

	// Check response refs and stub missing ones
	// Sort refs for deterministic output order
	responseRefs := make([]string, 0, len(collector.RefsByType[RefTypeResponse]))
	for ref := range collector.RefsByType[RefTypeResponse] {
		responseRefs = append(responseRefs, ref)
	}
	sort.Strings(responseRefs)
	for _, ref := range responseRefs {
		if !isLocalRef(ref) {
			continue
		}
		name := extractResponseNameFromRef(ref, parser.OASVersion20)
		if name == "" {
			continue
		}
		// Check if response exists
		if doc.Responses == nil || doc.Responses[name] == nil {
			f.stubResponseOAS2(doc, name, result)
		}
	}
}

// stubSchemaOAS2 creates a stub schema definition in an OAS 2.0 document.
func (f *Fixer) stubSchemaOAS2(doc *parser.OAS2Document, name string, result *FixResult) {
	// Initialize definitions map if nil
	if doc.Definitions == nil {
		doc.Definitions = make(map[string]*parser.Schema)
	}

	// Create empty stub schema
	stub := &parser.Schema{}

	// Add to definitions
	doc.Definitions[name] = stub

	// Record fix
	fix := Fix{
		Type:        FixTypeStubMissingRef,
		Path:        fmt.Sprintf("definitions.%s", name),
		Description: "Created stub schema for missing reference " + pathutil.DefinitionRef(name),
		Before:      nil,
		After:       stub,
	}
	f.populateFixLocation(&fix)
	result.Fixes = append(result.Fixes, fix)
	result.FixCount++
}

// stubResponseOAS2 creates a stub response definition in an OAS 2.0 document.
func (f *Fixer) stubResponseOAS2(doc *parser.OAS2Document, name string, result *FixResult) {
	// Initialize responses map if nil
	if doc.Responses == nil {
		doc.Responses = make(map[string]*parser.Response)
	}

	// Get description from config, use default if empty
	description := f.StubConfig.ResponseDescription
	if description == "" {
		description = DefaultStubConfig().ResponseDescription
	}

	// Create stub response with required description
	stub := &parser.Response{
		Description: description,
	}

	// Add to responses
	doc.Responses[name] = stub

	// Record fix
	fix := Fix{
		Type:        FixTypeStubMissingRef,
		Path:        fmt.Sprintf("responses.%s", name),
		Description: "Created stub response for missing reference " + pathutil.ResponseRef(name, true),
		Before:      nil,
		After:       stub,
	}
	f.populateFixLocation(&fix)
	result.Fixes = append(result.Fixes, fix)
	result.FixCount++
}

// stubMissingRefsOAS3 creates stub definitions for unresolved local references
// in an OAS 3.x document.
func (f *Fixer) stubMissingRefsOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc == nil {
		return
	}

	// Collect all refs in the document
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Check schema refs and stub missing ones
	// Sort refs for deterministic output order
	schemaRefs := make([]string, 0, len(collector.RefsByType[RefTypeSchema]))
	for ref := range collector.RefsByType[RefTypeSchema] {
		schemaRefs = append(schemaRefs, ref)
	}
	sort.Strings(schemaRefs)
	for _, ref := range schemaRefs {
		if !isLocalRef(ref) {
			continue
		}
		name := ExtractSchemaNameFromRef(ref, doc.OASVersion)
		if name == "" {
			continue
		}
		// Check if schema exists
		if doc.Components == nil || doc.Components.Schemas == nil || doc.Components.Schemas[name] == nil {
			f.stubSchemaOAS3(doc, name, result)
		}
	}

	// Check response refs and stub missing ones
	// Sort refs for deterministic output order
	responseRefs := make([]string, 0, len(collector.RefsByType[RefTypeResponse]))
	for ref := range collector.RefsByType[RefTypeResponse] {
		responseRefs = append(responseRefs, ref)
	}
	sort.Strings(responseRefs)
	for _, ref := range responseRefs {
		if !isLocalRef(ref) {
			continue
		}
		name := extractResponseNameFromRef(ref, doc.OASVersion)
		if name == "" {
			continue
		}
		// Check if response exists
		if doc.Components == nil || doc.Components.Responses == nil || doc.Components.Responses[name] == nil {
			f.stubResponseOAS3(doc, name, result)
		}
	}
}

// stubSchemaOAS3 creates a stub schema definition in an OAS 3.x document.
func (f *Fixer) stubSchemaOAS3(doc *parser.OAS3Document, name string, result *FixResult) {
	// Initialize Components if nil
	if doc.Components == nil {
		doc.Components = &parser.Components{}
	}

	// Initialize schemas map if nil
	if doc.Components.Schemas == nil {
		doc.Components.Schemas = make(map[string]*parser.Schema)
	}

	// Create empty stub schema
	stub := &parser.Schema{}

	// Add to schemas
	doc.Components.Schemas[name] = stub

	// Record fix
	fix := Fix{
		Type:        FixTypeStubMissingRef,
		Path:        fmt.Sprintf("components.schemas.%s", name),
		Description: "Created stub schema for missing reference " + pathutil.SchemaRef(name),
		Before:      nil,
		After:       stub,
	}
	f.populateFixLocation(&fix)
	result.Fixes = append(result.Fixes, fix)
	result.FixCount++
}

// stubResponseOAS3 creates a stub response definition in an OAS 3.x document.
func (f *Fixer) stubResponseOAS3(doc *parser.OAS3Document, name string, result *FixResult) {
	// Initialize Components if nil
	if doc.Components == nil {
		doc.Components = &parser.Components{}
	}

	// Initialize responses map if nil
	if doc.Components.Responses == nil {
		doc.Components.Responses = make(map[string]*parser.Response)
	}

	// Get description from config, use default if empty
	description := f.StubConfig.ResponseDescription
	if description == "" {
		description = DefaultStubConfig().ResponseDescription
	}

	// Create stub response with required description
	stub := &parser.Response{
		Description: description,
	}

	// Add to responses
	doc.Components.Responses[name] = stub

	// Record fix
	fix := Fix{
		Type:        FixTypeStubMissingRef,
		Path:        fmt.Sprintf("components.responses.%s", name),
		Description: "Created stub response for missing reference " + pathutil.ResponseRef(name, false),
		Before:      nil,
		After:       stub,
	}
	f.populateFixLocation(&fix)
	result.Fixes = append(result.Fixes, fix)
	result.FixCount++
}
