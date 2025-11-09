package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	semver "github.com/hashicorp/go-version"
	"gopkg.in/yaml.v3"
)

const (
	// HTTP status code validation constants
	statusCodeLength     = 3   // Standard length of HTTP status codes (e.g., "200", "404")
	minStatusCode        = 100 // Minimum valid HTTP status code
	maxStatusCode        = 599 // Maximum valid HTTP status code
	wildcardChar         = 'X' // Wildcard character used in status code patterns (e.g., "2XX")
	minWildcardFirstChar = '1' // Minimum first digit for wildcard patterns
	maxWildcardFirstChar = '5' // Maximum first digit for wildcard patterns
)

// Parser handles OpenAPI specification parsing
type Parser struct {
	// ResolveRefs determines whether to resolve $ref references
	ResolveRefs bool
	// ValidateStructure determines whether to perform basic structure validation
	ValidateStructure bool
}

// New creates a new Parser instance with default settings
func New() *Parser {
	return &Parser{
		ResolveRefs:       false,
		ValidateStructure: true,
	}
}

// ParseResult contains the parsed OpenAPI specification and metadata.
// This structure provides both the raw parsed data and version-specific
// typed representations of the OpenAPI document.
type ParseResult struct {
	// Version is the detected OAS version string (e.g., "2.0", "3.0.3", "3.1.0")
	Version string
	// Data contains the raw parsed data as a map, potentially with resolved $refs
	Data map[string]interface{}
	// Document contains the version-specific parsed document:
	// - *OAS2Document for OpenAPI 2.0
	// - *OAS3Document for OpenAPI 3.x
	Document interface{}
	// Errors contains any parsing or validation errors encountered
	Errors []error
	// Warnings contains non-fatal issues such as ref resolution failures
	Warnings []string
	// OASVersion is the enumerated version of the OpenAPI specification
	OASVersion OASVersion
}

// Parse parses an OpenAPI specification file
func (p *Parser) Parse(specPath string) (*ParseResult, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	// Get the directory of the spec file for resolving relative refs
	baseDir := filepath.Dir(specPath)
	return p.parseBytesWithBaseDir(data, baseDir)
}

// ParseReader parses an OpenAPI specification from an io.Reader
func (p *Parser) ParseReader(r io.Reader) (*ParseResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return p.ParseBytes(data)
}

// ParseBytes parses an OpenAPI specification from a byte slice
// For external references to work, use Parse() with a file path instead
func (p *Parser) ParseBytes(data []byte) (*ParseResult, error) {
	return p.parseBytesWithBaseDir(data, ".")
}

// parseBytesWithBaseDir parses data with a specified base directory for ref resolution
func (p *Parser) parseBytesWithBaseDir(data []byte, baseDir string) (*ParseResult, error) {
	result := &ParseResult{
		Errors:   make([]error, 0),
		Warnings: make([]string, 0),
	}

	// First pass: parse to generic map to detect OAS version
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
	}

	// Resolve references if enabled (before semver-specific parsing)
	if p.ResolveRefs {
		resolver := NewRefResolver(baseDir)
		if err := resolver.ResolveAllRefs(rawData); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ref resolution warning: %v", err))
		}
	}

	result.Data = rawData

	// Detect semver
	version, err := p.detectVersion(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to detect OAS semver: %w", err)
	}
	result.Version = version

	// Prepare data for version-specific parsing
	// Only re-marshal if we resolved refs (to avoid unnecessary overhead)
	//
	// Performance trade-off: When ResolveRefs is enabled, we must re-marshal the
	// rawData map after reference resolution to ensure the resolved content is
	// available to the version-specific parsers. This adds overhead (especially
	// for large documents), but is necessary for correct reference resolution.
	// When ResolveRefs is disabled, we skip this step and use the original data.
	var parseData []byte
	if p.ResolveRefs {
		// Re-marshal the data with resolved refs
		parseData, err = yaml.Marshal(rawData)
		if err != nil {
			return nil, fmt.Errorf("failed to re-marshal data: %w", err)
		}
	} else {
		// Use original data directly
		parseData = data
	}

	// Parse to semver-specific structure
	doc, oasVersion, err := p.parseVersionSpecific(parseData, version)
	if err != nil {
		return nil, err
	}
	result.Document = doc
	result.OASVersion = oasVersion

	// Validate structure if enabled
	if p.ValidateStructure {
		validationErrors := p.validateStructure(result)
		result.Errors = append(result.Errors, validationErrors...)
	}

	return result, nil
}

