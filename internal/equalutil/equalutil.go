package equalutil

// EqualPtr compares two pointers of any comparable type for equality.
// Both nil returns true, both non-nil with equal values returns true.
func EqualPtr[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
