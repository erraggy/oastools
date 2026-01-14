package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// diffSchemasUnified compares schema maps
func (d *Differ) diffSchemasUnified(source, target map[string]*parser.Schema, path string, result *DiffResult) {
	// Find removed schemas
	for name, sourceSchema := range source {
		targetSchema, exists := target[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeRemoved, CategorySchema,
				SeverityError, nil, nil, fmt.Sprintf("schema %q removed", name))
			continue
		}

		// Compare schema details
		d.diffSchemaUnified(sourceSchema, targetSchema, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Find added schemas
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, nil, fmt.Sprintf("schema %q added", name))
		}
	}
}

// diffSchemaUnified compares individual Schema objects
func (d *Differ) diffSchemaUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// Use recursive diffing with cycle detection
	visited := newSchemaVisited()
	d.diffSchemaRecursiveUnified(source, target, path, visited, result)
}

// diffSchemaRecursiveUnified performs recursive schema comparison with cycle detection
func (d *Differ) diffSchemaRecursiveUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	// Nil handling
	if source == nil && target == nil {
		return
	}
	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target, "schema added")
		return
	}
	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategorySchema,
			SeverityError, source, nil, "schema removed")
		return
	}

	// Cycle detection
	if visited.enter(source, target, path) {
		return
	}
	defer visited.leave(source, target)

	// Compare metadata
	d.diffSchemaMetadataUnified(source, target, path, result)

	// Compare type and format
	d.diffSchemaTypeUnified(source, target, path, result)

	// Compare constraints
	d.diffSchemaNumericConstraintsUnified(source, target, path, result)
	d.diffSchemaStringConstraintsUnified(source, target, path, result)
	d.diffSchemaArrayConstraintsUnified(source, target, path, result)
	d.diffSchemaObjectConstraintsUnified(source, target, path, result)

	// Compare required fields
	d.diffSchemaRequiredFieldsUnified(source, target, path, result)

	// Compare OAS-specific fields
	d.diffSchemaOASFieldsUnified(source, target, path, result)

	// Compare enum values
	d.diffEnumUnified(source.Enum, target.Enum, path+".enum", result)

	// Compare recursive/complex fields
	d.diffSchemaPropertiesUnified(source.Properties, target.Properties, source.Required, target.Required, path, visited, result)
	d.diffSchemaItemsUnified(source.Items, target.Items, path, visited, result)
	d.diffSchemaAdditionalPropertiesUnified(source.AdditionalProperties, target.AdditionalProperties, path, visited, result)

	// Compare composition fields
	d.diffSchemaAllOfUnified(source.AllOf, target.AllOf, path, visited, result)
	d.diffSchemaAnyOfUnified(source.AnyOf, target.AnyOf, path, visited, result)
	d.diffSchemaOneOfUnified(source.OneOf, target.OneOf, path, visited, result)
	d.diffSchemaNotUnified(source.Not, target.Not, path, visited, result)

	// Compare conditional schemas
	d.diffSchemaConditionalUnified(source.If, source.Then, source.Else, target.If, target.Then, target.Else, path, visited, result)

	// JSON Schema 2020-12 fields
	d.diffSchemaUnevaluatedPropertiesUnified(source.UnevaluatedProperties, target.UnevaluatedProperties, path, visited, result)
	d.diffSchemaUnevaluatedItemsUnified(source.UnevaluatedItems, target.UnevaluatedItems, path, visited, result)
	d.diffSchemaContentFieldsUnified(source, target, path, visited, result)
	d.diffSchemaPrefixItemsUnified(source.PrefixItems, target.PrefixItems, path, visited, result)
	d.diffSchemaContainsUnified(source.Contains, target.Contains, path, visited, result)
	d.diffSchemaPropertyNamesUnified(source.PropertyNames, target.PropertyNames, path, visited, result)
	d.diffSchemaDependentSchemasUnified(source.DependentSchemas, target.DependentSchemas, path, visited, result)

	// Compare extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffSchemaMetadataUnified compares schema metadata fields
func (d *Differ) diffSchemaMetadataUnified(source, target *parser.Schema, path string, result *DiffResult) {
	if source.Title != target.Title {
		d.addChange(result, path+".title", ChangeTypeModified, CategorySchema,
			SeverityInfo, source.Title, target.Title, "schema title changed")
	}

	if source.Description != target.Description {
		d.addChange(result, path+".description", ChangeTypeModified, CategorySchema,
			SeverityInfo, source.Description, target.Description, "schema description changed")
	}
}

