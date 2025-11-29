package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erraggy/oastools/parser"
	"gopkg.in/yaml.v3"
)

// Builder is the main entry point for constructing OAS documents.
// It maintains internal state for accumulated components and reflection cache.
//
// Concurrency: Builder instances are not safe for concurrent use.
// Create separate Builder instances for concurrent operations.
type Builder struct {
	// Configuration
	version parser.OASVersion

	// Document sections
	info     *parser.Info
	servers  []*parser.Server
	paths    parser.Paths
	tags     []*parser.Tag
	security []parser.SecurityRequirement

	// Components (tracked separately for deduplication)
	schemas         map[string]*parser.Schema
	responses       map[string]*parser.Response
	parameters      map[string]*parser.Parameter
	requestBodies   map[string]*parser.RequestBody
	securitySchemes map[string]*parser.SecurityScheme

	// Reflection cache for schema generation
	schemaCache *schemaCache

	// Tracking
	operationIDs map[string]bool // Track used operation IDs for uniqueness
	errors       []error         // Accumulated validation errors
}

// New creates a new Builder instance for the specified OAS version.
// For type-safe document building, prefer NewOAS2() or NewOAS3() with
// their corresponding BuildOAS2() or BuildOAS3() methods.
//
// Example:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
func New(version parser.OASVersion) *Builder {
	return &Builder{
		version:         version,
		paths:           make(parser.Paths),
		schemas:         make(map[string]*parser.Schema),
		responses:       make(map[string]*parser.Response),
		parameters:      make(map[string]*parser.Parameter),
		requestBodies:   make(map[string]*parser.RequestBody),
		securitySchemes: make(map[string]*parser.SecurityScheme),
		schemaCache:     newSchemaCache(),
		operationIDs:    make(map[string]bool),
		errors:          make([]error, 0),
	}
}

// NewOAS2 creates a new Builder for OAS 2.0 (Swagger) documents.
// Use BuildOAS2() to get the typed *parser.OAS2Document.
//
// Schema references use "#/definitions/" format.
//
// Example:
//
//	spec := builder.NewOAS2().
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS2()
func NewOAS2() *Builder {
	return New(parser.OASVersion20)
}

// NewOAS3 creates a new Builder for OAS 3.x documents.
// Use BuildOAS3() to get the typed *parser.OAS3Document.
//
// Schema references use "#/components/schemas/" format.
// The version parameter specifies the exact OAS 3.x version (e.g., OASVersion320).
//
// Example:
//
//	spec := builder.NewOAS3(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS3()
func NewOAS3(version parser.OASVersion) *Builder {
	// Validate that it's an OAS 3.x version
	if version == parser.OASVersion20 || version == parser.Unknown {
		// Default to latest OAS 3.x if invalid version provided
		version = parser.OASVersion320
	}
	return New(version)
}

// NewWithInfo creates a Builder with pre-configured Info.
// For type-safe document building, prefer NewOAS2() or NewOAS3() with SetInfo().
//
// Example:
//
//	info := &parser.Info{Title: "My API", Version: "1.0.0"}
//	spec := builder.NewWithInfo(parser.OASVersion320, info)
func NewWithInfo(version parser.OASVersion, info *parser.Info) *Builder {
	b := New(version)
	b.info = info
	return b
}

// SetInfo sets the Info object for the document.
func (b *Builder) SetInfo(info *parser.Info) *Builder {
	b.info = info
	return b
}

// SetTitle sets the title in the Info object.
func (b *Builder) SetTitle(title string) *Builder {
	if b.info == nil {
		b.info = &parser.Info{}
	}
	b.info.Title = title
	return b
}

// SetVersion sets the version in the Info object.
// Note: This is the API version, not the OpenAPI specification version.
func (b *Builder) SetVersion(version string) *Builder {
	if b.info == nil {
		b.info = &parser.Info{}
	}
	b.info.Version = version
	return b
}

// SetDescription sets the description in the Info object.
func (b *Builder) SetDescription(desc string) *Builder {
	if b.info == nil {
		b.info = &parser.Info{}
	}
	b.info.Description = desc
	return b
}

