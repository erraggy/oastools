package parser

import (
	"fmt"

	semver "github.com/hashicorp/go-version"
)

// OASVersion represents each canonical version of the OpenAPI Specification that may be found at:
// https://github.com/OAI/OpenAPI-Specification/releases
type OASVersion int

const (
	// Unknown represents an unknown or invalid OAS version
	Unknown OASVersion = iota
	// OASVersion20 OpenAPI Specification Version 2.0 (Swagger)
	OASVersion20
	// OASVersion300 OpenAPI Specification Version 3.0.0
	OASVersion300
	// OASVersion301  OpenAPI Specification Version 3.0.1
	OASVersion301
	// OASVersion302  OpenAPI Specification Version 3.0.2
	OASVersion302
	// OASVersion303  OpenAPI Specification Version 3.0.3
	OASVersion303
	// OASVersion304  OpenAPI Specification Version 3.0.4
	OASVersion304
	// OASVersion310  OpenAPI Specification Version 3.1.0
	OASVersion310
	// OASVersion311  OpenAPI Specification Version 3.1.1
	OASVersion311
	// OASVersion312  OpenAPI Specification Version 3.1.2
	OASVersion312
	// OASVersion320  OpenAPI Specification Version 3.2.0
	OASVersion320
)

var (
	versionToString = map[OASVersion]string{
		OASVersion20:  "2.0",
		OASVersion300: "3.0.0",
		OASVersion301: "3.0.1",
		OASVersion302: "3.0.2",
		OASVersion303: "3.0.3",
		OASVersion304: "3.0.4",
		OASVersion310: "3.1.0",
		OASVersion311: "3.1.1",
		OASVersion312: "3.1.2",
		OASVersion320: "3.2.0",
	}

	stringToVersion = func() map[string]OASVersion {
		m := make(map[string]OASVersion, 10)
		for k, v := range versionToString {
			m[v] = k
		}
		return m
	}()
)

func (v OASVersion) String() string {
	if s, ok := versionToString[v]; ok {
		return s
	}
	return "unknown"
}

// IsValid returns true if this is a valid version
func (v OASVersion) IsValid() bool {
	_, ok := versionToString[v]
	return ok
}

// ParseVersion will attempt to parse the string s into an OASVersion, and returns false if not valid.
// This function supports:
// 1. Exact version matches (e.g., "2.0", "3.0.3")
// 2. Future patch versions in known major.minor series (e.g., "3.0.5" maps to "3.0.4")
// 3. Pre-release versions (e.g., "3.0.0-rc0") map to closest match without exceeding base version
//
// For example:
// - "3.0.5" (not yet released) maps to OASVersion304 (3.0.4) - latest in 3.0.x series
// - "3.0.0-rc0" maps to OASVersion300 (3.0.0) - the base version
// - "3.0.5-rc1" maps to OASVersion304 (3.0.4) - closest without exceeding 3.0.5
func ParseVersion(s string) (OASVersion, bool) {
	// First try exact match
	if v, ok := stringToVersion[s]; ok {
		return v, ok
	}

	// Special case: "2.0" doesn't need patch version handling
	if s == "2.0" {
		return OASVersion20, true
	}

	// Try to parse as semver and map to known major.minor series
	ver, err := semver.NewVersion(s)
	if err != nil {
		return Unknown, false
	}

	// Extract major.minor.patch from the base version (stripping pre-release suffix)
	segments := ver.Segments()
	if len(segments) < 2 {
		return Unknown, false
	}

	// Handle 2.x versions - only 2.0 is supported
	if segments[0] == 2 {
		if segments[1] == 0 {
			return OASVersion20, true
		}
		return Unknown, false
	}

	// Handle 3.x versions - find closest match without exceeding the base version
	if segments[0] == 3 {
		// Get the base version string (without pre-release suffix)
		baseVersion := fmt.Sprintf("%d.%d.%d", segments[0], segments[1], segments[2])

		// Try exact match on base version first (handles RC of known versions)
		if v, ok := stringToVersion[baseVersion]; ok {
			return v, true
		}

		// Find the closest version in this major.minor series that doesn't exceed baseVersion
		return findClosestVersion(segments[0], segments[1], segments[2])
	}

	return Unknown, false
}

// findClosestVersion finds the closest known version that doesn't exceed major.minor.patch
func findClosestVersion(major, minor, patch int) (OASVersion, bool) {
	// Iterate through all known versions in descending order
	// and find the highest one that doesn't exceed the target version
	var candidates []struct {
		version OASVersion
		major   int
		minor   int
		patch   int
	}

	for oasVer, verStr := range versionToString {
		if oasVer == OASVersion20 {
			continue // Skip 2.0
		}

		// Parse the version string
		v, err := semver.NewVersion(verStr)
		if err != nil {
			continue
		}

		segs := v.Segments()
		if len(segs) < 3 {
			continue
		}

		// Only consider versions in the same major.minor series
		if segs[0] == major && segs[1] == minor {
			candidates = append(candidates, struct {
				version OASVersion
				major   int
				minor   int
				patch   int
			}{oasVer, segs[0], segs[1], segs[2]})
		}
	}

	// Find the highest patch version that doesn't exceed the target
	var bestMatch OASVersion
	bestPatch := -1

	for _, cand := range candidates {
		if cand.patch <= patch && cand.patch > bestPatch {
			bestMatch = cand.version
			bestPatch = cand.patch
		}
	}

	if bestPatch >= 0 {
		return bestMatch, true
	}

	return Unknown, false
}
