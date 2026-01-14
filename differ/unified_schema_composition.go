package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// diffSchemaAllOfUnified compares allOf composition schemas
func (d *Differ) diffSchemaAllOfUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	allOfPath := path + ".allOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", allOfPath, i)
		if i < len(target) {
			d.diffSchemaRecursiveUnified(sourceSchema, target[i], schemaPath, visited, result)
		} else {
			// Schema removed - relaxes validation
			d.addChange(result, schemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityInfo, sourceSchema, nil, fmt.Sprintf("allOf schema at index %d removed", i))
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", allOfPath, i)
		// Adding makes validation stricter
		d.addChange(result, schemaPath, ChangeTypeAdded, CategorySchema,
			SeverityError, nil, target[i], fmt.Sprintf("allOf schema at index %d added", i))
	}
}

// diffSchemaAnyOfUnified compares anyOf composition schemas
func (d *Differ) diffSchemaAnyOfUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	anyOfPath := path + ".anyOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", anyOfPath, i)
		if i < len(target) {
			d.diffSchemaRecursiveUnified(sourceSchema, target[i], schemaPath, visited, result)
		} else {
			// Removing reduces choices
			d.addChange(result, schemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceSchema, nil, fmt.Sprintf("anyOf schema at index %d removed", i))
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", anyOfPath, i)
		// Adding provides more choices
		d.addChange(result, schemaPath, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target[i], fmt.Sprintf("anyOf schema at index %d added", i))
	}
}

// diffSchemaOneOfUnified compares oneOf composition schemas
func (d *Differ) diffSchemaOneOfUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	oneOfPath := path + ".oneOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", oneOfPath, i)
		if i < len(target) {
			d.diffSchemaRecursiveUnified(sourceSchema, target[i], schemaPath, visited, result)
		} else {
			// Changes exclusive validation
			d.addChange(result, schemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceSchema, nil, fmt.Sprintf("oneOf schema at index %d removed", i))
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", oneOfPath, i)
		// Changes exclusive validation
		d.addChange(result, schemaPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target[i], fmt.Sprintf("oneOf schema at index %d added", i))
	}
}

// diffSchemaNotUnified compares not schemas
func (d *Differ) diffSchemaNotUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	notPath := path + ".not"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, notPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target, "not schema added")
		return
	}

	if target == nil {
		d.addChange(result, notPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "not schema removed")
		return
	}

	d.diffSchemaRecursiveUnified(source, target, notPath, visited, result)
}

// diffSchemaConditionalUnified compares conditional schemas (if/then/else)
func (d *Differ) diffSchemaConditionalUnified(sourceIf, sourceThen, sourceElse, targetIf, targetThen, targetElse *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	// Compare if condition
	if sourceIf != nil || targetIf != nil {
		ifPath := path + ".if"
		if sourceIf == nil {
			d.addChange(result, ifPath, ChangeTypeAdded, CategorySchema,
				SeverityWarning, nil, targetIf, "conditional if schema added")
		} else if targetIf == nil {
			d.addChange(result, ifPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceIf, nil, "conditional if schema removed")
		} else {
			d.diffSchemaRecursiveUnified(sourceIf, targetIf, ifPath, visited, result)
		}
	}

	// Compare then branch
	if sourceThen != nil || targetThen != nil {
		thenPath := path + ".then"
		if sourceThen == nil {
			d.addChange(result, thenPath, ChangeTypeAdded, CategorySchema,
				SeverityWarning, nil, targetThen, "conditional then schema added")
		} else if targetThen == nil {
			d.addChange(result, thenPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceThen, nil, "conditional then schema removed")
		} else {
			d.diffSchemaRecursiveUnified(sourceThen, targetThen, thenPath, visited, result)
		}
	}

	// Compare else branch
	if sourceElse != nil || targetElse != nil {
		elsePath := path + ".else"
		if sourceElse == nil {
			d.addChange(result, elsePath, ChangeTypeAdded, CategorySchema,
				SeverityWarning, nil, targetElse, "conditional else schema added")
		} else if targetElse == nil {
			d.addChange(result, elsePath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceElse, nil, "conditional else schema removed")
		} else {
			d.diffSchemaRecursiveUnified(sourceElse, targetElse, elsePath, visited, result)
		}
	}
}

