package validator

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/paramutil"
	"github.com/erraggy/oastools/parser"
)

// validateOAS3 performs OAS 3.x specific validation
func (v *Validator) validateOAS3(doc *parser.OAS3Document, result *ValidationResult) {
	version := doc.OpenAPI
	var baseURL string

	// Determine the correct spec URL based on version
	switch doc.OASVersion {
	case parser.OASVersion300:
		baseURL = "https://spec.openapis.org/oas/v3.0.0.html"
	case parser.OASVersion301:
		baseURL = "https://spec.openapis.org/oas/v3.0.1.html"
	case parser.OASVersion302:
		baseURL = "https://spec.openapis.org/oas/v3.0.2.html"
	case parser.OASVersion303:
		baseURL = "https://spec.openapis.org/oas/v3.0.3.html"
	case parser.OASVersion304:
		baseURL = "https://spec.openapis.org/oas/v3.0.4.html"
	case parser.OASVersion310:
		baseURL = "https://spec.openapis.org/oas/v3.1.0.html"
	case parser.OASVersion311:
		baseURL = "https://spec.openapis.org/oas/v3.1.1.html"
	case parser.OASVersion312:
		baseURL = "https://spec.openapis.org/oas/v3.1.2.html"
	case parser.OASVersion320:
		baseURL = "https://spec.openapis.org/oas/v3.2.0.html"
	default:
		baseURL = fmt.Sprintf("https://spec.openapis.org/oas/v%s.html", version)
	}

	// Validate required fields in info object
	v.validateOAS3Info(doc, result, baseURL)

	// Validate servers
	v.validateOAS3Servers(doc, result, baseURL)

	// Validate paths and operations
	v.validateOAS3Paths(doc, result, baseURL)

	// Validate components
	v.validateOAS3Components(doc, result, baseURL)

	// Validate webhooks (OAS 3.1+)
	v.validateOAS3Webhooks(doc, result, baseURL)

	// Validate reusable path items (OAS 3.1+)
	v.validateOAS3ComponentPathItems(doc, result, baseURL)

	// Validate path parameters match path templates
	v.validateOAS3PathParameterConsistency(doc, result, baseURL)

	// Validate security requirements reference existing schemes
	v.validateOAS3SecurityRequirements(doc, result, baseURL)

	// Validate duplicate operationIds
	v.validateOAS3OperationIds(doc, result, baseURL)

	// Validate the OAS 3.2 cross-field constraints (see oas32.go)
	v.validateOAS32Document(doc, result)
	v.validateOAS32FieldsNotYetIntroduced(doc, result)

	// Validate all $ref values point to valid components
	v.validateOAS3Refs(doc, result, baseURL)
}

// validateOAS3Info validates the info object in OAS 3.x
func (v *Validator) validateOAS3Info(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Info == nil {
		v.addError(result, "info", "Document must have an info object",
			withSpecRef(fmt.Sprintf("%s#info-object", baseURL)),
			withField("info"),
		)
		return
	}
	v.validateInfoObject(doc.Info, result, baseURL, true)
}

// validateOAS3OperationIds validates that operationIds are unique across the document
func (v *Validator) validateOAS3OperationIds(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	operationIds := make(map[string]string) // map of operationId -> path where first seen

	// Check paths
	if doc.Paths != nil {
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, doc.OASVersion)

			v.checkDuplicateOperationIds(operations, "paths", pathPattern, operationIds, result, baseURL)
		}
	}

	// Check webhooks (OAS 3.1+)
	for webhookName, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, doc.OASVersion)

		v.checkDuplicateOperationIds(operations, "webhooks", webhookName, operationIds, result, baseURL)
	}

	// Check reusable path items (OAS 3.1+).
	//
	// `operationId` must be unique among all operations described in the API:
	// https://spec.openapis.org/oas/v3.2.0.html#operation-id
	//
	// Applying that to `components.pathItems` is a decision, so it is recorded
	// here: each entry counts exactly once, however many places `$ref` it, and
	// whether or not anything does.
	//
	// Once rather than per use site, because a path item `$ref` is preserved
	// verbatim rather than resolved in place. A use site carries no operations of
	// its own, so it cannot double-count against the declaration, and counting
	// resolved sites instead would make a twice-referenced path item collide with
	// itself.
	//
	// An unreferenced entry counts too. It describes nothing today, but an
	// `operationId` duplicating a live one is one `$ref` away from mattering.
	//
	// Swept last so a collision with `paths` or `webhooks` is reported at the
	// components site, and in sorted name order so the same one of a colliding
	// pair is named on every run.
	if doc.Components != nil {
		for _, name := range slices.Sorted(maps.Keys(doc.Components.PathItems)) {
			pathItem := doc.Components.PathItems[name]
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, doc.OASVersion)

			v.checkDuplicateOperationIds(operations, "components.pathItems", name, operationIds, result, baseURL)
		}
	}
}