// diffSchemaTypeUnified compares schema type and format fields
func (d *Differ) diffSchemaTypeUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// Type can be string or []string in OAS 3.1+
	sourceTypeStr := formatSchemaType(source.Type)
	targetTypeStr := formatSchemaType(target.Type)
	if sourceTypeStr != targetTypeStr {
		d.addChange(result, path+".type", ChangeTypeModified, CategorySchema,
			SeverityError, source.Type, target.Type, "schema type changed")
	}

	if source.Format != target.Format {
		d.addChange(result, path+".format", ChangeTypeModified, CategorySchema,
			SeverityWarning, source.Format, target.Format, "schema format changed")
	}
}

// diffSchemaNumericConstraintsUnified compares numeric validation constraints
func (d *Differ) diffSchemaNumericConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MultipleOf
	if source.MultipleOf != nil && target.MultipleOf != nil && *source.MultipleOf != *target.MultipleOf {
		d.addChange(result, path+".multipleOf", ChangeTypeModified, CategorySchema,
			SeverityWarning, *source.MultipleOf, *target.MultipleOf, "multipleOf constraint changed")
	}

	// Maximum
	if source.Maximum != nil && target.Maximum != nil && *source.Maximum != *target.Maximum {
		// Tightening (lowering max) is error, relaxing is warning
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.Maximum < *source.Maximum {
			severity = SeverityError
		}
		d.addChange(result, path+".maximum", ChangeTypeModified, CategorySchema,
			severity, *source.Maximum, *target.Maximum, "maximum constraint changed")
	} else if source.Maximum == nil && target.Maximum != nil {
		d.addChange(result, path+".maximum", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.Maximum, "maximum constraint added")
	}

	// Minimum
	if source.Minimum != nil && target.Minimum != nil && *source.Minimum != *target.Minimum {
		// Tightening (raising min) is error, relaxing is warning
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.Minimum > *source.Minimum {
			severity = SeverityError
		}
		d.addChange(result, path+".minimum", ChangeTypeModified, CategorySchema,
			severity, *source.Minimum, *target.Minimum, "minimum constraint changed")
	} else if source.Minimum == nil && target.Minimum != nil {
		d.addChange(result, path+".minimum", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.Minimum, "minimum constraint added")
	}
}

// diffSchemaStringConstraintsUnified compares string validation constraints
func (d *Differ) diffSchemaStringConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxLength
	if source.MaxLength != nil && target.MaxLength != nil && *source.MaxLength != *target.MaxLength {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MaxLength < *source.MaxLength {
			severity = SeverityError
		}
		d.addChange(result, path+".maxLength", ChangeTypeModified, CategorySchema,
			severity, *source.MaxLength, *target.MaxLength, "maxLength constraint changed")
	} else if source.MaxLength == nil && target.MaxLength != nil {
		d.addChange(result, path+".maxLength", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MaxLength, "maxLength constraint added")
	}

	// MinLength
	if source.MinLength != nil && target.MinLength != nil && *source.MinLength != *target.MinLength {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MinLength > *source.MinLength {
			severity = SeverityError
		}
		d.addChange(result, path+".minLength", ChangeTypeModified, CategorySchema,
			severity, *source.MinLength, *target.MinLength, "minLength constraint changed")
	} else if source.MinLength == nil && target.MinLength != nil {
		d.addChange(result, path+".minLength", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MinLength, "minLength constraint added")
	}

	// Pattern
	if source.Pattern != target.Pattern {
		if source.Pattern != "" || target.Pattern != "" {
			severity := SeverityWarning
			if d.Mode == ModeBreaking && source.Pattern == "" && target.Pattern != "" {
				severity = SeverityError
			}
			d.addChange(result, path+".pattern", ChangeTypeModified, CategorySchema,
				severity, source.Pattern, target.Pattern, "pattern constraint changed")
		}
	}
}

// diffSchemaArrayConstraintsUnified compares array validation constraints
func (d *Differ) diffSchemaArrayConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxItems
	if source.MaxItems != nil && target.MaxItems != nil && *source.MaxItems != *target.MaxItems {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MaxItems < *source.MaxItems {
			severity = SeverityError
		}
		d.addChange(result, path+".maxItems", ChangeTypeModified, CategorySchema,
			severity, *source.MaxItems, *target.MaxItems, "maxItems constraint changed")
	} else if source.MaxItems == nil && target.MaxItems != nil {
		d.addChange(result, path+".maxItems", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MaxItems, "maxItems constraint added")
	}

	// MinItems
	if source.MinItems != nil && target.MinItems != nil && *source.MinItems != *target.MinItems {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MinItems > *source.MinItems {
			severity = SeverityError
		}
		d.addChange(result, path+".minItems", ChangeTypeModified, CategorySchema,
			severity, *source.MinItems, *target.MinItems, "minItems constraint changed")
	} else if source.MinItems == nil && target.MinItems != nil {
		d.addChange(result, path+".minItems", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MinItems, "minItems constraint added")
	}

	// UniqueItems
	if source.UniqueItems != target.UniqueItems {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && !source.UniqueItems && target.UniqueItems {
			severity = SeverityError
		}
		d.addChange(result, path+".uniqueItems", ChangeTypeModified, CategorySchema,
			severity, source.UniqueItems, target.UniqueItems, "uniqueItems constraint changed")
	}
}

