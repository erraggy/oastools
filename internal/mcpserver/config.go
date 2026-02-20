package mcpserver

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

// serverConfig holds all configurable MCP server defaults.
// Loaded once at startup from environment variables via loadConfig().
type serverConfig struct {
	// Cache settings.
	CacheEnabled       bool
	CacheMaxSize       int
	CacheFileTTL       time.Duration
	CacheURLTTL        time.Duration
	CacheContentTTL    time.Duration
	CacheSweepInterval time.Duration

	// Walk tool defaults.
	WalkLimit       int
	WalkDetailLimit int

	// Join tool defaults.
	JoinPathStrategy   string
	JoinSchemaStrategy string

	// Validate tool defaults.
	ValidateStrict     bool
	ValidateNoWarnings bool
}

// cfg is the active server configuration, initialized at package load time.
var cfg = loadConfig()

// loadConfig reads configuration from OASTOOLS_* environment variables.
// Invalid values log a warning and fall back to the hardcoded default.
func loadConfig() *serverConfig {
	return &serverConfig{
		CacheEnabled:       envBool("OASTOOLS_CACHE_ENABLED", true),
		CacheMaxSize:       envInt("OASTOOLS_CACHE_MAX_SIZE", 10),
		CacheFileTTL:       envDuration("OASTOOLS_CACHE_FILE_TTL", 15*time.Minute),
		CacheURLTTL:        envDuration("OASTOOLS_CACHE_URL_TTL", 5*time.Minute),
		CacheContentTTL:    envDuration("OASTOOLS_CACHE_CONTENT_TTL", 15*time.Minute),
		CacheSweepInterval: envDuration("OASTOOLS_CACHE_SWEEP_INTERVAL", 60*time.Second),
		WalkLimit:          envInt("OASTOOLS_WALK_LIMIT", 100),
		WalkDetailLimit:    envInt("OASTOOLS_WALK_DETAIL_LIMIT", 25),
		JoinPathStrategy:   envStrategy("OASTOOLS_JOIN_PATH_STRATEGY"),
		JoinSchemaStrategy: envStrategy("OASTOOLS_JOIN_SCHEMA_STRATEGY"),
		ValidateStrict:     envBool("OASTOOLS_VALIDATE_STRICT", false),
		ValidateNoWarnings: envBool("OASTOOLS_VALIDATE_NO_WARNINGS", false),
	}
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn("invalid bool env var, using default", "key", key, "value", v, "default", fallback) //nolint:gosec // G706: values are structured log fields, not format strings
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		slog.Warn("invalid int env var, using default", "key", key, "value", v, "default", fallback) //nolint:gosec // G706: values are structured log fields, not format strings
		return fallback
	}
	return n
}

// validJoinStrategies is the set of recognised collision strategy values.
// Must stay in sync with joiner.CollisionStrategy constants.
var validJoinStrategies = map[string]bool{
	"accept-left": true, "accept-right": true,
	"fail": true, "fail-on-paths": true,
	"rename-left": true, "rename-right": true,
	"deduplicate": true,
}

func envStrategy(key string) string {
	v := os.Getenv(key)
	if v == "" {
		return ""
	}
	if !validJoinStrategies[v] {
		slog.Warn("invalid strategy env var, ignoring", "key", key, "value", v) //nolint:gosec // G706: values are structured log fields, not format strings
		return ""
	}
	return v
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		slog.Warn("invalid duration env var, using default", "key", key, "value", v, "default", fallback) //nolint:gosec // G706: values are structured log fields, not format strings
		return fallback
	}
	return d
}
