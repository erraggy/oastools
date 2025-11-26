package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// version represents a semantic version with major, minor, and patch components.
// It supports comparison and parsing of standard semver strings (e.g., "3.0.1", "3.1.0-rc1").
type version struct {
	major      int
	minor      int
	patch      int
	prerelease string
}

// parseVersion parses a semantic version string into a version struct.
// Supports standard semver format: "major.minor.patch" with optional "-prerelease" suffix.
// Examples: "2.0", "3.0.1", "3.1.0-rc1"
func parseVersion(s string) (*version, error) {
	// Split off pre-release suffix if present
	var prerelease string
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	// Split version components
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	// Parse major with bounds checking
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 || major > math.MaxInt32 {
		return nil, fmt.Errorf("invalid major version: %q", parts[0])
	}

	// Parse minor with bounds checking
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 || minor > math.MaxInt32 {
		return nil, fmt.Errorf("invalid minor version: %q", parts[1])
	}

	// Parse patch (optional, defaults to 0) with bounds checking
	patch := 0
	if len(parts) == 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil || patch < 0 || patch > math.MaxInt32 {
			return nil, fmt.Errorf("invalid patch version: %q", parts[2])
		}
	}

	return &version{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: prerelease,
	}, nil
}

// segments returns the version components as a slice [major, minor, patch].
func (v *version) segments() []int {
	return []int{v.major, v.minor, v.patch}
}

// lessThan returns true if v < other.
// Pre-release versions are compared lexicographically if major.minor.patch are equal.
func (v *version) lessThan(other *version) bool {
	if v.major != other.major {
		return v.major < other.major
	}
	if v.minor != other.minor {
		return v.minor < other.minor
	}
	if v.patch != other.patch {
		return v.patch < other.patch
	}
	// If major.minor.patch are equal, compare pre-release
	// Pre-release version has lower precedence than normal version
	if v.prerelease == "" && other.prerelease != "" {
		return false // v is release, other is pre-release
	}
	if v.prerelease != "" && other.prerelease == "" {
		return true // v is pre-release, other is release
	}
	// Both have pre-release or both don't
	// Note: This uses simplified lexicographic comparison, which is sufficient
	// for OpenAPI version strings (e.g., "3.0.0-rc1" < "3.0.0-rc2").
	// The full semver spec has more complex pre-release precedence rules.
	return v.prerelease < other.prerelease
}

// greaterThanOrEqual returns true if v >= other.
func (v *version) greaterThanOrEqual(other *version) bool {
	return !v.lessThan(other)
}
