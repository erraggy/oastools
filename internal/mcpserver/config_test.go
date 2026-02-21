package mcpserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// clearOASTOOLSEnv clears all OASTOOLS_* env vars to isolate tests from the ambient environment.
func clearOASTOOLSEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"OASTOOLS_CACHE_ENABLED", "OASTOOLS_CACHE_MAX_SIZE",
		"OASTOOLS_CACHE_FILE_TTL", "OASTOOLS_CACHE_URL_TTL",
		"OASTOOLS_CACHE_CONTENT_TTL", "OASTOOLS_CACHE_SWEEP_INTERVAL",
		"OASTOOLS_WALK_LIMIT", "OASTOOLS_WALK_DETAIL_LIMIT",
		"OASTOOLS_JOIN_PATH_STRATEGY", "OASTOOLS_JOIN_SCHEMA_STRATEGY",
		"OASTOOLS_VALIDATE_STRICT", "OASTOOLS_VALIDATE_NO_WARNINGS",
		"OASTOOLS_MAX_INLINE_SIZE", "OASTOOLS_MAX_LIMIT",
		"OASTOOLS_MAX_JOIN_SPECS", "OASTOOLS_ALLOW_PRIVATE_IPS",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	clearOASTOOLSEnv(t)

	c := loadConfig()

	assert.True(t, c.CacheEnabled)
	assert.Equal(t, 10, c.CacheMaxSize)
	assert.Equal(t, 15*time.Minute, c.CacheFileTTL)
	assert.Equal(t, 5*time.Minute, c.CacheURLTTL)
	assert.Equal(t, 15*time.Minute, c.CacheContentTTL)
	assert.Equal(t, 60*time.Second, c.CacheSweepInterval)
	assert.Equal(t, 100, c.WalkLimit)
	assert.Equal(t, 25, c.WalkDetailLimit)
	assert.Empty(t, c.JoinPathStrategy)
	assert.Empty(t, c.JoinSchemaStrategy)
	assert.False(t, c.ValidateStrict)
	assert.False(t, c.ValidateNoWarnings)
	assert.Equal(t, int64(10*1024*1024), c.MaxInlineSize)
	assert.Equal(t, 1000, c.MaxLimit)
	assert.Equal(t, 20, c.MaxJoinSpecs)
	assert.False(t, c.AllowPrivateIPs)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	clearOASTOOLSEnv(t)
	t.Setenv("OASTOOLS_CACHE_ENABLED", "false")
	t.Setenv("OASTOOLS_CACHE_MAX_SIZE", "50")
	t.Setenv("OASTOOLS_CACHE_FILE_TTL", "30m")
	t.Setenv("OASTOOLS_CACHE_URL_TTL", "2m")
	t.Setenv("OASTOOLS_CACHE_CONTENT_TTL", "10m")
	t.Setenv("OASTOOLS_CACHE_SWEEP_INTERVAL", "30s")
	t.Setenv("OASTOOLS_WALK_LIMIT", "200")
	t.Setenv("OASTOOLS_WALK_DETAIL_LIMIT", "50")
	t.Setenv("OASTOOLS_JOIN_PATH_STRATEGY", "accept-left")
	t.Setenv("OASTOOLS_JOIN_SCHEMA_STRATEGY", "rename-right")
	t.Setenv("OASTOOLS_VALIDATE_STRICT", "true")
	t.Setenv("OASTOOLS_VALIDATE_NO_WARNINGS", "true")
	t.Setenv("OASTOOLS_MAX_INLINE_SIZE", "5242880")
	t.Setenv("OASTOOLS_MAX_LIMIT", "500")
	t.Setenv("OASTOOLS_MAX_JOIN_SPECS", "50")
	t.Setenv("OASTOOLS_ALLOW_PRIVATE_IPS", "true")

	c := loadConfig()

	assert.False(t, c.CacheEnabled)
	assert.Equal(t, 50, c.CacheMaxSize)
	assert.Equal(t, 30*time.Minute, c.CacheFileTTL)
	assert.Equal(t, 2*time.Minute, c.CacheURLTTL)
	assert.Equal(t, 10*time.Minute, c.CacheContentTTL)
	assert.Equal(t, 30*time.Second, c.CacheSweepInterval)
	assert.Equal(t, 200, c.WalkLimit)
	assert.Equal(t, 50, c.WalkDetailLimit)
	assert.Equal(t, "accept-left", c.JoinPathStrategy)
	assert.Equal(t, "rename-right", c.JoinSchemaStrategy)
	assert.True(t, c.ValidateStrict)
	assert.True(t, c.ValidateNoWarnings)
	assert.Equal(t, int64(5242880), c.MaxInlineSize)
	assert.Equal(t, 500, c.MaxLimit)
	assert.Equal(t, 50, c.MaxJoinSpecs)
	assert.True(t, c.AllowPrivateIPs)
}

func TestLoadConfig_InvalidValues_UseDefaults(t *testing.T) {
	clearOASTOOLSEnv(t)
	t.Setenv("OASTOOLS_CACHE_MAX_SIZE", "banana")
	t.Setenv("OASTOOLS_CACHE_FILE_TTL", "not-a-duration")
	t.Setenv("OASTOOLS_CACHE_ENABLED", "maybe")
	t.Setenv("OASTOOLS_WALK_LIMIT", "-5")
	t.Setenv("OASTOOLS_JOIN_PATH_STRATEGY", "typo")
	t.Setenv("OASTOOLS_MAX_INLINE_SIZE", "abc")
	t.Setenv("OASTOOLS_MAX_LIMIT", "0")
	t.Setenv("OASTOOLS_MAX_JOIN_SPECS", "-1")

	c := loadConfig()

	// Invalid values should fall back to defaults.
	assert.True(t, c.CacheEnabled)
	assert.Equal(t, 10, c.CacheMaxSize)
	assert.Equal(t, 15*time.Minute, c.CacheFileTTL)
	assert.Equal(t, 100, c.WalkLimit)
	assert.Empty(t, c.JoinPathStrategy, "invalid strategy should fall back to empty")
	assert.Equal(t, int64(10*1024*1024), c.MaxInlineSize)
	assert.Equal(t, 1000, c.MaxLimit)
	assert.Equal(t, 20, c.MaxJoinSpecs)
}

func TestLoadConfig_PartialOverrides(t *testing.T) {
	clearOASTOOLSEnv(t)
	// Only override some values; others stay at defaults.
	t.Setenv("OASTOOLS_WALK_LIMIT", "42")
	t.Setenv("OASTOOLS_CACHE_URL_TTL", "10m")

	c := loadConfig()

	assert.Equal(t, 42, c.WalkLimit)
	assert.Equal(t, 10*time.Minute, c.CacheURLTTL)
	// Unchanged defaults:
	assert.Equal(t, 25, c.WalkDetailLimit)
	assert.Equal(t, 15*time.Minute, c.CacheFileTTL)
	assert.True(t, c.CacheEnabled)
}
