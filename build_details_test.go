package oastools

import (
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
