package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// validateRef validates that a $ref string points to a valid location in the document
func (v *Validator) validateRef(ref, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if ref == "" {
		return
	}

	// Check for refs that reference empty schema names (end with /)
	if strings.HasSuffix(ref, "/") {
		v.addError(result, path,
			fmt.Sprintf("$ref %q references an empty schema name", ref),
			withSpecRef(baseURL),
			withField("$ref"),
			withValue(ref),
		)
		return
	}

	// Only validate local references (starting with #/)
	// External references (file paths, URLs) are handled by the parser's resolver
	if !strings.HasPrefix(ref, "#/") {
		// External reference - we don't validate these here
		return
	}

	// Check if the reference exists in the valid refs map
	if !validRefs[ref] {
		v.addError(result, path,
			fmt.Sprintf("$ref '%s' does not resolve to a valid component in the document", ref),
			withSpecRef(baseURL),
			withField("$ref"),
			withValue(ref),
		)
	}
}

// buildOAS2ValidRefs builds a map of all valid $ref paths in an OAS 2.0 document
func buildOAS2ValidRefs(doc *parser.OAS2Document) map[string]bool {
	validRefs := make(map[string]bool)

	// Add definitions
	for name := range doc.Definitions {
		validRefs[fmt.Sprintf("#/definitions/%s", name)] = true
	}

	// Add parameters
	for name := range doc.Parameters {
		validRefs[fmt.Sprintf("#/parameters/%s", name)] = true
	}

	// Add responses
	for name := range doc.Responses {
		validRefs[fmt.Sprintf("#/responses/%s", name)] = true
	}

	// Add security definitions
	for name := range doc.SecurityDefinitions {
		validRefs[fmt.Sprintf("#/securityDefinitions/%s", name)] = true
	}

	return validRefs
}

// buildOAS3ValidRefs builds a map of all valid $ref paths in an OAS 3.x document
func buildOAS3ValidRefs(doc *parser.OAS3Document) map[string]bool {
	validRefs := make(map[string]bool)

	if doc.Components == nil {
		return validRefs
	}

	// Add schemas
	for name := range doc.Components.Schemas {
		validRefs[fmt.Sprintf("#/components/schemas/%s", name)] = true
	}

	// Add responses
	for name := range doc.Components.Responses {
		validRefs[fmt.Sprintf("#/components/responses/%s", name)] = true
	}

	// Add parameters
	for name := range doc.Components.Parameters {
		validRefs[fmt.Sprintf("#/components/parameters/%s", name)] = true
	}

	// Add examples
	for name := range doc.Components.Examples {
		validRefs[fmt.Sprintf("#/components/examples/%s", name)] = true
	}

	// Add request bodies
	for name := range doc.Components.RequestBodies {
		validRefs[fmt.Sprintf("#/components/requestBodies/%s", name)] = true
	}

	// Add headers
	for name := range doc.Components.Headers {
		validRefs[fmt.Sprintf("#/components/headers/%s", name)] = true
	}

	// Add security schemes
	for name := range doc.Components.SecuritySchemes {
		validRefs[fmt.Sprintf("#/components/securitySchemes/%s", name)] = true
	}

	// Add links
	for name := range doc.Components.Links {
		validRefs[fmt.Sprintf("#/components/links/%s", name)] = true
	}

	// Add callbacks
	for name := range doc.Components.Callbacks {
		validRefs[fmt.Sprintf("#/components/callbacks/%s", name)] = true
	}

	// Add path items (OAS 3.1+)
	for name := range doc.Components.PathItems {
		validRefs[fmt.Sprintf("#/components/pathItems/%s", name)] = true
	}

	return validRefs
}

