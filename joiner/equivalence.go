package joiner

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/equalutil"
	"github.com/erraggy/oastools/parser"
)

// comparePath is a stack-based path builder that minimizes allocations.
// Instead of creating a new string on each recursive call, it appends
// segments to a slice and only builds the full string when needed.
type comparePath struct {
	segments []string
}

func (p *comparePath) push(segment string) {
	p.segments = append(p.segments, segment)
}

func (p *comparePath) pop() {
	if len(p.segments) > 0 {
		p.segments = p.segments[:len(p.segments)-1]
	}
}

func (p *comparePath) String() string {
	return strings.Join(p.segments, ".")
}

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

// isEmptySchema reports whether a schema has no structural constraints.
// A schema is considered "empty" if it has no type, format, properties,
// validation rules, or composition keywords. Metadata fields (title,
// description, example, deprecated, extensions) are NOT considered constraints.
//
// Constraint fields checked:
//   - Basic: Type, Format, Enum, Const, Pattern, Required
//   - OAS-specific: Nullable, ReadOnly, WriteOnly, CollectionFormat
//   - Object: Properties, AdditionalProperties, MinProperties, MaxProperties,
//     PatternProperties, DependentRequired
//   - Array: Items, MinItems, MaxItems, UniqueItems, AdditionalItems,
//     MaxContains, MinContains
//   - Numeric: Minimum, Maximum, MultipleOf, ExclusiveMinimum, ExclusiveMaximum
//   - String: MinLength, MaxLength
//   - Composition: AllOf, AnyOf, OneOf, Not
//   - Conditional: If, Then, Else
//   - JSON Schema 2020-12: UnevaluatedProperties, UnevaluatedItems,
//     ContentEncoding, ContentMediaType, ContentSchema, PrefixItems,
//     Contains, PropertyNames, DependentSchemas
//
// Empty schemas are semantically distinct even when structurally identical,
// because they serve different purposes depending on context (placeholders,
// "any type" markers, context-specific wildcards). Returning false for nil
// schemas prevents nil-pointer panics in callers.
func isEmptySchema(s *parser.Schema) bool {
	if s == nil {
		return false
	}

	// Basic type constraints
	if s.Type != nil {
		return false
	}
	if s.Format != "" {
		return false
	}
	if len(s.Enum) > 0 {
		return false
	}
	if s.Const != nil {
		return false
	}
	if s.Pattern != "" {
		return false
	}
	if len(s.Required) > 0 {
		return false
	}

	// OAS-specific constraints
	if s.Nullable {
		return false
	}
	if s.ReadOnly {
		return false
	}
	if s.WriteOnly {
		return false
	}
	if s.CollectionFormat != "" {
		return false
	}

	// Properties and object constraints
	if len(s.Properties) > 0 {
		return false
	}
	if s.AdditionalProperties != nil {
		return false
	}
	if s.MinProperties != nil {
		return false
	}
	if s.MaxProperties != nil {
		return false
	}
	if len(s.PatternProperties) > 0 {
		return false
	}
	if len(s.DependentRequired) > 0 {
		return false
	}

	// Array constraints
	if s.Items != nil {
		return false
	}
	if s.MinItems != nil {
		return false
	}
	if s.MaxItems != nil {
		return false
	}
	if s.UniqueItems {
		return false
	}
	if s.AdditionalItems != nil {
		return false
	}
	if s.MaxContains != nil {
		return false
	}
	if s.MinContains != nil {
		return false
	}

	// Numeric constraints
	if s.Minimum != nil {
		return false
	}
	if s.Maximum != nil {
		return false
	}
	if s.MultipleOf != nil {
		return false
	}
	if s.ExclusiveMinimum != nil {
		return false
	}
	if s.ExclusiveMaximum != nil {
		return false
	}

	// String constraints
	if s.MinLength != nil {
		return false
	}
	if s.MaxLength != nil {
		return false
	}

	// Composition
	if len(s.AllOf) > 0 {
		return false
	}
	if len(s.AnyOf) > 0 {
		return false
	}
	if len(s.OneOf) > 0 {
		return false
	}
	if s.Not != nil {
		return false
	}

	// Conditional composition
	if s.If != nil {
		return false
	}
	if s.Then != nil {
		return false
	}
	if s.Else != nil {
		return false
	}

	// JSON Schema 2020-12 fields
	if s.UnevaluatedProperties != nil {
		return false
	}
	if s.UnevaluatedItems != nil {
		return false
	}
	if s.ContentEncoding != "" {
		return false
	}
	if s.ContentMediaType != "" {
		return false
	}
	if s.ContentSchema != nil {
		return false
	}
	if len(s.PrefixItems) > 0 {
		return false
	}
	if s.Contains != nil {
		return false
	}
	if s.PropertyNames != nil {
		return false
	}
	if len(s.DependentSchemas) > 0 {
		return false
	}

	return true
}

