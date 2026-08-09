package parser

// This file contains helper functions for deep copying OAS-typed polymorphic fields.
// These helpers understand the OAS specification semantics for fields that use
// any types but have well-defined possible types per the spec.

// deepCopySchemaType handles Schema.Type which can be:
// - string (OAS 2.0, 3.0, 3.1)
// - []string (OAS 3.1+ for type arrays like ["string", "null"])
func deepCopySchemaType(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		return t // strings are immutable
	case []string:
		cp := make([]string, len(t))
		copy(cp, t)
		return cp
	case []any:
		// YAML may unmarshal as []any instead of []string
		cp := make([]any, len(t))
		copy(cp, t)
		return cp
	default:
		return v // Unknown type, return as-is
	}
}

// deepCopySchemaOrBool handles fields that can be *Schema or bool:
// - Schema.Items (OAS 3.1+: bool for additionalItems semantics)
// - Schema.AdditionalProperties
// - Schema.AdditionalItems
func deepCopySchemaOrBool(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case bool:
		return t
	case *Schema:
		if t == nil {
			return nil
		}
		return t.DeepCopy()
	default:
		return v // Unknown type, return as-is
	}
}

// deepCopyBoolOrNumber handles ExclusiveMinimum/ExclusiveMaximum:
// - bool (OAS 2.0, 3.0)
// - float64/number (OAS 3.1+ JSON Schema Draft 2020-12)
func deepCopyBoolOrNumber(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t
	case int:
		return t
	case int64:
		return t
	default:
		return v
	}
}

// deepCopyJSONValue recursively deep copies any JSON-compatible value.
// This handles Default, Example, Const, and other fields that can hold
// arbitrary JSON values.
func deepCopyJSONValue(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string, bool, float64, int, int64, float32, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return t // Primitives copy by value
	case []any:
		cp := make([]any, len(t))
		for i, item := range t {
			cp[i] = deepCopyJSONValue(item)
		}
		return cp
	case map[string]any:
		cp := make(map[string]any, len(t))
		for k, item := range t {
			cp[k] = deepCopyJSONValue(item)
		}
		return cp
	default:
		// Unknown type - could be custom types in extensions
		// Return as-is (shallow copy)
		return v
	}
}

// deepCopyEnumSlice deep copies a []any slice containing enum values.
// Enum values are typically JSON primitives but may contain nested structures.
func deepCopyEnumSlice(v []any) []any {
	if v == nil {
		return nil
	}
	cp := make([]any, len(v))
	for i, item := range v {
		cp[i] = deepCopyJSONValue(item)
	}
	return cp
}

// deepCopyExtensions deep copies a map[string]any containing x-* extensions.
// Extension values can be any JSON-compatible value.
func deepCopyExtensions(v map[string]any) map[string]any {
	if v == nil {
		return nil
	}
	cp := make(map[string]any, len(v))
	for k, item := range v {
		cp[k] = deepCopyJSONValue(item)
	}
	return cp
}

// DeepCopyExtensions deep copies an Extra map, the field every OAS object uses
// to carry specification extensions and any other field the struct does not
// name. A nil map copies to nil, so an object with no extensions does not gain
// an empty one.
//
// Use it when building an object field by field from an existing one: assigning
// the map directly leaves both objects sharing it.
func DeepCopyExtensions(v map[string]any) map[string]any {
	return deepCopyExtensions(v)
}

// DeepCopySecurityRequirements deep copies a slice of SecurityRequirement, both
// the requirement maps and the scope slices inside them. A nil slice copies to
// nil.
func DeepCopySecurityRequirements(v []SecurityRequirement) []SecurityRequirement {
	return deepCopySecurityRequirements(v)
}

// deepCopyPaths deep copies a Paths map (map[string]*PathItem).
func deepCopyPaths(v Paths) Paths {
	if v == nil {
		return nil
	}
	cp := make(Paths, len(v))
	for k, item := range v {
		// Every key is assigned, including one whose value is nil. An empty
		// path item is a present path, so skipping it would drop the path from
		// the document rather than copy it.
		cp[k] = item.DeepCopy()
	}
	return cp
}

// deepCopySecurityRequirements deep copies a slice of SecurityRequirement.
func deepCopySecurityRequirements(v []SecurityRequirement) []SecurityRequirement {
	if v == nil {
		return nil
	}
	cp := make([]SecurityRequirement, len(v))
	for i, req := range v {
		if req != nil {
			cp[i] = make(SecurityRequirement, len(req))
			for k, scopes := range req {
				// Every key is assigned, including one whose scopes are nil.
				// Skipping it would drop the scheme from the requirement, which
				// is a different document, not a smaller copy of the same one.
				if scopes == nil {
					cp[i][k] = nil
					continue
				}
				cpScopes := make([]string, len(scopes))
				copy(cpScopes, scopes)
				cp[i][k] = cpScopes
			}
		}
	}
	return cp
}

// deepCopyServerVariables deep copies a map of ServerVariable (value type, not pointer).
func deepCopyServerVariables(v map[string]ServerVariable) map[string]ServerVariable {
	if v == nil {
		return nil
	}
	cp := make(map[string]ServerVariable, len(v))
	for k, sv := range v {
		cpSV := ServerVariable{
			Default:     sv.Default,
			Description: sv.Description,
		}
		if sv.Enum != nil {
			cpSV.Enum = make([]string, len(sv.Enum))
			copy(cpSV.Enum, sv.Enum)
		}
		if sv.Extra != nil {
			cpSV.Extra = deepCopyExtensions(sv.Extra)
		}
		cp[k] = cpSV
	}
	return cp
}

// deepCopyStringMap deep copies a map[string]string.
func deepCopyStringMap(v map[string]string) map[string]string {
	if v == nil {
		return nil
	}
	cp := make(map[string]string, len(v))
	for k, val := range v {
		cp[k] = val
	}
	return cp
}

// deepCopyDependentRequired deep copies a map[string][]string.
func deepCopyDependentRequired(v map[string][]string) map[string][]string {
	if v == nil {
		return nil
	}
	cp := make(map[string][]string, len(v))
	for k, val := range v {
		// Every key is assigned, so a nil value stays a present key.
		if val == nil {
			cp[k] = nil
			continue
		}
		cpVal := make([]string, len(val))
		copy(cpVal, val)
		cp[k] = cpVal
	}
	return cp
}

// deepCopyVocabulary deep copies a map[string]bool.
func deepCopyVocabulary(v map[string]bool) map[string]bool {
	if v == nil {
		return nil
	}
	cp := make(map[string]bool, len(v))
	for k, val := range v {
		cp[k] = val
	}
	return cp
}

// deepCopyCallbacks deep copies a map[string]*Callback.
// Callback is a type alias for map[string]*PathItem.
func deepCopyCallbacks(v map[string]*Callback) map[string]*Callback {
	if v == nil {
		return nil
	}
	cp := make(map[string]*Callback, len(v))
	for k, callback := range v {
		// Every key is assigned, so a nil value stays a present key.
		if callback == nil {
			cp[k] = nil
			continue
		}
		cpCallback := deepCopyCallback(*callback)
		cp[k] = &cpCallback
	}
	return cp
}

// deepCopyCallback deep copies a Callback (map[string]*PathItem).
func deepCopyCallback(v Callback) Callback {
	if v == nil {
		return nil
	}
	cp := make(Callback, len(v))
	for k, item := range v {
		// Every key is assigned, so a nil value stays a present key.
		cp[k] = item.DeepCopy()
	}
	return cp
}