// validateSchemaRefs recursively validates all $ref values in a schema
func (v *Validator) validateSchemaRefs(schema *parser.Schema, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if schema == nil {
		return
	}

	// Validate the $ref in this schema
	if schema.Ref != "" {
		v.validateRef(schema.Ref, path, validRefs, result, baseURL)
	}

	// Recursively validate nested schemas
	// Properties
	for propName, propSchema := range schema.Properties {
		if propSchema != nil {
			propPath := fmt.Sprintf("%s.properties.%s", path, propName)
			v.validateSchemaRefs(propSchema, propPath, validRefs, result, baseURL)
		}
	}

	// Pattern properties
	for propName, propSchema := range schema.PatternProperties {
		if propSchema != nil {
			propPath := fmt.Sprintf("%s.patternProperties.%s", path, propName)
			v.validateSchemaRefs(propSchema, propPath, validRefs, result, baseURL)
		}
	}

	// Additional properties
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			addPropsPath := fmt.Sprintf("%s.additionalProperties", path)
			v.validateSchemaRefs(addProps, addPropsPath, validRefs, result, baseURL)
		}
	}

	// Items
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			itemsPath := fmt.Sprintf("%s.items", path)
			v.validateSchemaRefs(items, itemsPath, validRefs, result, baseURL)
		}
	}

	// AllOf, AnyOf, OneOf
	for i, subSchema := range schema.AllOf {
		if subSchema != nil {
			subPath := fmt.Sprintf("%s.allOf[%d]", path, i)
			v.validateSchemaRefs(subSchema, subPath, validRefs, result, baseURL)
		}
	}

	for i, subSchema := range schema.AnyOf {
		if subSchema != nil {
			subPath := fmt.Sprintf("%s.anyOf[%d]", path, i)
			v.validateSchemaRefs(subSchema, subPath, validRefs, result, baseURL)
		}
	}

	for i, subSchema := range schema.OneOf {
		if subSchema != nil {
			subPath := fmt.Sprintf("%s.oneOf[%d]", path, i)
			v.validateSchemaRefs(subSchema, subPath, validRefs, result, baseURL)
		}
	}

	// Not
	if schema.Not != nil {
		notPath := fmt.Sprintf("%s.not", path)
		v.validateSchemaRefs(schema.Not, notPath, validRefs, result, baseURL)
	}

	// Additional items
	if schema.AdditionalItems != nil {
		if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
			addItemsPath := fmt.Sprintf("%s.additionalItems", path)
			v.validateSchemaRefs(addItems, addItemsPath, validRefs, result, baseURL)
		}
	}

	// Prefix items (JSON Schema Draft 2020-12)
	for i, prefixItem := range schema.PrefixItems {
		if prefixItem != nil {
			prefixPath := fmt.Sprintf("%s.prefixItems[%d]", path, i)
			v.validateSchemaRefs(prefixItem, prefixPath, validRefs, result, baseURL)
		}
	}

	// Contains, PropertyNames (JSON Schema Draft 2020-12)
	if schema.Contains != nil {
		v.validateSchemaRefs(schema.Contains, fmt.Sprintf("%s.contains", path), validRefs, result, baseURL)
	}

	if schema.PropertyNames != nil {
		v.validateSchemaRefs(schema.PropertyNames, fmt.Sprintf("%s.propertyNames", path), validRefs, result, baseURL)
	}

	// Dependent schemas (JSON Schema Draft 2020-12)
	for name, depSchema := range schema.DependentSchemas {
		if depSchema != nil {
			depPath := fmt.Sprintf("%s.dependentSchemas.%s", path, name)
			v.validateSchemaRefs(depSchema, depPath, validRefs, result, baseURL)
		}
	}

	// If/Then/Else (JSON Schema Draft 2020-12, OAS 3.1+)
	if schema.If != nil {
		v.validateSchemaRefs(schema.If, fmt.Sprintf("%s.if", path), validRefs, result, baseURL)
	}
	if schema.Then != nil {
		v.validateSchemaRefs(schema.Then, fmt.Sprintf("%s.then", path), validRefs, result, baseURL)
	}
	if schema.Else != nil {
		v.validateSchemaRefs(schema.Else, fmt.Sprintf("%s.else", path), validRefs, result, baseURL)
	}

	// $defs (JSON Schema Draft 2020-12)
	for name, defSchema := range schema.Defs {
		if defSchema != nil {
			defPath := fmt.Sprintf("%s.$defs.%s", path, name)
			v.validateSchemaRefs(defSchema, defPath, validRefs, result, baseURL)
		}
	}
}

// validateParameterRef validates a parameter's $ref if present
func (v *Validator) validateParameterRef(param *parser.Parameter, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if param == nil {
		return
	}

	if param.Ref != "" {
		v.validateRef(param.Ref, path, validRefs, result, baseURL)
	}

	// Also validate schema refs within the parameter
	if param.Schema != nil {
		v.validateSchemaRefs(param.Schema, fmt.Sprintf("%s.schema", path), validRefs, result, baseURL)
	}
}