// SetTermsOfService sets the terms of service URL in the Info object.
func (b *Builder) SetTermsOfService(url string) *Builder {
	if b.info == nil {
		b.info = &parser.Info{}
	}
	b.info.TermsOfService = url
	return b
}

// SetContact sets the contact information in the Info object.
func (b *Builder) SetContact(contact *parser.Contact) *Builder {
	if b.info == nil {
		b.info = &parser.Info{}
	}
	b.info.Contact = contact
	return b
}

// SetLicense sets the license information in the Info object.
func (b *Builder) SetLicense(license *parser.License) *Builder {
	if b.info == nil {
		b.info = &parser.Info{}
	}
	b.info.License = license
	return b
}

// Build creates the final OAS document based on the version specified in New().
// Returns either *parser.OAS2Document or *parser.OAS3Document depending on the version.
// Returns an error if required fields are missing or validation errors occurred.
//
// For type-safe access without casting, use BuildOAS2() or BuildOAS3() instead.
func (b *Builder) Build() (any, error) {
	if b.version == parser.OASVersion20 {
		return b.BuildOAS2()
	}
	return b.BuildOAS3()
}

// BuildOAS2 creates an OAS 2.0 (Swagger) document.
// Returns an error if the builder was created with an OAS 3.x version,
// or if required fields are missing.
//
// Example:
//
//	spec := builder.New(parser.OASVersion20).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS2()
//	// doc is *parser.OAS2Document - no type assertion needed
func (b *Builder) BuildOAS2() (*parser.OAS2Document, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	if b.version != parser.OASVersion20 {
		return nil, fmt.Errorf("builder: BuildOAS2() called but builder was created with version %s; use BuildOAS3() instead", b.version)
	}

	// Build paths - only include if non-empty
	var paths parser.Paths
	if len(b.paths) > 0 {
		paths = b.paths
	}

	// Create document
	doc := &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: b.version,
		Info:       b.info,
		Paths:      paths,
		Tags:       b.tags,
		Security:   b.security,
	}

	// Add definitions (schemas)
	if len(b.schemas) > 0 {
		doc.Definitions = b.schemas
	}

	// Add parameters
	if len(b.parameters) > 0 {
		doc.Parameters = b.parameters
	}

	// Add responses
	if len(b.responses) > 0 {
		doc.Responses = b.responses
	}

	// Add security definitions
	if len(b.securitySchemes) > 0 {
		doc.SecurityDefinitions = b.securitySchemes
	}

	return doc, nil
}

// BuildOAS3 creates an OAS 3.x document.
// Returns an error if the builder was created with OAS 2.0 version,
// or if required fields are missing.
//
// Example:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS3()
//	// doc is *parser.OAS3Document - no type assertion needed
func (b *Builder) BuildOAS3() (*parser.OAS3Document, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	if b.version == parser.OASVersion20 {
		return nil, fmt.Errorf("builder: BuildOAS3() called but builder was created with OAS 2.0; use BuildOAS2() instead")
	}

	// Build components
	var components *parser.Components
	if len(b.schemas) > 0 || len(b.responses) > 0 || len(b.parameters) > 0 ||
		len(b.requestBodies) > 0 || len(b.securitySchemes) > 0 {
		components = &parser.Components{}
		if len(b.schemas) > 0 {
			components.Schemas = b.schemas
		}
		if len(b.responses) > 0 {
			components.Responses = b.responses
		}
		if len(b.parameters) > 0 {
			components.Parameters = b.parameters
		}
		if len(b.requestBodies) > 0 {
			components.RequestBodies = b.requestBodies
		}
		if len(b.securitySchemes) > 0 {
			components.SecuritySchemes = b.securitySchemes
		}
	}

	// Build paths - only include if non-empty
	var paths parser.Paths
	if len(b.paths) > 0 {
		paths = b.paths
	}

	// Create document
	doc := &parser.OAS3Document{
		OpenAPI:    b.version.String(),
		OASVersion: b.version,
		Info:       b.info,
		Servers:    b.servers,
		Paths:      paths,
		Components: components,
		Tags:       b.tags,
		Security:   b.security,
	}

	return doc, nil
}

