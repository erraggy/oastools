package stringutil

import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail checks if s is a valid email address.
func IsValidEmail(s string) bool {
	return emailRegex.MatchString(s)
}
