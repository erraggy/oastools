package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Parser handles OpenAPI specification parsing
type Parser struct {
	// ResolveRefs determines whether to resolve $ref references
	ResolveRefs bool
	// ValidateStructure determines whether to perform basic structure validation
	ValidateStructure bool
}

// New creates a new Parser instance with default settings
func New() *Parser {
	return &Parser{
		ResolveRefs:       false,
		ValidateStructure: true,
	}
}

// ParseResult contains the parsed OpenAPI specification and metadata
type ParseResult struct {
	// Version is the detected OAS version (e.g., "2.0", "3.0.3", "3.1.0")
	Version string
	// Data contains the raw parsed data as a map
	Data map[string]interface{}
	// Document contains the parsed document (type depends on version)
	Document interface{}
	// Errors contains any parsing or validation errors
	Errors []error
	// Warnings contains non-fatal issues
	Warnings []string
}

// Parse parses an OpenAPI specification file
func (p *Parser) Parse(specPath string) (*ParseResult, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	// Get the directory of the spec file for resolving relative refs
	baseDir := filepath.Dir(specPath)
	return p.parseBytesWithBaseDir(data, baseDir)
}

// ParseReader parses an OpenAPI specification from an io.Reader
func (p *Parser) ParseReader(r io.Reader) (*ParseResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return p.ParseBytes(data)
}

// ParseBytes parses an OpenAPI specification from a byte slice
// For external references to work, use Parse() with a file path instead
func (p *Parser) ParseBytes(data []byte) (*ParseResult, error) {
	return p.parseBytesWithBaseDir(data, ".")
}

// parseBytesWithBaseDir parses data with a specified base directory for ref resolution
func (p *Parser) parseBytesWithBaseDir(data []byte, baseDir string) (*ParseResult, error) {
	result := &ParseResult{
		Errors:   make([]error, 0),
		Warnings: make([]string, 0),
	}

	// First pass: parse to generic map to detect version
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
	}

	// Resolve references if enabled (before version-specific parsing)
	if p.ResolveRefs {
		resolver := NewRefResolver(baseDir)
		if err := resolver.ResolveAllRefs(rawData); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ref resolution warning: %v", err))
		}
	}

	result.Data = rawData

	// Detect version
	version, err := p.detectVersion(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to detect OAS version: %w", err)
	}
	result.Version = version

	// Re-marshal the data (potentially with resolved refs) for version-specific parsing
	resolvedData, err := yaml.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshal data: %w", err)
	}

	// Parse to version-specific structure
	doc, err := p.parseVersionSpecific(resolvedData, version)
	if err != nil {
		result.Errors = append(result.Errors, err)
	} else {
		result.Document = doc
	}

	// Validate structure if enabled
	if p.ValidateStructure {
		validationErrors := p.validateStructure(result)
		result.Errors = append(result.Errors, validationErrors...)
	}

	return result, nil
}

// detectVersion determines the OAS version from the raw data
func (p *Parser) detectVersion(data map[string]interface{}) (string, error) {
	// Check for OAS 2.0 (Swagger)
	if swagger, ok := data["swagger"].(string); ok {
		return swagger, nil
	}

	// Check for OAS 3.x
	if openapi, ok := data["openapi"].(string); ok {
		return openapi, nil
	}

	// Neither field was found - provide helpful error message
	return "", fmt.Errorf("[Version Detection] unable to detect OpenAPI version: document must contain either 'swagger: \"2.0\"' (for OAS 2.0) or 'openapi: \"3.x.x\"' (for OAS 3.x) at the root level")
}

// parseVersionSpecific parses the data into a version-specific structure
func (p *Parser) parseVersionSpecific(data []byte, version string) (interface{}, error) {
	switch {
	case version == "2.0":
		var doc OAS2Document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("[OAS 2.0 Parser] failed to parse document structure: %w", err)
		}
		return &doc, nil

	case version >= "3.0.0" && version < "4.0.0":
		var doc OAS3Document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("[OAS %s Parser] failed to parse document structure: %w", version, err)
		}
		return &doc, nil

	default:
		return nil, fmt.Errorf("[Parser Error] unsupported OpenAPI version: %s (only 2.0 and 3.x versions are supported)", version)
	}
}

// validateStructure performs basic structure validation
func (p *Parser) validateStructure(result *ParseResult) []error {
	errors := make([]error, 0)

	// Validate required fields based on version
	switch {
	case result.Version == "2.0":
		doc, ok := result.Document.(*OAS2Document)
		if !ok {
			errors = append(errors, fmt.Errorf("[Parser Error] internal error: document type mismatch for OAS 2.0 (expected *OAS2Document, got %T)", result.Document))
			return errors
		}
		errors = append(errors, p.validateOAS2(doc)...)

	case result.Version >= "3.0.0" && result.Version < "4.0.0":
		doc, ok := result.Document.(*OAS3Document)
		if !ok {
			errors = append(errors, fmt.Errorf("[Parser Error] internal error: document type mismatch for OAS 3.x (expected *OAS3Document, got %T)", result.Document))
			return errors
		}
		errors = append(errors, p.validateOAS3(doc)...)

	default:
		errors = append(errors, fmt.Errorf("[Parser Error] unsupported OpenAPI version: %s (only versions 2.0 and 3.x are supported)", result.Version))
	}

	return errors
}

