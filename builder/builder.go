package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
	"go.yaml.in/yaml/v4"
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
	info         *parser.Info
	servers      []*parser.Server
	paths        parser.Paths
	tags         []*parser.Tag
	security     []parser.SecurityRequirement
	externalDocs *parser.ExternalDocs
	webhooks     map[string]*parser.PathItem // OAS 3.1+ only

	// Components (tracked separately for deduplication)
	schemas         map[string]*parser.Schema
	responses       map[string]*parser.Response
	parameters      map[string]*parser.Parameter
	requestBodies   map[string]*parser.RequestBody
	securitySchemes map[string]*parser.SecurityScheme

	// Reflection cache for schema generation
	schemaCache *schemaCache

	// Tracking
	operationIDs         map[string]bool              // Track used operation IDs for uniqueness
	operationIDLocations map[string]operationLocation // Track where each operationID was first defined
	errors               []error                      // Accumulated errors

	// Schema naming configuration
	namer       *schemaNamer
	configError error // Stores configuration errors (e.g., invalid templates)

	// Semantic deduplication
	dedupeEnabled bool              // Whether deduplication is enabled
	schemaAliases map[string]string // Maps alias names to canonical names
}

// New creates a new Builder instance for the specified OAS version.
// Use BuildOAS2() for OAS 2.0 (Swagger) or BuildOAS3() for OAS 3.x documents.
//
// Options can be provided to customize schema naming:
//
//	// Use PascalCase naming (e.g., "ModelsUser" instead of "models.User")
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
//	)
//
// The builder does not perform OAS specification validation. Use the validator
// package to validate built documents.
//
// Example:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS3()
func New(version parser.OASVersion, opts ...BuilderOption) *Builder {
	// Apply options to get configuration
	cfg := defaultBuilderConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Create namer from config
	namer := newSchemaNamer()
	namer.strategy = cfg.namingStrategy
	namer.genericConfig = cfg.genericConfig
	namer.template = cfg.namingTemplate
	namer.fn = cfg.namingFunc

	return &Builder{
		version:              version,
		paths:                make(parser.Paths),
		webhooks:             make(map[string]*parser.PathItem),
		schemas:              make(map[string]*parser.Schema),
		responses:            make(map[string]*parser.Response),
		parameters:           make(map[string]*parser.Parameter),
		requestBodies:        make(map[string]*parser.RequestBody),
		securitySchemes:      make(map[string]*parser.SecurityScheme),
		schemaCache:          newSchemaCache(),
		operationIDs:         make(map[string]bool),
		operationIDLocations: make(map[string]operationLocation),
		errors:               make([]error, 0),
		namer:                namer,
		configError:          cfg.templateError,
		dedupeEnabled:        cfg.semanticDeduplication,
		schemaAliases:        make(map[string]string),
	}
}

// NewWithInfo creates a Builder with pre-configured Info.
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

// DeduplicateSchemas identifies semantically identical schemas and consolidates
// them to a single canonical schema. This method should be called after all
// schemas have been added but before Build*() methods.
//
// Schemas are considered semantically identical if they have the same structural
// properties (type, format, properties, constraints, etc.), ignoring metadata
// fields like title, description, and examples.
//
// The canonical schema name is selected alphabetically. For example, if
// "Address" and "Location" schemas are identical, "Address" becomes canonical
// and all references to "Location" are rewritten to point to "Address".
//
// Returns the Builder for method chaining.
//
// Note: This method is automatically called by Build*() methods when
// WithSemanticDeduplication(true) is set. Call it manually only if you need
// to inspect schemaAliases before building.
func (b *Builder) DeduplicateSchemas() *Builder {
	if len(b.schemas) < 2 {
		return b
	}

	// Create compare function using joiner's deep comparison
	compare := func(left, right *parser.Schema) bool {
		result := joiner.CompareSchemas(left, right, joiner.EquivalenceModeDeep)
		return result.Equivalent
	}

	// Create deduplicator and run deduplication
	config := schemautil.DefaultDeduplicationConfig()
	deduper := schemautil.NewSchemaDeduplicator(config, compare)

	result, err := deduper.Deduplicate(b.schemas)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("builder: schema deduplication failed: %w", err))
		return b
	}

	// Apply results
	b.schemas = result.CanonicalSchemas
	b.schemaAliases = result.Aliases

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

