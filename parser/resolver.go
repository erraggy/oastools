package parser

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	// MaxRefDepth is the maximum depth allowed for nested $ref resolution
	// This prevents stack overflow from deeply nested (but non-circular) references
	MaxRefDepth = 100

	// MaxCachedDocuments is the maximum number of external documents to cache
	// This prevents memory exhaustion from documents with many external references
	MaxCachedDocuments = 100

	// MaxFileSize is the maximum size (in bytes) allowed for external reference files
	// This prevents resource exhaustion from loading arbitrarily large files
	// Set to 10MB which should be sufficient for most OpenAPI documents
	MaxFileSize = 10 * 1024 * 1024 // 10MB
)

// HTTPFetcher is a function type for fetching content from HTTP/HTTPS URLs
// Returns the response body, content-type header, and any error
type HTTPFetcher func(url string) ([]byte, string, error)

// RefResolver handles $ref resolution in OpenAPI documents
type RefResolver struct {
	// visited tracks visited refs to prevent circular reference loops
	visited map[string]bool
	// resolving tracks refs currently being resolved in the recursion stack
	resolving map[string]bool
	// documents caches loaded external documents
	documents map[string]map[string]any
	// baseDir is the base directory for resolving relative file paths
	baseDir string
	// baseURL is the base URL for resolving relative HTTP references
	// When set, relative refs in HTTP-loaded documents resolve against this URL
	baseURL string
	// httpFetch is the function used to fetch HTTP/HTTPS URLs
	// If nil, HTTP references will return an error
	httpFetch HTTPFetcher
	// hasCircularRefs is set to true when circular references are detected
	// This is used to skip re-marshaling which would cause infinite loops
	hasCircularRefs bool
}

// NewRefResolver creates a new reference resolver for local and file-based refs
func NewRefResolver(baseDir string) *RefResolver {
	return &RefResolver{
		visited:   make(map[string]bool),
		resolving: make(map[string]bool),
		documents: make(map[string]map[string]any),
		baseDir:   baseDir,
	}
}

// NewRefResolverWithHTTP creates a reference resolver with HTTP/HTTPS support
// The baseURL is used for resolving relative refs when the source is an HTTP URL
// The fetcher function is called to retrieve content from HTTP/HTTPS URLs
func NewRefResolverWithHTTP(baseDir, baseURL string, fetcher HTTPFetcher) *RefResolver {
	return &RefResolver{
		visited:   make(map[string]bool),
		resolving: make(map[string]bool),
		documents: make(map[string]map[string]any),
		baseDir:   baseDir,
		baseURL:   baseURL,
		httpFetch: fetcher,
	}
}

// ResolveLocal resolves local references within a document
// Local refs are in the format: #/path/to/component
func (r *RefResolver) ResolveLocal(doc map[string]any, ref string) (any, error) {
	// Check for circular references
	if r.visited[ref] {
		return nil, fmt.Errorf("circular reference detected: %s", ref)
	}
	r.visited[ref] = true
	defer func() { r.visited[ref] = false }()

	// Remove the leading # if present
	ref = strings.TrimPrefix(ref, "#")
	if ref == "" || ref == "/" {
		return doc, nil
	}

	// Split the reference path
	parts := strings.Split(strings.TrimPrefix(ref, "/"), "/")

	// Traverse the document
	current := any(doc)
	for i, part := range parts {
		// Unescape JSON Pointer tokens
		part = unescapeJSONPointer(part)

		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("reference not found: #/%s (missing key: %s)", strings.Join(parts[:i+1], "/"), part)
			}
			current = next

		case []any:
			// Handle array indexing (not common in OAS but valid JSON Pointer)
			return nil, fmt.Errorf("array indexing not supported in reference: #/%s", strings.Join(parts, "/"))

		default:
			return nil, fmt.Errorf("cannot traverse into type %T at #/%s", v, strings.Join(parts[:i], "/"))
		}
	}

	return current, nil
}