// validateOperationResponses validates all responses for an operation.
// This handles both default and status code responses.
func (v *Validator) validateOperationResponses(op *parser.Operation, opPath string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if op.Responses == nil {
		return
	}
	if op.Responses.Default != nil {
		responsePath := fmt.Sprintf("%s.responses.default", opPath)
		v.validateResponseRef(op.Responses.Default, responsePath, validRefs, result, baseURL)
	}
	for code, response := range op.Responses.Codes {
		if response != nil {
			responsePath := fmt.Sprintf("%s.responses.%s", opPath, code)
			v.validateResponseRef(response, responsePath, validRefs, result, baseURL)
		}
	}
}

// validateResponseRef validates a response's $ref if present
func (v *Validator) validateResponseRef(response *parser.Response, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if response == nil {
		return
	}

	if response.Ref != "" {
		v.validateRef(response.Ref, path, validRefs, result, baseURL)
	}

	// Validate schema refs in the response
	if response.Schema != nil {
		v.validateSchemaRefs(response.Schema, fmt.Sprintf("%s.schema", path), validRefs, result, baseURL)
	}

	// Validate content schemas (OAS 3.x)
	for mediaType, mediaTypeObj := range response.Content {
		if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
			schemaPath := fmt.Sprintf("%s.content.%s.schema", path, mediaType)
			v.validateSchemaRefs(mediaTypeObj.Schema, schemaPath, validRefs, result, baseURL)
		}
	}

	// Validate headers
	for headerName, header := range response.Headers {
		if header != nil {
			headerPath := fmt.Sprintf("%s.headers.%s", path, headerName)
			if header.Ref != "" {
				v.validateRef(header.Ref, headerPath, validRefs, result, baseURL)
			}
			if header.Schema != nil {
				v.validateSchemaRefs(header.Schema, fmt.Sprintf("%s.schema", headerPath), validRefs, result, baseURL)
			}
		}
	}

	// Validate links (OAS 3.x)
	for linkName, link := range response.Links {
		if link != nil && link.Ref != "" {
			linkPath := fmt.Sprintf("%s.links.%s", path, linkName)
			v.validateRef(link.Ref, linkPath, validRefs, result, baseURL)
		}
	}
}

// validateRequestBodyRef validates a request body's $ref if present
func (v *Validator) validateRequestBodyRef(requestBody *parser.RequestBody, path string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if requestBody == nil {
		return
	}

	if requestBody.Ref != "" {
		v.validateRef(requestBody.Ref, path, validRefs, result, baseURL)
	}

	// Validate content schemas
	for mediaType, mediaTypeObj := range requestBody.Content {
		if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
			schemaPath := fmt.Sprintf("%s.content.%s.schema", path, mediaType)
			v.validateSchemaRefs(mediaTypeObj.Schema, schemaPath, validRefs, result, baseURL)
		}
	}
}

// validateOAS2Refs validates all $ref values in an OAS 2.0 document
func (v *Validator) validateOAS2Refs(doc *parser.OAS2Document, result *ValidationResult, baseURL string) {
	// Build the map of valid reference paths
	validRefs := buildOAS2ValidRefs(doc)

	// Validate refs in definitions
	for name, schema := range doc.Definitions {
		if schema != nil {
			path := fmt.Sprintf("definitions.%s", name)
			v.validateSchemaRefs(schema, path, validRefs, result, baseURL)
		}
	}

	// Validate refs in parameters
	for name, param := range doc.Parameters {
		if param != nil {
			path := fmt.Sprintf("parameters.%s", name)
			v.validateParameterRef(param, path, validRefs, result, baseURL)
		}
	}

	// Validate refs in responses
	for name, response := range doc.Responses {
		if response != nil {
			path := fmt.Sprintf("responses.%s", name)
			v.validateResponseRef(response, path, validRefs, result, baseURL)
		}
	}

	// Validate refs in paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

		// Validate path-level parameters
		for i, param := range pathItem.Parameters {
			if param != nil {
				paramPath := fmt.Sprintf("%s.parameters[%d]", pathPrefix, i)
				v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
			}
		}

		// Validate each operation
		operations := parser.GetOperations(pathItem, parser.OASVersion20)
		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := fmt.Sprintf("%s.%s", pathPrefix, method)

			// Validate operation parameters
			for i, param := range op.Parameters {
				if param != nil {
					paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
					v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
				}
			}

			// Validate operation responses
			v.validateOperationResponses(op, opPath, validRefs, result, baseURL)
		}
	}
}

