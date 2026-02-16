package parser

import (
	"math"
	"strings"
)

// extractExtensionsFromMap collects x-* keys from a map into an extension map.
// Returns nil if no extensions found (not an empty map).
func extractExtensionsFromMap(m map[string]any) map[string]any {
	var extra map[string]any
	for k, v := range m {
		if isExtensionKey(k) {
			if extra == nil {
				extra = make(map[string]any)
			}
			extra[k] = v
		}
	}
	return extra
}

// mapGetStringSlice extracts a []string from m[key], handling the []any that
// yaml.Unmarshal / json.Unmarshal produce.
func mapGetStringSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// mapGetFloat64Ptr extracts a *float64 from m[key].
// Handles both float64 (from JSON) and int (from YAML) numeric values.
func mapGetFloat64Ptr(m map[string]any, key string) *float64 {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case float64:
		return &n
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	case uint64:
		f := float64(n)
		return &f
	case uint:
		f := float64(n)
		return &f
	case uint32:
		f := float64(n)
		return &f
	case uint16:
		f := float64(n)
		return &f
	case uint8:
		f := float64(n)
		return &f
	default:
		return nil
	}
}

// mapGetIntPtr extracts a *int from m[key].
// Handles both float64 (from JSON) and int (from YAML) numeric values.
func mapGetIntPtr(m map[string]any, key string) *int {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case int:
		return &n
	case int64:
		i := int(n)
		return &i
	case uint64:
		if n > math.MaxInt {
			return nil
		}
		i := int(n)
		return &i
	case uint:
		if n > math.MaxInt {
			return nil
		}
		i := int(n)
		return &i
	case uint32:
		i := int(n)
		return &i
	case uint16:
		i := int(n)
		return &i
	case uint8:
		i := int(n)
		return &i
	default:
		return nil
	}
}

// mapGetBoolPtr extracts a *bool from m[key].
func mapGetBoolPtr(m map[string]any, key string) *bool {
	v, ok := m[key]
	if !ok {
		return nil
	}
	if b, ok := v.(bool); ok {
		return &b
	}
	return nil
}

// mapGetStringMap extracts a map[string]string from m[key].
func mapGetStringMap(m map[string]any, key string) map[string]string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	sub, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(sub))
	for k, val := range sub {
		if s, ok := val.(string); ok {
			result[k] = s
		}
	}
	return result
}

// mapGetBoolMap extracts a map[string]bool from m[key].
func mapGetBoolMap(m map[string]any, key string) map[string]bool {
	v, ok := m[key]
	if !ok {
		return nil
	}
	sub, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]bool, len(sub))
	for k, val := range sub {
		if b, ok := val.(bool); ok {
			result[k] = b
		}
	}
	return result
}

// mapGetDependentRequired extracts a map[string][]string from m[key].
func mapGetDependentRequired(m map[string]any, key string) map[string][]string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	sub, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string][]string, len(sub))
	for k, val := range sub {
		if arr, ok := val.([]any); ok {
			strs := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					strs = append(strs, s)
				}
			}
			result[k] = strs
		}
	}
	return result
}

// decodeSchemaOrBool decodes a value that can be either a *Schema (map) or a
// bool, as used by Schema.Items, Schema.AdditionalProperties, etc.
func decodeSchemaOrBool(v any) any {
	switch val := v.(type) {
	case map[string]any:
		s := new(Schema)
		s.decodeFromMap(val)
		return s
	case []any:
		// OAS 2.0 tuple validation: items can be an array of schemas
		schemas := make([]*Schema, 0, len(val))
		for _, elem := range val {
			if m, ok := elem.(map[string]any); ok {
				s := new(Schema)
				s.decodeFromMap(m)
				schemas = append(schemas, s)
			}
		}
		return schemas
	default:
		return v // bool, nil, etc.
	}
}

// decodePaths decodes a map[string]any into a Paths value.
func decodePaths(m map[string]any) Paths {
	if m == nil {
		return nil
	}
	result := make(Paths, len(m))
	for k, v := range m {
		if sub, ok := v.(map[string]any); ok {
			pi := new(PathItem)
			pi.decodeFromMap(sub)
			result[k] = pi
		}
	}
	return result
}

// decodeCallback decodes a map[string]any into a Callback value.
func decodeCallback(m map[string]any) *Callback {
	if m == nil {
		return nil
	}
	cb := make(Callback, len(m))
	for k, v := range m {
		if sub, ok := v.(map[string]any); ok {
			pi := new(PathItem)
			pi.decodeFromMap(sub)
			cb[k] = pi
		}
	}
	return &cb
}

// decodeSecurityRequirements decodes a []any into []SecurityRequirement.
func decodeSecurityRequirements(arr []any) []SecurityRequirement {
	if arr == nil {
		return nil
	}
	result := make([]SecurityRequirement, 0, len(arr))
	for _, item := range arr {
		if im, ok := item.(map[string]any); ok {
			sr := make(SecurityRequirement, len(im))
			for sk, sv := range im {
				if sarr, ok := sv.([]any); ok {
					strs := make([]string, 0, len(sarr))
					for _, s := range sarr {
						if str, ok := s.(string); ok {
							strs = append(strs, str)
						}
					}
					sr[sk] = strs
				}
			}
			result = append(result, sr)
		}
	}
	return result
}

// isExtensionKey returns true if the key starts with "x-".
func isExtensionKey(key string) bool {
	return strings.HasPrefix(key, "x-")
}
