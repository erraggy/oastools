package joiner

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// EquivalenceMode defines how deeply to compare schemas
type EquivalenceMode string

const (
	// EquivalenceModeNone disables equivalence detection
	EquivalenceModeNone EquivalenceMode = "none"
	// EquivalenceModeShallow compares only top-level schema properties
	EquivalenceModeShallow EquivalenceMode = "shallow"
	// EquivalenceModeDeep recursively compares all nested schemas
	EquivalenceModeDeep EquivalenceMode = "deep"
)

// ValidEquivalenceModes returns all valid equivalence mode strings
func ValidEquivalenceModes() []string {
	return []string{
		string(EquivalenceModeNone),
		string(EquivalenceModeShallow),
		string(EquivalenceModeDeep),
	}
}

// IsValidEquivalenceMode checks if an equivalence mode string is valid
func IsValidEquivalenceMode(mode string) bool {
	switch EquivalenceMode(mode) {
	case EquivalenceModeNone, EquivalenceModeShallow, EquivalenceModeDeep:
		return true
	default:
		return false
	}
}

// EquivalenceResult contains the outcome of schema comparison
type EquivalenceResult struct {
	Equivalent  bool
	Differences []SchemaDifference
}

// CompareSchemas compares two schemas for structural equivalence
// Ignores: description, title, example, deprecated, and extension fields (x-*)
func CompareSchemas(left, right *parser.Schema, mode EquivalenceMode) EquivalenceResult {
	if mode == EquivalenceModeNone {
		return EquivalenceResult{Equivalent: false}
	}

	result := EquivalenceResult{
		Differences: make([]SchemaDifference, 0),
	}

	// Handle nil schemas
	if left == nil && right == nil {
		result.Equivalent = true
		return result
	}
	if left == nil || right == nil {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        "",
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "schema presence mismatch (one is nil)",
		})
		result.Equivalent = false
		return result
	}

	// Track visited pointers to handle circular references
	visited := make(map[pointerPair]bool)

	if mode == EquivalenceModeShallow {
		compareShallow(left, right, "", &result)
	} else {
		compareDeep(left, right, "", &result, visited)
	}

	result.Equivalent = len(result.Differences) == 0
	return result
}

// pointerPair tracks schema pointer pairs to detect cycles
type pointerPair struct {
	left  uintptr
	right uintptr
}

// compareCommonFields compares schema fields common to both shallow and deep comparison.
// This helper eliminates duplication between compareShallow and compareDeep.
func compareCommonFields(left, right *parser.Schema, path string, result *EquivalenceResult) {
	// Compare type
	if !equalTypes(left.Type, right.Type) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "type"),
			LeftValue:   left.Type,
			RightValue:  right.Type,
			Description: "type mismatch",
		})
	}

	// Compare format
	if left.Format != right.Format {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "format"),
			LeftValue:   left.Format,
			RightValue:  right.Format,
			Description: "format mismatch",
		})
	}

	// Compare required arrays (order-independent)
	if !equalStringSlices(left.Required, right.Required) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "required"),
			LeftValue:   left.Required,
			RightValue:  right.Required,
			Description: "required fields mismatch",
		})
	}

	// Compare enum (order matters for enum)
	if !reflect.DeepEqual(left.Enum, right.Enum) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "enum"),
			LeftValue:   left.Enum,
			RightValue:  right.Enum,
			Description: "enum values mismatch",
		})
	}

	// Compare property names (shallow - don't compare nested schemas)
	if !equalPropertyNames(left.Properties, right.Properties) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "properties"),
			LeftValue:   getPropertyNames(left.Properties),
			RightValue:  getPropertyNames(right.Properties),
			Description: "property names mismatch",
		})
	}
}

// compareShallow compares only the top-level properties of schemas
func compareShallow(left, right *parser.Schema, path string, result *EquivalenceResult) {
	compareCommonFields(left, right, path, result)
}

