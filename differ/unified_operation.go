package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// diffParameterUnified compares individual Parameter objects
func (d *Differ) diffParameterUnified(source, target *parser.Parameter, path string, result *DiffResult) {
	// Required changed
	if source.Required != target.Required {
		// Making optional parameter required is error, making required optional is warning
		d.addChangeConditional(result, path+".required", ChangeTypeModified, CategoryParameter,
			!source.Required && target.Required, SeverityError, SeverityWarning,
			source.Required, target.Required, fmt.Sprintf("required changed from %v to %v", source.Required, target.Required))
	}

	// Type changed
	if source.Type != target.Type {
		// Check for compatible type changes
		severity := SeverityWarning
		if d.Mode == ModeBreaking && !isCompatibleTypeChange(source.Type, target.Type) {
			severity = SeverityError
		}
		d.addChange(result, path+".type", ChangeTypeModified, CategoryParameter,
			severity, source.Type, target.Type, fmt.Sprintf("type changed from %q to %q", source.Type, target.Type))
	}

	// Format changed
	if source.Format != target.Format {
		d.addChange(result, path+".format", ChangeTypeModified, CategoryParameter,
			SeverityWarning, source.Format, target.Format, fmt.Sprintf("format changed from %q to %q", source.Format, target.Format))
	}

	// Schema comparison (OAS 3.x)
	if source.Schema != nil || target.Schema != nil {
		if source.Schema != nil && target.Schema != nil {
			d.diffSchemaUnified(source.Schema, target.Schema, path+".schema", result)
		} else if source.Schema == nil {
			d.addChange(result, path+".schema", ChangeTypeAdded, CategoryParameter,
				SeverityInfo, nil, target.Schema, "parameter schema added")
		} else {
			d.addChange(result, path+".schema", ChangeTypeRemoved, CategoryParameter,
				SeverityWarning, source.Schema, nil, "parameter schema removed")
		}
	}

	// Compare Parameter extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffParametersUnified compares Parameter slices
func (d *Differ) diffParametersUnified(source, target []*parser.Parameter, path string, result *DiffResult) {
	// Build maps by name+in for easier comparison
	sourceMap := make(map[string]*parser.Parameter)
	for _, param := range source {
		key := param.Name + ":" + param.In
		sourceMap[key] = param
	}

	targetMap := make(map[string]*parser.Parameter)
	for _, param := range target {
		key := param.Name + ":" + param.In
		targetMap[key] = param
	}

	// Find removed parameters
	for key, sourceParam := range sourceMap {
		if _, exists := targetMap[key]; !exists {
			severity := SeverityWarning
			if d.Mode == ModeBreaking && sourceParam.Required {
				severity = SeverityError
			}
			d.addChange(result, fmt.Sprintf("%s[%s]", path, key), ChangeTypeRemoved, CategoryParameter,
				severity, sourceParam, nil, fmt.Sprintf("parameter %q in %s removed", sourceParam.Name, sourceParam.In))
		}
	}

	// Find added or modified parameters
	for key, targetParam := range targetMap {
		sourceParam, exists := sourceMap[key]
		if !exists {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && targetParam.Required {
				severity = SeverityWarning
			}
			d.addChange(result, fmt.Sprintf("%s[%s]", path, key), ChangeTypeAdded, CategoryParameter,
				severity, nil, targetParam, fmt.Sprintf("parameter %q in %s added", targetParam.Name, targetParam.In))
			continue
		}

		// Compare parameter details
		d.diffParameterUnified(sourceParam, targetParam, fmt.Sprintf("%s[%s]", path, key), result)
	}
}

// diffRequestBodyUnified compares RequestBody objects (OAS 3.x)
func (d *Differ) diffRequestBodyUnified(source, target *parser.RequestBody, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && target.Required {
			severity = SeverityWarning
		}
		d.addChange(result, path, ChangeTypeAdded, CategoryRequestBody,
			severity, nil, target, "request body added")
		return
	}

	if target == nil {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && source.Required {
			severity = SeverityError
		}
		d.addChange(result, path, ChangeTypeRemoved, CategoryRequestBody,
			severity, source, nil, "request body removed")
		return
	}

	// Required changed
	if source.Required != target.Required {
		// Making optional required is error, making required optional is info
		d.addChangeConditional(result, path+".required", ChangeTypeModified, CategoryRequestBody,
			!source.Required && target.Required, SeverityError, SeverityInfo,
			source.Required, target.Required, fmt.Sprintf("required changed from %v to %v", source.Required, target.Required))
	}

	// Compare content media types
	d.diffRequestBodyContentUnified(source.Content, target.Content, path, result)

	// Compare RequestBody extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffRequestBodyContentUnified compares request body content maps
func (d *Differ) diffRequestBodyContentUnified(source, target map[string]*parser.MediaType, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed media types
	for mediaType := range source {
		if _, exists := target[mediaType]; !exists {
			d.addChange(result, fmt.Sprintf("%s.content.%s", path, mediaType), ChangeTypeRemoved, CategoryRequestBody,
				SeverityError, nil, nil, fmt.Sprintf("request body media type %q removed", mediaType))
		}
	}

	// Find added or modified media types
	for mediaType, targetMedia := range target {
		sourceMedia, exists := source[mediaType]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.content.%s", path, mediaType), ChangeTypeAdded, CategoryRequestBody,
				SeverityInfo, nil, nil, fmt.Sprintf("request body media type %q added", mediaType))
			continue
		}

		// Compare media type details
		d.diffRequestBodyMediaTypeUnified(sourceMedia, targetMedia, fmt.Sprintf("%s.content.%s", path, mediaType), result)
	}
}

