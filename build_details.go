package oastools

import "fmt"

var (
	// version is set via ldflags during build by GoReleaser
	// For development builds, this will show "dev"
	version = "dev"
)

// Version returns the compiled version or 'dev' if run from source
func Version() string {
	return version
}

// UserAgent returns the User-Agent string to use
func UserAgent() string {
	return fmt.Sprintf("oastools/%s", version)
}
