// Package main generates decodeFromMap methods for parser package types.
//
// This generator uses go/types to introspect parser package struct types that
// have an Extra map[string]any field, and generates decodeFromMap methods that
// populate struct fields from a map[string]any. This avoids the expensive
// marshal/unmarshal roundtrip when resolving $ref references.
//
// Usage:
//
//	go run ./internal/codegen/decode
//	go run ./internal/codegen/decode -check  # verify freshness
//
// Or via go generate:
//
//	//go:generate go run ../internal/codegen/decode
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

// decodeTarget holds metadata for a single struct type that will get a
// generated decodeFromMap method.
type decodeTarget struct {
	Name   string
	Fields []fieldInfo
}

// fieldInfo describes one struct field and the decode strategy to use.
type fieldInfo struct {
	FieldName string // Go field name
	JSONKey   string // JSON/YAML key from struct tag
	Strategy  string // decode strategy key
	ElemType  string // element type name for slices/maps of OAS structs
}

// polymorphicFields maps "StructName.FieldName" to true for fields where
// `any` means "*Schema or bool" rather than plain any.
var polymorphicFields = map[string]bool{
	"Schema.Items":                 true,
	"Schema.AdditionalItems":       true,
	"Schema.AdditionalProperties":  true,
	"Schema.UnevaluatedItems":      true,
	"Schema.UnevaluatedProperties": true,
}

// oasStructTypes is populated during discovery with the names of all struct
// types in the parser package that have an Extra map[string]any field.
var oasStructTypes = map[string]bool{}