// detectVersion determines the OAS semver from the raw data
func (p *Parser) detectVersion(data map[string]interface{}) (string, error) {
	// Check for OAS 2.0 (Swagger)
	if swagger, ok := data["swagger"].(string); ok {
		return swagger, nil
	}

	// Check for OAS 3.x
	if openapi, ok := data["openapi"].(string); ok {
		return openapi, nil
	}

	// Neither field was found - provide helpful error message
	return "", fmt.Errorf("version detection: unable to detect OpenAPI version: document must contain either 'swagger: \"2.0\"' (for OAS 2.0) or 'openapi: \"3.x.x\"' (for OAS 3.x) at the root level")
}

// parseSemVer parses a semver string into a semantic semver
func parseSemVer(v string) (*semver.Version, error) {
	return semver.NewVersion(v)
}

// versionInRangeExclusive checks if a semver string is within the specified range: minVersion <= v < maxVersion
// If maxVersion is empty string, no upper bound is enforced (v >= minVersion)
func versionInRangeExclusive(v, minVersion, maxVersion string) bool {
	ver, err := parseSemVer(v)
	if err != nil {
		return false
	}

	min, err := parseSemVer(minVersion)
	if err != nil {
		return false
	}

	// Check lower bound
	if !ver.GreaterThanOrEqual(min) {
		return false
	}

	// If maxVersion is empty, no upper bound
	if maxVersion == "" {
		return true
	}

	max, err := parseSemVer(maxVersion)
	if err != nil {
		return false
	}
	return ver.LessThan(max)
}

// parseVersionSpecific parses the data into a semver-specific structure
func (p *Parser) parseVersionSpecific(data []byte, version string) (interface{}, OASVersion, error) {
	v, ok := ParseVersion(version)
	if !ok {
		return nil, 0, fmt.Errorf("invalid OAS semver: %s", version)
	}
	switch v {
	case OASVersion20:
		var doc OAS2Document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, 0, fmt.Errorf("oas 2.0 parser: failed to parse document structure: %w", err)
		}
		doc.OASVersion = v
		return &doc, v, nil

	case OASVersion300, OASVersion301, OASVersion302, OASVersion303, OASVersion304, OASVersion310, OASVersion311, OASVersion312, OASVersion320:
		var doc OAS3Document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, 0, fmt.Errorf("oas %s parser: failed to parse document structure: %w", version, err)
		}
		doc.OASVersion = v
		return &doc, v, nil

	default:
		return nil, 0, fmt.Errorf("parser: unsupported OpenAPI version: %s (only 2.0 and 3.x versions are supported)", version)
	}
}

// validateStructure performs basic structure validation
func (p *Parser) validateStructure(result *ParseResult) []error {
	errors := make([]error, 0)

	// Validate required fields based on semver
	switch {
	case result.OASVersion == OASVersion20:
		doc, ok := result.Document.(*OAS2Document)
		if !ok {
			errors = append(errors, fmt.Errorf("parser: internal error: document type mismatch for OAS 2.0 (expected *OAS2Document, got %T)", result.Document))
			return errors
		}
		errors = append(errors, p.validateOAS2(doc)...)

	case result.OASVersion.IsValid():
		doc, ok := result.Document.(*OAS3Document)
		if !ok {
			errors = append(errors, fmt.Errorf("parser: internal error: document type mismatch for OAS 3.x (expected *OAS3Document, got %T)", result.Document))
			return errors
		}
		errors = append(errors, p.validateOAS3(doc)...)

	default:
		errors = append(errors, fmt.Errorf("parser: unsupported OpenAPI version: %s (only versions 2.0 and 3.x are supported)", result.Version))
	}

	return errors
}