// SetExternalDocs sets the external documentation for the document.
// This is used for providing additional documentation at the document level.
func (b *Builder) SetExternalDocs(externalDocs *parser.ExternalDocs) *Builder {
	b.externalDocs = externalDocs
	return b
}

// AddWebhook adds a webhook to the specification (OAS 3.1+ only).
// Webhooks are callbacks that are triggered by the API provider.
// Returns an error during Build if used with OAS versions earlier than 3.1.
//
// Example:
//
//	spec.AddWebhook("newUser", http.MethodPost,
//	    builder.WithResponse(http.StatusOK, UserCreatedEvent{}),
//	)
func (b *Builder) AddWebhook(name, method string, opts ...OperationOption) *Builder {
	// Create operation config with defaults
	cfg := &operationConfig{
		responses: make(map[string]*responseBuilder),
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	// Check for duplicate operation ID (shared namespace with operations)
	if cfg.operationID != "" {
		if first, exists := b.operationIDLocations[cfg.operationID]; exists {
			b.errors = append(b.errors, NewDuplicateWebhookOperationIDError(cfg.operationID, name, method, &first))
		} else {
			b.operationIDLocations[cfg.operationID] = operationLocation{
				Method:    method,
				Path:      name,
				IsWebhook: true,
			}
		}
		b.operationIDs[cfg.operationID] = true
	}

	// Unwrap and process request body
	var requestBody *parser.RequestBody
	if cfg.requestBody != nil {
		requestBody = cfg.requestBody.body
		if cfg.requestBody.bType != nil {
			schema := b.generateSchema(cfg.requestBody.bType)
			for contentType := range requestBody.Content {
				requestBody.Content[contentType].Schema = schema
			}
		}
	}

	// Unwrap and process responses
	responseMap := make(map[string]*parser.Response)
	for code, respBuilder := range cfg.responses {
		resp := respBuilder.response
		if respBuilder.rType != nil {
			schema := b.generateSchema(respBuilder.rType)
			for contentType := range resp.Content {
				resp.Content[contentType].Schema = schema
			}
		}
		responseMap[code] = resp
	}

	// Process parameters using shared helper
	parameters := b.processParameters(cfg.parameters)

	// Process form parameters (webhooks are OAS 3.1+ only, so always use request body)
	if len(cfg.formParams) > 0 {
		formSchema := b.buildFormParamSchema(cfg.formParams)
		contentType := "application/x-www-form-urlencoded"
		if hasFileParam(cfg.formParams) {
			contentType = "multipart/form-data"
		}
		requestBody = addFormParamsToRequestBody(requestBody, formSchema, contentType)
	}

	// Build responses object
	responses := buildResponsesFromMap(responseMap)

	// Build security
	var security []parser.SecurityRequirement
	if cfg.noSecurity {
		security = []parser.SecurityRequirement{{}}
	} else if len(cfg.security) > 0 {
		security = cfg.security
	}

	// Build Operation struct
	op := &parser.Operation{
		OperationID: cfg.operationID,
		Summary:     cfg.summary,
		Description: cfg.description,
		Tags:        cfg.tags,
		Parameters:  parameters,
		RequestBody: requestBody,
		Responses:   responses,
		Security:    security,
		Deprecated:  cfg.deprecated,
	}

	// Get or create PathItem for webhook
	pathItem, exists := b.webhooks[name]
	if !exists {
		pathItem = &parser.PathItem{}
		b.webhooks[name] = pathItem
	}

	// Assign operation to method
	b.setOperation(pathItem, method, name, op)

	return b
}

// BuildOAS2 creates an OAS 2.0 (Swagger) document.
// Returns an error if the builder was created with an OAS 3.x version,
// or if required fields are missing.
//
// The builder does not perform OAS specification validation. Use the validator
// package to validate built documents.
//
// Example:
//
//	spec := builder.New(parser.OASVersion20).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS2()
//	// doc is *parser.OAS2Document - no type assertion needed
func (b *Builder) BuildOAS2() (*parser.OAS2Document, error) {
	if b.configError != nil {
		return nil, fmt.Errorf("builder: configuration error: %w", b.configError)
	}

	if b.version != parser.OASVersion20 {
		return nil, fmt.Errorf("builder: BuildOAS2() called but builder was created with version %s; use BuildOAS3() instead", b.version)
	}

	// Apply semantic deduplication if enabled
	if b.dedupeEnabled {
		b.DeduplicateSchemas()
	}

	if err := b.checkErrors(); err != nil {
		return nil, err
	}

	// Build paths - only include if non-empty
	var paths parser.Paths
	if len(b.paths) > 0 {
		paths = b.paths
	}

	// Create document
	doc := &parser.OAS2Document{
		Swagger:      "2.0",
		OASVersion:   b.version,
		Info:         b.info,
		Paths:        paths,
		Tags:         b.tags,
		Security:     b.security,
		ExternalDocs: b.externalDocs,
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

	// Rewrite references if schema aliases exist
	if err := b.applySchemaAliases(doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// BuildOAS3 creates an OAS 3.x document.
// Returns an error if the builder was created with OAS 2.0 version,
// or if required fields are missing.
//
// The builder does not perform OAS specification validation. Use the validator
// package to validate built documents.
//
// Example:
//
//	spec := builder.New(parser.OASVersion320).
//		SetTitle("My API").
//		SetVersion("1.0.0")
//	doc, err := spec.BuildOAS3()
//	// doc is *parser.OAS3Document - no type assertion needed
func (b *Builder) BuildOAS3() (*parser.OAS3Document, error) {
	if b.configError != nil {
		return nil, fmt.Errorf("builder: configuration error: %w", b.configError)
	}

	if b.version == parser.OASVersion20 {
		return nil, fmt.Errorf("builder: BuildOAS3() called but builder was created with OAS 2.0; use BuildOAS2() instead")
	}

	// Apply semantic deduplication if enabled
	if b.dedupeEnabled {
		b.DeduplicateSchemas()
	}

	if err := b.checkErrors(); err != nil {
		return nil, err
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
		OpenAPI:      b.version.String(),
		OASVersion:   b.version,
		Info:         b.info,
		Servers:      b.servers,
		Paths:        paths,
		Components:   components,
		Tags:         b.tags,
		Security:     b.security,
		ExternalDocs: b.externalDocs,
	}

	// Add webhooks (OAS 3.1+ only)
	if len(b.webhooks) > 0 {
		doc.Webhooks = b.webhooks
	}

	// Rewrite references if schema aliases exist
	if err := b.applySchemaAliases(doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// applySchemaAliases rewrites schema references using the deduplicated alias map.
// This is shared between BuildOAS2 and BuildOAS3.
func (b *Builder) applySchemaAliases(doc interface{}) error {
	if len(b.schemaAliases) == 0 {
		return nil
	}
	rewriter := joiner.NewSchemaRewriter()
	for alias, canonical := range b.schemaAliases {
		rewriter.RegisterRename(alias, canonical, b.version)
	}
	if err := rewriter.RewriteDocument(doc); err != nil {
		return fmt.Errorf("builder: failed to rewrite deduplicated references: %w", err)
	}
	return nil
}

// checkErrors checks for accumulated errors during building.
// The builder does not perform OAS specification validation.
// Use the validator package to validate built documents.
//
// Returns a BuilderErrors collection if there are errors, which provides
// detailed locality information for each error including component type,
// HTTP method, path, and operationID context.
func (b *Builder) checkErrors() error {
	if len(b.errors) == 0 {
		return nil
	}

	// Convert to BuilderErrors for structured output
	builderErrs := make(BuilderErrors, 0, len(b.errors))
	for _, err := range b.errors {
		var be *BuilderError
		if errors.As(err, &be) {
			builderErrs = append(builderErrs, be)
		} else {
			// Wrap legacy errors (e.g., from ConstraintError)
			// Set only Cause to preserve the error chain for errors.Unwrap()
			// The Message is derived from Cause.Error() in BuilderError.Error()
			builderErrs = append(builderErrs, &BuilderError{
				Cause: err,
			})
		}
	}
	return builderErrs
}

// BuildResult creates a ParseResult for compatibility with other packages.
// This is useful for validating the built document with the validator package.
//
// The builder does not perform OAS specification validation. Use the validator
// package to validate built documents:
//
//	result, err := spec.BuildResult()
//	if err != nil { return err }
//	valResult, err := validator.ValidateParsed(result)
func (b *Builder) BuildResult() (*parser.ParseResult, error) {
	var doc any
	var err error

	if b.version == parser.OASVersion20 {
		doc, err = b.BuildOAS2()
	} else {
		doc, err = b.BuildOAS3()
	}
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
	var doc any
	var err error

	if b.version == parser.OASVersion20 {
		doc, err = b.BuildOAS2()
	} else {
		doc, err = b.BuildOAS3()
	}
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(doc)
}

// MarshalJSON returns the document as JSON bytes.
func (b *Builder) MarshalJSON() ([]byte, error) {
	var doc any
	var err error

	if b.version == parser.OASVersion20 {
		doc, err = b.BuildOAS2()
	} else {
		doc, err = b.BuildOAS3()
	}
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
//
// For OAS 2.0 documents, use FromOAS2Document instead.
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
	b.externalDocs = doc.ExternalDocs

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

// FromOAS2Document creates a builder from an existing OAS2Document (Swagger 2.0).
// This allows modifying an existing document by adding operations.
//
// For OAS 3.x documents, use FromDocument instead.
func FromOAS2Document(doc *parser.OAS2Document) *Builder {
	b := New(parser.OASVersion20)
	b.info = doc.Info
	b.tags = doc.Tags
	b.security = doc.Security
	b.externalDocs = doc.ExternalDocs

	// Copy paths
	if doc.Paths != nil {
		for path, item := range doc.Paths {
			b.paths[path] = item
		}
	}

	// Copy definitions (schemas)
	if doc.Definitions != nil {
		for name, schema := range doc.Definitions {
			b.schemas[name] = schema
		}
	}

	// Copy parameters
	if doc.Parameters != nil {
		for name, param := range doc.Parameters {
			b.parameters[name] = param
		}
	}

	// Copy responses
	if doc.Responses != nil {
		for name, resp := range doc.Responses {
			b.responses[name] = resp
		}
	}

	// Copy security definitions
	if doc.SecurityDefinitions != nil {
		for name, ss := range doc.SecurityDefinitions {
			b.securitySchemes[name] = ss
		}
	}

	return b
}

// addFormParamsToRequestBody adds form parameters to a request body.
// If requestBody is nil, a new one is created. Otherwise, the form content type is added.
// The contentType parameter specifies which content type to use (e.g., "application/x-www-form-urlencoded" or "multipart/form-data").
func addFormParamsToRequestBody(requestBody *parser.RequestBody, formSchema *parser.Schema, contentType string) *parser.RequestBody {
	if requestBody != nil {
		// Merge form parameters into existing request body
		if requestBody.Content == nil {
			requestBody.Content = make(map[string]*parser.MediaType)
		}
		requestBody.Content[contentType] = &parser.MediaType{
			Schema: formSchema,
		}
		return requestBody
	}

	// Create new request body for form parameters
	return &parser.RequestBody{
		Content: map[string]*parser.MediaType{
			contentType: {
				Schema: formSchema,
			},
		},
	}
}
