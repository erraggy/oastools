// This file implements server URL conversion between OAS 2.0 host/basePath/schemes
// and OAS 3.x servers array format.

package converter

import (
	"fmt"
	"net/url"

	"github.com/erraggy/oastools/internal/pathutil"
)

// parseServerURL extracts host, basePath, and schemes from an OAS 3.x server URL
// Returns host, basePath, schemes, and error
func parseServerURL(serverURL string) (host, basePath string, schemes []string, err error) {
	// Handle server variables by replacing them with defaults or placeholders like:
	// http://example.com/foo/{parameter}/bar ==> http://example.com/foo/placeholder/bar
	// For simplicity, we'll strip variables for now and parse the base URL since the rest of the path is ignored here
	cleanURL := pathutil.PathParamRegex.ReplaceAllString(serverURL, "placeholder")

	// Parse the URL
	u, err := url.Parse(cleanURL)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Extract components
	if u.Scheme != "" {
		schemes = []string{u.Scheme}
	}

	host = u.Host
	basePath = u.Path
	if basePath == "" {
		basePath = "/"
	}

	return host, basePath, schemes, nil
}