// diffRequestBodyMediaTypeUnified compares request body MediaType objects
func (d *Differ) diffRequestBodyMediaTypeUnified(source, target *parser.MediaType, path string, result *DiffResult) {
	// Compare schemas if present
	if source.Schema != nil && target.Schema != nil {
		d.diffSchemaUnified(source.Schema, target.Schema, path+".schema", result)
	} else if source.Schema != nil && target.Schema == nil {
		d.addChange(result, path+".schema", ChangeTypeRemoved, CategoryRequestBody,
			SeverityError, nil, nil, "request body schema removed")
	} else if source.Schema == nil && target.Schema != nil {
		d.addChange(result, path+".schema", ChangeTypeAdded, CategoryRequestBody,
			SeverityWarning, nil, nil, "request body schema added")
	}

	// Compare MediaType extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffOperationUnified compares Operation objects
func (d *Differ) diffOperationUnified(source, target *parser.Operation, path string, result *DiffResult) {
	// Compare operationId - important for code generation, considered a breaking change when modified
	if source.OperationID != target.OperationID {
		d.addChangeWithKey(result, path+".operationId", ChangeTypeModified, CategoryOperation,
			SeverityWarning, source.OperationID, target.OperationID,
			fmt.Sprintf("operationId changed from %q to %q", source.OperationID, target.OperationID),
			"operationId")
	}

	// Compare deprecated flag
	if source.Deprecated != target.Deprecated {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && !source.Deprecated && target.Deprecated {
			severity = SeverityWarning
		}
		d.addChange(result, path+".deprecated", ChangeTypeModified, CategoryOperation,
			severity, source.Deprecated, target.Deprecated, fmt.Sprintf("deprecated changed from %v to %v", source.Deprecated, target.Deprecated))
	}

	// Compare parameters
	d.diffParametersUnified(source.Parameters, target.Parameters, path+".parameters", result)

	// Compare responses
	d.diffResponsesUnified(source.Responses, target.Responses, path+".responses", result)

	// Compare request body (OAS 3.x)
	if source.RequestBody != nil || target.RequestBody != nil {
		d.diffRequestBodyUnified(source.RequestBody, target.RequestBody, path+".requestBody", result)
	}

	// Compare Operation extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffPathItemUnified compares PathItem objects
func (d *Differ) diffPathItemUnified(source, target *parser.PathItem, path string, result *DiffResult) {
	operations := map[string]struct {
		source *parser.Operation
		target *parser.Operation
	}{
		"get":     {source.Get, target.Get},
		"put":     {source.Put, target.Put},
		"post":    {source.Post, target.Post},
		"delete":  {source.Delete, target.Delete},
		"options": {source.Options, target.Options},
		"head":    {source.Head, target.Head},
		"patch":   {source.Patch, target.Patch},
		"trace":   {source.Trace, target.Trace},
	}

	for method, ops := range operations {
		opPath := fmt.Sprintf("%s.%s", path, method)

		if ops.source == nil && ops.target == nil {
			continue
		}

		if ops.source == nil && ops.target != nil {
			d.addChange(result, opPath, ChangeTypeAdded, CategoryOperation,
				SeverityInfo, nil, ops.target, fmt.Sprintf("operation %s added", method))
			continue
		}

		if ops.source != nil && ops.target == nil {
			d.addChange(result, opPath, ChangeTypeRemoved, CategoryOperation,
				SeverityError, ops.source, nil, fmt.Sprintf("operation %s removed", method))
			continue
		}

		// Compare operations
		d.diffOperationUnified(ops.source, ops.target, opPath, result)
	}

	// Compare PathItem extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffPathsUnified compares Paths objects
func (d *Differ) diffPathsUnified(source, target parser.Paths, path string, result *DiffResult) {
	// Find removed paths
	for pathName, sourceItem := range source {
		targetItem, exists := target[pathName]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, pathName), ChangeTypeRemoved, CategoryEndpoint,
				SeverityCritical, sourceItem, nil, fmt.Sprintf("endpoint %q removed", pathName))
			continue
		}

		// Compare path items
		d.diffPathItemUnified(sourceItem, targetItem, fmt.Sprintf("%s.%s", path, pathName), result)
	}

	// Find added paths
	for pathName, targetItem := range target {
		if _, exists := source[pathName]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, pathName), ChangeTypeAdded, CategoryEndpoint,
				SeverityInfo, nil, targetItem, fmt.Sprintf("endpoint %q added", pathName))
		}
	}
}