// diffSchemaUnevaluatedPropertiesUnified compares unevaluatedProperties (JSON Schema 2020-12)
func (d *Differ) diffSchemaUnevaluatedPropertiesUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)
	fieldPath := path + ".unevaluatedProperties"

	// Handle unknown types
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("unevaluatedProperties has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("unevaluatedProperties has unexpected type in target: %T", target))
		return
	}

	// Both nil - no change
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// Added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, fieldPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, "unevaluatedProperties constraint added")
		return
	}

	// Removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		d.addChange(result, fieldPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "unevaluatedProperties constraint removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, "unevaluatedProperties type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		d.diffSchemaRecursiveUnified(source.(*parser.Schema), target.(*parser.Schema), fieldPath, visited, result)
	case schemaAdditionalPropsTypeBool:
		if source.(bool) != target.(bool) {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && source.(bool) && !target.(bool) {
				severity = SeverityError
			}
			d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
				severity, source, target, fmt.Sprintf("unevaluatedProperties changed from %v to %v", source, target))
		}
	case schemaAdditionalPropsTypeNil, schemaAdditionalPropsTypeUnknown:
		// Already handled above
	}
}

// diffSchemaUnevaluatedItemsUnified compares unevaluatedItems (JSON Schema 2020-12)
func (d *Differ) diffSchemaUnevaluatedItemsUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)
	fieldPath := path + ".unevaluatedItems"

	// Handle unknown types
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("unevaluatedItems has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("unevaluatedItems has unexpected type in target: %T", target))
		return
	}

	// Both nil - no change
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// Added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, fieldPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, "unevaluatedItems constraint added")
		return
	}

	// Removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		d.addChange(result, fieldPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "unevaluatedItems constraint removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, "unevaluatedItems type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		d.diffSchemaRecursiveUnified(source.(*parser.Schema), target.(*parser.Schema), fieldPath, visited, result)
	case schemaAdditionalPropsTypeBool:
		if source.(bool) != target.(bool) {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && source.(bool) && !target.(bool) {
				severity = SeverityError
			}
			d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
				severity, source, target, fmt.Sprintf("unevaluatedItems changed from %v to %v", source, target))
		}
	case schemaAdditionalPropsTypeNil, schemaAdditionalPropsTypeUnknown:
		// Already handled above
	}
}

// diffSchemaContentFieldsUnified compares content keywords (JSON Schema 2020-12)
func (d *Differ) diffSchemaContentFieldsUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	// ContentEncoding
	if source.ContentEncoding != target.ContentEncoding {
		if source.ContentEncoding == "" {
			d.addChange(result, path+".contentEncoding", ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target.ContentEncoding, "contentEncoding added")
		} else if target.ContentEncoding == "" {
			d.addChange(result, path+".contentEncoding", ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source.ContentEncoding, nil, "contentEncoding removed")
		} else {
			d.addChange(result, path+".contentEncoding", ChangeTypeModified, CategorySchema,
				SeverityWarning, source.ContentEncoding, target.ContentEncoding,
				fmt.Sprintf("contentEncoding changed from %q to %q", source.ContentEncoding, target.ContentEncoding))
		}
	}

	// ContentMediaType
	if source.ContentMediaType != target.ContentMediaType {
		if source.ContentMediaType == "" {
			d.addChange(result, path+".contentMediaType", ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target.ContentMediaType, "contentMediaType added")
		} else if target.ContentMediaType == "" {
			d.addChange(result, path+".contentMediaType", ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source.ContentMediaType, nil, "contentMediaType removed")
		} else {
			d.addChange(result, path+".contentMediaType", ChangeTypeModified, CategorySchema,
				SeverityWarning, source.ContentMediaType, target.ContentMediaType,
				fmt.Sprintf("contentMediaType changed from %q to %q", source.ContentMediaType, target.ContentMediaType))
		}
	}

	// ContentSchema
	if source.ContentSchema != nil || target.ContentSchema != nil {
		contentSchemaPath := path + ".contentSchema"
		if source.ContentSchema == nil {
			d.addChange(result, contentSchemaPath, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target.ContentSchema, "contentSchema added")
		} else if target.ContentSchema == nil {
			d.addChange(result, contentSchemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source.ContentSchema, nil, "contentSchema removed")
		} else {
			d.diffSchemaRecursiveUnified(source.ContentSchema, target.ContentSchema, contentSchemaPath, visited, result)
		}
	}
}

