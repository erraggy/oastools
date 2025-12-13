package jsonpath

import (
	"fmt"
	"reflect"
)

// evalFilter evaluates a filter expression against a value.
//
// The value is typically a map[string]any representing an object.
// Returns true if the filter matches, false otherwise.
func evalFilter(value any, expr *FilterExpr) bool {
	if expr == nil {
		return true
	}

	// Get the field value from the object
	fieldValue := getFieldValue(value, expr.Field)

	// Compare using the operator
	return compare(fieldValue, expr.Operator, expr.Value)
}

// getFieldValue extracts a field value from an object.
//
// Supports nested fields using dot notation (e.g., "outer.inner").
func getFieldValue(obj any, field string) any {
	m, ok := obj.(map[string]any)
	if !ok {
		return nil
	}

	// Handle nested fields (e.g., "x-custom.nested")
	// For now, we only support single-level field access
	val, exists := m[field]
	if !exists {
		return nil
	}
	return val
}

// compare performs a comparison between two values using the given operator.
func compare(left any, op string, right any) bool {
	// Handle nil cases
	if left == nil && right == nil {
		return op == "==" || op == "<=" || op == ">="
	}
	if left == nil || right == nil {
		return op == "!="
	}

	// Normalize types for comparison
	leftNorm := normalizeValue(left)
	rightNorm := normalizeValue(right)

	switch op {
	case "==":
		return valuesEqual(leftNorm, rightNorm)
	case "!=":
		return !valuesEqual(leftNorm, rightNorm)
	case "<":
		return compareLess(leftNorm, rightNorm)
	case "<=":
		return compareLess(leftNorm, rightNorm) || valuesEqual(leftNorm, rightNorm)
	case ">":
		return compareLess(rightNorm, leftNorm)
	case ">=":
		return compareLess(rightNorm, leftNorm) || valuesEqual(leftNorm, rightNorm)
	default:
		return false
	}
}

// normalizeValue converts values to comparable types.
//
// This handles the common case where YAML unmarshaling produces different
// numeric types (int vs int64 vs float64).
func normalizeValue(v any) any {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return float64(val)
	default:
		return v
	}
}

// valuesEqual checks if two normalized values are equal.
func valuesEqual(left, right any) bool {
	// Direct comparison for simple types
	if left == right {
		return true
	}

	// Handle float comparison with tolerance for integer values
	leftFloat, leftIsFloat := left.(float64)
	rightFloat, rightIsFloat := right.(float64)
	if leftIsFloat && rightIsFloat {
		return leftFloat == rightFloat
	}

	// String comparison
	leftStr, leftIsStr := left.(string)
	rightStr, rightIsStr := right.(string)
	if leftIsStr && rightIsStr {
		return leftStr == rightStr
	}

	// Boolean comparison
	leftBool, leftIsBool := left.(bool)
	rightBool, rightIsBool := right.(bool)
	if leftIsBool && rightIsBool {
		return leftBool == rightBool
	}

	// Use reflection for complex types
	return reflect.DeepEqual(left, right)
}

// compareLess checks if left < right for ordered types.
func compareLess(left, right any) bool {
	// Numeric comparison
	leftFloat, leftIsFloat := left.(float64)
	rightFloat, rightIsFloat := right.(float64)
	if leftIsFloat && rightIsFloat {
		return leftFloat < rightFloat
	}

	// String comparison
	leftStr, leftIsStr := left.(string)
	rightStr, rightIsStr := right.(string)
	if leftIsStr && rightIsStr {
		return leftStr < rightStr
	}

	// Cannot compare - return false
	return false
}

// String returns a string representation of the filter expression.
func (f *FilterExpr) String() string {
	return fmt.Sprintf("@.%s %s %v", f.Field, f.Operator, f.Value)
}
