package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// buildTypesFileData builds the template data for types.go file generation.
// This is the main entry point for template-based type generation.
func (cg *oas3CodeGenerator) buildTypesFileData() *TypesFileData {
	data := &TypesFileData{}

	// Build header with package name and imports
	data.Header = cg.buildHeaderData()

	// Process schemas from components
	var schemas []schemaEntry
	if cg.doc.Components != nil && cg.doc.Components.Schemas != nil {
		for name, schema := range cg.doc.Components.Schemas {
			if schema == nil {
				continue
			}
			// Check for duplicate type names (e.g., "user_profile" and "UserProfile" both become "UserProfile")
			typeName := toTypeName(name)
			if cg.generatedTypes[typeName] {
				cg.addIssue(fmt.Sprintf("components.schemas.%s", name),
					fmt.Sprintf("duplicate type name %s - skipping", typeName), SeverityWarning)
				continue
			}
			cg.generatedTypes[typeName] = true

			schemas = append(schemas, schemaEntry{name: name, schema: schema})
			cg.schemaNames["#/components/schemas/"+name] = typeName
		}
	}

	// Sort schemas for deterministic output
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].name < schemas[j].name
	})

	// Build type definitions
	for _, entry := range schemas {
		typeDef := cg.buildTypeDefinition(entry.name, entry.schema)
		data.Types = append(data.Types, typeDef)
		cg.result.GeneratedTypes++
	}

	return data
}

// buildHeaderData builds the header data with package name and imports.
func (cg *oas3CodeGenerator) buildHeaderData() HeaderData {
	imports := make(map[string]bool)

	// Check if we need time or encoding/json imports
	if cg.doc.Components != nil && cg.doc.Components.Schemas != nil {
		for _, schema := range cg.doc.Components.Schemas {
			if needsTimeImport(schema) {
				imports["time"] = true
			}
			// Check if schema has discriminator (which generates UnmarshalJSON)
			if hasDiscriminator(schema) {
				imports["encoding/json"] = true
			}
		}
	}

	// Convert to sorted slice
	importList := make([]string, 0, len(imports))
	for imp := range imports {
		importList = append(importList, imp)
	}
	sort.Strings(importList)

	return HeaderData{
		PackageName: cg.result.PackageName,
		Imports:     importList,
	}
}

// hasDiscriminator checks if a schema has a discriminator that will generate UnmarshalJSON
func hasDiscriminator(schema *parser.Schema) bool {
	if schema == nil {
		return false
	}
	// Check oneOf/anyOf with discriminator
	if (len(schema.OneOf) > 0 || len(schema.AnyOf) > 0) &&
		schema.Discriminator != nil &&
		schema.Discriminator.PropertyName != "" {
		return true
	}
	return false
}

// buildTypeDefinition builds a TypeDefinition from a schema.
// This determines which kind of type to generate and calls the appropriate builder.
func (cg *oas3CodeGenerator) buildTypeDefinition(name string, schema *parser.Schema) TypeDefinition {
	typeName := toTypeName(name)

	// Handle $ref - creates alias
	if schema.Ref != "" {
		return cg.buildAliasTypeDefinition(typeName, schema)
	}

	// Determine schema type
	schemaType := getSchemaType(schema)

	switch schemaType {
	case "object":
		return cg.buildStructTypeDefinition(typeName, name, schema)

	case "array":
		return cg.buildArrayAliasTypeDefinition(typeName, schema)

	case "string":
		// Check for enum
		if len(schema.Enum) > 0 {
			return cg.buildEnumTypeDefinition(typeName, schema)
		}
		return cg.buildStringAliasTypeDefinition(typeName, schema)

	case "integer":
		return cg.buildIntegerAliasTypeDefinition(typeName, schema)

	case "number":
		return cg.buildNumberAliasTypeDefinition(typeName, schema)

	case "boolean":
		return cg.buildBooleanAliasTypeDefinition(typeName, schema)

	default:
		// Handle allOf, oneOf, anyOf
		if len(schema.AllOf) > 0 {
			return cg.buildAllOfTypeDefinition(typeName, schema)
		}
		if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
			return cg.buildOneOfTypeDefinition(typeName, name, schema)
		}
		// Default to any
		return cg.buildAnyAliasTypeDefinition(typeName)
	}
}