// validateOAS3Servers validates server objects in OAS 3.x
func (v *Validator) validateOAS3Servers(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	for i, server := range doc.Servers {
		path := "servers[" + strconv.Itoa(i) + "]"

		if server.URL == "" {
			v.addError(result, path, "Server must have a url",
				withSpecRef(fmt.Sprintf("%s#server-object", baseURL)),
				withField("url"),
			)
		}

		// Validate server variables
		for varName, varObj := range server.Variables {
			varPath := path + ".variables." + varName

			if varObj.Default == "" {
				v.addError(result, varPath, "Server variable must have a default value",
					withSpecRef(fmt.Sprintf("%s#server-variable-object", baseURL)),
					withField("default"),
				)
			}

			// If enum is specified, default must be in enum
			if len(varObj.Enum) > 0 && !slices.Contains(varObj.Enum, varObj.Default) {
				v.addError(result, varPath,
					fmt.Sprintf("Server variable default value '%s' must be one of the enum values", varObj.Default),
					withSpecRef(fmt.Sprintf("%s#server-variable-object", baseURL)),
					withField("default"),
					withValue(varObj.Default),
				)
			}
		}
	}
}

// validateOAS3Paths validates paths in OAS 3.x
func (v *Validator) validateOAS3Paths(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Paths == nil {
		return
	}

	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		pathPrefix := "paths." + pathPattern

		// Validate path pattern starts with "/"
		if !strings.HasPrefix(pathPattern, "/") {
			v.addError(result, pathPrefix, "Path must start with '/'",
				withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
				withValue(pathPattern),
			)
		}

		// Validate path template is well-formed
		if err := validatePathTemplate(pathPattern); err != nil {
			v.addError(result, pathPrefix, fmt.Sprintf("Invalid path template: %s", err),
				withSpecRef(fmt.Sprintf("%s#paths-object", baseURL)),
				withValue(pathPattern),
			)
		}

		// Warning: trailing slash in path (REST best practice)
		checkTrailingSlash(v, pathPattern, result, baseURL)

		// Validate QUERY method is only used in OAS 3.2+
		if pathItem.Query != nil && doc.OASVersion < parser.OASVersion320 {
			v.addError(result, pathPrefix+".query",
				fmt.Sprintf("QUERY method is only supported in OAS 3.2+, but document is version %s", doc.OASVersion),
				withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
				withField("query"),
			)
		}

		// Validate each operation
		operations := parser.GetOperations(pathItem, doc.OASVersion)

		// The OAS 3.2 traversal rules ride along on the operations map this pass
		// already built, rather than taking a traversal of their own (see oas32.go).
		v.validateOAS32PathItem(pathItem, pathPrefix, doc.OASVersion, operations, result)

		// A path item carries parameters of its own, shared by every operation in it.
		v.validateParameterListSchemas(pathItem.Parameters, pathPrefix, result)

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := pathPrefix + "." + method
			v.validateOAS3Operation(op, opPath, result, baseURL)

			// Warning: recommend description
			if v.IncludeWarnings && op.Description == "" && op.Summary == "" {
				v.addWarning(result, opPath, "Operation should have a description or summary for better documentation",
					withSpecRef(fmt.Sprintf("%s#operation-object", baseURL)),
					withField("description"),
				)
			}
		}
	}
}