// isValidStatusCode checks if a status code string is valid per OAS spec
// Valid formats: "200", "404", "2XX", "4XX", "5XX", "default", etc.
func isValidStatusCode(code string) bool {
	if code == "" {
		return false
	}

	// Check if it's the "default" response
	if code == ResponseDefault {
		return true
	}

	// Check if it's a wildcard pattern (e.g., "2XX", "4XX")
	if len(code) == statusCodeLength && code[1] == wildcardChar && code[2] == wildcardChar {
		// First character should be a digit 1-5
		if code[0] >= minWildcardFirstChar && code[0] <= maxWildcardFirstChar {
			return true
		}
		return false
	}

	// Check if it's a valid HTTP status code (100-599)
	if len(code) != statusCodeLength {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	// Convert to int and validate range
	statusCode := 0
	for _, c := range code {
		statusCode = statusCode*10 + int(c-'0')
	}
	return statusCode >= minStatusCode && statusCode <= maxStatusCode
}

// validateOAS2 validates an OAS 2.0 document
func (p *Parser) validateOAS2(doc *OAS2Document) []error {
	errors := make([]error, 0)

	// Validate swagger version field
	if doc.Swagger == "" {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required root field 'swagger': must be set to \"2.0\""))
	} else if doc.Swagger != "2.0" {
		errors = append(errors, fmt.Errorf("oas 2.0: invalid 'swagger' field value: expected \"2.0\", got \"%s\"", doc.Swagger))
	}

	// Validate info object
	errors = append(errors, p.validateOAS2Info(doc.Info)...)

	// Validate paths object
	if doc.Paths == nil {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required root field 'paths': Paths object is required per spec (https://spec.openapis.org/oas/v2.0.html#pathsObject)"))
	} else {
		errors = append(errors, p.validateOAS2Paths(doc.Paths)...)
	}

	return errors
}

func (p *Parser) validateOAS2Info(info *Info) []error {
	errors := make([]error, 0)
	if info == nil {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required root field 'info': Info object is required per spec (https://spec.openapis.org/oas/v2.0.html#infoObject)"))
	} else {
		if info.Title == "" {
			errors = append(errors, fmt.Errorf("oas 2.0: missing required field 'info.title': Info object must have a title per spec"))
		}
		if info.Version == "" {
			errors = append(errors, fmt.Errorf("oas 2.0: missing required field 'info.version': Info object must have a version string per spec"))
		}
	}
	return errors
}

func (p *Parser) validateOAS2Paths(paths map[string]*PathItem) []error {
	errors := make([]error, 0)
	operationIDs := make(map[string]string)

	for pathPattern, pathItem := range paths {
		if pathItem == nil {
			continue
		}

		// Validate path pattern
		if pathPattern != "" && pathPattern[0] != '/' {
			errors = append(errors, fmt.Errorf("oas 2.0: invalid path pattern 'paths.%s': path must begin with '/'", pathPattern))
		}

		// Check all operations in this path
		errors = append(errors, p.validateOAS2PathItem(pathItem, pathPattern, operationIDs)...)
	}

	return errors
}

func (p *Parser) validateOAS2PathItem(pathItem *PathItem, pathPattern string, operationIDs map[string]string) []error {
	errors := make([]error, 0)
	operations := map[string]*Operation{
		"get":     pathItem.Get,
		"put":     pathItem.Put,
		"post":    pathItem.Post,
		"delete":  pathItem.Delete,
		"options": pathItem.Options,
		"head":    pathItem.Head,
		"patch":   pathItem.Patch,
	}

	for method, op := range operations {
		if op == nil {
			continue
		}

		opPath := fmt.Sprintf("paths.%s.%s", pathPattern, method)
		errors = append(errors, p.validateOAS2Operation(op, opPath, operationIDs)...)
	}

	return errors
}

func (p *Parser) validateOAS2Operation(op *Operation, opPath string, operationIDs map[string]string) []error {
	errors := make([]error, 0)

	// Validate operationId uniqueness
	if op.OperationID != "" {
		if existingPath, exists := operationIDs[op.OperationID]; exists {
			errors = append(errors, fmt.Errorf("oas 2.0: duplicate operationId '%s' at '%s': previously defined at '%s'",
				op.OperationID, opPath, existingPath))
		} else {
			operationIDs[op.OperationID] = opPath
		}
	}

	// Validate responses object exists
	if op.Responses == nil {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required field '%s.responses': Operation must have a responses object", opPath))
	} else {
		// Validate status codes in responses
		for code := range op.Responses.Codes {
			if !isValidStatusCode(code) {
				errors = append(errors, fmt.Errorf("oas 2.0: invalid status code '%s' in '%s.responses': must be a valid HTTP status code (e.g., \"200\", \"404\") or wildcard pattern (e.g., \"2XX\")", code, opPath))
			}
		}
	}

	// Validate parameters
	for i, param := range op.Parameters {
		if param == nil {
			continue
		}
		errors = append(errors, p.validateOAS2Parameter(param, opPath, i)...)
	}

	return errors
}

func (p *Parser) validateOAS2Parameter(param *Parameter, opPath string, index int) []error {
	errors := make([]error, 0)
	paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, index)

	if param.Name == "" {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required field '%s.name': Parameter must have a name", paramPath))
	}
	if param.In == "" {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required field '%s.in': Parameter must specify location (query, header, path, formData, body)", paramPath))
	} else {
		validLocations := map[string]bool{
			ParamInQuery:    true,
			ParamInHeader:   true,
			ParamInPath:     true,
			ParamInFormData: true,
			ParamInBody:     true,
		}
		if !validLocations[param.In] {
			errors = append(errors, fmt.Errorf("oas 2.0: invalid value for '%s.in': \"%s\" is not a valid parameter location (must be query, header, path, formData, or body)", paramPath, param.In))
		}
	}

	return errors
}

