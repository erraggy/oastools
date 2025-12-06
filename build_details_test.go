package oastools

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVersion verifies that Version() returns the version variable.
// In normal builds, this is set via ldflags by GoReleaser.
// In development, it defaults to "dev".
func TestVersion(t *testing.T) {
	result := Version()

	// Should not be empty
	assert.NotEmpty(t, result, "Version() should not return empty string")

	// Should be either "dev" (development) or a semantic version (e.g., "v1.2.3")
	// We can't assert exact value since it changes per build, but we can verify format
	assert.True(t,
		result == "dev" || strings.HasPrefix(result, "v"),
		"Version() should be 'dev' or start with 'v', got: %s", result)
}

// TestCommit verifies that Commit() returns the commit variable.
// In normal builds, this is set via ldflags by GoReleaser.
// In development, it defaults to "unknown".
func TestCommit(t *testing.T) {
	result := Commit()

	// Should not be empty
	assert.NotEmpty(t, result, "Commit() should not return empty string")

	// Should be either "unknown" (development) or a git short hash (7+ hex chars)
	if result != "unknown" {
		// Git short hash is typically 7+ hex characters
		assert.GreaterOrEqual(t, len(result), 7,
			"Commit() should be at least 7 characters for a git hash, got: %s", result)
		// Verify it's valid hex
		for _, ch := range result {
			assert.True(t, (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'),
				"Commit() should contain only hex characters, got: %s", result)
		}
	}
}

// TestBuildTime verifies that BuildTime() returns the buildTime variable.
// In normal builds, this is set via ldflags by GoReleaser in RFC3339 format.
// In development, it defaults to "unknown".
func TestBuildTime(t *testing.T) {
	result := BuildTime()

	// Should not be empty
	assert.NotEmpty(t, result, "BuildTime() should not return empty string")

	// Should be either "unknown" (development) or an RFC3339 timestamp
	if result != "unknown" {
		// RFC3339 timestamps contain 'T' separator and include timezone
		assert.Contains(t, result, "T",
			"BuildTime() should be RFC3339 format containing 'T', got: %s", result)
	}
}

// TestGoVersion verifies that GoVersion() returns the runtime Go version.
func TestGoVersion(t *testing.T) {
	result := GoVersion()

	// Should not be empty
	assert.NotEmpty(t, result, "GoVersion() should not return empty string")

	// Should match runtime.Version()
	assert.Equal(t, runtime.Version(), result,
		"GoVersion() should match runtime.Version()")

	// Should start with "go"
	assert.True(t, strings.HasPrefix(result, "go"),
		"GoVersion() should start with 'go', got: %s", result)
}

// TestUserAgent verifies that UserAgent() returns a properly formatted User-Agent string.
func TestUserAgent(t *testing.T) {
	result := UserAgent()

	// Should not be empty
	assert.NotEmpty(t, result, "UserAgent() should not return empty string")

	// Should have format "oastools/{version}"
	assert.True(t, strings.HasPrefix(result, "oastools/"),
		"UserAgent() should start with 'oastools/', got: %s", result)

	// Should contain the version
	version := Version()
	expected := "oastools/" + version
	assert.Equal(t, expected, result,
		"UserAgent() should be 'oastools/%s', got: %s", version, result)
}

// TestUserAgentConsistency verifies that UserAgent() uses the same version as Version().
func TestUserAgentConsistency(t *testing.T) {
	version := Version()
	userAgent := UserAgent()

	// UserAgent should contain the version string
	assert.Contains(t, userAgent, version,
		"UserAgent() should contain the version from Version()")

	// Extract version from user agent (after "oastools/")
	parts := strings.SplitN(userAgent, "/", 2)
	assert.Len(t, parts, 2, "UserAgent() should have format 'oastools/{version}'")

	extractedVersion := parts[1]
	assert.Equal(t, version, extractedVersion,
		"Version extracted from UserAgent() should match Version()")
}

// TestVersionFormat verifies that the version string follows expected patterns.
func TestVersionFormat(t *testing.T) {
	version := Version()

	// Development version
	if version == "dev" {
		assert.Equal(t, "dev", version, "Development version should be exactly 'dev'")
		return
	}

	// Release version should start with 'v' and contain digits
	assert.True(t, strings.HasPrefix(version, "v"),
		"Release version should start with 'v', got: %s", version)

	// Should contain at least one digit (part of semver)
	hasDigit := false
	for _, ch := range version {
		if ch >= '0' && ch <= '9' {
			hasDigit = true
			break
		}
	}
	assert.True(t, hasDigit, "Release version should contain at least one digit, got: %s", version)
}

// TestUserAgentFormat verifies that the UserAgent string has no whitespace or special characters.
func TestUserAgentFormat(t *testing.T) {
	userAgent := UserAgent()

	// Should not contain whitespace
	assert.NotContains(t, userAgent, " ", "UserAgent() should not contain spaces")
	assert.NotContains(t, userAgent, "\t", "UserAgent() should not contain tabs")
	assert.NotContains(t, userAgent, "\n", "UserAgent() should not contain newlines")

	// Should not contain other problematic characters for HTTP headers
	assert.NotContains(t, userAgent, "\r", "UserAgent() should not contain carriage returns")
	assert.NotContains(t, userAgent, "\x00", "UserAgent() should not contain null bytes")
}

// TestBuildInfo verifies that BuildInfo() returns a formatted string with all build metadata.
func TestBuildInfo(t *testing.T) {
	result := BuildInfo()

	// Should not be empty
	assert.NotEmpty(t, result, "BuildInfo() should not return empty string")

	// Should contain all build metadata labels
	assert.Contains(t, result, "Version:", "BuildInfo() should contain 'Version:'")
	assert.Contains(t, result, "Commit:", "BuildInfo() should contain 'Commit:'")
	assert.Contains(t, result, "Build Time:", "BuildInfo() should contain 'Build Time:'")
	assert.Contains(t, result, "Go Version:", "BuildInfo() should contain 'Go Version:'")

	// Should contain actual values from the individual functions
	assert.Contains(t, result, Version(), "BuildInfo() should contain Version()")
	assert.Contains(t, result, Commit(), "BuildInfo() should contain Commit()")
	assert.Contains(t, result, BuildTime(), "BuildInfo() should contain BuildTime()")
	assert.Contains(t, result, GoVersion(), "BuildInfo() should contain GoVersion()")
}