// buildStructTypeDefinition builds a struct type definition from an object schema.
func (cg *oas3CodeGenerator) buildStructTypeDefinition(typeName, originalName string, schema *parser.Schema) TypeDefinition {
	structData := &StructData{
		TypeName:     typeName,
		OriginalName: originalName,
	}

	// Set comment
	if schema.Description != "" {
		structData.Comment = cleanDescription(schema.Description)
	}

	// Build fields from properties
	if schema.Properties != nil {
		var propNames []string
		for propName := range schema.Properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		// Track used field names to avoid duplicates (e.g., @id and id both become Id)
		usedFieldNames := make(map[string]int)

		for _, propName := range propNames {
			propSchema := schema.Properties[propName]
			if propSchema == nil {
				continue
			}

			field := cg.buildFieldData(propName, propSchema, isRequired(schema.Required, propName))

			// Check for self-reference (recursive type) - needs pointer indirection
			// e.g., type UserGroup struct { Children UserGroup } is invalid, needs *UserGroup
			if isSelfReference(propSchema, typeName) &&
				!strings.HasPrefix(field.Type, "*") &&
				!strings.HasPrefix(field.Type, "[]") {
				field.Type = "*" + field.Type
			}

			// Handle duplicate field names (e.g., @id and id both become Id)
			baseName := field.Name
			if count, exists := usedFieldNames[baseName]; exists {
				// Append suffix to make unique
				field.Name = fmt.Sprintf("%s%d", baseName, count+1)
			}
			usedFieldNames[baseName]++

			structData.Fields = append(structData.Fields, field)
		}
	}

	// Handle additionalProperties
	if schema.AdditionalProperties != nil {
		var addPropsType string
		switch addProps := schema.AdditionalProperties.(type) {
		case *parser.Schema:
			addPropsType = cg.schemaToGoType(addProps, true)
		case map[string]interface{}:
			addPropsType = schemaTypeFromMap(addProps)
		case bool:
			if addProps {
				addPropsType = "any"
			}
			// If false, don't add AdditionalProperties field
		default:
			// Unknown type, default to any
			addPropsType = "any"
		}
		// Only set HasAdditionalProps if we have a valid type
		if addPropsType != "" {
			structData.HasAdditionalProps = true
			structData.AdditionalPropsType = addPropsType
		}
	}

	return TypeDefinition{
		Kind:   "struct",
		Struct: structData,
	}
}

// buildFieldData builds field data for a struct field.
func (cg *oas3CodeGenerator) buildFieldData(propName string, propSchema *parser.Schema, required bool) FieldData {
	goType := cg.schemaToGoType(propSchema, required)
	fieldName := toFieldName(propName)

	jsonTag := propName
	if !required {
		jsonTag += ",omitempty"
	}

	// Build struct tags
	tags := fmt.Sprintf("json:%q", jsonTag)
	if cg.g.IncludeValidation {
		if validateTag := cg.buildValidateTag(propSchema, required); validateTag != "" {
			tags += fmt.Sprintf(" validate:%q", validateTag)
		}
	}

	field := FieldData{
		Name: fieldName,
		Type: goType,
		Tags: tags,
	}

	if propSchema.Description != "" {
		field.Comment = cleanDescription(propSchema.Description)
	}

	return field
}

// buildEnumTypeDefinition builds an enum type definition from a string schema with enum values.
func (cg *oas3CodeGenerator) buildEnumTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	enumData := &EnumData{
		TypeName: typeName,
		BaseType: "string",
	}

	if schema.Description != "" {
		enumData.Comment = cleanDescription(schema.Description)
	}

	// Build enum values
	for _, e := range schema.Enum {
		enumVal := fmt.Sprintf("%v", e)
		enumName := typeName + toFieldName(enumVal)
		enumData.Values = append(enumData.Values, EnumValueData{
			ConstName: enumName,
			Type:      typeName,
			Value:     enumVal,
		})
	}

	return TypeDefinition{
		Kind: "enum",
		Enum: enumData,
	}
}