// validateOAS3 validates an OAS 3.x document
func (p *Parser) validateOAS3(doc *OAS3Document) []error {
	errors := make([]error, 0)
	version := doc.OpenAPI

	// Validate openapi version field
	if doc.OpenAPI == "" {
		errors = append(errors, fmt.Errorf("oas 3.x: missing required root field 'openapi': must be set to a valid 3.x version (e.g., \"3.0.3\", \"3.1.0\")"))
	} else if !versionInRangeExclusive(doc.OpenAPI, "3.0.0", "4.0.0") {
		errors = append(errors, fmt.Errorf("oas %s: invalid 'openapi' field value: \"%s\" is not a valid 3.x version", version, doc.OpenAPI))
	}

	// Validate info object
	errors = append(errors, p.validateOAS3Info(doc.Info, version)...)

	// Validate paths object - required in 3.0.x, optional in 3.1+
	errors = append(errors, p.validateOAS3PathsRequirement(doc, version)...)

	// Validate paths if present
	if doc.Paths != nil {
		errors = append(errors, p.validateOAS3Paths(doc.Paths, version)...)
	}

	// Validate webhooks if present (OAS 3.1+)
	if len(doc.Webhooks) > 0 && versionInRangeExclusive(doc.OpenAPI, "0.0.0", "3.1.0") {
		errors = append(errors, fmt.Errorf("oas %s: 'webhooks' field is only supported in OAS 3.1.0 and later, not in version %s", version, doc.OpenAPI))
	}

	return errors
}

func (p *Parser) validateOAS3Info(info *Info, version string) []error {
	errors := make([]error, 0)
	if info == nil {
		errors = append(errors, fmt.Errorf("oas %s: missing required root field 'info': Info object is required per spec (https://spec.openapis.org/oas/v3.0.0.html#info-object)", version))
	} else {
		if info.Title == "" {
			errors = append(errors, fmt.Errorf("oas %s: missing required field 'info.title': Info object must have a title per spec", version))
		}
		if info.Version == "" {
			errors = append(errors, fmt.Errorf("oas %s: missing required field 'info.version': Info object must have a version string per spec", version))
		}
	}
	return errors
}

func (p *Parser) validateOAS3PathsRequirement(doc *OAS3Document, version string) []error {
	errors := make([]error, 0)
	if versionInRangeExclusive(doc.OpenAPI, "3.0.0", "3.1.0") {
		if doc.Paths == nil {
			errors = append(errors, fmt.Errorf("oas %s: missing required root field 'paths': Paths object is required in OAS 3.0.x (https://spec.openapis.org/oas/v3.0.0.html#paths-object)", version))
		}
	} else if versionInRangeExclusive(doc.OpenAPI, "3.1.0", "") {
		// In OAS 3.1+, either paths or webhooks must be present
		if doc.Paths == nil && len(doc.Webhooks) == 0 {
			errors = append(errors, fmt.Errorf("oas %s: document must have either 'paths' or 'webhooks': at least one is required in OAS 3.1+", version))
		}
	}
	return errors
}

