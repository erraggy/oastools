// Package main generates DeepCopy methods for parser package types.
//
// This generator creates type-aware deep copy methods that properly handle:
// - Pointer fields (deep copy the pointed value)
// - Slice fields (create new slice and copy elements)
// - Map fields (create new map and copy entries)
// - OAS-typed polymorphic fields (any/interface{} with known types)
//
// Usage:
//
//	go run ./internal/codegen/deepcopy
//
// Or via go generate:
//
//	//go:generate go run ../internal/codegen/deepcopy
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// FieldConfig defines how a single field should be copied
type FieldConfig struct {
	Name       string // Field name
	Type       string // Go type
	CopyMethod string // How to copy: "value", "pointer", "prim_pointer", "slice", "map", "helper"
	Helper     string // Helper function name (if CopyMethod == "helper")
	ElemType   string // Element type for slices/maps
	KeyType    string // Key type for maps
}

// TypeConfig defines how a struct type should have DeepCopy generated
type TypeConfig struct {
	Name   string        // Type name
	Fields []FieldConfig // Fields requiring special handling
}

// Configuration for all parser types
var typeConfigs = []TypeConfig{
	{
		Name: "OAS2Document",
		Fields: []FieldConfig{
			{Name: "Info", Type: "*Info", CopyMethod: "pointer"},
			{Name: "Schemes", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Consumes", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Produces", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Paths", Type: "Paths", CopyMethod: "helper", Helper: "deepCopyPaths"},
			{Name: "Definitions", Type: "map[string]*Schema", CopyMethod: "map", KeyType: "string", ElemType: "*Schema"},
			{Name: "Parameters", Type: "map[string]*Parameter", CopyMethod: "map", KeyType: "string", ElemType: "*Parameter"},
			{Name: "Responses", Type: "map[string]*Response", CopyMethod: "map", KeyType: "string", ElemType: "*Response"},
			{Name: "SecurityDefinitions", Type: "map[string]*SecurityScheme", CopyMethod: "map", KeyType: "string", ElemType: "*SecurityScheme"},
			{Name: "Security", Type: "[]SecurityRequirement", CopyMethod: "helper", Helper: "deepCopySecurityRequirements"},
			{Name: "Tags", Type: "[]*Tag", CopyMethod: "slice", ElemType: "*Tag"},
			{Name: "ExternalDocs", Type: "*ExternalDocs", CopyMethod: "pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "OAS3Document",
		Fields: []FieldConfig{
			{Name: "Info", Type: "*Info", CopyMethod: "pointer"},
			{Name: "Servers", Type: "[]*Server", CopyMethod: "slice", ElemType: "*Server"},
			{Name: "Paths", Type: "Paths", CopyMethod: "helper", Helper: "deepCopyPaths"},
			{Name: "Webhooks", Type: "map[string]*PathItem", CopyMethod: "map", KeyType: "string", ElemType: "*PathItem"},
			{Name: "Components", Type: "*Components", CopyMethod: "pointer"},
			{Name: "Security", Type: "[]SecurityRequirement", CopyMethod: "helper", Helper: "deepCopySecurityRequirements"},
			{Name: "Tags", Type: "[]*Tag", CopyMethod: "slice", ElemType: "*Tag"},
			{Name: "ExternalDocs", Type: "*ExternalDocs", CopyMethod: "pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Info",
		Fields: []FieldConfig{
			{Name: "Contact", Type: "*Contact", CopyMethod: "pointer"},
			{Name: "License", Type: "*License", CopyMethod: "pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Contact",
		Fields: []FieldConfig{
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "License",
		Fields: []FieldConfig{
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "ExternalDocs",
		Fields: []FieldConfig{
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Tag",
		Fields: []FieldConfig{
			{Name: "ExternalDocs", Type: "*ExternalDocs", CopyMethod: "pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Server",
		Fields: []FieldConfig{
			{Name: "Variables", Type: "map[string]ServerVariable", CopyMethod: "helper", Helper: "deepCopyServerVariables"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "ServerVariable",
		Fields: []FieldConfig{
			{Name: "Enum", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Reference",
		Fields: []FieldConfig{
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Components",
		Fields: []FieldConfig{
			{Name: "Schemas", Type: "map[string]*Schema", CopyMethod: "map", KeyType: "string", ElemType: "*Schema"},
			{Name: "Responses", Type: "map[string]*Response", CopyMethod: "map", KeyType: "string", ElemType: "*Response"},
			{Name: "Parameters", Type: "map[string]*Parameter", CopyMethod: "map", KeyType: "string", ElemType: "*Parameter"},
			{Name: "Examples", Type: "map[string]*Example", CopyMethod: "map", KeyType: "string", ElemType: "*Example"},
			{Name: "RequestBodies", Type: "map[string]*RequestBody", CopyMethod: "map", KeyType: "string", ElemType: "*RequestBody"},
			{Name: "Headers", Type: "map[string]*Header", CopyMethod: "map", KeyType: "string", ElemType: "*Header"},
			{Name: "SecuritySchemes", Type: "map[string]*SecurityScheme", CopyMethod: "map", KeyType: "string", ElemType: "*SecurityScheme"},
			{Name: "Links", Type: "map[string]*Link", CopyMethod: "map", KeyType: "string", ElemType: "*Link"},
			{Name: "Callbacks", Type: "map[string]*Callback", CopyMethod: "helper", Helper: "deepCopyCallbacks"},
			{Name: "PathItems", Type: "map[string]*PathItem", CopyMethod: "map", KeyType: "string", ElemType: "*PathItem"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Schema",
		Fields: []FieldConfig{
			// OAS-typed any fields
			{Name: "Type", Type: "any", CopyMethod: "helper", Helper: "deepCopySchemaType"},
			{Name: "Items", Type: "any", CopyMethod: "helper", Helper: "deepCopySchemaOrBool"},
			{Name: "AdditionalProperties", Type: "any", CopyMethod: "helper", Helper: "deepCopySchemaOrBool"},
			{Name: "AdditionalItems", Type: "any", CopyMethod: "helper", Helper: "deepCopySchemaOrBool"},
			{Name: "ExclusiveMinimum", Type: "any", CopyMethod: "helper", Helper: "deepCopyBoolOrNumber"},
			{Name: "ExclusiveMaximum", Type: "any", CopyMethod: "helper", Helper: "deepCopyBoolOrNumber"},
			{Name: "Default", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Example", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Const", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Enum", Type: "[]any", CopyMethod: "helper", Helper: "deepCopyEnumSlice"},
			{Name: "Examples", Type: "[]any", CopyMethod: "helper", Helper: "deepCopyEnumSlice"},
			// Primitive pointer fields
			{Name: "MultipleOf", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Maximum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Minimum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "MaxLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MaxItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MaxContains", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinContains", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MaxProperties", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinProperties", Type: "*int", CopyMethod: "prim_pointer"},
			// Struct pointer fields
			{Name: "Discriminator", Type: "*Discriminator", CopyMethod: "pointer"},
			{Name: "XML", Type: "*XML", CopyMethod: "pointer"},
			{Name: "ExternalDocs", Type: "*ExternalDocs", CopyMethod: "pointer"},
			{Name: "Contains", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "PropertyNames", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "If", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Then", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Else", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Not", Type: "*Schema", CopyMethod: "pointer"},
			// Slice fields
			{Name: "Required", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "PrefixItems", Type: "[]*Schema", CopyMethod: "slice", ElemType: "*Schema"},
			{Name: "AllOf", Type: "[]*Schema", CopyMethod: "slice", ElemType: "*Schema"},
			{Name: "AnyOf", Type: "[]*Schema", CopyMethod: "slice", ElemType: "*Schema"},
			{Name: "OneOf", Type: "[]*Schema", CopyMethod: "slice", ElemType: "*Schema"},
			// Map fields
			{Name: "Properties", Type: "map[string]*Schema", CopyMethod: "map", KeyType: "string", ElemType: "*Schema"},
			{Name: "PatternProperties", Type: "map[string]*Schema", CopyMethod: "map", KeyType: "string", ElemType: "*Schema"},
			{Name: "DependentRequired", Type: "map[string][]string", CopyMethod: "helper", Helper: "deepCopyDependentRequired"},
			{Name: "DependentSchemas", Type: "map[string]*Schema", CopyMethod: "map", KeyType: "string", ElemType: "*Schema"},
			{Name: "Vocabulary", Type: "map[string]bool", CopyMethod: "helper", Helper: "deepCopyVocabulary"},
			{Name: "Defs", Type: "map[string]*Schema", CopyMethod: "map", KeyType: "string", ElemType: "*Schema"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Discriminator",
		Fields: []FieldConfig{
			{Name: "Mapping", Type: "map[string]string", CopyMethod: "helper", Helper: "deepCopyStringMap"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "XML",
		Fields: []FieldConfig{
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "PathItem",
		Fields: []FieldConfig{
			{Name: "Get", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Put", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Post", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Delete", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Options", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Head", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Patch", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Trace", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Query", Type: "*Operation", CopyMethod: "pointer"},
			{Name: "Servers", Type: "[]*Server", CopyMethod: "slice", ElemType: "*Server"},
			{Name: "Parameters", Type: "[]*Parameter", CopyMethod: "slice", ElemType: "*Parameter"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Operation",
		Fields: []FieldConfig{
			{Name: "Tags", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "ExternalDocs", Type: "*ExternalDocs", CopyMethod: "pointer"},
			{Name: "Parameters", Type: "[]*Parameter", CopyMethod: "slice", ElemType: "*Parameter"},
			{Name: "RequestBody", Type: "*RequestBody", CopyMethod: "pointer"},
			{Name: "Responses", Type: "*Responses", CopyMethod: "pointer"},
			{Name: "Callbacks", Type: "map[string]*Callback", CopyMethod: "helper", Helper: "deepCopyCallbacks"},
			{Name: "Security", Type: "[]SecurityRequirement", CopyMethod: "helper", Helper: "deepCopySecurityRequirements"},
			{Name: "Servers", Type: "[]*Server", CopyMethod: "slice", ElemType: "*Server"},
			{Name: "Consumes", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Produces", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Schemes", Type: "[]string", CopyMethod: "slice", ElemType: "string"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Parameter",
		Fields: []FieldConfig{
			{Name: "Explode", Type: "*bool", CopyMethod: "prim_pointer"},
			{Name: "Schema", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Example", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Examples", Type: "map[string]*Example", CopyMethod: "map", KeyType: "string", ElemType: "*Example"},
			{Name: "Content", Type: "map[string]*MediaType", CopyMethod: "map", KeyType: "string", ElemType: "*MediaType"},
			{Name: "Items", Type: "*Items", CopyMethod: "pointer"},
			{Name: "Default", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Maximum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Minimum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "MaxLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MaxItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "Enum", Type: "[]any", CopyMethod: "helper", Helper: "deepCopyEnumSlice"},
			{Name: "MultipleOf", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Header",
		Fields: []FieldConfig{
			{Name: "Explode", Type: "*bool", CopyMethod: "prim_pointer"},
			{Name: "Schema", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Example", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Examples", Type: "map[string]*Example", CopyMethod: "map", KeyType: "string", ElemType: "*Example"},
			{Name: "Content", Type: "map[string]*MediaType", CopyMethod: "map", KeyType: "string", ElemType: "*MediaType"},
			{Name: "Items", Type: "*Items", CopyMethod: "pointer"},
			{Name: "Default", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Maximum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Minimum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "MaxLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MaxItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "Enum", Type: "[]any", CopyMethod: "helper", Helper: "deepCopyEnumSlice"},
			{Name: "MultipleOf", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "RequestBody",
		Fields: []FieldConfig{
			{Name: "Content", Type: "map[string]*MediaType", CopyMethod: "map", KeyType: "string", ElemType: "*MediaType"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "MediaType",
		Fields: []FieldConfig{
			{Name: "Schema", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Example", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Examples", Type: "map[string]*Example", CopyMethod: "map", KeyType: "string", ElemType: "*Example"},
			{Name: "Encoding", Type: "map[string]*Encoding", CopyMethod: "map", KeyType: "string", ElemType: "*Encoding"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Encoding",
		Fields: []FieldConfig{
			{Name: "Headers", Type: "map[string]*Header", CopyMethod: "map", KeyType: "string", ElemType: "*Header"},
			{Name: "Explode", Type: "*bool", CopyMethod: "prim_pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Example",
		Fields: []FieldConfig{
			{Name: "Value", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Link",
		Fields: []FieldConfig{
			{Name: "Parameters", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
			{Name: "RequestBody", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Server", Type: "*Server", CopyMethod: "pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Response",
		Fields: []FieldConfig{
			{Name: "Headers", Type: "map[string]*Header", CopyMethod: "map", KeyType: "string", ElemType: "*Header"},
			{Name: "Content", Type: "map[string]*MediaType", CopyMethod: "map", KeyType: "string", ElemType: "*MediaType"},
			{Name: "Links", Type: "map[string]*Link", CopyMethod: "map", KeyType: "string", ElemType: "*Link"},
			{Name: "Schema", Type: "*Schema", CopyMethod: "pointer"},
			{Name: "Examples", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Responses",
		Fields: []FieldConfig{
			{Name: "Default", Type: "*Response", CopyMethod: "pointer"},
			{Name: "Codes", Type: "map[string]*Response", CopyMethod: "map", KeyType: "string", ElemType: "*Response"},
		},
	},
	{
		Name: "SecurityScheme",
		Fields: []FieldConfig{
			{Name: "Flows", Type: "*OAuthFlows", CopyMethod: "pointer"},
			{Name: "Scopes", Type: "map[string]string", CopyMethod: "helper", Helper: "deepCopyStringMap"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "OAuthFlows",
		Fields: []FieldConfig{
			{Name: "Implicit", Type: "*OAuthFlow", CopyMethod: "pointer"},
			{Name: "Password", Type: "*OAuthFlow", CopyMethod: "pointer"},
			{Name: "ClientCredentials", Type: "*OAuthFlow", CopyMethod: "pointer"},
			{Name: "AuthorizationCode", Type: "*OAuthFlow", CopyMethod: "pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "OAuthFlow",
		Fields: []FieldConfig{
			{Name: "Scopes", Type: "map[string]string", CopyMethod: "helper", Helper: "deepCopyStringMap"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
	{
		Name: "Items",
		Fields: []FieldConfig{
			{Name: "Items", Type: "*Items", CopyMethod: "pointer"},
			{Name: "Default", Type: "any", CopyMethod: "helper", Helper: "deepCopyJSONValue"},
			{Name: "Maximum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Minimum", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "MaxLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinLength", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MaxItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "MinItems", Type: "*int", CopyMethod: "prim_pointer"},
			{Name: "Enum", Type: "[]any", CopyMethod: "helper", Helper: "deepCopyEnumSlice"},
			{Name: "MultipleOf", Type: "*float64", CopyMethod: "prim_pointer"},
			{Name: "Extra", Type: "map[string]any", CopyMethod: "helper", Helper: "deepCopyExtensions"},
		},
	},
}

// Template for generating DeepCopy methods
const deepCopyTemplate = `// Code generated by internal/codegen/deepcopy; DO NOT EDIT.
//
// This file contains DeepCopy methods for parser package types.
// These methods provide type-aware deep copying that properly handles:
// - Pointer fields (deep copy the pointed value)
// - Slice fields (create new slice and copy elements)
// - Map fields (create new map and copy entries)
// - OAS-typed polymorphic fields (any/interface{} with known types)

package parser

{{range .Types}}
// DeepCopy creates a deep copy of {{.Name}}.
func (in *{{.Name}}) DeepCopy() *{{.Name}} {
	if in == nil {
		return nil
	}
	out := new({{.Name}})
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies {{.Name}} into out.
func (in *{{.Name}}) DeepCopyInto(out *{{.Name}}) {
	*out = *in
{{range .Fields}}{{if eq .CopyMethod "pointer"}}
	if in.{{.Name}} != nil {
		out.{{.Name}} = in.{{.Name}}.DeepCopy()
	}
{{else if eq .CopyMethod "prim_pointer"}}
	if in.{{.Name}} != nil {
		out.{{.Name}} = new({{stripPointer .Type}})
		*out.{{.Name}} = *in.{{.Name}}
	}
{{else if eq .CopyMethod "slice"}}{{if hasPointerElem .ElemType}}
	if in.{{.Name}} != nil {
		out.{{.Name}} = make({{.Type}}, len(in.{{.Name}}))
		for i, v := range in.{{.Name}} {
			if v != nil {
				out.{{.Name}}[i] = v.DeepCopy()
			}
		}
	}
{{else}}
	if in.{{.Name}} != nil {
		out.{{.Name}} = make({{.Type}}, len(in.{{.Name}}))
		copy(out.{{.Name}}, in.{{.Name}})
	}
{{end}}{{else if eq .CopyMethod "map"}}{{if hasPointerElem .ElemType}}
	if in.{{.Name}} != nil {
		out.{{.Name}} = make({{.Type}}, len(in.{{.Name}}))
		for k, v := range in.{{.Name}} {
			if v != nil {
				out.{{.Name}}[k] = v.DeepCopy()
			}
		}
	}
{{else}}
	if in.{{.Name}} != nil {
		out.{{.Name}} = make({{.Type}}, len(in.{{.Name}}))
		for k, v := range in.{{.Name}} {
			out.{{.Name}}[k] = v
		}
	}
{{end}}{{else if eq .CopyMethod "helper"}}
	out.{{.Name}} = {{.Helper}}(in.{{.Name}})
{{end}}{{end}}}

{{end}}`

// TemplateData holds the data passed to the template for code generation.
type TemplateData struct {
	Types []TypeConfig
}

func main() {
	// Parse template with helper functions
	funcMap := template.FuncMap{
		"hasPointerElem": func(elemType string) bool {
			return strings.HasPrefix(elemType, "*")
		},
		"stripPointer": func(ptrType string) string {
			return strings.TrimPrefix(ptrType, "*")
		},
	}

	tmpl, err := template.New("deepcopy").Funcs(funcMap).Parse(deepCopyTemplate)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Sort types alphabetically for consistent output
	sortedConfigs := make([]TypeConfig, len(typeConfigs))
	copy(sortedConfigs, typeConfigs)
	sort.Slice(sortedConfigs, func(i, j int) bool {
		return sortedConfigs[i].Name < sortedConfigs[j].Name
	})

	data := TemplateData{
		Types: sortedConfigs,
	}

	// Generate code
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error executing template: %v\n", err)
		os.Exit(1)
	}

	// Format generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error formatting code: %v\n", err)
		_, _ = fmt.Fprintf(os.Stderr, "Generated code:\n%s\n", buf.String())
		os.Exit(1)
	}

	// Write to file
	outputPath := filepath.Join("parser", "zz_generated_deepcopy.go")
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outputPath)
}
