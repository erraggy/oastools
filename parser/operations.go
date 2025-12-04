package parser

import "github.com/erraggy/oastools/internal/httputil"

// GetOperations extracts a map of all operations from a PathItem based on the OAS version.
// Returns a map with keys for HTTP methods and values pointing to the corresponding Operation (or nil if not defined).
// The returned map includes methods supported by the specified OAS version:
//   - OAS 2.0: get, put, post, delete, options, head, patch
//   - OAS 3.0-3.1: get, put, post, delete, options, head, patch, trace
//   - OAS 3.2+: get, put, post, delete, options, head, patch, trace, query
func GetOperations(pathItem *PathItem, version OASVersion) map[string]*Operation {
	ops := map[string]*Operation{
		httputil.MethodGet:     pathItem.Get,
		httputil.MethodPut:     pathItem.Put,
		httputil.MethodPost:    pathItem.Post,
		httputil.MethodDelete:  pathItem.Delete,
		httputil.MethodOptions: pathItem.Options,
		httputil.MethodHead:    pathItem.Head,
		httputil.MethodPatch:   pathItem.Patch,
	}

	// TRACE method is OAS 3.0+
	if version >= OASVersion300 {
		ops[httputil.MethodTrace] = pathItem.Trace
	}

	// QUERY method is OAS 3.2+
	if version >= OASVersion320 {
		ops[httputil.MethodQuery] = pathItem.Query
	}

	return ops
}

// GetOAS2Operations extracts a map of all operations from a PathItem.
// Returns a map with keys for HTTP methods (get, put, post, delete, options, head, patch)
// and values pointing to the corresponding Operation (or nil if not defined).
// This is used for OAS 2.0 paths that do not support the TRACE method.
//
// Deprecated: Use GetOperations(pathItem, OASVersion20) instead.
func GetOAS2Operations(pathItem *PathItem) map[string]*Operation {
	return GetOperations(pathItem, OASVersion20)
}

// GetOAS3Operations extracts a map of all operations from a PathItem.
// Returns a map with keys for HTTP methods (get, put, post, delete, options, head, patch, trace)
// and values pointing to the corresponding Operation (or nil if not defined).
// This is used for OAS 3.0 and 3.1 paths that support the TRACE method but not QUERY.
//
// Deprecated: Use GetOperations(pathItem, version) instead where version is OASVersion300 through OASVersion312.
func GetOAS3Operations(pathItem *PathItem) map[string]*Operation {
	return GetOperations(pathItem, OASVersion300)
}

// GetOAS32Operations extracts a map of all operations from a PathItem.
// Returns a map with keys for HTTP methods including QUERY.
// This is used for OAS 3.2+ paths that support the QUERY method.
//
// Deprecated: Use GetOperations(pathItem, OASVersion320) instead.
func GetOAS32Operations(pathItem *PathItem) map[string]*Operation {
	return GetOperations(pathItem, OASVersion320)
}
