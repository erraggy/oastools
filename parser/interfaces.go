package parser

// DocumentAccessor provides a unified read-only interface for accessing
// common fields across OAS 2.0 and OAS 3.x documents. This interface
// abstracts away version-specific differences for fields that have
// semantic equivalence between versions.
//
// # Fields with identical structure across versions
//
//   - Info, Paths, Tags, Security, ExternalDocs
//
// # Fields with semantic equivalence (different locations, same meaning)
//
//   - Schemas: OAS 2.0 doc.Definitions vs OAS 3.x doc.Components.Schemas
//   - SecuritySchemes: OAS 2.0 doc.SecurityDefinitions vs OAS 3.x doc.Components.SecuritySchemes
//   - Parameters: OAS 2.0 doc.Parameters vs OAS 3.x doc.Components.Parameters
//   - Responses: OAS 2.0 doc.Responses vs OAS 3.x doc.Components.Responses
//
// # Accessing version-specific fields
//
// For version-specific fields (Servers, Webhooks, RequestBodies, etc.),
// use the version-specific document types directly via [ParseResult.OAS2Document]
// or [ParseResult.OAS3Document] methods on the ParseResult.
//
// # Return value semantics
//
// Methods returning maps (GetPaths, GetSchemas, etc.) return nil when the field
// is not set or when a required parent object is nil (e.g., OAS3 Components).
// These methods return direct references to the underlying data structures, so
// callers should avoid modifying the returned values unless they intend to mutate
// the document. If you need to modify the data, make a copy first.
//
// # Example usage
//
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	if accessor := result.AsAccessor(); accessor != nil {
//	    // Works for both OAS 2.0 and OAS 3.x
//	    for path, item := range accessor.GetPaths() {
//	        fmt.Println("Path:", path)
//	    }
//	    for name, schema := range accessor.GetSchemas() {
//	        fmt.Println("Schema:", name)
//	    }
//	}
type DocumentAccessor interface {
	// GetInfo returns the API metadata.
	// Returns nil if Info is not set.
	GetInfo() *Info

	// GetPaths returns the path items map.
	// Returns nil if Paths is not set.
	GetPaths() Paths

	// GetTags returns the tag definitions.
	// Returns nil if Tags is not set.
	GetTags() []*Tag

	// GetSecurity returns the global security requirements.
	// Returns nil if Security is not set.
	GetSecurity() []SecurityRequirement

	// GetExternalDocs returns the external documentation reference.
	// Returns nil if ExternalDocs is not set.
	GetExternalDocs() *ExternalDocs

	// GetSchemas returns the schema definitions.
	// For OAS 2.0: returns doc.Definitions
	// For OAS 3.x: returns doc.Components.Schemas
	// Returns nil if the schema container is not set (OAS2 Definitions nil, or
	// OAS3 Components nil or Components.Schemas nil). An empty map is returned
	// only when the container exists but has no entries.
	GetSchemas() map[string]*Schema

	// GetSecuritySchemes returns the security scheme definitions.
	// For OAS 2.0: returns doc.SecurityDefinitions
	// For OAS 3.x: returns doc.Components.SecuritySchemes
	// Returns nil if no security schemes are defined.
	GetSecuritySchemes() map[string]*SecurityScheme

	// GetParameters returns the reusable parameter definitions.
	// For OAS 2.0: returns doc.Parameters
	// For OAS 3.x: returns doc.Components.Parameters
	// Returns nil if no parameters are defined.
	GetParameters() map[string]*Parameter

	// GetResponses returns the reusable response definitions.
	// For OAS 2.0: returns doc.Responses
	// For OAS 3.x: returns doc.Components.Responses
	// Returns nil if no responses are defined.
	GetResponses() map[string]*Response

	// GetVersion returns the OASVersion enum for this document.
	GetVersion() OASVersion

	// GetVersionString returns the version string (e.g., "2.0", "3.0.3", "3.1.0").
	GetVersionString() string

	// SchemaRefPrefix returns the JSON reference prefix for schemas.
	// For OAS 2.0: returns "#/definitions/"
	// For OAS 3.x: returns "#/components/schemas/"
	SchemaRefPrefix() string
}

// Compile-time interface verification
var (
	_ DocumentAccessor = (*OAS2Document)(nil)
	_ DocumentAccessor = (*OAS3Document)(nil)
)

// ----- OAS2Document implementation -----

// GetInfo returns the API metadata for OAS 2.0 documents.
func (d *OAS2Document) GetInfo() *Info {
	if d == nil {
		return nil
	}
	return d.Info
}

// GetPaths returns the path items for OAS 2.0 documents.
func (d *OAS2Document) GetPaths() Paths {
	if d == nil {
		return nil
	}
	return d.Paths
}

// GetTags returns the tag definitions for OAS 2.0 documents.
func (d *OAS2Document) GetTags() []*Tag {
	if d == nil {
		return nil
	}
	return d.Tags
}

// GetSecurity returns the global security requirements for OAS 2.0 documents.
func (d *OAS2Document) GetSecurity() []SecurityRequirement {
	if d == nil {
		return nil
	}
	return d.Security
}

// GetExternalDocs returns the external documentation for OAS 2.0 documents.
func (d *OAS2Document) GetExternalDocs() *ExternalDocs {
	if d == nil {
		return nil
	}
	return d.ExternalDocs
}

// GetSchemas returns the schema definitions for OAS 2.0 documents.
// In OAS 2.0, schemas are stored in doc.Definitions.
func (d *OAS2Document) GetSchemas() map[string]*Schema {
	if d == nil {
		return nil
	}
	return d.Definitions
}

// GetSecuritySchemes returns the security scheme definitions for OAS 2.0 documents.
// In OAS 2.0, these are stored in doc.SecurityDefinitions.
func (d *OAS2Document) GetSecuritySchemes() map[string]*SecurityScheme {
	if d == nil {
		return nil
	}
	return d.SecurityDefinitions
}

// GetParameters returns the reusable parameter definitions for OAS 2.0 documents.
func (d *OAS2Document) GetParameters() map[string]*Parameter {
	if d == nil {
		return nil
	}
	return d.Parameters
}

// GetResponses returns the reusable response definitions for OAS 2.0 documents.
func (d *OAS2Document) GetResponses() map[string]*Response {
	if d == nil {
		return nil
	}
	return d.Responses
}

// GetVersion returns the OASVersion for OAS 2.0 documents.
func (d *OAS2Document) GetVersion() OASVersion {
	if d == nil {
		return Unknown
	}
	return d.OASVersion
}

// GetVersionString returns the version string for OAS 2.0 documents.
func (d *OAS2Document) GetVersionString() string {
	if d == nil {
		return ""
	}
	return d.Swagger
}

// SchemaRefPrefix returns the JSON reference prefix for OAS 2.0 schemas.
func (d *OAS2Document) SchemaRefPrefix() string {
	return "#/definitions/"
}

// ----- OAS3Document implementation -----

// GetInfo returns the API metadata for OAS 3.x documents.
func (d *OAS3Document) GetInfo() *Info {
	if d == nil {
		return nil
	}
	return d.Info
}

// GetPaths returns the path items for OAS 3.x documents.
func (d *OAS3Document) GetPaths() Paths {
	if d == nil {
		return nil
	}
	return d.Paths
}

// GetTags returns the tag definitions for OAS 3.x documents.
func (d *OAS3Document) GetTags() []*Tag {
	if d == nil {
		return nil
	}
	return d.Tags
}

// GetSecurity returns the global security requirements for OAS 3.x documents.
func (d *OAS3Document) GetSecurity() []SecurityRequirement {
	if d == nil {
		return nil
	}
	return d.Security
}

// GetExternalDocs returns the external documentation for OAS 3.x documents.
func (d *OAS3Document) GetExternalDocs() *ExternalDocs {
	if d == nil {
		return nil
	}
	return d.ExternalDocs
}

// GetSchemas returns the schema definitions for OAS 3.x documents.
// In OAS 3.x, schemas are stored in doc.Components.Schemas.
func (d *OAS3Document) GetSchemas() map[string]*Schema {
	if d == nil || d.Components == nil {
		return nil
	}
	return d.Components.Schemas
}

// GetSecuritySchemes returns the security scheme definitions for OAS 3.x documents.
// In OAS 3.x, these are stored in doc.Components.SecuritySchemes.
func (d *OAS3Document) GetSecuritySchemes() map[string]*SecurityScheme {
	if d == nil || d.Components == nil {
		return nil
	}
	return d.Components.SecuritySchemes
}

// GetParameters returns the reusable parameter definitions for OAS 3.x documents.
// In OAS 3.x, these are stored in doc.Components.Parameters.
func (d *OAS3Document) GetParameters() map[string]*Parameter {
	if d == nil || d.Components == nil {
		return nil
	}
	return d.Components.Parameters
}

// GetResponses returns the reusable response definitions for OAS 3.x documents.
// In OAS 3.x, these are stored in doc.Components.Responses.
func (d *OAS3Document) GetResponses() map[string]*Response {
	if d == nil || d.Components == nil {
		return nil
	}
	return d.Components.Responses
}

// GetVersion returns the OASVersion for OAS 3.x documents.
func (d *OAS3Document) GetVersion() OASVersion {
	if d == nil {
		return Unknown
	}
	return d.OASVersion
}

// GetVersionString returns the version string for OAS 3.x documents.
func (d *OAS3Document) GetVersionString() string {
	if d == nil {
		return ""
	}
	return d.OpenAPI
}

// SchemaRefPrefix returns the JSON reference prefix for OAS 3.x schemas.
func (d *OAS3Document) SchemaRefPrefix() string {
	return "#/components/schemas/"
}

// ----- ParseResult helper -----

// AsAccessor returns a DocumentAccessor for version-agnostic access to the parsed document.
// Returns nil if the document type is unknown or nil.
//
// This method provides a convenient way to work with parsed documents without
// needing to check the version and perform type assertions:
//
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	if accessor := result.AsAccessor(); accessor != nil {
//	    schemas := accessor.GetSchemas() // Works for both OAS 2.0 and 3.x
//	}
func (pr *ParseResult) AsAccessor() DocumentAccessor {
	if pr == nil {
		return nil
	}
	switch doc := pr.Document.(type) {
	case *OAS3Document:
		return doc
	case *OAS2Document:
		return doc
	default:
		return nil
	}
}
