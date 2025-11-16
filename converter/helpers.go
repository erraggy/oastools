package converter

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/erraggy/oastools/parser"
)

// deepCopyOAS3Document creates a deep copy of an OAS3 document
func (c *Converter) deepCopyOAS3Document(src *parser.OAS3Document) (*parser.OAS3Document, error) {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(src)
	if err != nil {
		// This should never happen with valid documents
		return nil, fmt.Errorf("failed to marshal src document: %w", err)
	}

	var dst parser.OAS3Document
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dst document: %w", err)
	}

	// Restore OASVersion which may not round-trip through JSON
	dst.OASVersion = src.OASVersion

	return &dst, nil
}

// parseServerURL extracts host, basePath, and schemes from an OAS 3.x server URL
// Returns host, basePath, schemes, and error
func parseServerURL(serverURL string) (host, basePath string, schemes []string, err error) {
	// Handle server variables by replacing them with defaults or placeholders like:
	// http://example.com/foo/{parameter}/bar ==> http://example.com/foo/placeholder/bar
	// For simplicity, we'll strip variables for now and parse the base URL since the rest of the path is ignored here
	cleanURL := regexp.MustCompile(`\{[^}]+}`).ReplaceAllString(serverURL, "placeholder")

	// Parse the URL
	u, err := url.Parse(cleanURL)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Extract components
	if u.Scheme != "" {
		schemes = []string{u.Scheme}
	}

	host = u.Host
	basePath = u.Path
	if basePath == "" {
		basePath = "/"
	}

	return host, basePath, schemes, nil
}

// convertOAS2SchemaToOAS3 converts an OAS 2.0 schema to OAS 3.x format
func (c *Converter) convertOAS2SchemaToOAS3(schema *parser.Schema) *parser.Schema {
	if schema == nil {
		return nil
	}

	// For now, schemas are compatible between OAS 2.0 and 3.x
	// Deep copy to avoid mutations
	return c.deepCopySchema(schema)
}

// convertOAS3SchemaToOAS2 converts an OAS 3.x schema to OAS 2.0 format
func (c *Converter) convertOAS3SchemaToOAS2(schema *parser.Schema, result *ConversionResult, path string) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Check for OAS 3.1+ features that may not be compatible with OAS 2.0
	converted := c.deepCopySchema(schema)

	// Check for nullable (OAS 3.0+)
	if schema.Nullable {
		c.addIssueWithContext(result, path, "Schema uses 'nullable' which is OAS 3.0+",
			"Consider using 'x-nullable' extension for OAS 2.0 compatibility")
	}

	return converted
}

// deepCopySchema creates a deep copy of a schema
func (c *Converter) deepCopySchema(src *parser.Schema) *parser.Schema {
	if src == nil {
		return nil
	}

	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}

	var dst parser.Schema
	if err := json.Unmarshal(data, &dst); err != nil {
		return src
	}

	return &dst
}

// getDefaultMediaType returns a default media type if none is specified
func getDefaultMediaType() string {
	return "application/json"
}

// mergeStringArrays merges multiple string arrays, removing duplicates
func mergeStringArrays(arrays ...[]string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, arr := range arrays {
		for _, item := range arr {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		}
	}

	return result
}

// convertOAS2ParameterToOAS3 converts an OAS 2.0 parameter to OAS 3.x format
func (c *Converter) convertOAS2ParameterToOAS3(param *parser.Parameter, result *ConversionResult, path string) *parser.Parameter {
	if param == nil {
		return nil
	}

	converted := &parser.Parameter{
		Name:            param.Name,
		In:              param.In,
		Description:     param.Description,
		Required:        param.Required,
		Deprecated:      param.Deprecated,
		AllowEmptyValue: param.AllowEmptyValue,
	}

	// Handle schema
	if param.Schema != nil {
		converted.Schema = c.convertOAS2SchemaToOAS3(param.Schema)
	} else if param.Type != "" {
		// Convert type/format to schema
		converted.Schema = &parser.Schema{
			Type:   param.Type,
			Format: param.Format,
		}

		// Handle collection format
		if param.CollectionFormat != "" && param.CollectionFormat != "csv" {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Parameter uses collectionFormat '%s'", param.CollectionFormat),
				"OAS 3.x uses 'style' and 'explode' instead; 'csv' format maps to style=form")
		}
	}

	// AllowEmptyValue was removed in OAS 3.0
	if param.AllowEmptyValue {
		c.addIssueWithContext(result, path, "Parameter uses 'allowEmptyValue'",
			"This field was removed in OAS 3.0")
	}

	return converted
}