// buildAliasTypeDefinition builds a type alias from a $ref schema.
func (cg *oas3CodeGenerator) buildAliasTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	refType := cg.resolveRef(schema.Ref)

	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: refType,
		Comment:    fmt.Sprintf("is an alias for %s.", refType),
		IsDefined:  false,
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildArrayAliasTypeDefinition builds a defined type (not alias) for array types.
func (cg *oas3CodeGenerator) buildArrayAliasTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	itemType := cg.getArrayItemType(schema)

	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: "[]" + itemType,
		IsDefined:  true, // Arrays use defined types, not type aliases
	}

	if schema.Description != "" {
		aliasData.Comment = cleanDescription(schema.Description)
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildStringAliasTypeDefinition builds a type alias for string types.
func (cg *oas3CodeGenerator) buildStringAliasTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	goType := stringFormatToGoType(schema.Format)

	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: goType,
		IsDefined:  false,
	}

	if schema.Description != "" {
		aliasData.Comment = cleanDescription(schema.Description)
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildIntegerAliasTypeDefinition builds a type alias for integer types.
func (cg *oas3CodeGenerator) buildIntegerAliasTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	goType := integerFormatToGoType(schema.Format)

	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: goType,
		IsDefined:  false,
	}

	if schema.Description != "" {
		aliasData.Comment = cleanDescription(schema.Description)
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildNumberAliasTypeDefinition builds a type alias for number types.
func (cg *oas3CodeGenerator) buildNumberAliasTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	goType := numberFormatToGoType(schema.Format)

	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: goType,
		IsDefined:  false,
	}

	if schema.Description != "" {
		aliasData.Comment = cleanDescription(schema.Description)
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildBooleanAliasTypeDefinition builds a type alias for boolean types.
func (cg *oas3CodeGenerator) buildBooleanAliasTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: "bool",
		IsDefined:  false,
	}

	if schema.Description != "" {
		aliasData.Comment = cleanDescription(schema.Description)
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildAnyAliasTypeDefinition builds a type alias for any/unknown types.
func (cg *oas3CodeGenerator) buildAnyAliasTypeDefinition(typeName string) TypeDefinition {
	aliasData := &AliasData{
		TypeName:   typeName,
		TargetType: "any",
		IsDefined:  false,
	}

	return TypeDefinition{
		Kind:  "alias",
		Alias: aliasData,
	}
}

// buildAllOfTypeDefinition builds a struct type definition for allOf composition.
func (cg *oas3CodeGenerator) buildAllOfTypeDefinition(typeName string, schema *parser.Schema) TypeDefinition {
	allOfData := &AllOfData{
		TypeName: typeName,
		Comment:  "combines multiple schemas.",
	}

	for _, subSchema := range schema.AllOf {
		if subSchema.Ref != "" {
			// Embedded type
			refType := cg.resolveRef(subSchema.Ref)
			allOfData.EmbeddedTypes = append(allOfData.EmbeddedTypes, refType)
		} else if subSchema.Properties != nil {
			// Inline properties
			var propNames []string
			for propName := range subSchema.Properties {
				propNames = append(propNames, propName)
			}
			sort.Strings(propNames)

			for _, propName := range propNames {
				propSchema := subSchema.Properties[propName]
				if propSchema == nil {
					continue
				}
				field := cg.buildFieldData(propName, propSchema, isRequired(subSchema.Required, propName))
				allOfData.Fields = append(allOfData.Fields, field)
			}
		}
	}

	return TypeDefinition{
		Kind:  "allof",
		AllOf: allOfData,
	}
}

// buildOneOfTypeDefinition builds a struct type definition for oneOf/anyOf union types.
func (cg *oas3CodeGenerator) buildOneOfTypeDefinition(typeName, originalName string, schema *parser.Schema) TypeDefinition {
	oneOfData := &OneOfData{
		TypeName: typeName,
		Comment:  "represents a union type.",
	}

	schemas := schema.OneOf
	if len(schemas) == 0 {
		schemas = schema.AnyOf
	}

	// Handle discriminator
	if schema.Discriminator != nil && schema.Discriminator.PropertyName != "" {
		oneOfData.Discriminator = schema.Discriminator.PropertyName
		oneOfData.DiscriminatorField = toFieldName(schema.Discriminator.PropertyName)
		oneOfData.DiscriminatorJSONName = schema.Discriminator.PropertyName
		oneOfData.HasUnmarshal = true

		// Build unmarshal cases from discriminator mapping
		if schema.Discriminator.Mapping != nil {
			for value, ref := range schema.Discriminator.Mapping {
				typeName := cg.resolveRef(ref)
				oneOfData.UnmarshalCases = append(oneOfData.UnmarshalCases, UnmarshalCase{
					Value:    value,
					TypeName: typeName,
				})
			}
			// Sort for deterministic output
			sort.Slice(oneOfData.UnmarshalCases, func(i, j int) bool {
				return oneOfData.UnmarshalCases[i].Value < oneOfData.UnmarshalCases[j].Value
			})
		}
	}

	// Build variants
	for i, subSchema := range schemas {
		if subSchema.Ref != "" {
			refType := cg.resolveRef(subSchema.Ref)
			oneOfData.Variants = append(oneOfData.Variants, OneOfVariant{
				Name: refType,
				Type: "*" + refType,
			})
		} else {
			oneOfData.Variants = append(oneOfData.Variants, OneOfVariant{
				Name: fmt.Sprintf("Variant%d", i),
				Type: "any",
			})
		}
	}

	cg.addIssue("components.schemas."+originalName, "union types (oneOf/anyOf) are generated as structs with pointer fields", SeverityInfo)

	return TypeDefinition{
		Kind:  "oneof",
		OneOf: oneOfData,
	}
}