// validateOAS3Operation validates an operation in OAS 3.x
func (v *Validator) validateOAS3Operation(op *parser.Operation, path string, result *ValidationResult, baseURL string) {
	// Validate request body if present
	if op.RequestBody != nil {
		v.validateOAS3RequestBody(op.RequestBody, path+".requestBody", result, baseURL)
	}

	// Validate response status codes
	v.validateResponseStatusCodes(op.Responses, path, result, baseURL)

	// Every schema this operation carries beyond the request body, which
	// validateOAS3RequestBody already reaches. See schema_traversal.go.
	v.validateOAS3OperationSchemas(op, path, result)
}

// validateOAS3RequestBody validates a request body in OAS 3.x
func (v *Validator) validateOAS3RequestBody(requestBody *parser.RequestBody, path string, result *ValidationResult, baseURL string) {
	if requestBody == nil {
		return
	}

	// Skip validation if this is a $ref
	if requestBody.Ref != "" {
		return
	}

	// RequestBody must have content
	if len(requestBody.Content) == 0 {
		v.addError(result, path, "RequestBody must have a content object with at least one media type",
			withSpecRef(fmt.Sprintf("%s#request-body-object", baseURL)),
			withField("content"),
		)
		return
	}

	// Validate each media type
	for mediaType, mediaTypeObj := range requestBody.Content {
		mediaTypePath := path + ".content." + mediaType

		// Validate media type format
		if !isValidMediaType(mediaType) {
			v.addError(result, mediaTypePath, fmt.Sprintf("Invalid media type: %s", mediaType),
				withSpecRef(fmt.Sprintf("%s#request-body-object", baseURL)),
				withValue(mediaType),
			)
		}

		// Both the schema and the 3.2 itemSchema beside it, so a request body is
		// covered the same way every other media type position is. See
		// schema_traversal.go.
		v.validateMediaTypeSchemas(mediaTypeObj, mediaTypePath, result)
	}
}

// validateOAS3Components validates components in OAS 3.x
func (v *Validator) validateOAS3Components(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Components == nil {
		return
	}

	// Checked across every section first, so a nil value cannot cause its name
	// to go unchecked by the loops below, several of which skip nil entries.
	v.validateOAS3ComponentNames(doc.Components, result, baseURL)

	// The schemas held by the sections other than `schemas`. See
	// schema_traversal.go.
	v.validateOAS3ComponentSchemas(doc.Components, result)

	// Validate schemas
	for name, schema := range doc.Components.Schemas {
		v.validateSchemaName(name, "components.schemas", result)
		if schema == nil {
			continue
		}
		v.validateSchema(schema, "components.schemas."+name, result)
	}

	// Validate responses
	for name, response := range doc.Components.Responses {
		if response == nil {
			continue
		}

		if response.Description == "" {
			v.addError(result, "components.responses."+name, "Response must have a description",
				withSpecRef(fmt.Sprintf("%s#response-object", baseURL)),
				withField("description"),
			)
		}
	}

	// Validate request bodies
	for name, requestBody := range doc.Components.RequestBodies {
		if requestBody == nil {
			continue
		}
		v.validateOAS3RequestBody(requestBody, "components.requestBodies."+name, result, baseURL)
	}

	// Validate parameters
	for name, param := range doc.Components.Parameters {
		if param == nil {
			continue
		}
		// A pure $ref alias to another parameter definition is valid OAS, and
		// arrives from the parser with every sibling field empty because
		// references are preserved verbatim for lossless round-trips. Checking
		// the fields below would read those empties as missing content. The
		// referenced definition is validated in its own right, so nothing is
		// lost. Mirrors validateOAS3RequestBody, which returns early for the
		// same reason.
		if param.Ref != "" {
			continue
		}

		path := "components.parameters." + name

		// Parameters must have either schema or content (but not both)
		hasSchema := param.Schema != nil
		hasContent := len(param.Content) > 0

		if !hasSchema && !hasContent {
			v.addError(result, path, "Parameter must have either a schema or content",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			)
		}

		if hasSchema && hasContent {
			v.addError(result, path, "Parameter must not have both schema and content",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
			)
		}

		// Path parameters must have required: true
		if paramutil.NeedsRequiredTrue(param) {
			v.addError(result, path, "Path parameters must have required: true",
				withSpecRef(fmt.Sprintf("%s#parameter-object", baseURL)),
				withField("required"),
			)
		}
	}

	// Validate security schemes
	for name, secScheme := range doc.Components.SecuritySchemes {
		if secScheme == nil {
			continue
		}
		v.validateOAS3SecurityScheme(secScheme, "components.securitySchemes."+name, result, baseURL)
	}
}

