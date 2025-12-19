package parser

import "github.com/erraggy/oastools/internal/httputil"

// GetOperations extracts a map of all operations from a PathItem based on the OAS version.
// Returns a map with keys for HTTP methods and values pointing to the corresponding Operation (or nil if not defined).
// The returned map includes methods supported by the specified OAS version:
//   - OAS 2.0: get, put, post, delete, options, head, patch
//   - OAS 3.0-3.1: get, put, post, delete, options, head, patch, trace
//   - OAS 3.2+: get, put, post, delete, options, head, patch, trace, query, plus any additionalOperations
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

	// QUERY method and additionalOperations are OAS 3.2+
	if version >= OASVersion320 {
		ops[httputil.MethodQuery] = pathItem.Query

		// Include any custom methods from additionalOperations
		for method, op := range pathItem.AdditionalOperations {
			ops[method] = op
		}
	}

	return ops
}