// compareDeep recursively compares all schema properties
func compareDeep(left, right *parser.Schema, path string, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Check for circular references
	pair := pointerPair{
		left:  reflect.ValueOf(left).Pointer(),
		right: reflect.ValueOf(right).Pointer(),
	}
	if visited[pair] {
		return // Already compared this pair
	}
	visited[pair] = true

	// Compare common fields (type, format, required, enum, propertyNames)
	compareCommonFields(left, right, path, result)

	// Compare pattern (deep only)
	if left.Pattern != right.Pattern {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "pattern"),
			LeftValue:   left.Pattern,
			RightValue:  right.Pattern,
			Description: "pattern mismatch",
		})
	}

	// Compare const
	if !reflect.DeepEqual(left.Const, right.Const) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "const"),
			LeftValue:   left.Const,
			RightValue:  right.Const,
			Description: "const value mismatch",
		})
	}

	// Compare numeric constraints
	if !equalPointerFloat(left.Minimum, right.Minimum) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "minimum"),
			LeftValue:   left.Minimum,
			RightValue:  right.Minimum,
			Description: "minimum constraint mismatch",
		})
	}
	if !equalPointerFloat(left.Maximum, right.Maximum) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "maximum"),
			LeftValue:   left.Maximum,
			RightValue:  right.Maximum,
			Description: "maximum constraint mismatch",
		})
	}

	// Compare string constraints
	if !equalPointerInt(left.MinLength, right.MinLength) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "minLength"),
			LeftValue:   left.MinLength,
			RightValue:  right.MinLength,
			Description: "minLength constraint mismatch",
		})
	}
	if !equalPointerInt(left.MaxLength, right.MaxLength) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "maxLength"),
			LeftValue:   left.MaxLength,
			RightValue:  right.MaxLength,
			Description: "maxLength constraint mismatch",
		})
	}

	// Compare array constraints
	if !equalPointerInt(left.MinItems, right.MinItems) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "minItems"),
			LeftValue:   left.MinItems,
			RightValue:  right.MinItems,
			Description: "minItems constraint mismatch",
		})
	}
	if !equalPointerInt(left.MaxItems, right.MaxItems) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "maxItems"),
			LeftValue:   left.MaxItems,
			RightValue:  right.MaxItems,
			Description: "maxItems constraint mismatch",
		})
	}
	if left.UniqueItems != right.UniqueItems {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "uniqueItems"),
			LeftValue:   left.UniqueItems,
			RightValue:  right.UniqueItems,
			Description: "uniqueItems constraint mismatch",
		})
	}

	// Compare object constraints
	if !equalPointerInt(left.MinProperties, right.MinProperties) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "minProperties"),
			LeftValue:   left.MinProperties,
			RightValue:  right.MinProperties,
			Description: "minProperties constraint mismatch",
		})
	}
	if !equalPointerInt(left.MaxProperties, right.MaxProperties) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "maxProperties"),
			LeftValue:   left.MaxProperties,
			RightValue:  right.MaxProperties,
			Description: "maxProperties constraint mismatch",
		})
	}

	// Compare properties recursively (property names already checked by compareCommonFields)
	if equalPropertyNames(left.Properties, right.Properties) && left.Properties != nil {
		for name, leftProp := range left.Properties {
			rightProp := right.Properties[name]
			compareDeep(leftProp, rightProp, pathJoin(path, fmt.Sprintf("properties.%s", name)), result, visited)
		}
	}

	// Compare items (array item schema)
	compareItemsSchemas(left.Items, right.Items, path, result, visited)

	// Compare additionalProperties
	compareAdditionalPropertiesSchemas(left.AdditionalProperties, right.AdditionalProperties, path, result, visited)

	// Compare composition (allOf, anyOf, oneOf)
	compareSchemaArrays(left.AllOf, right.AllOf, pathJoin(path, "allOf"), result, visited)
	compareSchemaArrays(left.AnyOf, right.AnyOf, pathJoin(path, "anyOf"), result, visited)
	compareSchemaArrays(left.OneOf, right.OneOf, pathJoin(path, "oneOf"), result, visited)

	// Compare not
	if (left.Not == nil) != (right.Not == nil) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "not"),
			LeftValue:   left.Not != nil,
			RightValue:  right.Not != nil,
			Description: "not schema presence mismatch",
		})
	} else if left.Not != nil && right.Not != nil {
		compareDeep(left.Not, right.Not, pathJoin(path, "not"), result, visited)
	}

	// JSON Schema 2020-12 fields

	// Compare unevaluatedProperties (can be bool or *Schema)
	comparePolymorphicSchemas(left.UnevaluatedProperties, right.UnevaluatedProperties, pathJoin(path, "unevaluatedProperties"), result, visited)

	// Compare unevaluatedItems (can be bool or *Schema)
	comparePolymorphicSchemas(left.UnevaluatedItems, right.UnevaluatedItems, pathJoin(path, "unevaluatedItems"), result, visited)

	// Compare contentEncoding
	if left.ContentEncoding != right.ContentEncoding {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "contentEncoding"),
			LeftValue:   left.ContentEncoding,
			RightValue:  right.ContentEncoding,
			Description: "contentEncoding mismatch",
		})
	}

	// Compare contentMediaType
	if left.ContentMediaType != right.ContentMediaType {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "contentMediaType"),
			LeftValue:   left.ContentMediaType,
			RightValue:  right.ContentMediaType,
			Description: "contentMediaType mismatch",
		})
	}

	// Compare contentSchema
	if (left.ContentSchema == nil) != (right.ContentSchema == nil) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "contentSchema"),
			LeftValue:   left.ContentSchema != nil,
			RightValue:  right.ContentSchema != nil,
			Description: "contentSchema presence mismatch",
		})
	} else if left.ContentSchema != nil && right.ContentSchema != nil {
		compareDeep(left.ContentSchema, right.ContentSchema, pathJoin(path, "contentSchema"), result, visited)
	}

	// Compare prefixItems
	compareSchemaArrays(left.PrefixItems, right.PrefixItems, pathJoin(path, "prefixItems"), result, visited)

	// Compare contains
	if (left.Contains == nil) != (right.Contains == nil) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "contains"),
			LeftValue:   left.Contains != nil,
			RightValue:  right.Contains != nil,
			Description: "contains schema presence mismatch",
		})
	} else if left.Contains != nil && right.Contains != nil {
		compareDeep(left.Contains, right.Contains, pathJoin(path, "contains"), result, visited)
	}

	// Compare propertyNames
	if (left.PropertyNames == nil) != (right.PropertyNames == nil) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "propertyNames"),
			LeftValue:   left.PropertyNames != nil,
			RightValue:  right.PropertyNames != nil,
			Description: "propertyNames schema presence mismatch",
		})
	} else if left.PropertyNames != nil && right.PropertyNames != nil {
		compareDeep(left.PropertyNames, right.PropertyNames, pathJoin(path, "propertyNames"), result, visited)
	}

	// Compare dependentSchemas
	if !equalPropertyNames(left.DependentSchemas, right.DependentSchemas) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "dependentSchemas"),
			LeftValue:   getPropertyNames(left.DependentSchemas),
			RightValue:  getPropertyNames(right.DependentSchemas),
			Description: "dependentSchemas keys mismatch",
		})
	} else if left.DependentSchemas != nil && right.DependentSchemas != nil {
		for name, leftSchema := range left.DependentSchemas {
			rightSchema := right.DependentSchemas[name]
			compareDeep(leftSchema, rightSchema, pathJoin(path, fmt.Sprintf("dependentSchemas.%s", name)), result, visited)
		}
	}
}

