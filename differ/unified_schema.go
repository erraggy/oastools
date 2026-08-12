package differ

import (
	"fmt"

	"github.com/erraggy/oastools/internal/schemautil"
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
	d.diffSchemaAdditionalItemsUnified(source.AdditionalItems, target.AdditionalItems, path, visited, result)

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

// diffSchemaTupleUnified compares two OAS 2.0 tuple-form values element by
// element, using the same severities as [Differ.diffSchemaPrefixItemsUnified]:
// prefixItems is the JSON Schema 2020-12 spelling of the same construct, so an
// identical edit must report identically under either name.
func (d *Differ) diffSchemaTupleUnified(source, target []*parser.Schema, fieldPath, fieldName string, visited *schemaVisited, result *DiffResult) {
	maxLen := max(len(source), len(target))

	for i := range maxLen {
		itemPath := fieldPath + schemautil.IndexSuffix(i)
		switch {
		case i >= len(source):
			d.addChange(result, itemPath, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target[i], fmt.Sprintf("%s tuple element added", fieldName))
		case i >= len(target):
			d.addChange(result, itemPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[i], nil, fmt.Sprintf("%s tuple element removed", fieldName))
		default:
			d.diffSchemaRecursiveUnified(source[i], target[i], itemPath, visited, result)
		}
	}
}

// shapeChangeMessage describes a schema-or-bool field changing shape, naming
// both shapes in OAS terms rather than in Go type names.
func shapeChangeMessage(fieldName string, sourceKind, targetKind schemaOrBoolKind) string {
	return fmt.Sprintf("%s changed from %s to %s", fieldName,
		schemaOrBoolShapeName(sourceKind), schemaOrBoolShapeName(targetKind))
}

// diffSchemaItemsUnified compares schema Items field
func (d *Differ) diffSchemaItemsUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaOrBoolKind(source)
	targetType := getSchemaOrBoolKind(target)
	itemsPath := path + ".items"

	// Handle unknown types
	if sourceType == schemaOrBoolUnknown && targetType == schemaOrBoolUnknown {
		return
	}
	if sourceType == schemaOrBoolUnknown {
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, "items holds an unrecognized value in source")
		return
	}
	if targetType == schemaOrBoolUnknown {
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, "items holds an unrecognized value in target")
		return
	}

	// Both nil
	if sourceType == schemaOrBoolNil && targetType == schemaOrBoolNil {
		return
	}

	// Items added
	if sourceType == schemaOrBoolNil && targetType != schemaOrBoolNil {
		d.addChange(result, itemsPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target, "items schema added")
		return
	}

	// Items removed
	if sourceType != schemaOrBoolNil && targetType == schemaOrBoolNil {
		d.addChange(result, itemsPath, ChangeTypeRemoved, CategorySchema,
			SeverityError, source, nil, "items schema removed")
		return
	}

	// Shape changed
	if sourceType != targetType {
		severity := SeverityError
		if sourceType == schemaOrBoolBool && targetType == schemaOrBoolSchema {
			severity = SeverityWarning
		}
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			severity, source, target, shapeChangeMessage("items", sourceType, targetType))
		return
	}

	// Both same shape - compare
	switch sourceType {
	case schemaOrBoolSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, itemsPath, visited, result)
	case schemaOrBoolTuple:
		d.diffSchemaTupleUnified(source.([]*parser.Schema), target.([]*parser.Schema), itemsPath, "items", visited, result)
	case schemaOrBoolBool:
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
	case schemaOrBoolNil, schemaOrBoolUnknown:
		// Already handled above before the switch
	}
}

// diffSchemaAdditionalPropertiesUnified compares additionalProperties field
func (d *Differ) diffSchemaAdditionalPropertiesUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	d.diffSchemaAdditionalUnified(source, target, path, "additionalProperties", visited, result)
}

// diffSchemaAdditionalItemsUnified compares the additionalItems field, which
// carries the same severity policy as additionalProperties.
func (d *Differ) diffSchemaAdditionalItemsUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	d.diffSchemaAdditionalUnified(source, target, path, "additionalItems", visited, result)
}

// diffSchemaAdditionalUnified compares an additionalProperties or
// additionalItems field, named by fieldName.
func (d *Differ) diffSchemaAdditionalUnified(source, target any, path, fieldName string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaOrBoolKind(source)
	targetType := getSchemaOrBoolKind(target)
	addPropsPath := path + "." + fieldName

	// Handle unknown types
	if sourceType == schemaOrBoolUnknown && targetType == schemaOrBoolUnknown {
		return
	}
	if sourceType == schemaOrBoolUnknown {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("%s holds an unrecognized value in source", fieldName))
		return
	}
	if targetType == schemaOrBoolUnknown {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("%s holds an unrecognized value in target", fieldName))
		return
	}

	// Both nil
	if sourceType == schemaOrBoolNil && targetType == schemaOrBoolNil {
		return
	}

	// Constraint added
	if sourceType == schemaOrBoolNil && targetType != schemaOrBoolNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaOrBoolBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, addPropsPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, fmt.Sprintf("%s constraint added", fieldName))
		return
	}

	// Constraint removed
	if sourceType != schemaOrBoolNil && targetType == schemaOrBoolNil {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && sourceType == schemaOrBoolBool && !source.(bool) {
			severity = SeverityInfo
		}
		d.addChange(result, addPropsPath, ChangeTypeRemoved, CategorySchema,
			severity, source, nil, fmt.Sprintf("%s constraint removed", fieldName))
		return
	}

	// Shape changed
	if sourceType != targetType {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, shapeChangeMessage(fieldName, sourceType, targetType))
		return
	}

	// Both same shape - compare
	switch sourceType {
	case schemaOrBoolSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, addPropsPath, visited, result)
	case schemaOrBoolTuple:
		d.diffSchemaTupleUnified(source.([]*parser.Schema), target.([]*parser.Schema), addPropsPath, fieldName, visited, result)
	case schemaOrBoolBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && sourceBool && !targetBool {
				severity = SeverityError
			}
			d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
				severity, sourceBool, targetBool, fmt.Sprintf("%s changed from %v to %v", fieldName, sourceBool, targetBool))
		}
	case schemaOrBoolNil, schemaOrBoolUnknown:
		// Already handled above before the switch
	}
}