// ResolveExternal resolves external file references
// External refs are in the format: ./file.yaml#/path/to/component or file.yaml#/path/to/component
func (r *RefResolver) ResolveExternal(ref string) (any, error) {
	// Check for circular references
	if r.visited[ref] {
		return nil, fmt.Errorf("circular reference detected: %s", ref)
	}
	r.visited[ref] = true
	defer func() { r.visited[ref] = false }()

	// Split the reference into file path and internal path
	parts := strings.SplitN(ref, "#", 2)
	filePath := parts[0]
	internalPath := ""
	if len(parts) > 1 {
		internalPath = parts[1]
	}

	// Resolve the file path relative to baseDir
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Clean(filepath.Join(r.baseDir, filePath))
	}

	// Ensure the resolved path is within allowed directory
	absBase, err := filepath.Abs(r.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base directory: %w", err)
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Use filepath.Rel to detect path traversal attempts
	// This properly handles all cases including different volumes on Windows
	// (filepath.Rel returns an error when paths are on different drives)
	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", ref)
	}

	// Check if document is already loaded
	doc, ok := r.documents[filePath]
	if !ok {
		// Enforce cache size limit to prevent memory exhaustion
		if len(r.documents) >= MaxCachedDocuments {
			return nil, fmt.Errorf("exceeded maximum cached documents limit (%d): too many external references", MaxCachedDocuments)
		}

		// Check file size to prevent resource exhaustion
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat external file %s: %w", filePath, err)
		}
		if fileInfo.Size() > MaxFileSize {
			return nil, fmt.Errorf("external file %s exceeds maximum size limit (%d bytes): file is %d bytes",
				filePath, MaxFileSize, fileInfo.Size())
		}

		// Load the external document
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read external file %s: %w", filePath, err)
		}

		// Parse the external document
		var extDoc map[string]any
		if err := yaml.Unmarshal(data, &extDoc); err != nil {
			return nil, fmt.Errorf("failed to parse external file %s: %w", filePath, err)
		}

		r.documents[filePath] = extDoc
		doc = extDoc
	}

	// If there's no internal path, return the whole document
	if internalPath == "" {
		return doc, nil
	}

	// Resolve the internal reference
	return r.ResolveLocal(doc, "#"+internalPath)
}

// ResolveHTTP resolves HTTP/HTTPS URL references
// HTTP refs are in the format: https://example.com/api.yaml#/components/schemas/Pet
func (r *RefResolver) ResolveHTTP(ref string) (any, error) {
	// Check for circular references
	if r.visited[ref] {
		return nil, fmt.Errorf("circular reference detected: %s", ref)
	}
	r.visited[ref] = true
	defer func() { r.visited[ref] = false }()

	// Split the reference into URL and fragment (internal path)
	parts := strings.SplitN(ref, "#", 2)
	urlStr := parts[0]
	internalPath := ""
	if len(parts) > 1 {
		internalPath = parts[1]
	}

	// Check if document is already cached
	doc, ok := r.documents[urlStr]
	if !ok {
		// Enforce cache size limit to prevent memory exhaustion
		if len(r.documents) >= MaxCachedDocuments {
			return nil, fmt.Errorf("exceeded maximum cached documents limit (%d): too many external references", MaxCachedDocuments)
		}

		// Fetch the URL content
		data, _, err := r.httpFetch(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch HTTP reference %s: %w", urlStr, err)
		}

		// Enforce file size limit
		if int64(len(data)) > MaxFileSize {
			return nil, fmt.Errorf("HTTP response from %s exceeds maximum size limit (%d bytes): response is %d bytes",
				urlStr, MaxFileSize, len(data))
		}

		// Parse the document (YAML parser handles both YAML and JSON)
		var extDoc map[string]any
		if err := yaml.Unmarshal(data, &extDoc); err != nil {
			return nil, fmt.Errorf("failed to parse HTTP reference %s: %w", urlStr, err)
		}

		r.documents[urlStr] = extDoc
		doc = extDoc
	}

	// If there's no internal path, return the whole document
	if internalPath == "" {
		return doc, nil
	}

	// Resolve the internal reference
	return r.ResolveLocal(doc, "#"+internalPath)
}

// Resolve resolves a $ref reference (local, file, or HTTP)
func (r *RefResolver) Resolve(doc map[string]any, ref string) (any, error) {
	// Check if it's a local reference (starts with #)
	if strings.HasPrefix(ref, "#") {
		return r.ResolveLocal(doc, ref)
	}

	// Check if it's an HTTP(S) URL
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		if r.httpFetch == nil {
			return nil, fmt.Errorf("HTTP references require HTTP fetcher to be configured: %s", ref)
		}
		return r.ResolveHTTP(ref)
	}

	// Check if we have a base URL and this is a relative reference
	// (not starting with # and not an absolute URL)
	if r.baseURL != "" {
		// Resolve relative path against base URL
		resolved, err := r.resolveRelativeURL(ref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve relative URL %s: %w", ref, err)
		}
		return r.ResolveHTTP(resolved)
	}

	// Otherwise, treat it as an external file reference
	return r.ResolveExternal(ref)
}