// validateOAS3SecurityScheme validates a security scheme in OAS 3.x
func (v *Validator) validateOAS3SecurityScheme(scheme *parser.SecurityScheme, path string, result *ValidationResult, baseURL string) {
	if scheme.Type == "" {
		v.addError(result, path, "Security scheme must have a type",
			withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
			withField("type"),
		)
		return
	}

	switch scheme.Type {
	case securitySchemeTypeAPIKey:
		if scheme.Name == "" {
			v.addError(result, path, "API key security scheme must have a name",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("name"),
			)
		}
		if scheme.In == "" {
			v.addError(result, path, "API key security scheme must specify 'in' (query, header, or cookie)",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("in"),
			)
		}
	case "http":
		if scheme.Scheme == "" {
			v.addError(result, path, "HTTP security scheme must have a scheme (e.g., 'basic', 'bearer')",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("scheme"),
			)
		}
	case securitySchemeTypeOAuth2:
		if scheme.Flows == nil {
			v.addError(result, path, "OAuth2 security scheme must have flows",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("flows"),
			)
		} else {
			v.validateOAuth2Flows(scheme.Flows, path, result, baseURL)
		}
	case "openIdConnect":
		if scheme.OpenIDConnectURL == "" {
			v.addError(result, path, "OpenID Connect security scheme must have openIdConnectUrl",
				withSpecRef(fmt.Sprintf("%s#security-scheme-object", baseURL)),
				withField("openIdConnectUrl"),
			)
		}
	}
}

// validateOAuth2Flows validates OAuth2 flows in OAS 3.x
func (v *Validator) validateOAuth2Flows(flows *parser.OAuthFlows, path string, result *ValidationResult, baseURL string) {
	// Validate implicit flow
	if flows.Implicit != nil {
		flowPath := path + ".flows.implicit"
		if flows.Implicit.AuthorizationURL == "" {
			v.addError(result, flowPath, "Implicit flow must have authorizationUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
			)
		} else if !isValidURL(flows.Implicit.AuthorizationURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for authorizationUrl: %s", flows.Implicit.AuthorizationURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
				withValue(flows.Implicit.AuthorizationURL),
			)
		}
	}

	// Validate password flow
	if flows.Password != nil {
		flowPath := path + ".flows.password"
		if flows.Password.TokenURL == "" {
			v.addError(result, flowPath, "Password flow must have tokenUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
			)
		} else if !isValidURL(flows.Password.TokenURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.Password.TokenURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
				withValue(flows.Password.TokenURL),
			)
		}
	}

	// Validate clientCredentials flow
	if flows.ClientCredentials != nil {
		flowPath := path + ".flows.clientCredentials"
		if flows.ClientCredentials.TokenURL == "" {
			v.addError(result, flowPath, "Client credentials flow must have tokenUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
			)
		} else if !isValidURL(flows.ClientCredentials.TokenURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.ClientCredentials.TokenURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
				withValue(flows.ClientCredentials.TokenURL),
			)
		}
	}

	// Validate authorizationCode flow
	if flows.AuthorizationCode != nil {
		flowPath := path + ".flows.authorizationCode"
		if flows.AuthorizationCode.AuthorizationURL == "" {
			v.addError(result, flowPath, "Authorization code flow must have authorizationUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
			)
		} else if !isValidURL(flows.AuthorizationCode.AuthorizationURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for authorizationUrl: %s", flows.AuthorizationCode.AuthorizationURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("authorizationUrl"),
				withValue(flows.AuthorizationCode.AuthorizationURL),
			)
		}
		if flows.AuthorizationCode.TokenURL == "" {
			v.addError(result, flowPath, "Authorization code flow must have tokenUrl",
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
			)
		} else if !isValidURL(flows.AuthorizationCode.TokenURL) {
			v.addError(result, flowPath,
				fmt.Sprintf("Invalid URL format for tokenUrl: %s", flows.AuthorizationCode.TokenURL),
				withSpecRef(fmt.Sprintf("%s#oauth-flows-object", baseURL)),
				withField("tokenUrl"),
				withValue(flows.AuthorizationCode.TokenURL),
			)
		}
	}
}

