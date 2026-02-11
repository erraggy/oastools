package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
)

// validateInfoObject validates the info object fields shared between OAS 2.0 and 3.x.
// Set validateSPDX to true for OAS 3.1+ to validate the SPDX license identifier.
func (v *Validator) validateInfoObject(info *parser.Info, result *ValidationResult, baseURL string, validateSPDX bool) {
	if info.Title == "" {
		v.addError(result, "info.title", "Info object must have a title",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("title"),
		)
	}

	if info.Version == "" {
		v.addError(result, "info.version", "Info object must have a version",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("version"),
		)
	}

	// Validate contact information if present
	if info.Contact != nil {
		if info.Contact.URL != "" && !isValidURL(info.Contact.URL) {
			v.addError(result, "info.contact.url", fmt.Sprintf("Invalid URL format: %s", info.Contact.URL),
				withSpecRef(fmt.Sprintf("%s#contact-object", baseURL)),
				withField("url"),
				withValue(info.Contact.URL),
			)
		}
		if info.Contact.Email != "" && !isValidEmail(info.Contact.Email) {
			v.addError(result, "info.contact.email", fmt.Sprintf("Invalid email format: %s", info.Contact.Email),
				withSpecRef(fmt.Sprintf("%s#contact-object", baseURL)),
				withField("email"),
				withValue(info.Contact.Email),
			)
		}
	}

	// Validate license information if present
	if info.License != nil {
		if info.License.URL != "" && !isValidURL(info.License.URL) {
			v.addError(result, "info.license.url", fmt.Sprintf("Invalid URL format: %s", info.License.URL),
				withSpecRef(fmt.Sprintf("%s#license-object", baseURL)),
				withField("url"),
				withValue(info.License.URL),
			)
		}
		// SPDX license identifier validation (OAS 3.1+)
		if validateSPDX && info.License.Identifier != "" && !validateSPDXLicense(info.License.Identifier) {
			v.addError(result, "info.license.identifier", fmt.Sprintf("Invalid SPDX license identifier format: %s", info.License.Identifier),
				withSpecRef(fmt.Sprintf("%s#license-object", baseURL)),
				withField("identifier"),
				withValue(info.License.Identifier),
			)
		}
	}
}

// validateResponseStatusCodes validates HTTP status codes in an operation's responses.
// This helper is shared by both OAS 2.0 and OAS 3.x operation validators.
func (v *Validator) validateResponseStatusCodes(responses *parser.Responses, path string, result *ValidationResult, baseURL string) {
	if responses == nil || responses.Codes == nil {
		return
	}

	hasSuccess := false
	for code := range responses.Codes {
		// Validate HTTP status code format
		if !httputil.ValidateStatusCode(code) {
			v.addError(result, fmt.Sprintf("%s.responses.%s", path, code),
				fmt.Sprintf("Invalid HTTP status code: %s", code),
				withSpecRef(fmt.Sprintf("%s#responses-object", baseURL)),
				withValue(code),
			)
		} else if v.StrictMode && !httputil.IsStandardStatusCode(code) {
			// In strict mode, warn about non-standard status codes
			v.addWarning(result, fmt.Sprintf("%s.responses.%s", path, code),
				fmt.Sprintf("Non-standard HTTP status code: %s (not defined in HTTP RFCs)", code),
				withSpecRef(fmt.Sprintf("%s#responses-object", baseURL)),
				withValue(code),
			)
		}

		if strings.HasPrefix(code, "2") || code == "default" {
			hasSuccess = true
		}
	}
	if !hasSuccess && v.StrictMode {
		v.addWarning(result, fmt.Sprintf("%s.responses", path),
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

		opPath := fmt.Sprintf("%s.%s.%s", pathType, pathPattern, method)

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