// String returns a human-readable representation of the equivalence result.
// Special case: When Equivalent is false but Differences is non-nil and empty,
// this indicates empty schemas that are structurally identical but semantically distinct.
func (r EquivalenceResult) String() string {
	if r.Equivalent {
		return "Schemas are equivalent"
	}
	if r.Differences != nil && len(r.Differences) == 0 {
		return "Schemas are non-equivalent (empty schemas are semantically distinct)"
	}
	var b strings.Builder
	b.WriteString("Schemas differ:\n")
	for _, d := range r.Differences {
		fmt.Fprintf(&b, "  - %s: %s\n", d.Path, d.Description)
	}
	return b.String()
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

	// Empty schemas are semantically distinct - never equivalent.
	// They serve different purposes depending on context (placeholders,
	// "any type" markers, context-specific wildcards) and should not be
	// consolidated during deduplication.
	if isEmptySchema(left) || isEmptySchema(right) {
		return EquivalenceResult{
			Equivalent:  false,
			Differences: []SchemaDifference{},
		}
	}

	// Track visited pointers to handle circular references
	visited := make(map[pointerPair]bool)

	// Use stack-based path builder to minimize allocations
	path := &comparePath{segments: make([]string, 0, 8)}

	if mode == EquivalenceModeShallow {
		compareShallow(left, right, path, &result)
	} else {
		compareDeep(left, right, path, &result, visited)
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
func compareCommonFields(left, right *parser.Schema, path *comparePath, result *EquivalenceResult) {
	// Compare type
	if !equalTypes(left.Type, right.Type) {
		path.push("type")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Type,
			RightValue:  right.Type,
			Description: "type mismatch",
		})
		path.pop()
	}

	// Compare format
	if left.Format != right.Format {
		path.push("format")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Format,
			RightValue:  right.Format,
			Description: "format mismatch",
		})
		path.pop()
	}

	// Compare required arrays (order-independent)
	if !equalStringSlices(left.Required, right.Required) {
		path.push("required")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Required,
			RightValue:  right.Required,
			Description: "required fields mismatch",
		})
		path.pop()
	}

	// Compare enum (order matters for enum)
	if !reflect.DeepEqual(left.Enum, right.Enum) {
		path.push("enum")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Enum,
			RightValue:  right.Enum,
			Description: "enum values mismatch",
		})
		path.pop()
	}

	// Compare property names (shallow - don't compare nested schemas)
	if !equalPropertyNames(left.Properties, right.Properties) {
		path.push("properties")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   getPropertyNames(left.Properties),
			RightValue:  getPropertyNames(right.Properties),
			Description: "property names mismatch",
		})
		path.pop()
	}
}

// compareShallow compares only the top-level properties of schemas
func compareShallow(left, right *parser.Schema, path *comparePath, result *EquivalenceResult) {
	compareCommonFields(left, right, path, result)
}

