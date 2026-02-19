package parser

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/erraggy/oastools"
)

// FormatBytes formats a byte count into a human-readable string using binary units (KiB, MiB, etc.)
func FormatBytes(size int64) string {
	// Handle negative values
	if size < 0 {
		return fmt.Sprintf("%d B", size)
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}

	// Use proper binary unit notation (KiB, MiB, GiB, etc.)
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

// detectFormatFromPath detects the source format from a file path
func detectFormatFromPath(path string) SourceFormat {
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return SourceFormatJSON
	case ".yaml", ".yml":
		return SourceFormatYAML
	default:
		return SourceFormatUnknown
	}
}

// detectFormatFromContent attempts to detect the format from the content bytes
// JSON typically starts with '{' or '[', while YAML does not
func detectFormatFromContent(data []byte) SourceFormat {
	// Trim leading whitespace
	trimmed := bytes.TrimLeft(data, " \t\n\r")

	if len(trimmed) == 0 {
		return SourceFormatUnknown
	}

	// JSON objects/arrays start with { or [
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return SourceFormatJSON
	}

	// Otherwise assume YAML (could be more sophisticated, but this covers most cases)
	return SourceFormatYAML
}

// isURL determines if the given path is a URL (http:// or https://)
func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// fetchURL fetches content from a URL and returns the bytes and Content-Type header
func (p *Parser) fetchURL(urlStr string) ([]byte, string, error) {
	// Create HTTP client with timeout
	// Use custom client if provided, otherwise create default
	var client *http.Client
	if p.HTTPClient != nil {
		client = p.HTTPClient
		if p.InsecureSkipVerify {
			p.log().Warn("InsecureSkipVerify ignored when HTTPClient provided; configure TLS on your client's transport")
		}
	} else if p.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested insecure mode
				MinVersion:         tls.VersionTLS12,
			},
		}
		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Create request
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("parser: failed to create request: %w", err)
	}

	// Set user agent (use default if not set)
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = oastools.UserAgent()
	}
	req.Header.Set("User-Agent", userAgent)

	// Execute request
	resp, err := client.Do(req) //nolint:gosec // G704 - URL is user-provided input (CLI parser)
	if err != nil {
		return nil, "", fmt.Errorf("parser: failed to fetch URL: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("parser: HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("parser: failed to read response body: %w", err)
	}

	// Return data and Content-Type header
	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// detectFormatFromURL attempts to detect the format from a URL path and Content-Type header
func detectFormatFromURL(urlStr string, contentType string) SourceFormat {
	// First try to detect from URL path extension
	parsedURL, err := url.Parse(urlStr)
	if err == nil && parsedURL.Path != "" {
		format := detectFormatFromPath(parsedURL.Path)
		if format != SourceFormatUnknown {
			return format
		}
	}

	// Try to detect from Content-Type header
	if contentType != "" {
		contentType = strings.ToLower(contentType)
		// Remove charset and other parameters
		if idx := strings.Index(contentType, ";"); idx != -1 {
			contentType = contentType[:idx]
		}
		contentType = strings.TrimSpace(contentType)

		switch contentType {
		case "application/json":
			return SourceFormatJSON
		case "application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml":
			return SourceFormatYAML
		}
	}

	return SourceFormatUnknown
}
