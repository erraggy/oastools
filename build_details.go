package oastools

import (
	"fmt"
	"runtime"
)

var (
	// version is set via ldflags during build by GoReleaser
	// For development builds, this will show "dev"
	version = "dev"

	// commit is the git commit hash, set via ldflags during build
	// For development builds, this will show "unknown"
	commit = "unknown"

	// buildTime is the build timestamp in RFC3339 format, set via ldflags during build
	// For development builds, this will show "unknown"
	buildTime = "unknown"
)

// Version returns the compiled version or 'dev' if run from source
func Version() string {
	return version
}

// Commit returns the git commit hash or 'unknown' if run from source
func Commit() string {
	return commit
}

// BuildTime returns the build timestamp or 'unknown' if run from source
func BuildTime() string {
	return buildTime
}

// GoVersion returns the Go version used to build the binary
func GoVersion() string {
	return runtime.Version()
}

// UserAgent returns the User-Agent string to use
func UserAgent() string {
	return fmt.Sprintf("oastools/%s", version)
}

// BuildInfo returns a formatted string with all build metadata
func BuildInfo() string {
	return fmt.Sprintf("Version: %s\nCommit: %s\nBuild Time: %s\nGo Version: %s",
		Version(), Commit(), BuildTime(), GoVersion())
}