// validateOAS3Refs validates all $ref values in an OAS 3.x document
func (v *Validator) validateOAS3Refs(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	// Build the map of valid reference paths
	validRefs := buildOAS3ValidRefs(doc)

	// Validate refs in components
	if doc.Components != nil {
		// Validate schemas
		for name, schema := range doc.Components.Schemas {
			if schema != nil {
				path := fmt.Sprintf("components.schemas.%s", name)
				v.validateSchemaRefs(schema, path, validRefs, result, baseURL)
			}
		}

		// Validate parameters
		for name, param := range doc.Components.Parameters {
			if param != nil {
				path := fmt.Sprintf("components.parameters.%s", name)
				v.validateParameterRef(param, path, validRefs, result, baseURL)
			}
		}

		// Validate responses
		for name, response := range doc.Components.Responses {
			if response != nil {
				path := fmt.Sprintf("components.responses.%s", name)
				v.validateResponseRef(response, path, validRefs, result, baseURL)
			}
		}

		// Validate request bodies
		for name, requestBody := range doc.Components.RequestBodies {
			if requestBody != nil {
				path := fmt.Sprintf("components.requestBodies.%s", name)
				v.validateRequestBodyRef(requestBody, path, validRefs, result, baseURL)
			}
		}

		// Validate headers
		for name, header := range doc.Components.Headers {
			if header != nil {
				headerPath := fmt.Sprintf("components.headers.%s", name)
				if header.Ref != "" {
					v.validateRef(header.Ref, headerPath, validRefs, result, baseURL)
				}
				if header.Schema != nil {
					v.validateSchemaRefs(header.Schema, fmt.Sprintf("%s.schema", headerPath), validRefs, result, baseURL)
				}
			}
		}
	}

	// Validate refs in paths
	if doc.Paths != nil {
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			pathPrefix := fmt.Sprintf("paths.%s", pathPattern)

			// Validate PathItem $ref
			if pathItem.Ref != "" {
				v.validateRef(pathItem.Ref, pathPrefix, validRefs, result, baseURL)
			}

			// Validate path-level parameters
			for i, param := range pathItem.Parameters {
				if param != nil {
					paramPath := fmt.Sprintf("%s.parameters[%d]", pathPrefix, i)
					v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
				}
			}

			// Validate each operation
			v.validatePathItemOperationRefs(pathItem, pathPrefix, doc.OASVersion, validRefs, result, baseURL)
		}
	}

	// Validate refs in webhooks (OAS 3.1+)
	for webhookName, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("webhooks.%s", webhookName)

		// Validate PathItem $ref
		if pathItem.Ref != "" {
			v.validateRef(pathItem.Ref, pathPrefix, validRefs, result, baseURL)
		}

		// Validate webhook operations
		v.validatePathItemOperationRefs(pathItem, pathPrefix, doc.OASVersion, validRefs, result, baseURL)
	}
}

// validatePathItemOperationRefs validates $ref values within all operations of a PathItem.
// This is used by both paths and webhooks validation to avoid code duplication.
func (v *Validator) validatePathItemOperationRefs(pathItem *parser.PathItem, pathPrefix string, version parser.OASVersion, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	operations := parser.GetOperations(pathItem, version)
	for method, op := range operations {
		if op == nil {
			continue
		}

		opPath := fmt.Sprintf("%s.%s", pathPrefix, method)

		// Validate operation parameters
		for i, param := range op.Parameters {
			if param != nil {
				paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
				v.validateParameterRef(param, paramPath, validRefs, result, baseURL)
			}
		}

		// Validate request body
		if op.RequestBody != nil {
			requestBodyPath := fmt.Sprintf("%s.requestBody", opPath)
			v.validateRequestBodyRef(op.RequestBody, requestBodyPath, validRefs, result, baseURL)
		}

		// Validate operation responses
		v.validateOperationResponses(op, opPath, validRefs, result, baseURL)
	}
}