// diffComponentsUnified compares Components objects (OAS 3.x)
func (d *Differ) diffComponentsUnified(source, target *parser.Components, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, nil, "components added")
		return
	}

	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategorySchema,
			SeverityError, nil, nil, "components removed")
		return
	}

	// Compare schemas
	d.diffSchemasUnified(source.Schemas, target.Schemas, path+".schemas", result)

	// Compare security schemes
	d.diffSecuritySchemesUnified(source.SecuritySchemes, target.SecuritySchemes, path+".securitySchemes", result)

	// Compare mediaTypes (OAS 3.2+)
	d.diffMediaTypesUnified(source.MediaTypes, target.MediaTypes, path+".mediaTypes", result)

	// Compare Components extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffMediaTypesUnified compares reusable MediaType definitions (OAS 3.2+)
func (d *Differ) diffMediaTypesUnified(source, target map[string]*parser.MediaType, path string, result *DiffResult) {
	// Build key sets
	sourceKeys := make(map[string]bool)
	targetKeys := make(map[string]bool)
	for k := range source {
		sourceKeys[k] = true
	}
	for k := range target {
		targetKeys[k] = true
	}

	// Check for removed media types
	for key := range sourceKeys {
		if !targetKeys[key] {
			d.addChange(result, path+"."+key, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[key], nil, fmt.Sprintf("mediaType %q removed", key))
		}
	}

	// Check for added media types
	for key := range targetKeys {
		if !sourceKeys[key] {
			d.addChange(result, path+"."+key, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target[key], fmt.Sprintf("mediaType %q added", key))
		}
	}

	// Compare existing media types
	for key := range sourceKeys {
		if targetKeys[key] {
			d.diffMediaTypeUnified(source[key], target[key], path+"."+key, result)
		}
	}
}
