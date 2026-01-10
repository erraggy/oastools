package parser

// ptr returns a pointer to a float64 value.
// Used in tests for pointer fields like Maximum, Minimum, MultipleOf.
func ptr(v float64) *float64 {
	return &v
}

// intPtr returns a pointer to an int value.
// Used in tests for pointer fields like MaxLength, MinLength, MaxItems, etc.
func intPtr(v int) *int {
	return &v
}

// boolPtr returns a pointer to a bool value.
// Used in tests for optional boolean fields.
func boolPtr(v bool) *bool {
	return &v
}