func (p *Parser) validateOAS3Paths(paths map[string]*PathItem, version string) []error {
	errors := make([]error, 0)
	operationIDs := make(map[string]string)

	for pathPattern, pathItem := range paths {
		if pathItem == nil {
			continue
		}

		// Validate path pattern
		if pathPattern != "" && pathPattern[0] != '/' {
			errors = append(errors, fmt.Errorf("oas %s: invalid path pattern 'paths.%s': path must begin with '/'", version, pathPattern))
		}

		// Check all operations in this path
		errors = append(errors, p.validateOAS3PathItem(pathItem, pathPattern, operationIDs, version)...)
	}

	return errors
}

func (p *Parser) validateOAS3PathItem(pathItem *PathItem, pathPattern string, operationIDs map[string]string, version string) []error {
	errors := make([]error, 0)
	operations := map[string]*Operation{
		"get":     pathItem.Get,
		"put":     pathItem.Put,
		"post":    pathItem.Post,
		"delete":  pathItem.Delete,
		"options": pathItem.Options,
		"head":    pathItem.Head,
		"patch":   pathItem.Patch,
		"trace":   pathItem.Trace,
	}

	for method, op := range operations {
		if op == nil {
			continue
		}

		opPath := fmt.Sprintf("paths.%s.%s", pathPattern, method)
		errors = append(errors, p.validateOAS3Operation(op, opPath, operationIDs, version)...)
	}

	return errors
}

func (p *Parser) validateOAS3Operation(op *Operation, opPath string, operationIDs map[string]string, version string) []error {
	errors := make([]error, 0)

	// Validate operationId uniqueness
	if op.OperationID != "" {
		if existingPath, exists := operationIDs[op.OperationID]; exists {
			errors = append(errors, fmt.Errorf("oas %s: duplicate operationId '%s' at '%s': previously defined at '%s' (operationIds must be unique across all operations)",
				version, op.OperationID, opPath, existingPath))
		} else {
			operationIDs[op.OperationID] = opPath
		}
	}

	// Validate responses object exists
	if op.Responses == nil {
		errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.responses': Operation must have a responses object", version, opPath))
	} else {
		// Validate status codes in responses
		for code := range op.Responses.Codes {
			if !isValidStatusCode(code) {
				errors = append(errors, fmt.Errorf("oas %s: invalid status code '%s' in '%s.responses': must be a valid HTTP status code (e.g., \"200\", \"404\") or wildcard pattern (e.g., \"2XX\")", version, code, opPath))
			}
		}
	}

	// Validate parameters
	for i, param := range op.Parameters {
		if param == nil {
			continue
		}
		errors = append(errors, p.validateOAS3Parameter(param, opPath, i, version)...)
	}

	// Validate requestBody if present
	if op.RequestBody != nil {
		rbPath := fmt.Sprintf("%s.requestBody", opPath)
		if len(op.RequestBody.Content) == 0 {
			errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.content': RequestBody must have at least one media type", version, rbPath))
		}
	}

	return errors
}

func (p *Parser) validateOAS3Parameter(param *Parameter, opPath string, index int, version string) []error {
	errors := make([]error, 0)
	paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, index)

	if param.Name == "" {
		errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.name': Parameter must have a name", version, paramPath))
	}
	if param.In == "" {
		errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.in': Parameter must specify location (query, header, path, cookie)", version, paramPath))
	} else {
		validLocations := map[string]bool{
			ParamInQuery:  true,
			ParamInHeader: true,
			ParamInPath:   true,
			ParamInCookie: true,
		}
		if !validLocations[param.In] {
			errors = append(errors, fmt.Errorf("oas %s: invalid value for '%s.in': \"%s\" is not a valid parameter location (must be query, header, path, or cookie)", version, paramPath, param.In))
		}
	}

	// Path parameters must be required
	if param.In == ParamInPath && !param.Required {
		errors = append(errors, fmt.Errorf("oas %s: invalid parameter '%s': path parameters must have 'required: true' per spec", version, paramPath))
	}

	return errors
}
