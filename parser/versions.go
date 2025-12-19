package parser

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

// seriesInfo pre-computed info for a major.minor version series
type seriesInfo struct {
	// patches maps patch version -> OASVersion for this series
	// e.g., for 3.0.x series: {0: OASVersion300, 1: OASVersion301, ...}
	patches map[int]OASVersion
	// maxPatch is the highest known patch version in this series
	maxPatch int
}

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

	// versionSeriesLookup maps "major.minor" -> seriesInfo for O(1) future version lookups
	// e.g., "3.0" -> {patches: {0: 300, 1: 301, 2: 302, 3: 303, 4: 304}, maxPatch: 4}
	versionSeriesLookup = func() map[string]seriesInfo {
		m := make(map[string]seriesInfo)
		for oasVer, verStr := range versionToString {
			if oasVer == OASVersion20 {
				continue // 2.0 is special case
			}

			v, err := parseVersion(verStr)
			if err != nil {
				continue
			}

			segs := v.segments()
			if len(segs) < 3 {
				continue
			}

			key := seriesKey(segs[0], segs[1])
			info, exists := m[key]
			if !exists {
				info = seriesInfo{patches: make(map[int]OASVersion), maxPatch: -1}
			}
			info.patches[segs[2]] = oasVer
			if segs[2] > info.maxPatch {
				info.maxPatch = segs[2]
			}
			m[key] = info
		}
		return m
	}()
)

// seriesKey returns a string key for major.minor lookup (e.g., "3.0", "3.1")
func seriesKey(major, minor int) string {
	// Pre-allocate for common case of single-digit numbers
	buf := make([]byte, 0, 5)
	if major >= 10 {
		buf = append(buf, byte('0'+major/10))
	}
	buf = append(buf, byte('0'+major%10))
	buf = append(buf, '.')
	if minor >= 10 {
		buf = append(buf, byte('0'+minor/10))
	}
	buf = append(buf, byte('0'+minor%10))
	return string(buf)
}

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
	// First try exact match (handles all known versions including "2.0")
	if v, ok := stringToVersion[s]; ok {
		return v, true
	}

	// Try to parse as semver and map to known major.minor series
	ver, err := parseVersion(s)
	if err != nil {
		return Unknown, false
	}

	// Extract major.minor.patch from the base version (stripping pre-release suffix)
	segments := ver.segments()
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

	// Handle 3.x versions - find closest match using pre-computed lookup
	if segments[0] == 3 {
		return findClosestVersion(segments[0], segments[1], segments[2])
	}

	return Unknown, false
}

// findClosestVersion finds the closest known version that doesn't exceed major.minor.patch
// Uses pre-computed versionSeriesLookup for O(1) lookups
func findClosestVersion(major, minor, patch int) (OASVersion, bool) {
	key := seriesKey(major, minor)
	info, exists := versionSeriesLookup[key]
	if !exists {
		return Unknown, false
	}

	// If exact patch exists, return it
	if v, ok := info.patches[patch]; ok {
		return v, true
	}

	// If requested patch exceeds our max, return the max
	if patch > info.maxPatch {
		return info.patches[info.maxPatch], true
	}

	// Find the highest patch version that doesn't exceed the target
	// This handles cases like requesting 3.0.3 when we have 0, 1, 2, 4 (skip 3)
	for p := patch; p >= 0; p-- {
		if v, ok := info.patches[p]; ok {
			return v, true
		}
	}

	return Unknown, false
}
