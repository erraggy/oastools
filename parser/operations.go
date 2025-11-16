package parser

import "github.com/erraggy/oastools/internal/httputil"

// GetOAS2Operations extracts a map of all operations from a PathItem.
// Returns a map with keys for HTTP methods (get, put, post, delete, options, head, patch)
// and values pointing to the corresponding Operation (or nil if not defined).
// This is used for OAS 2.0 paths that do not support the TRACE method.
func GetOAS2Operations(pathItem *PathItem) map[string]*Operation {
	return map[string]*Operation{
		httputil.MethodGet:     pathItem.Get,
		httputil.MethodPut:     pathItem.Put,
		httputil.MethodPost:    pathItem.Post,
		httputil.MethodDelete:  pathItem.Delete,
		httputil.MethodOptions: pathItem.Options,
		httputil.MethodHead:    pathItem.Head,
		httputil.MethodPatch:   pathItem.Patch,
	}
}

// GetOAS3Operations extracts a map of all operations from a PathItem.
// Returns a map with keys for HTTP methods (get, put, post, delete, options, head, patch, trace)
// and values pointing to the corresponding Operation (or nil if not defined).
// This is used for OAS 3.x paths that support the TRACE method.
func GetOAS3Operations(pathItem *PathItem) map[string]*Operation {
	return map[string]*Operation{
		httputil.MethodGet:     pathItem.Get,
		httputil.MethodPut:     pathItem.Put,
		httputil.MethodPost:    pathItem.Post,
		httputil.MethodDelete:  pathItem.Delete,
		httputil.MethodOptions: pathItem.Options,
		httputil.MethodHead:    pathItem.Head,
		httputil.MethodPatch:   pathItem.Patch,
		httputil.MethodTrace:   pathItem.Trace,
	}
}