// diffSchemaObjectConstraintsUnified compares object validation constraints
func (d *Differ) diffSchemaObjectConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxProperties
	if source.MaxProperties != nil && target.MaxProperties != nil && *source.MaxProperties != *target.MaxProperties {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MaxProperties < *source.MaxProperties {
			severity = SeverityError
		}
		d.addChange(result, path+".maxProperties", ChangeTypeModified, CategorySchema,
			severity, *source.MaxProperties, *target.MaxProperties, "maxProperties constraint changed")
	} else if source.MaxProperties == nil && target.MaxProperties != nil {
		d.addChange(result, path+".maxProperties", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MaxProperties, "maxProperties constraint added")
	}

	// MinProperties
	if source.MinProperties != nil && target.MinProperties != nil && *source.MinProperties != *target.MinProperties {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MinProperties > *source.MinProperties {
			severity = SeverityError
		}
		d.addChange(result, path+".minProperties", ChangeTypeModified, CategorySchema,
			severity, *source.MinProperties, *target.MinProperties, "minProperties constraint changed")
	} else if source.MinProperties == nil && target.MinProperties != nil {
		d.addChange(result, path+".minProperties", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MinProperties, "minProperties constraint added")
	}
}

// diffSchemaRequiredFieldsUnified compares required field lists
func (d *Differ) diffSchemaRequiredFieldsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	sourceRequired := make(map[string]bool)
	for _, req := range source.Required {
		sourceRequired[req] = true
	}
	targetRequired := make(map[string]bool)
	for _, req := range target.Required {
		targetRequired[req] = true
	}

	// Removed required fields - relaxing
	for req := range sourceRequired {
		if !targetRequired[req] {
			d.addChange(result, fmt.Sprintf("%s.required[%s]", path, req), ChangeTypeRemoved, CategorySchema,
				SeverityInfo, nil, nil, fmt.Sprintf("required field %q removed", req))
		}
	}

	// Added required fields - stricter
	for req := range targetRequired {
		if !sourceRequired[req] {
			d.addChange(result, fmt.Sprintf("%s.required[%s]", path, req), ChangeTypeAdded, CategorySchema,
				SeverityError, nil, nil, fmt.Sprintf("required field %q added", req))
		}
	}
}

// diffSchemaOASFieldsUnified compares OAS-specific schema fields
func (d *Differ) diffSchemaOASFieldsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// Nullable
	if source.Nullable != target.Nullable {
		// Removing nullable is breaking (was accepting null, now not)
		severity := SeverityWarning
		if d.Mode == ModeBreaking && source.Nullable && !target.Nullable {
			severity = SeverityError
		}
		d.addChange(result, path+".nullable", ChangeTypeModified, CategorySchema,
			severity, source.Nullable, target.Nullable, "nullable changed")
	}

	// ReadOnly
	if source.ReadOnly != target.ReadOnly {
		d.addChange(result, path+".readOnly", ChangeTypeModified, CategorySchema,
			SeverityWarning, source.ReadOnly, target.ReadOnly, "readOnly changed")
	}

	// WriteOnly
	if source.WriteOnly != target.WriteOnly {
		d.addChange(result, path+".writeOnly", ChangeTypeModified, CategorySchema,
			SeverityWarning, source.WriteOnly, target.WriteOnly, "writeOnly changed")
	}

	// Deprecated
	if source.Deprecated != target.Deprecated {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && !source.Deprecated && target.Deprecated {
			severity = SeverityWarning
		}
		d.addChange(result, path+".deprecated", ChangeTypeModified, CategorySchema,
			severity, source.Deprecated, target.Deprecated, "deprecated status changed")
	}
}

// diffEnumUnified compares enum values
func (d *Differ) diffEnumUnified(source, target []any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	sourceMap := make(map[string]struct{})
	for _, val := range source {
		sourceMap[anyToString(val)] = struct{}{}
	}

	targetMap := make(map[string]struct{})
	for _, val := range target {
		targetMap[anyToString(val)] = struct{}{}
	}

	// Removed enum values - restricts valid values
	for val := range sourceMap {
		if _, ok := targetMap[val]; !ok {
			d.addChange(result, path, ChangeTypeRemoved, CategoryParameter,
				SeverityError, nil, nil, fmt.Sprintf("enum value %q removed", val))
		}
	}

	// Added enum values - expands valid values
	for val := range targetMap {
		if _, ok := sourceMap[val]; !ok {
			d.addChange(result, path, ChangeTypeAdded, CategoryParameter,
				SeverityInfo, nil, nil, fmt.Sprintf("enum value %q added", val))
		}
	}
}

