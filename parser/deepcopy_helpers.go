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

// deepCopyPaths deep copies a Paths map (map[string]*PathItem).
func deepCopyPaths(v Paths) Paths {
	if v == nil {
		return nil
	}
	cp := make(Paths, len(v))
	for k, item := range v {
		if item != nil {
			cp[k] = item.DeepCopy()
		}
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
				if scopes != nil {
					cpScopes := make([]string, len(scopes))
					copy(cpScopes, scopes)
					cp[i][k] = cpScopes
				}
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
		if val != nil {
			cpVal := make([]string, len(val))
			copy(cpVal, val)
			cp[k] = cpVal
		}
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
		if callback != nil {
			cpCallback := deepCopyCallback(*callback)
			cp[k] = &cpCallback
		}
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
		if item != nil {
			cp[k] = item.DeepCopy()
		}
	}
	return cp
}