func main() {
	check := flag.Bool("check", false, "Compare generated output with existing file and exit non-zero if stale")
	flag.Parse()

	// Determine paths relative to working directory. The generator can be
	// invoked from the project root (go run ./internal/codegen/decode) or
	// from the parser directory (go generate).
	parserDir := "parser"
	outputPath := filepath.Join("parser", "zz_generated_decode.go")
	if _, err := os.Stat("parser"); os.IsNotExist(err) {
		// Likely running from the parser directory via go generate
		parserDir = "."
		outputPath = "zz_generated_decode.go"
	}

	// Resolve the absolute path for the overlay key. go/packages uses
	// absolute paths internally, so the overlay key must match.
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		fatal("failed to resolve absolute path for %s: %v", outputPath, err)
	}

	// Use an overlay to replace the generated file with a minimal stub.
	// This prevents stale method signatures from causing type errors
	// during package loading, while keeping decode_helpers.go compilable
	// by providing the three methods it references.
	stub := []byte(`package parser

func (x *OAS2Document) decodeFromMap(m map[string]any) {}
func (x *OAS3Document) decodeFromMap(m map[string]any) {}
func (x *Schema) decodeFromMap(m map[string]any)       {}
func (x *PathItem) decodeFromMap(m map[string]any)     {}
func (x *Response) decodeFromMap(m map[string]any)     {}
`)

	// Load the parser package using go/types
	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedName,
		Dir:     parserDir,
		Overlay: map[string][]byte{absOutput: stub},
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		fatal("failed to load parser package: %v", err)
	}
	if len(pkgs) == 0 {
		fatal("no packages found")
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		for _, e := range pkg.Errors {
			fmt.Fprintf(os.Stderr, "package error: %v\n", e)
		}
		fatal("package has errors")
	}

	scope := pkg.Types.Scope()

	// Phase 1: Discover all struct types with Extra map[string]any field
	var targetNames []string
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		st, ok := tn.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		if hasExtraField(st) {
			oasStructTypes[name] = true
			targetNames = append(targetNames, name)
		}
	}
	sort.Strings(targetNames)

	// Responses lacks Extra but has a hand-written decodeFromMap method,
	// so it must be registered as an OAS struct type for classifyField to
	// recognize *Responses fields (e.g., Operation.Responses) as oas_ptr.
	oasStructTypes["Responses"] = true

	// Phase 2: For each discovered type, introspect fields and classify
	var targets []decodeTarget
	for _, name := range targetNames {
		// Responses gets a hand-written method
		if name == "Responses" {
			continue
		}

		obj := scope.Lookup(name)
		st := obj.Type().Underlying().(*types.Struct)

		var fields []fieldInfo
		for i := range st.NumFields() {
			f := st.Field(i)
			tag := st.Tag(i)

			// Skip unexported fields
			if !f.Exported() {
				continue
			}

			fieldName := f.Name()

			// Parse the JSON key from the struct tag
			jsonKey := parseJSONKey(tag)
			if jsonKey == "" {
				// No json tag or json:"-" — skip
				continue
			}

			// Classify the field type into a decode strategy
			strategy, elemType := classifyField(name, fieldName, f.Type())
			if strategy == "" {
				// Warn about skipped fields so silent data loss is visible.
				// Fields intentionally skipped (Extra, OASVersion) are
				// handled before classifyField returns ("", "").
				if fieldName != "Extra" {
					fmt.Fprintf(os.Stderr, "warning: skipping %s.%s (type %s): no decode strategy\n",
						name, fieldName, types.TypeString(f.Type(), nil))
				}
				continue
			}

			fields = append(fields, fieldInfo{
				FieldName: fieldName,
				JSONKey:   jsonKey,
				Strategy:  strategy,
				ElemType:  elemType,
			})
		}

		targets = append(targets, decodeTarget{
			Name:   name,
			Fields: fields,
		})
	}

	// Phase 3: Generate code using template
	tmpl, err := template.New("decode").Parse(decodeTemplate)
	if err != nil {
		fatal("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, targets); err != nil {
		fatal("failed to execute template: %v", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fatal("failed to format generated code: %v\n\nGenerated code:\n%s", err, buf.String())
	}

	if *check {
		existing, err := os.ReadFile(outputPath)
		if err != nil {
			fatal("failed to read existing file %s: %v", outputPath, err)
		}
		if !bytes.Equal(existing, formatted) {
			fatal("%s is stale; run 'go generate ./parser/' to regenerate", outputPath)
		}
		fmt.Printf("%s is up to date\n", outputPath)
		return
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		fatal("failed to write %s: %v", outputPath, err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

// hasExtraField returns true if the struct has a field named "Extra" of type
// map[string]any.
func hasExtraField(st *types.Struct) bool {
	for i := range st.NumFields() {
		f := st.Field(i)
		if f.Name() != "Extra" {
			continue
		}
		mt, ok := f.Type().(*types.Map)
		if !ok {
			return false
		}
		keyBasic, ok := mt.Key().(*types.Basic)
		if !ok || keyBasic.Kind() != types.String {
			return false
		}
		// Use Unalias because `any` is a type alias for interface{} in Go 1.22+
		_, ok = types.Unalias(mt.Elem()).(*types.Interface)
		return ok
	}
	return false
}

// parseJSONKey extracts the JSON key from a struct tag string.
// Returns "" for fields tagged json:"-" or without a json tag.
func parseJSONKey(tag string) string {
	st := reflect.StructTag(tag)
	jsonTag := st.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return ""
	}
	key, _, _ := strings.Cut(jsonTag, ",")
	if key == "-" {
		return ""
	}
	return key
}

// classifyField determines the decode strategy for a field based on its Go type.
// Returns the strategy key and an optional element type name (for OAS struct
// slices/maps). Returns ("", "") for fields that should be skipped.
func classifyField(structName, fieldName string, t types.Type) (strategy, elemType string) {
	// 1. Check polymorphic override (Schema.Items, etc.)
	if polymorphicFields[structName+"."+fieldName] {
		return "polymorphic_schema", ""
	}

	// 2. Check for Extra field — handled separately
	if fieldName == "Extra" {
		return "", ""
	}

	// Unwrap type aliases (e.g., `any` is an alias for `interface{}` in Go 1.22+)
	t = types.Unalias(t)

	switch typ := t.(type) {
	case *types.Basic:
		// 3. Basic types: string, bool
		switch typ.Kind() {
		case types.String:
			return "string", ""
		case types.Bool:
			return "bool", ""
		default:
			return "", ""
		}

	case *types.Pointer:
		// 4. Pointer types: *bool, *int, *float64, *OASStruct
		elem := typ.Elem()
		if basic, ok := elem.(*types.Basic); ok {
			switch basic.Kind() {
			case types.Bool:
				return "ptr_bool", ""
			case types.Int:
				return "ptr_int", ""
			case types.Float64:
				return "ptr_float64", ""
			}
		}
		if named, ok := elem.(*types.Named); ok {
			name := named.Obj().Name()
			if oasStructTypes[name] {
				return "oas_ptr", name
			}
		}
		return "", ""

	case *types.Interface:
		// 5. Interface type (any) — if not polymorphic (checked above), plain assign
		return "any", ""

	case *types.Named:
		// 6. Named types
		name := typ.Obj().Name()
		switch name {
		case "Paths":
			return "paths", ""
		case "OASVersion":
			return "", "" // skip
		default:
			// Check if it's a slice of SecurityRequirement
			if name == "SecurityRequirement" {
				return "", "" // handled below as slice element
			}
			return "", ""
		}

	case *types.Slice:
		// 7. Slice types
		elem := types.Unalias(typ.Elem())

		// []string
		if basic, ok := elem.(*types.Basic); ok && basic.Kind() == types.String {
			return "string_slice", ""
		}

		// []any
		if _, ok := elem.(*types.Interface); ok {
			return "any_slice", ""
		}

		// []*T where T is an OAS struct
		if ptr, ok := elem.(*types.Pointer); ok {
			if named, ok := ptr.Elem().(*types.Named); ok {
				name := named.Obj().Name()
				if oasStructTypes[name] {
					return "oas_slice", name
				}
			}
		}

		// []SecurityRequirement
		if named, ok := elem.(*types.Named); ok {
			if named.Obj().Name() == "SecurityRequirement" {
				return "security_reqs", ""
			}
		}

		return "", ""

	case *types.Map:
		// 8. Map types
		keyBasic, ok := typ.Key().(*types.Basic)
		if !ok || keyBasic.Kind() != types.String {
			return "", ""
		}

		valType := types.Unalias(typ.Elem())

		// map[string]string
		if basic, ok := valType.(*types.Basic); ok && basic.Kind() == types.String {
			return "string_map", ""
		}

		// map[string]bool
		if basic, ok := valType.(*types.Basic); ok && basic.Kind() == types.Bool {
			return "bool_map", ""
		}

		// map[string]any
		if _, ok := valType.(*types.Interface); ok {
			return "any_map", ""
		}

		// map[string][]string (DependentRequired)
		if sliceType, ok := valType.(*types.Slice); ok {
			if basic, ok := sliceType.Elem().(*types.Basic); ok && basic.Kind() == types.String {
				return "dependent_required", ""
			}
		}

		// map[string]ServerVariable (value type, not pointer)
		if named, ok := valType.(*types.Named); ok {
			if named.Obj().Name() == "ServerVariable" {
				return "server_variable_map", ""
			}
		}

		// map[string]*T where T is OAS struct
		if ptr, ok := valType.(*types.Pointer); ok {
			if named, ok := ptr.Elem().(*types.Named); ok {
				name := named.Obj().Name()
				// map[string]*Callback is special (Callback is map type, not struct)
				if name == "Callback" {
					return "callbacks_map", ""
				}
				if oasStructTypes[name] {
					return "oas_map", name
				}
			}
		}

		return "", ""
	}

	return "", ""
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

const decodeTemplate = `// Code generated by internal/codegen/decode; DO NOT EDIT.
//
// This file contains decodeFromMap methods for parser package types.
// These methods populate struct fields directly from a map[string]any,
// avoiding the expensive marshal/unmarshal roundtrip used during $ref resolution.

package parser

import "github.com/erraggy/oastools/internal/httputil"
{{range .}}
func (x *{{.Name}}) decodeFromMap(m map[string]any) {
{{- range .Fields}}
{{- if eq .Strategy "string"}}
	x.{{.FieldName}}, _ = m["{{.JSONKey}}"].(string)
{{- else if eq .Strategy "bool"}}
	x.{{.FieldName}}, _ = m["{{.JSONKey}}"].(bool)
{{- else if eq .Strategy "ptr_bool"}}
	x.{{.FieldName}} = mapGetBoolPtr(m, "{{.JSONKey}}")
{{- else if eq .Strategy "ptr_int"}}
	x.{{.FieldName}} = mapGetIntPtr(m, "{{.JSONKey}}")
{{- else if eq .Strategy "ptr_float64"}}
	x.{{.FieldName}} = mapGetFloat64Ptr(m, "{{.JSONKey}}")
{{- else if eq .Strategy "any"}}
	x.{{.FieldName}} = m["{{.JSONKey}}"]
{{- else if eq .Strategy "polymorphic_schema"}}
	x.{{.FieldName}} = decodeSchemaOrBool(m["{{.JSONKey}}"])
{{- else if eq .Strategy "string_slice"}}
	x.{{.FieldName}} = mapGetStringSlice(m, "{{.JSONKey}}")
{{- else if eq .Strategy "any_slice"}}
	if arr, ok := m["{{.JSONKey}}"].([]any); ok {
		x.{{.FieldName}} = arr
	}
{{- else if eq .Strategy "oas_slice"}}
	if arr, ok := m["{{.JSONKey}}"].([]any); ok {
		x.{{.FieldName}} = make([]*{{.ElemType}}, 0, len(arr))
		for _, item := range arr {
			if sub, ok := item.(map[string]any); ok {
				elem := new({{.ElemType}})
				elem.decodeFromMap(sub)
				x.{{.FieldName}} = append(x.{{.FieldName}}, elem)
			}
		}
	}
{{- else if eq .Strategy "oas_ptr"}}
	if sub, ok := m["{{.JSONKey}}"].(map[string]any); ok {
		x.{{.FieldName}} = new({{.ElemType}})
		x.{{.FieldName}}.decodeFromMap(sub)
	}
{{- else if eq .Strategy "oas_map"}}
	if sub, ok := m["{{.JSONKey}}"].(map[string]any); ok {
		x.{{.FieldName}} = make(map[string]*{{.ElemType}}, len(sub))
		for k, v := range sub {
			if vm, ok := v.(map[string]any); ok {
				elem := new({{.ElemType}})
				elem.decodeFromMap(vm)
				x.{{.FieldName}}[k] = elem
			}
		}
	}
{{- else if eq .Strategy "string_map"}}
	x.{{.FieldName}} = mapGetStringMap(m, "{{.JSONKey}}")
{{- else if eq .Strategy "bool_map"}}
	x.{{.FieldName}} = mapGetBoolMap(m, "{{.JSONKey}}")
{{- else if eq .Strategy "dependent_required"}}
	x.{{.FieldName}} = mapGetDependentRequired(m, "{{.JSONKey}}")
{{- else if eq .Strategy "any_map"}}
	if sub, ok := m["{{.JSONKey}}"].(map[string]any); ok {
		x.{{.FieldName}} = sub
	}
{{- else if eq .Strategy "server_variable_map"}}
	if sub, ok := m["{{.JSONKey}}"].(map[string]any); ok {
		x.{{.FieldName}} = make(map[string]ServerVariable, len(sub))
		for k, v := range sub {
			if vm, ok := v.(map[string]any); ok {
				var sv ServerVariable
				sv.decodeFromMap(vm)
				x.{{.FieldName}}[k] = sv
			}
		}
	}
{{- else if eq .Strategy "paths"}}
	if sub, ok := m["{{.JSONKey}}"].(map[string]any); ok {
		x.{{.FieldName}} = decodePaths(sub)
	}
{{- else if eq .Strategy "callbacks_map"}}
	if sub, ok := m["{{.JSONKey}}"].(map[string]any); ok {
		x.{{.FieldName}} = make(map[string]*Callback, len(sub))
		for k, v := range sub {
			if vm, ok := v.(map[string]any); ok {
				x.{{.FieldName}}[k] = decodeCallback(vm)
			}
		}
	}
{{- else if eq .Strategy "security_reqs"}}
	if arr, ok := m["{{.JSONKey}}"].([]any); ok {
		x.{{.FieldName}} = decodeSecurityRequirements(arr)
	}
{{- end}}
{{- end}}
	x.Extra = extractExtensionsFromMap(m)
}
{{end}}
func (x *Responses) decodeFromMap(m map[string]any) {
	x.Codes = make(map[string]*Response)
	for key, value := range m {
		sub, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if key == "default" {
			x.Default = new(Response)
			x.Default.decodeFromMap(sub)
		} else if httputil.ValidateStatusCode(key) {
			resp := new(Response)
			resp.decodeFromMap(sub)
			x.Codes[key] = resp
		}
	}
}
`