// resolveRelativeURL resolves a relative reference against the baseURL
func (r *RefResolver) resolveRelativeURL(ref string) (string, error) {
	// Split ref into path and fragment
	parts := strings.SplitN(ref, "#", 2)
	relPath := parts[0]
	fragment := ""
	if len(parts) > 1 {
		fragment = "#" + parts[1]
	}

	// Parse the base URL
	base, err := url.Parse(r.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Resolve the relative path against the base URL's directory
	// Use path.Dir to get the directory of the base URL path
	base.Path = path.Join(path.Dir(base.Path), relPath)

	return base.String() + fragment, nil
}

// unescapeJSONPointer unescapes JSON Pointer tokens
// Per RFC 6901, ~1 represents / and ~0 represents ~
func unescapeJSONPointer(token string) string {
	token = strings.ReplaceAll(token, "~1", "/")
	token = strings.ReplaceAll(token, "~0", "~")
	return token
}

// ResolveAllRefs walks through the entire document and resolves all $ref references
func (r *RefResolver) ResolveAllRefs(doc map[string]any) error {
	// Reset circular ref flag for each resolution pass
	r.hasCircularRefs = false
	return r.resolveRefsRecursive(doc, doc, 0)
}

// resolveRefsRecursive recursively walks through the document structure and resolves $ref
func (r *RefResolver) resolveRefsRecursive(root, current any, depth int) error {
	// Prevent stack overflow from deeply nested structures
	if depth > MaxRefDepth {
		return fmt.Errorf("exceeded maximum reference depth (%d): structure too deeply nested", MaxRefDepth)
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid root type: expected map[string]any, got %T", root)
	}
	switch v := current.(type) {
	case map[string]any:
		// Check if this object has a $ref field
		if ref, ok := v["$ref"].(string); ok {
			// Check for $ref pointing to document root (always circular)
			if ref == "#" || ref == "#/" {
				// Leave the $ref in place - resolving it would create infinite recursion
				r.hasCircularRefs = true
				return nil
			}

			// Check if we're already resolving this reference (circular dependency)
			if r.resolving[ref] {
				// Leave the $ref in place rather than trying to expand it infinitely
				// This allows circular references to exist in the document
				r.hasCircularRefs = true
				return nil
			}

			// Mark this ref as being resolved. The defer cleanup runs when
			// resolveRefsRecursive returns (either from line 263 or line 269's error),
			// which correctly maintains the resolving state across recursive calls.
			r.resolving[ref] = true
			defer func() { delete(r.resolving, ref) }()

			// Resolve the reference
			resolved, err := r.Resolve(rootMap, ref)
			if err != nil {
				return fmt.Errorf("failed to resolve $ref %s: %w", ref, err)
			}

			// Replace the current object with the resolved content
			// Note: This modifies the map in place
			for k := range v {
				if k != "$ref" {
					delete(v, k)
				}
			}
			resolvedMap, ok := resolved.(map[string]any)
			if !ok {
				return fmt.Errorf("resolved $ref %s is not an object (got %T)", ref, resolved)
			}
			for k, val := range resolvedMap {
				v[k] = val
			}
			delete(v, "$ref")

			// Continue resolving in the newly resolved content
			return r.resolveRefsRecursive(root, v, depth+1)
		}

		// Recursively process all values in the map
		for _, val := range v {
			if err := r.resolveRefsRecursive(root, val, depth+1); err != nil {
				return err
			}
		}

	case []any:
		// Recursively process all items in the array
		for _, item := range v {
			if err := r.resolveRefsRecursive(root, item, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// HasCircularRefs returns true if circular references were detected during resolution.
// When true, the resolved data contains actual Go circular references and cannot be
// safely serialized with yaml.Marshal (which would cause an infinite loop).
func (r *RefResolver) HasCircularRefs() bool {
	return r.hasCircularRefs
}