// validate checks that all required fields are present.
func (b *Builder) validate() error {
	// Check accumulated errors
	if len(b.errors) > 0 {
		var errMsgs []string
		for _, err := range b.errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("builder: %d error(s): %s", len(b.errors), strings.Join(errMsgs, "; "))
	}

	// Validate required fields
	if b.info == nil {
		return fmt.Errorf("builder: info is required")
	}
	if b.info.Title == "" {
		return fmt.Errorf("builder: info.title is required")
	}
	if b.info.Version == "" {
		return fmt.Errorf("builder: info.version is required")
	}

	return nil
}

// BuildResult creates a ParseResult for compatibility with other packages.
// This is useful for validating the built document with the validator package.
func (b *Builder) BuildResult() (*parser.ParseResult, error) {
	doc, err := b.Build()
	if err != nil {
		return nil, err
	}

	return &parser.ParseResult{
		SourcePath:   "builder",
		SourceFormat: parser.SourceFormatYAML,
		Version:      b.version.String(),
		OASVersion:   b.version,
		Document:     doc,
		Errors:       make([]error, 0),
		Warnings:     make([]string, 0),
	}, nil
}

// MarshalYAML returns the document as YAML bytes.
func (b *Builder) MarshalYAML() ([]byte, error) {
	doc, err := b.Build()
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(doc)
}

// MarshalJSON returns the document as JSON bytes.
func (b *Builder) MarshalJSON() ([]byte, error) {
	doc, err := b.Build()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}

// outputFileMode is the file permission mode for output files (owner read/write only)
const outputFileMode = 0600

// WriteFile writes the document to a file.
// The format is inferred from the file extension (.json for JSON, .yaml/.yml for YAML).
func (b *Builder) WriteFile(path string) error {
	var data []byte
	var err error

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		data, err = b.MarshalJSON()
	case ".yaml", ".yml":
		data, err = b.MarshalYAML()
	default:
		// Default to YAML
		data, err = b.MarshalYAML()
	}

	if err != nil {
		return fmt.Errorf("builder: failed to marshal document: %w", err)
	}

	if err := os.WriteFile(path, data, outputFileMode); err != nil {
		return fmt.Errorf("builder: failed to write file: %w", err)
	}

	return nil
}

// getOrCreatePathItem gets or creates a PathItem for the given path.
func (b *Builder) getOrCreatePathItem(path string) *parser.PathItem {
	if pathItem, exists := b.paths[path]; exists {
		return pathItem
	}
	pathItem := &parser.PathItem{}
	b.paths[path] = pathItem
	return pathItem
}

// RegisterType registers a Go type and returns a $ref to it.
// The schema is automatically generated via reflection and added to components.schemas.
func (b *Builder) RegisterType(v any) *parser.Schema {
	return b.generateSchema(v)
}

// RegisterTypeAs registers a Go type with a custom schema name.
func (b *Builder) RegisterTypeAs(name string, v any) *parser.Schema {
	schema := b.generateSchemaInternal(v, name)
	return schema
}

// FromDocument creates a builder from an existing OAS3Document.
// This allows modifying an existing document by adding operations.
func FromDocument(doc *parser.OAS3Document) *Builder {
	version, ok := parser.ParseVersion(doc.OpenAPI)
	if !ok {
		version = parser.OASVersion320 // Default to latest
	}

	b := New(version)
	b.info = doc.Info
	b.servers = doc.Servers
	b.tags = doc.Tags
	b.security = doc.Security

	// Copy paths
	if doc.Paths != nil {
		for path, item := range doc.Paths {
			b.paths[path] = item
		}
	}

	// Copy components
	if doc.Components != nil {
		if doc.Components.Schemas != nil {
			for name, schema := range doc.Components.Schemas {
				b.schemas[name] = schema
			}
		}
		if doc.Components.Responses != nil {
			for name, resp := range doc.Components.Responses {
				b.responses[name] = resp
			}
		}
		if doc.Components.Parameters != nil {
			for name, param := range doc.Components.Parameters {
				b.parameters[name] = param
			}
		}
		if doc.Components.RequestBodies != nil {
			for name, rb := range doc.Components.RequestBodies {
				b.requestBodies[name] = rb
			}
		}
		if doc.Components.SecuritySchemes != nil {
			for name, ss := range doc.Components.SecuritySchemes {
				b.securitySchemes[name] = ss
			}
		}
	}

	return b
}