// compareDeep recursively compares all schema properties
func compareDeep(left, right *parser.Schema, path *comparePath, result *EquivalenceResult, visited map[pointerPair]bool) {
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
		path.push("pattern")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Pattern,
			RightValue:  right.Pattern,
			Description: "pattern mismatch",
		})
		path.pop()
	}

	// Compare const
	if !reflect.DeepEqual(left.Const, right.Const) {
		path.push("const")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Const,
			RightValue:  right.Const,
			Description: "const value mismatch",
		})
		path.pop()
	}

	// Compare numeric constraints
	if !equalutil.EqualPtr(left.Minimum, right.Minimum) {
		path.push("minimum")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Minimum,
			RightValue:  right.Minimum,
			Description: "minimum constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.Maximum, right.Maximum) {
		path.push("maximum")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Maximum,
			RightValue:  right.Maximum,
			Description: "maximum constraint mismatch",
		})
		path.pop()
	}

	// Compare string constraints
	if !equalutil.EqualPtr(left.MinLength, right.MinLength) {
		path.push("minLength")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MinLength,
			RightValue:  right.MinLength,
			Description: "minLength constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.MaxLength, right.MaxLength) {
		path.push("maxLength")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MaxLength,
			RightValue:  right.MaxLength,
			Description: "maxLength constraint mismatch",
		})
		path.pop()
	}

	// Compare array constraints
	if !equalutil.EqualPtr(left.MinItems, right.MinItems) {
		path.push("minItems")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MinItems,
			RightValue:  right.MinItems,
			Description: "minItems constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.MaxItems, right.MaxItems) {
		path.push("maxItems")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MaxItems,
			RightValue:  right.MaxItems,
			Description: "maxItems constraint mismatch",
		})
		path.pop()
	}
	if left.UniqueItems != right.UniqueItems {
		path.push("uniqueItems")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.UniqueItems,
			RightValue:  right.UniqueItems,
			Description: "uniqueItems constraint mismatch",
		})
		path.pop()
	}

	// Compare object constraints
	if !equalutil.EqualPtr(left.MinProperties, right.MinProperties) {
		path.push("minProperties")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MinProperties,
			RightValue:  right.MinProperties,
			Description: "minProperties constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.MaxProperties, right.MaxProperties) {
		path.push("maxProperties")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MaxProperties,
			RightValue:  right.MaxProperties,
			Description: "maxProperties constraint mismatch",
		})
		path.pop()
	}

	// Compare properties recursively (property names already checked by compareCommonFields)
	if equalPropertyNames(left.Properties, right.Properties) && left.Properties != nil {
		path.push("properties")
		for name, leftProp := range left.Properties {
			rightProp := right.Properties[name]
			path.push(name)
			compareDeep(leftProp, rightProp, path, result, visited)
			path.pop()
		}
		path.pop()
	}

	// Compare items (array item schema)
	compareItemsSchemas(left.Items, right.Items, path, result, visited)

	// Compare additionalProperties
	compareAdditionalPropertiesSchemas(left.AdditionalProperties, right.AdditionalProperties, path, result, visited)

	// Compare composition (allOf, anyOf, oneOf)
	path.push("allOf")
	compareSchemaArrays(left.AllOf, right.AllOf, path, result, visited)
	path.pop()
	path.push("anyOf")
	compareSchemaArrays(left.AnyOf, right.AnyOf, path, result, visited)
	path.pop()
	path.push("oneOf")
	compareSchemaArrays(left.OneOf, right.OneOf, path, result, visited)
	path.pop()

	// Compare not
	if (left.Not == nil) != (right.Not == nil) {
		path.push("not")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Not != nil,
			RightValue:  right.Not != nil,
			Description: "not schema presence mismatch",
		})
		path.pop()
	} else if left.Not != nil && right.Not != nil {
		path.push("not")
		compareDeep(left.Not, right.Not, path, result, visited)
		path.pop()
	}

	// JSON Schema 2020-12 fields

	// Compare unevaluatedProperties (can be bool or *Schema)
	path.push("unevaluatedProperties")
	comparePolymorphicSchemas(left.UnevaluatedProperties, right.UnevaluatedProperties, path, result, visited)
	path.pop()

	// Compare unevaluatedItems (can be bool or *Schema)
	path.push("unevaluatedItems")
	comparePolymorphicSchemas(left.UnevaluatedItems, right.UnevaluatedItems, path, result, visited)
	path.pop()

	// Compare contentEncoding
	if left.ContentEncoding != right.ContentEncoding {
		path.push("contentEncoding")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.ContentEncoding,
			RightValue:  right.ContentEncoding,
			Description: "contentEncoding mismatch",
		})
		path.pop()
	}

	// Compare contentMediaType
	if left.ContentMediaType != right.ContentMediaType {
		path.push("contentMediaType")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.ContentMediaType,
			RightValue:  right.ContentMediaType,
			Description: "contentMediaType mismatch",
		})
		path.pop()
	}

	// Compare contentSchema
	if (left.ContentSchema == nil) != (right.ContentSchema == nil) {
		path.push("contentSchema")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.ContentSchema != nil,
			RightValue:  right.ContentSchema != nil,
			Description: "contentSchema presence mismatch",
		})
		path.pop()
	} else if left.ContentSchema != nil && right.ContentSchema != nil {
		path.push("contentSchema")
		compareDeep(left.ContentSchema, right.ContentSchema, path, result, visited)
		path.pop()
	}

	// Compare prefixItems
	path.push("prefixItems")
	compareSchemaArrays(left.PrefixItems, right.PrefixItems, path, result, visited)
	path.pop()

	// Compare contains
	if (left.Contains == nil) != (right.Contains == nil) {
		path.push("contains")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Contains != nil,
			RightValue:  right.Contains != nil,
			Description: "contains schema presence mismatch",
		})
		path.pop()
	} else if left.Contains != nil && right.Contains != nil {
		path.push("contains")
		compareDeep(left.Contains, right.Contains, path, result, visited)
		path.pop()
	}

	// Compare propertyNames
	if (left.PropertyNames == nil) != (right.PropertyNames == nil) {
		path.push("propertyNames")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.PropertyNames != nil,
			RightValue:  right.PropertyNames != nil,
			Description: "propertyNames schema presence mismatch",
		})
		path.pop()
	} else if left.PropertyNames != nil && right.PropertyNames != nil {
		path.push("propertyNames")
		compareDeep(left.PropertyNames, right.PropertyNames, path, result, visited)
		path.pop()
	}

	// Compare dependentSchemas
	if !equalPropertyNames(left.DependentSchemas, right.DependentSchemas) {
		path.push("dependentSchemas")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   getPropertyNames(left.DependentSchemas),
			RightValue:  getPropertyNames(right.DependentSchemas),
			Description: "dependentSchemas keys mismatch",
		})
		path.pop()
	} else if left.DependentSchemas != nil && right.DependentSchemas != nil {
		path.push("dependentSchemas")
		for name, leftSchema := range left.DependentSchemas {
			rightSchema := right.DependentSchemas[name]
			path.push(name)
			compareDeep(leftSchema, rightSchema, path, result, visited)
			path.pop()
		}
		path.pop()
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

func compareItemsSchemas(left, right any, path *comparePath, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		path.push("items")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "items presence mismatch",
		})
		path.pop()
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		path.push("items")
		compareDeep(leftSchema, rightSchema, path, result, visited)
		path.pop()
		return
	}

	// Both booleans
	leftBool, leftIsBool := left.(bool)
	rightBool, rightIsBool := right.(bool)
	if leftIsBool && rightIsBool {
		if leftBool != rightBool {
			path.push("items")
			result.Differences = append(result.Differences, SchemaDifference{
				Path:        path.String(),
				LeftValue:   leftBool,
				RightValue:  rightBool,
				Description: "items boolean value mismatch",
			})
			path.pop()
		}
		return
	}

	// Type mismatch
	path.push("items")
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "items type mismatch",
	})
	path.pop()
}