// convertOAS3ParameterToOAS2 converts an OAS 3.x parameter to OAS 2.0 format
func (c *Converter) convertOAS3ParameterToOAS2(param *parser.Parameter, result *ConversionResult, path string) *parser.Parameter {
	if param == nil {
		return nil
	}

	// Cookie parameters are not supported in OAS 2.0
	if param.In == "cookie" {
		c.addIssue(result, path, "Cookie parameters are not supported in OAS 2.0", SeverityCritical)
		return nil
	}

	converted := &parser.Parameter{
		Name:        param.Name,
		In:          param.In,
		Description: param.Description,
		Required:    param.Required,
	}

	// Convert schema to type/format
	if param.Schema != nil {
		schema := c.convertOAS3SchemaToOAS2(param.Schema, result, fmt.Sprintf("%s.schema", path))
		converted.Schema = schema

		// Extract type and format for non-body parameters
		if param.In != "body" && schema != nil {
			// Type can be string or []string, extract as string if possible
			if typeStr, ok := schema.Type.(string); ok {
				converted.Type = typeStr
			}
			converted.Format = schema.Format
		}
	}

	// Check for OAS 3.x style/explode parameters
	if param.Style != "" {
		c.addIssueWithContext(result, path,
			fmt.Sprintf("Parameter uses style '%s'", param.Style),
			"OAS 2.0 uses 'collectionFormat' instead")
	}

	return converted
}

// convertOAS2ResponseToOAS3Old converts an OAS 2.0 response to OAS 3.x format
func (c *Converter) convertOAS2ResponseToOAS3Old(response *parser.Response, produces []string) *parser.Response {
	if response == nil {
		return nil
	}

	converted := &parser.Response{
		Description: response.Description,
		Headers:     response.Headers,
	}

	// Convert schema to content
	if response.Schema != nil {
		converted.Content = make(map[string]*parser.MediaType)

		// Use produces array or default to application/json
		mediaTypes := produces
		if len(mediaTypes) == 0 {
			mediaTypes = []string{getDefaultMediaType()}
		}

		for _, mediaType := range mediaTypes {
			converted.Content[mediaType] = &parser.MediaType{
				Schema: c.convertOAS2SchemaToOAS3(response.Schema),
			}
		}
	}

	return converted
}

// convertOAS3ResponseToOAS2 converts an OAS 3.x response to OAS 2.0 format
func (c *Converter) convertOAS3ResponseToOAS2(response *parser.Response, result *ConversionResult, path string) (*parser.Response, []string) {
	if response == nil {
		return nil, nil
	}

	converted := &parser.Response{
		Description: response.Description,
		Headers:     response.Headers,
	}

	var produces []string

	// Convert content to schema
	if len(response.Content) > 0 {
		// Take the first media type's schema
		var firstMediaType string
		var firstContent *parser.MediaType

		for mt, content := range response.Content {
			if firstMediaType == "" {
				firstMediaType = mt
				firstContent = content
			}
			produces = append(produces, mt)
		}

		if len(response.Content) > 1 {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Response has multiple media types (%d), using first (%s)", len(response.Content), firstMediaType),
				"OAS 2.0 responses have a single schema; use 'produces' array to specify multiple content types")
		}

		if firstContent != nil && firstContent.Schema != nil {
			converted.Schema = c.convertOAS3SchemaToOAS2(firstContent.Schema, result, fmt.Sprintf("%s.content.%s.schema", path, firstMediaType))
		}
	}

	// Check for links (OAS 3.x only)
	if len(response.Links) > 0 {
		c.addIssue(result, path, "Response contains links which are not supported in OAS 2.0", SeverityCritical)
	}

	return converted, produces
}
