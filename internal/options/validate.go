// Package options provides shared utilities for option validation across packages.
package options

import "fmt"

// ValidateSingleInputSource ensures exactly one input source is specified.
// sources is a variadic list of booleans indicating whether each source is set.
// noSourceMsg is the error message when no source is specified.
// multiSourceMsg is the error message when multiple sources are specified.
// Returns an error if zero or more than one input source is specified.
func ValidateSingleInputSource(noSourceMsg, multiSourceMsg string, sources ...bool) error {
	sourceCount := 0
	for _, hasSource := range sources {
		if hasSource {
			sourceCount++
		}
	}

	if sourceCount == 0 {
		return fmt.Errorf("%s", noSourceMsg)
	}
	if sourceCount > 1 {
		return fmt.Errorf("%s", multiSourceMsg)
	}

	return nil
}