// validateOAS3Webhooks validates webhooks in OAS 3.1+
func (v *Validator) validateOAS3Webhooks(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if len(doc.Webhooks) == 0 {
		return
	}

	for webhookName, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}

		pathPrefix := "webhooks." + webhookName

		// Validate each operation in the webhook
		operations := parser.GetOperations(pathItem, doc.OASVersion)

		v.validateOAS32PathItem(pathItem, pathPrefix, doc.OASVersion, operations, result)

		// A path item carries parameters of its own, shared by every operation in it.
		v.validateParameterListSchemas(pathItem.Parameters, pathPrefix, result)

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := pathPrefix + "." + method
			v.validateOAS3Operation(op, opPath, result, baseURL)
		}
	}
}

// validateOAS3ComponentPathItems validates the reusable path items under
// components.pathItems (OAS 3.1+).
//
// A reusable path item is reached by $ref from webhooks, from paths, or from
// another path item, and describes operations as genuine as any other, so its
// operations get the same operation-level validation as those under paths and
// webhooks. Before this ran, only the component *names* were checked
// (checkComponentNames) and nothing inside them was: a request body with no
// content, or a malformed response status code, went unreported.
//
// Deliberately excluded: the path-template consistency checks
// (reportUndeclaredPathParams and warnUnusedPathParams). The spec scopes the
// `name` rule for `in: "path"` to a template expression in the Paths Object:
// https://spec.openapis.org/oas/v3.2.0.html#parameter-name
//
// A reusable path item has no `path` of its own, so running those checks here
// would warn about every well-formed path parameter. Webhooks are excluded for
// the same reason.
//
// [Validator.validatePathParamsRequired] is not a template check and does apply:
// `required: true` is a property of the parameter alone.
func (v *Validator) validateOAS3ComponentPathItems(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Components == nil || len(doc.Components.PathItems) == 0 {
		return
	}

	for name, pathItem := range doc.Components.PathItems {
		if pathItem == nil {
			continue
		}

		prefix := "components.pathItems." + name

		// Path-item parameters apply to every operation in the item, so they are
		// checked once here rather than once per operation — the same requirement
		// validatePathParamsRequired documents for its callers.
		v.validatePathParamsRequired(pathItem.Parameters, prefix, result, baseURL)

		operations := parser.GetOperations(pathItem, doc.OASVersion)

		v.validateOAS32PathItem(pathItem, prefix, doc.OASVersion, operations, result)

		// A path item carries parameters of its own, shared by every operation in it.
		v.validateParameterListSchemas(pathItem.Parameters, prefix, result)

		for method, op := range operations {
			if op == nil {
				continue
			}
			v.validateOAS3Operation(op, prefix+"."+method, result, baseURL)
		}
	}
}