func compareAdditionalPropertiesSchemas(left, right any, path *comparePath, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		path.push("additionalProperties")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "additionalProperties presence mismatch",
		})
		path.pop()
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		path.push("additionalProperties")
		compareDeep(leftSchema, rightSchema, path, result, visited)
		path.pop()
		return
	}

	// Both booleans
	leftBool, leftIsBool := left.(bool)
	rightBool, rightIsBool := right.(bool)
	if leftIsBool && rightIsBool {
		if leftBool != rightBool {
			path.push("additionalProperties")
			result.Differences = append(result.Differences, SchemaDifference{
				Path:        path.String(),
				LeftValue:   leftBool,
				RightValue:  rightBool,
				Description: "additionalProperties boolean value mismatch",
			})
			path.pop()
		}
		return
	}

	// Type mismatch
	path.push("additionalProperties")
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "additionalProperties type mismatch",
	})
	path.pop()
}

// comparePolymorphicSchemas compares schema fields that can be bool or *Schema (e.g., unevaluatedProperties, unevaluatedItems)
func comparePolymorphicSchemas(left, right any, path *comparePath, result *EquivalenceResult, visited map[pointerPair]bool) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
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
				Path:        path.String(),
				LeftValue:   leftBool,
				RightValue:  rightBool,
				Description: "boolean value mismatch",
			})
		}
		return
	}

	// Type mismatch
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "type mismatch",
	})
}

func compareSchemaArrays(left, right []*parser.Schema, path *comparePath, result *EquivalenceResult, visited map[pointerPair]bool) {
	if len(left) != len(right) {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   len(left),
			RightValue:  len(right),
			Description: "schema array length mismatch",
		})
		return
	}

	for i := range left {
		// Use strconv.Itoa instead of fmt.Sprintf for better performance
		path.push("[" + strconv.Itoa(i) + "]")
		compareDeep(left[i], right[i], path, result, visited)
		path.pop()
	}
}

// pathJoin is kept for backward compatibility but internal code uses comparePath.
// This function is still used by tests that call it directly.
func pathJoin(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
