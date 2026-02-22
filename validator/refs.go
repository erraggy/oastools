package validator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/pathutil"
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
	capacity := len(doc.Definitions) + len(doc.Parameters) + len(doc.Responses) + len(doc.SecurityDefinitions)
	validRefs := make(map[string]bool, capacity)

	// Add definitions
	for name := range doc.Definitions {
		validRefs[pathutil.DefinitionRef(name)] = true
	}

	// Add parameters
	for name := range doc.Parameters {
		validRefs[pathutil.ParameterRef(name, true)] = true
	}

	// Add responses
	for name := range doc.Responses {
		validRefs[pathutil.ResponseRef(name, true)] = true
	}

	// Add security definitions
	for name := range doc.SecurityDefinitions {
		validRefs[pathutil.SecuritySchemeRef(name, true)] = true
	}

	return validRefs
}

// buildOAS3ValidRefs builds a map of all valid $ref paths in an OAS 3.x document
func buildOAS3ValidRefs(doc *parser.OAS3Document) map[string]bool {
	if doc.Components == nil {
		return make(map[string]bool)
	}

	capacity := len(doc.Components.Schemas) +
		len(doc.Components.Responses) +
		len(doc.Components.Parameters) +
		len(doc.Components.Examples) +
		len(doc.Components.RequestBodies) +
		len(doc.Components.Headers) +
		len(doc.Components.SecuritySchemes) +
		len(doc.Components.Links) +
		len(doc.Components.Callbacks) +
		len(doc.Components.PathItems)
	validRefs := make(map[string]bool, capacity)

	// Add schemas
	for name := range doc.Components.Schemas {
		validRefs[pathutil.SchemaRef(name)] = true
	}

	// Add responses
	for name := range doc.Components.Responses {
		validRefs[pathutil.ResponseRef(name, false)] = true
	}

	// Add parameters
	for name := range doc.Components.Parameters {
		validRefs[pathutil.ParameterRef(name, false)] = true
	}

	// Add examples
	for name := range doc.Components.Examples {
		validRefs[pathutil.ExampleRef(name)] = true
	}

	// Add request bodies
	for name := range doc.Components.RequestBodies {
		validRefs[pathutil.RequestBodyRef(name)] = true
	}

	// Add headers
	for name := range doc.Components.Headers {
		validRefs[pathutil.HeaderRef(name)] = true
	}

	// Add security schemes
	for name := range doc.Components.SecuritySchemes {
		validRefs[pathutil.SecuritySchemeRef(name, false)] = true
	}

	// Add links
	for name := range doc.Components.Links {
		validRefs[pathutil.LinkRef(name)] = true
	}

	// Add callbacks
	for name := range doc.Components.Callbacks {
		validRefs[pathutil.CallbackRef(name)] = true
	}

	// Add path items (OAS 3.1+)
	for name := range doc.Components.PathItems {
		validRefs[pathutil.PathItemRef(name)] = true
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
			v.validateSchemaRefs(propSchema, path+".properties."+propName, validRefs, result, baseURL)
		}
	}

	// Pattern properties
	for propName, propSchema := range schema.PatternProperties {
		if propSchema != nil {
			v.validateSchemaRefs(propSchema, path+".patternProperties."+propName, validRefs, result, baseURL)
		}
	}

	// Additional properties
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			v.validateSchemaRefs(addProps, path+".additionalProperties", validRefs, result, baseURL)
		}
	}

	// Items
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			v.validateSchemaRefs(items, path+".items", validRefs, result, baseURL)
		}
	}

	// AllOf, AnyOf, OneOf
	for i, subSchema := range schema.AllOf {
		if subSchema != nil {
			v.validateSchemaRefs(subSchema, path+".allOf["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
		}
	}

	for i, subSchema := range schema.AnyOf {
		if subSchema != nil {
			v.validateSchemaRefs(subSchema, path+".anyOf["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
		}
	}

	for i, subSchema := range schema.OneOf {
		if subSchema != nil {
			v.validateSchemaRefs(subSchema, path+".oneOf["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
		}
	}

	// Not
	if schema.Not != nil {
		v.validateSchemaRefs(schema.Not, path+".not", validRefs, result, baseURL)
	}

	// Additional items
	if schema.AdditionalItems != nil {
		if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
			v.validateSchemaRefs(addItems, path+".additionalItems", validRefs, result, baseURL)
		}
	}

	// Prefix items (JSON Schema Draft 2020-12)
	for i, prefixItem := range schema.PrefixItems {
		if prefixItem != nil {
			v.validateSchemaRefs(prefixItem, path+".prefixItems["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
		}
	}

	// Contains, PropertyNames (JSON Schema Draft 2020-12)
	if schema.Contains != nil {
		v.validateSchemaRefs(schema.Contains, path+".contains", validRefs, result, baseURL)
	}

	if schema.PropertyNames != nil {
		v.validateSchemaRefs(schema.PropertyNames, path+".propertyNames", validRefs, result, baseURL)
	}

	// Dependent schemas (JSON Schema Draft 2020-12)
	for name, depSchema := range schema.DependentSchemas {
		if depSchema != nil {
			v.validateSchemaRefs(depSchema, path+".dependentSchemas."+name, validRefs, result, baseURL)
		}
	}

	// If/Then/Else (JSON Schema Draft 2020-12, OAS 3.1+)
	if schema.If != nil {
		v.validateSchemaRefs(schema.If, path+".if", validRefs, result, baseURL)
	}
	if schema.Then != nil {
		v.validateSchemaRefs(schema.Then, path+".then", validRefs, result, baseURL)
	}
	if schema.Else != nil {
		v.validateSchemaRefs(schema.Else, path+".else", validRefs, result, baseURL)
	}

	// $defs (JSON Schema Draft 2020-12)
	for name, defSchema := range schema.Defs {
		if defSchema != nil {
			v.validateSchemaRefs(defSchema, path+".$defs."+name, validRefs, result, baseURL)
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
		v.validateSchemaRefs(param.Schema, path+".schema", validRefs, result, baseURL)
	}
}

// validateOperationResponses validates all responses for an operation.
// This handles both default and status code responses.
func (v *Validator) validateOperationResponses(op *parser.Operation, opPath string, validRefs map[string]bool, result *ValidationResult, baseURL string) {
	if op.Responses == nil {
		return
	}
	if op.Responses.Default != nil {
		v.validateResponseRef(op.Responses.Default, opPath+".responses.default", validRefs, result, baseURL)
	}
	for code, response := range op.Responses.Codes {
		if response != nil {
			v.validateResponseRef(response, opPath+".responses."+code, validRefs, result, baseURL)
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
		v.validateSchemaRefs(response.Schema, path+".schema", validRefs, result, baseURL)
	}

	// Validate content schemas (OAS 3.x)
	for mediaType, mediaTypeObj := range response.Content {
		if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
			v.validateSchemaRefs(mediaTypeObj.Schema, path+".content."+mediaType+".schema", validRefs, result, baseURL)
		}
	}

	// Validate headers
	for headerName, header := range response.Headers {
		if header != nil {
			headerPath := path + ".headers." + headerName
			if header.Ref != "" {
				v.validateRef(header.Ref, headerPath, validRefs, result, baseURL)
			}
			if header.Schema != nil {
				v.validateSchemaRefs(header.Schema, headerPath+".schema", validRefs, result, baseURL)
			}
		}
	}

	// Validate links (OAS 3.x)
	for linkName, link := range response.Links {
		if link != nil && link.Ref != "" {
			v.validateRef(link.Ref, path+".links."+linkName, validRefs, result, baseURL)
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
			v.validateSchemaRefs(mediaTypeObj.Schema, path+".content."+mediaType+".schema", validRefs, result, baseURL)
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
			v.validateSchemaRefs(schema, "definitions."+name, validRefs, result, baseURL)
		}
	}

	// Validate refs in parameters
	for name, param := range doc.Parameters {
		if param != nil {
			v.validateParameterRef(param, "parameters."+name, validRefs, result, baseURL)
		}
	}

	// Validate refs in responses
	for name, response := range doc.Responses {
		if response != nil {
			v.validateResponseRef(response, "responses."+name, validRefs, result, baseURL)
		}
	}

	// Validate refs in paths
	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		pathPrefix := "paths." + pathPattern

		// Validate path-level parameters
		for i, param := range pathItem.Parameters {
			if param != nil {
				v.validateParameterRef(param, pathPrefix+".parameters["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
			}
		}

		// Validate each operation
		operations := parser.GetOperations(pathItem, parser.OASVersion20)
		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := pathPrefix + "." + method

			// Validate operation parameters
			for i, param := range op.Parameters {
				if param != nil {
					v.validateParameterRef(param, opPath+".parameters["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
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
				v.validateSchemaRefs(schema, "components.schemas."+name, validRefs, result, baseURL)
			}
		}

		// Validate parameters
		for name, param := range doc.Components.Parameters {
			if param != nil {
				v.validateParameterRef(param, "components.parameters."+name, validRefs, result, baseURL)
			}
		}

		// Validate responses
		for name, response := range doc.Components.Responses {
			if response != nil {
				v.validateResponseRef(response, "components.responses."+name, validRefs, result, baseURL)
			}
		}

		// Validate request bodies
		for name, requestBody := range doc.Components.RequestBodies {
			if requestBody != nil {
				v.validateRequestBodyRef(requestBody, "components.requestBodies."+name, validRefs, result, baseURL)
			}
		}

		// Validate headers
		for name, header := range doc.Components.Headers {
			if header != nil {
				headerPath := "components.headers." + name
				if header.Ref != "" {
					v.validateRef(header.Ref, headerPath, validRefs, result, baseURL)
				}
				if header.Schema != nil {
					v.validateSchemaRefs(header.Schema, headerPath+".schema", validRefs, result, baseURL)
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

			pathPrefix := "paths." + pathPattern

			// Validate PathItem $ref
			if pathItem.Ref != "" {
				v.validateRef(pathItem.Ref, pathPrefix, validRefs, result, baseURL)
			}

			// Validate path-level parameters
			for i, param := range pathItem.Parameters {
				if param != nil {
					v.validateParameterRef(param, pathPrefix+".parameters["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
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

		pathPrefix := "webhooks." + webhookName

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

		opPath := pathPrefix + "." + method

		// Validate operation parameters
		for i, param := range op.Parameters {
			if param != nil {
				v.validateParameterRef(param, opPath+".parameters["+strconv.Itoa(i)+"]", validRefs, result, baseURL)
			}
		}

		// Validate request body
		if op.RequestBody != nil {
			v.validateRequestBodyRef(op.RequestBody, opPath+".requestBody", validRefs, result, baseURL)
		}

		// Validate operation responses
		v.validateOperationResponses(op, opPath, validRefs, result, baseURL)
	}
}