// diffSchemaPropertiesUnified compares schema properties maps
func (d *Differ) diffSchemaPropertiesUnified(source, target map[string]*parser.Schema, sourceRequired, targetRequired []string, path string, visited *schemaVisited, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed properties
	for name, sourceSchema := range source {
		propPath := fmt.Sprintf("%s.properties.%s", path, name)
		if targetSchema, exists := target[name]; !exists {
			// Severity depends on whether it was required
			severity := SeverityWarning
			if d.Mode == ModeBreaking && isPropertyRequired(name, sourceRequired) {
				severity = SeverityError
			}
			d.addChange(result, propPath, ChangeTypeRemoved, CategorySchema,
				severity, sourceSchema, nil, fmt.Sprintf("property %q removed", name))
		} else {
			// Property exists in both - recursive comparison
			d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, propPath, visited, result)
		}
	}

	// Find added properties
	for name, targetSchema := range target {
		if _, exists := source[name]; !exists {
			propPath := fmt.Sprintf("%s.properties.%s", path, name)
			// Severity depends on whether it's required
			severity := SeverityInfo
			if d.Mode == ModeBreaking && isPropertyRequired(name, targetRequired) {
				severity = SeverityWarning
			}
			d.addChange(result, propPath, ChangeTypeAdded, CategorySchema,
				severity, nil, targetSchema, fmt.Sprintf("property %q added", name))
		}
	}
}

// diffSchemaItemsUnified compares schema Items field
func (d *Differ) diffSchemaItemsUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaItemsType(source)
	targetType := getSchemaItemsType(target)
	itemsPath := path + ".items"

	// Handle unknown types
	if sourceType == schemaItemsTypeUnknown && targetType == schemaItemsTypeUnknown {
		return
	}
	if sourceType == schemaItemsTypeUnknown {
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("items has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaItemsTypeUnknown {
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("items has unexpected type in target: %T", target))
		return
	}

	// Both nil
	if sourceType == schemaItemsTypeNil && targetType == schemaItemsTypeNil {
		return
	}

	// Items added
	if sourceType == schemaItemsTypeNil && targetType != schemaItemsTypeNil {
		d.addChange(result, itemsPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target, "items schema added")
		return
	}

	// Items removed
	if sourceType != schemaItemsTypeNil && targetType == schemaItemsTypeNil {
		d.addChange(result, itemsPath, ChangeTypeRemoved, CategorySchema,
			SeverityError, source, nil, "items schema removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		severity := SeverityError
		if sourceType == schemaItemsTypeBool && targetType == schemaItemsTypeSchema {
			severity = SeverityWarning
		}
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			severity, source, target, "items type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaItemsTypeSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, itemsPath, visited, result)
	case schemaItemsTypeBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityWarning
			if d.Mode == ModeBreaking && sourceBool && !targetBool {
				severity = SeverityError
			}
			d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
				severity, sourceBool, targetBool, fmt.Sprintf("items changed from %v to %v", sourceBool, targetBool))
		}
	case schemaItemsTypeNil, schemaItemsTypeUnknown:
		// Already handled above before the switch
	}
}

// diffSchemaAdditionalPropertiesUnified compares additionalProperties field
func (d *Differ) diffSchemaAdditionalPropertiesUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)
	addPropsPath := path + ".additionalProperties"

	// Handle unknown types
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("additionalProperties has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("additionalProperties has unexpected type in target: %T", target))
		return
	}

	// Both nil
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// additionalProperties added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, addPropsPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, "additionalProperties constraint added")
		return
	}

	// additionalProperties removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && sourceType == schemaAdditionalPropsTypeBool && !source.(bool) {
			severity = SeverityInfo
		}
		d.addChange(result, addPropsPath, ChangeTypeRemoved, CategorySchema,
			severity, source, nil, "additionalProperties constraint removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, "additionalProperties type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, addPropsPath, visited, result)
	case schemaAdditionalPropsTypeBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && sourceBool && !targetBool {
				severity = SeverityError
			}
			d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
				severity, sourceBool, targetBool, fmt.Sprintf("additionalProperties changed from %v to %v", sourceBool, targetBool))
		}
	case schemaAdditionalPropsTypeNil, schemaAdditionalPropsTypeUnknown:
		// Already handled above before the switch
	}
}