// Helper functions

func equalTypes(left, right any) bool {
	// Handle nil cases
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	// Handle string type
	leftStr, leftIsStr := left.(string)
	rightStr, rightIsStr := right.(string)
	if leftIsStr && rightIsStr {
		return leftStr == rightStr
	}

	// Handle array type (OAS 3.1+)
	leftArr, leftIsArr := left.([]string)
	rightArr, rightIsArr := right.([]string)
	if leftIsArr && rightIsArr {
		return equalStringSlices(leftArr, rightArr)
	}

	// Handle any slice that might contain strings
	leftIface, leftIsIface := left.([]any)
	rightIface, rightIsIface := right.([]any)
	if leftIsIface && rightIsIface {
		if len(leftIface) != len(rightIface) {
			return false
		}
		leftStrings := make([]string, len(leftIface))
		rightStrings := make([]string, len(rightIface))
		for i, v := range leftIface {
			if s, ok := v.(string); ok {
				leftStrings[i] = s
			} else {
				return false
			}
		}
		for i, v := range rightIface {
			if s, ok := v.(string); ok {
				rightStrings[i] = s
			} else {
				return false
			}
		}
		return equalStringSlices(leftStrings, rightStrings)
	}

	// Different types
	return false
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	// Create sorted copies for order-independent comparison
	leftCopy := make([]string, len(left))
	rightCopy := make([]string, len(right))
	copy(leftCopy, left)
	copy(rightCopy, right)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func equalPointerFloat(left, right *float64) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func equalPointerInt(left, right *int) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func equalPropertyNames(left, right map[string]*parser.Schema) bool {
	if len(left) != len(right) {
		return false
	}
	for name := range left {
		if _, exists := right[name]; !exists {
			return false
		}
	}
	return true
}

func getPropertyNames(properties map[string]*parser.Schema) []string {
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func compareItemsSchemas(left, right any, path string, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "items"),
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "items presence mismatch",
		})
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		compareDeep(leftSchema, rightSchema, pathJoin(path, "items"), result, visited)
		return
	}

	// Both booleans
	leftBool, leftIsBool := left.(bool)
	rightBool, rightIsBool := right.(bool)
	if leftIsBool && rightIsBool {
		if leftBool != rightBool {
			result.Differences = append(result.Differences, SchemaDifference{
				Path:        pathJoin(path, "items"),
				LeftValue:   leftBool,
				RightValue:  rightBool,
				Description: "items boolean value mismatch",
			})
		}
		return
	}

	// Type mismatch
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        pathJoin(path, "items"),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "items type mismatch",
	})
}