// validateOAS3PathParameterConsistency checks that path parameters match the path template
func (v *Validator) validateOAS3PathParameterConsistency(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	if doc.Paths == nil {
		return
	}

	// Parameters may be hoisted into components.parameters and referenced by
	// $ref, in which case Name and In only exist on the definition.
	resolver := paramutil.NewOAS3Resolver(doc)
	validRefs := buildOAS3ValidRefs(doc)

	// Checked before the paths below, whose use-site checks defer downstream
	// breaks to the definition that carries them.
	if doc.Components != nil {
		v.validateParameterDefinitionRefs(doc.Components.Parameters, "components.parameters",
			resolver, validRefs, result, baseURL)
	}

	for pathPattern, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Extract parameter names from path template
		pathParams := extractPathParameters(pathPattern)

		// Check all operations in this path
		operations := parser.GetOperations(pathItem, doc.OASVersion)
		pathPrefix := "paths." + pathPattern

		// Path-level parameters apply to every operation in the item, so they
		// are checked once here rather than once per operation.
		v.validatePathParamsRequired(pathItem.Parameters, pathPrefix, result, baseURL)
		v.validateParameterRefs(pathItem.Parameters, pathPrefix, resolver, validRefs, result, baseURL)

		// A path-item parameter is declared once, so an unused one is warned
		// about once here rather than repeated for every operation in the item.
		// Resolving the item's parameters here also keeps them out of the
		// operation loop below.
		itemDeclared, itemComplete := resolver.DeclaredPathParams(pathItem.Parameters)
		v.warnUnusedPathParams(itemDeclared, pathParams, pathPrefix, result, baseURL)

		for method, op := range operations {
			if op == nil {
				continue
			}

			opPath := pathPrefix + "." + method
			v.validatePathParamsRequired(op.Parameters, opPath, result, baseURL)
			v.validateParameterRefs(op.Parameters, opPath, resolver, validRefs, result, baseURL)

			decls := declaresPathParams(resolver, itemDeclared, itemComplete, op.Parameters)
			v.reportUndeclaredPathParams(decls, pathParams, opPath, result, baseURL)

			// Warn only about parameters this operation declares itself; the
			// path item's were warned about once, above.
			v.warnUnusedPathParams(decls.operationOnly(), pathParams, opPath, result, baseURL)
		}
	}
}

// warnUnusedPathParams warns about path parameters declared at prefix that the
// path template never uses.
//
// Deliberately not gated on the resolver reporting a complete set: declared is
// a lower bound, so an unresolved $ref can only suppress a warning, never
// invent one. Gating it would lose warnings for no gain.
func (v *Validator) warnUnusedPathParams(
	declared, pathParams map[string]bool,
	prefix string,
	result *ValidationResult,
	baseURL string,
) {
	for paramName := range declared {
		if !pathParams[paramName] {
			v.addWarning(result, prefix,
				fmt.Sprintf("Parameter '%s' is declared as path parameter but not used in path template", paramName),
				withSpecRef(fmt.Sprintf("%s#path-item-object", baseURL)),
				withValue(paramName),
			)
		}
	}
}

// validateOAS3SecurityRequirements validates security requirements reference existing schemes
func (v *Validator) validateOAS3SecurityRequirements(doc *parser.OAS3Document, result *ValidationResult, baseURL string) {
	// Get available security schemes
	availableSchemes := make(map[string]bool)
	if doc.Components != nil {
		for name := range doc.Components.SecuritySchemes {
			availableSchemes[name] = true
		}
	}

	// Validate root-level security requirements
	for i, secReq := range doc.Security {
		for schemeName := range secReq {
			if !availableSchemes[schemeName] {
				v.addError(result, "security["+strconv.Itoa(i)+"]."+schemeName,
					fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
					withSpecRef(fmt.Sprintf("%s#security-requirement-object", baseURL)),
					withValue(schemeName),
				)
			}
		}
	}

	// Validate operation-level security requirements
	if doc.Paths != nil {
		for pathPattern, pathItem := range doc.Paths {
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, doc.OASVersion)

			for method, op := range operations {
				if op == nil {
					continue
				}

				for i, secReq := range op.Security {
					for schemeName := range secReq {
						if !availableSchemes[schemeName] {
							v.addError(result, "paths."+pathPattern+"."+method+".security["+strconv.Itoa(i)+"]."+schemeName,
								fmt.Sprintf("Security requirement references undefined security scheme: %s", schemeName),
								withSpecRef(fmt.Sprintf("%s#security-requirement-object", baseURL)),
								withValue(schemeName),
							)
						}
					}
				}
			}
		}
	}
}