// validateOAS2 validates an OAS 2.0 document
func (p *Parser) validateOAS2(doc *OAS2Document) []error {
	errors := make([]error, 0)

	// Validate swagger version field
	if doc.Swagger == "" {
		errors = append(errors, fmt.Errorf("[OAS 2.0] missing required root field 'swagger': must be set to \"2.0\""))
	} else if doc.Swagger != "2.0" {
		errors = append(errors, fmt.Errorf("[OAS 2.0] invalid 'swagger' field value: expected \"2.0\", got \"%s\"", doc.Swagger))
	}

	// Validate info object
	if doc.Info == nil {
		errors = append(errors, fmt.Errorf("[OAS 2.0] missing required root field 'info': Info object is required per spec (https://spec.openapis.org/oas/v2.0.html#infoObject)"))
	} else {
		if doc.Info.Title == "" {
			errors = append(errors, fmt.Errorf("[OAS 2.0] missing required field 'info.title': Info object must have a title per spec"))
		}
		if doc.Info.Version == "" {
			errors = append(errors, fmt.Errorf("[OAS 2.0] missing required field 'info.version': Info object must have a version string per spec"))
		}
	}

	// Validate paths object
	if doc.Paths == nil {
		errors = append(errors, fmt.Errorf("[OAS 2.0] missing required root field 'paths': Paths object is required per spec (https://spec.openapis.org/oas/v2.0.html#pathsObject)"))
	} else {
		// Validate individual paths and operations
		operationIDs := make(map[string]string)
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			// Validate path pattern
			if pathPattern != "" && pathPattern[0] != '/' {
				errors = append(errors, fmt.Errorf("[OAS 2.0] invalid path pattern 'paths.%s': path must begin with '/'", pathPattern))
			}

			// Check all operations in this path
			operations := map[string]*Operation{
				"get":     pathItem.Get,
				"put":     pathItem.Put,
				"post":    pathItem.Post,
				"delete":  pathItem.Delete,
				"options": pathItem.Options,
				"head":    pathItem.Head,
				"patch":   pathItem.Patch,
			}

			for method, op := range operations {
				if op == nil {
					continue
				}

				opPath := fmt.Sprintf("paths.%s.%s", pathPattern, method)

				// Validate operationId uniqueness
				if op.OperationID != "" {
					if existingPath, exists := operationIDs[op.OperationID]; exists {
						errors = append(errors, fmt.Errorf("[OAS 2.0] duplicate operationId '%s' at '%s': previously defined at '%s'",
							op.OperationID, opPath, existingPath))
					} else {
						operationIDs[op.OperationID] = opPath
					}
				}

				// Validate responses object exists
				if op.Responses == nil {
					errors = append(errors, fmt.Errorf("[OAS 2.0] missing required field '%s.responses': Operation must have a responses object", opPath))
				}

				// Validate parameters
				for i, param := range op.Parameters {
					if param == nil {
						continue
					}
					paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
					if param.Name == "" {
						errors = append(errors, fmt.Errorf("[OAS 2.0] missing required field '%s.name': Parameter must have a name", paramPath))
					}
					if param.In == "" {
						errors = append(errors, fmt.Errorf("[OAS 2.0] missing required field '%s.in': Parameter must specify location (query, header, path, formData, body)", paramPath))
					} else {
						validLocations := map[string]bool{"query": true, "header": true, "path": true, "formData": true, "body": true}
						if !validLocations[param.In] {
							errors = append(errors, fmt.Errorf("[OAS 2.0] invalid value for '%s.in': \"%s\" is not a valid parameter location (must be query, header, path, formData, or body)", paramPath, param.In))
						}
					}
				}
			}
		}
	}

	return errors
}