func compareAdditionalPropertiesSchemas(left, right any, path string, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        pathJoin(path, "additionalProperties"),
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "additionalProperties presence mismatch",
		})
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		compareDeep(leftSchema, rightSchema, pathJoin(path, "additionalProperties"), result, visited)
		return
	}

	// Both booleans
	leftBool, leftIsBool := left.(bool)
	rightBool, rightIsBool := right.(bool)
	if leftIsBool && rightIsBool {
		if leftBool != rightBool {
			result.Differences = append(result.Differences, SchemaDifference{
				Path:        pathJoin(path, "additionalProperties"),
				LeftValue:   leftBool,
				RightValue:  rightBool,
				Description: "additionalProperties boolean value mismatch",
			})
		}
		return
	}

	// Type mismatch
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        pathJoin(path, "additionalProperties"),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "additionalProperties type mismatch",
	})
}

// comparePolymorphicSchemas compares schema fields that can be bool or *Schema (e.g., unevaluatedProperties, unevaluatedItems)
func comparePolymorphicSchemas(left, right any, path string, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path,
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "schema presence mismatch",
		})
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		compareDeep(leftSchema, rightSchema, path, result, visited)
		return
	}

	// Both booleans
	leftBool, leftIsBool := left.(bool)
	rightBool, rightIsBool := right.(bool)
	if leftIsBool && rightIsBool {
		if leftBool != rightBool {
			result.Differences = append(result.Differences, SchemaDifference{
				Path:        path,
				LeftValue:   leftBool,
				RightValue:  rightBool,
				Description: "boolean value mismatch",
			})
		}
		return
	}

	// Type mismatch
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        path,
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "type mismatch",
	})
}

func compareSchemaArrays(left, right []*parser.Schema, path string, result *EquivalenceResult, visited map[pointerPair]bool) {
	if len(left) != len(right) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path,
			LeftValue:   len(left),
			RightValue:  len(right),
			Description: "schema array length mismatch",
		})
		return
	}

	for i := range left {
		compareDeep(left[i], right[i], fmt.Sprintf("%s[%d]", path, i), result, visited)
	}
}

func pathJoin(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
