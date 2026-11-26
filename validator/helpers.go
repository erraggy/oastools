package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/internal/inforules"
	"github.com/erraggy/oastools/parser"
)

// validateInfoObject reports the internal/inforules problems for info, which
// the builder checks too.
func (v *Validator) validateInfoObject(info *parser.Info, version parser.OASVersion, result *ValidationResult, baseURL string) {
	for _, p := range inforules.Check(info, version) {
		opts := []func(*ValidationError){
			withSpecRef(fmt.Sprintf("%s#%s", baseURL, p.Object)),
			withField(p.Field),
		}
		if p.Value != "" {
			opts = append(opts, withValue(p.Value))
		}
		v.addError(result, p.Path, p.Message, opts...)
	}
}

// wildcardResponseRangesPermitted reports whether the document version defines
// wildcard response ranges (1XX through 5XX) as Responses Object keys.
//
// OAS 3.0 introduced them. The 2.0 Responses Object admits one property per HTTP
// status code and says nothing about ranges, so a 2.0 document writing "2XX"
// names a key its own version does not define. An unrecognized version permits
// them, matching the other version gates.
func wildcardResponseRangesPermitted(version parser.OASVersion) bool {
	return !version.IsValid() || version >= parser.OASVersion300
}

// statusCodeKeyProblem returns the diagnostic for a Responses Object key that is
// no status code, or "" when the key is one. Pass the answer of
// [wildcardResponseRangesPermitted] as allowWildcards.
//
// A numeric code is legal in every version, so only the wildcard range needs the
// version, and it gets its own message: "2XX" is a key most tooling accepts, and
// naming the version that introduced it says more than calling it malformed.
func statusCodeKeyProblem(code string, allowWildcards bool) string {
	switch {
	case httputil.IsNumericStatusCode(code):
		return ""
	case httputil.IsWildcardStatusCode(code):
		if allowWildcards {
			return ""
		}
		return fmt.Sprintf("Invalid HTTP status code: %s (wildcard ranges were introduced in OAS 3.0; OAS 2.0 defines one property per HTTP status code)", code)
	default:
		return fmt.Sprintf("Invalid HTTP status code: %s", code)
	}
}

// validateResponseStatusCodes validates HTTP status codes in an operation's responses.
// This helper is shared by both OAS 2.0 and OAS 3.x operation validators, so it
// reads the version off the Validator rather than taking it as an argument.
func (v *Validator) validateResponseStatusCodes(responses *parser.Responses, path string, result *ValidationResult, baseURL string) {
	if responses == nil {
		return
	}

	// A document declaring only `default` covers every code, so it satisfies
	// the success check below. The decoders route that key to Responses.Default
	// rather than into Codes, so it is not observable from the loop.
	hasSuccess := responses.Default != nil

	allowWildcards := wildcardResponseRangesPermitted(v.oasVersion)

	for code := range responses.Codes {
		// Validate the HTTP status code format, which the document's version
		// scopes: a numeric code is legal everywhere, a wildcard only from 3.0.
		if problem := statusCodeKeyProblem(code, allowWildcards); problem != "" {
			v.addError(result, path+".responses."+code, problem,
				withSpecRef(fmt.Sprintf("%s#responses-object", baseURL)),
				withValue(code),
			)
			// A key already reported as malformed says nothing about which
			// responses the operation defines, so it cannot answer the success
			// question below either.
			continue
		}

		// In strict mode, warn about non-standard status codes. The question is
		// whether the code is one the HTTP RFCs register, which only a numeric
		// code can be: a wildcard range names a class of codes, so the registry
		// has nothing to say about it and a permitted range is not irregular.
		if v.StrictMode && httputil.IsNumericStatusCode(code) && !httputil.IsStandardStatusCode(code) {
			v.addWarning(result, path+".responses."+code,
				fmt.Sprintf("Non-standard HTTP status code: %s (not defined in HTTP RFCs)", code),
				withSpecRef(fmt.Sprintf("%s#responses-object", baseURL)),
				withValue(code),
			)
		}

		if httputil.IsSuccessStatusCode(code) {
			hasSuccess = true
		}
	}
	if !hasSuccess && v.StrictMode {
		v.addWarning(result, path+".responses",
			"Operation should define at least one successful response (2XX or default)",
			withSpecRef(fmt.Sprintf("%s#responses-object", baseURL)),
		)
	}
}

// checkDuplicateOperationIds checks for duplicate operationIds in a set of operations
// and reports errors when found. Updates the operationIds map as it processes operations.
func (v *Validator) checkDuplicateOperationIds(
	operations map[string]*parser.Operation,
	pathType string,
	pathPattern string,
	operationIds map[string]string,
	result *ValidationResult,
	baseURL string,
) {
	for method, op := range operations {
		if op == nil || op.OperationID == "" {
			continue
		}

		opPath := pathType + "." + pathPattern + "." + method

		if firstSeenAt, exists := operationIds[op.OperationID]; exists {
			// Determine the correct spec reference based on path type
			specRef := fmt.Sprintf("%s#operation-object", baseURL)
			if pathType == "webhooks" || strings.Contains(baseURL, "v3") {
				specRef = fmt.Sprintf("%s#operation-object", baseURL)
			}

			v.addError(result, opPath,
				fmt.Sprintf("Duplicate operationId '%s' (first seen at %s)", op.OperationID, firstSeenAt),
				withSpecRef(specRef),
				withField("operationId"),
				withValue(op.OperationID),
			)
		} else {
			operationIds[op.OperationID] = opPath
		}
	}
}