// validateOAS3 validates an OAS 3.x document
func (p *Parser) validateOAS3(doc *OAS3Document) []error {
	errors := make([]error, 0)
	version := doc.OpenAPI

	// Validate openapi version field
	if doc.OpenAPI == "" {
		errors = append(errors, fmt.Errorf("[OAS 3.x] missing required root field 'openapi': must be set to a valid 3.x version (e.g., \"3.0.3\", \"3.1.0\")"))
	} else if doc.OpenAPI < "3.0.0" || doc.OpenAPI >= "4.0.0" {
		errors = append(errors, fmt.Errorf("[OAS %s] invalid 'openapi' field value: \"%s\" is not a valid 3.x version", version, doc.OpenAPI))
	}

	// Validate info object
	if doc.Info == nil {
		errors = append(errors, fmt.Errorf("[OAS %s] missing required root field 'info': Info object is required per spec (https://spec.openapis.org/oas/v3.0.0.html#info-object)", version))
	} else {
		if doc.Info.Title == "" {
			errors = append(errors, fmt.Errorf("[OAS %s] missing required field 'info.title': Info object must have a title per spec", version))
		}
		if doc.Info.Version == "" {
			errors = append(errors, fmt.Errorf("[OAS %s] missing required field 'info.version': Info object must have a version string per spec", version))
		}
	}

	// Validate paths object - required in 3.0.x, optional in 3.1+
	if doc.OpenAPI >= "3.0.0" && doc.OpenAPI < "3.1.0" {
		if doc.Paths == nil {
			errors = append(errors, fmt.Errorf("[OAS %s] missing required root field 'paths': Paths object is required in OAS 3.0.x (https://spec.openapis.org/oas/v3.0.0.html#paths-object)", version))
		}
	} else if doc.OpenAPI >= "3.1.0" {
		// In OAS 3.1+, either paths or webhooks must be present
		if doc.Paths == nil && len(doc.Webhooks) == 0 {
			errors = append(errors, fmt.Errorf("[OAS %s] document must have either 'paths' or 'webhooks': at least one is required in OAS 3.1+", version))
		}
	}

	// Validate paths if present
	if doc.Paths != nil {
		operationIDs := make(map[string]string)
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			// Validate path pattern
			if pathPattern != "" && pathPattern[0] != '/' {
				errors = append(errors, fmt.Errorf("[OAS %s] invalid path pattern 'paths.%s': path must begin with '/'", version, pathPattern))
			}

			// Check all operations in this path
			operations := map[string]*Operation{
				"get":     pathItem.Get,
				"put":     pathItem.Put,
				"post":    pathItem.Post,
				"delete":  pathItem.Delete,
				"options": pathItem.Options,
				"head":    pathItem.Head,
				"patch":   pathItem.Patch,
				"trace":   pathItem.Trace,
			}

			for method, op := range operations {
				if op == nil {
					continue
				}

				opPath := fmt.Sprintf("paths.%s.%s", pathPattern, method)

				// Validate operationId uniqueness
				if op.OperationID != "" {
					if existingPath, exists := operationIDs[op.OperationID]; exists {
						errors = append(errors, fmt.Errorf("[OAS %s] duplicate operationId '%s' at '%s': previously defined at '%s' (operationIds must be unique across all operations)",
							version, op.OperationID, opPath, existingPath))
					} else {
						operationIDs[op.OperationID] = opPath
					}
				}

				// Validate responses object exists
				if op.Responses == nil {
					errors = append(errors, fmt.Errorf("[OAS %s] missing required field '%s.responses': Operation must have a responses object", version, opPath))
				}

				// Validate parameters
				for i, param := range op.Parameters {
					if param == nil {
						continue
					}
					paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
					if param.Name == "" {
						errors = append(errors, fmt.Errorf("[OAS %s] missing required field '%s.name': Parameter must have a name", version, paramPath))
					}
					if param.In == "" {
						errors = append(errors, fmt.Errorf("[OAS %s] missing required field '%s.in': Parameter must specify location (query, header, path, cookie)", version, paramPath))
					} else {
						validLocations := map[string]bool{"query": true, "header": true, "path": true, "cookie": true}
						if !validLocations[param.In] {
							errors = append(errors, fmt.Errorf("[OAS %s] invalid value for '%s.in': \"%s\" is not a valid parameter location (must be query, header, path, or cookie)", version, paramPath, param.In))
						}
					}

					// Path parameters must be required
					if param.In == "path" && !param.Required {
						errors = append(errors, fmt.Errorf("[OAS %s] invalid parameter '%s': path parameters must have 'required: true' per spec", version, paramPath))
					}
				}

				// Validate requestBody if present
				if op.RequestBody != nil {
					rbPath := fmt.Sprintf("%s.requestBody", opPath)
					if op.RequestBody.Content == nil || len(op.RequestBody.Content) == 0 {
						errors = append(errors, fmt.Errorf("[OAS %s] missing required field '%s.content': RequestBody must have at least one media type", version, rbPath))
					}
				}
			}
		}
	}

	// Validate webhooks if present (OAS 3.1+)
	if len(doc.Webhooks) > 0 && doc.OpenAPI < "3.1.0" {
		errors = append(errors, fmt.Errorf("[OAS %s] 'webhooks' field is only supported in OAS 3.1.0 and later, not in version %s", version, doc.OpenAPI))
	}

	return errors
}