// diffSchemaPrefixItemsUnified compares prefixItems arrays (JSON Schema 2020-12)
func (d *Differ) diffSchemaPrefixItemsUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	prefixPath := path + ".prefixItems"

	// Both nil/empty
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Added
	if len(source) == 0 && len(target) > 0 {
		d.addChange(result, prefixPath, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target, "prefixItems added")
		return
	}

	// Removed
	if len(source) > 0 && len(target) == 0 {
		d.addChange(result, prefixPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "prefixItems removed")
		return
	}

	// Compare each item
	maxLen := max(len(source), len(target))

	for i := range maxLen {
		itemPath := fmt.Sprintf("%s[%d]", prefixPath, i)
		if i >= len(source) {
			d.addChange(result, itemPath, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target[i], "prefixItem added")
		} else if i >= len(target) {
			d.addChange(result, itemPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[i], nil, "prefixItem removed")
		} else {
			d.diffSchemaRecursiveUnified(source[i], target[i], itemPath, visited, result)
		}
	}
}

// diffSchemaContainsUnified compares contains schema (JSON Schema 2020-12)
func (d *Differ) diffSchemaContainsUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	containsPath := path + ".contains"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, containsPath, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target, "contains constraint added")
		return
	}

	if target == nil {
		d.addChange(result, containsPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "contains constraint removed")
		return
	}

	d.diffSchemaRecursiveUnified(source, target, containsPath, visited, result)
}

// diffSchemaPropertyNamesUnified compares propertyNames schema (JSON Schema 2020-12)
func (d *Differ) diffSchemaPropertyNamesUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	propNamesPath := path + ".propertyNames"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, propNamesPath, ChangeTypeAdded, CategorySchema,
			SeverityError, nil, target, "propertyNames constraint added")
		return
	}

	if target == nil {
		d.addChange(result, propNamesPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "propertyNames constraint removed")
		return
	}

	d.diffSchemaRecursiveUnified(source, target, propNamesPath, visited, result)
}

// diffSchemaDependentSchemasUnified compares dependentSchemas (JSON Schema 2020-12)
func (d *Differ) diffSchemaDependentSchemasUnified(source, target map[string]*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	depPath := path + ".dependentSchemas"

	// Build sets for comparison
	sourceKeys := make(map[string]bool)
	targetKeys := make(map[string]bool)
	for k := range source {
		sourceKeys[k] = true
	}
	for k := range target {
		targetKeys[k] = true
	}

	// Check for added/removed keys
	for key := range sourceKeys {
		if !targetKeys[key] {
			d.addChange(result, depPath+"."+key, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[key], nil, fmt.Sprintf("dependentSchema %q removed", key))
		}
	}
	for key := range targetKeys {
		if !sourceKeys[key] {
			d.addChange(result, depPath+"."+key, ChangeTypeAdded, CategorySchema,
				SeverityError, nil, target[key], fmt.Sprintf("dependentSchema %q added", key))
		}
	}

	// Compare existing keys
	for key := range sourceKeys {
		if targetKeys[key] {
			d.diffSchemaRecursiveUnified(source[key], target[key], depPath+"."+key, visited, result)
		}
	}
}
