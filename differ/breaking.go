package differ

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// diffBreaking performs a diff that categorizes changes by severity
// and identifies breaking API changes
func (d *Differ) diffBreaking(source, target parser.ParseResult, result *DiffResult) {
	// Compare based on OAS version
	switch {
	case source.OASVersion == parser.OASVersion20 && target.OASVersion == parser.OASVersion20:
		d.diffOAS2Breaking(source.Document.(*parser.OAS2Document), target.Document.(*parser.OAS2Document), result)
	case source.OASVersion >= parser.OASVersion300 && target.OASVersion >= parser.OASVersion300:
		d.diffOAS3Breaking(source.Document.(*parser.OAS3Document), target.Document.(*parser.OAS3Document), result)
	default:
		// Cross-version comparison
		d.diffCrossVersionBreaking(source, target, result)
	}
}

// diffOAS2Breaking compares two OAS 2.0 documents with breaking change detection
func (d *Differ) diffOAS2Breaking(source, target *parser.OAS2Document, result *DiffResult) {
	basePath := "document"

	// Compare Info (non-breaking)
	d.diffInfoBreaking(source.Info, target.Info, basePath+".info", result)

	// Compare Host, BasePath, Schemes (warning - may affect clients)
	if source.Host != target.Host {
		result.Changes = append(result.Changes, Change{
			Path:     basePath + ".host",
			Type:     ChangeTypeModified,
			Category: CategoryServer,
			Severity: SeverityWarning,
			OldValue: source.Host,
			NewValue: target.Host,
			Message:  fmt.Sprintf("host changed from %q to %q", source.Host, target.Host),
		})
	}

	if source.BasePath != target.BasePath {
		result.Changes = append(result.Changes, Change{
			Path:     basePath + ".basePath",
			Type:     ChangeTypeModified,
			Category: CategoryServer,
			Severity: SeverityWarning,
			OldValue: source.BasePath,
			NewValue: target.BasePath,
			Message:  fmt.Sprintf("basePath changed from %q to %q", source.BasePath, target.BasePath),
		})
	}

	d.diffStringSlicesBreaking(source.Schemes, target.Schemes, basePath+".schemes", CategoryServer, "scheme", SeverityWarning, result)

	// Compare Paths - critical for breaking changes
	d.diffPathsBreaking(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Definitions
	d.diffSchemasBreaking(source.Definitions, target.Definitions, basePath+".definitions", result)

	// Compare Security Definitions
	d.diffSecuritySchemesBreaking(source.SecurityDefinitions, target.SecurityDefinitions, basePath+".securityDefinitions", result)

	// Compare Extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, basePath, result)
}

// diffOAS3Breaking compares two OAS 3.x documents with breaking change detection
func (d *Differ) diffOAS3Breaking(source, target *parser.OAS3Document, result *DiffResult) {
	basePath := "document"

	// Compare Info (non-breaking)
	d.diffInfoBreaking(source.Info, target.Info, basePath+".info", result)

	// Compare Servers (warning)
	d.diffServersBreaking(source.Servers, target.Servers, basePath+".servers", result)

	// Compare Paths - critical for breaking changes
	d.diffPathsBreaking(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Webhooks (OAS 3.1+)
	if source.Webhooks != nil || target.Webhooks != nil {
		d.diffWebhooksBreaking(source.Webhooks, target.Webhooks, basePath+".webhooks", result)
	}

	// Compare Components
	if source.Components != nil || target.Components != nil {
		d.diffComponentsBreaking(source.Components, target.Components, basePath+".components", result)
	}

	// Compare Extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, basePath, result)
}

// diffCrossVersionBreaking compares documents of different OAS versions
func (d *Differ) diffCrossVersionBreaking(source, target parser.ParseResult, result *DiffResult) {
	// Version change is informational
	result.Changes = append(result.Changes, Change{
		Path:     "document.openapi",
		Type:     ChangeTypeModified,
		Category: CategoryInfo,
		Severity: SeverityInfo,
		OldValue: source.Version,
		NewValue: target.Version,
		Message:  fmt.Sprintf("OAS version changed from %s to %s", source.Version, target.Version),
	})

	// Compare paths as they exist in both versions
	var sourcePaths, targetPaths parser.Paths

	switch source.OASVersion {
	case parser.OASVersion20:
		sourcePaths = source.Document.(*parser.OAS2Document).Paths
	default:
		sourcePaths = source.Document.(*parser.OAS3Document).Paths
	}

	switch target.OASVersion {
	case parser.OASVersion20:
		targetPaths = target.Document.(*parser.OAS2Document).Paths
	default:
		targetPaths = target.Document.(*parser.OAS3Document).Paths
	}

	d.diffPathsBreaking(sourcePaths, targetPaths, "document.paths", result)
}

// diffInfoBreaking compares Info objects (non-breaking changes)
func (d *Differ) diffInfoBreaking(source, target *parser.Info, path string, result *DiffResult) {
	if source == nil || target == nil {
		return
	}

	if source.Title != target.Title {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".title",
			Type:     ChangeTypeModified,
			Category: CategoryInfo,
			Severity: SeverityInfo,
			OldValue: source.Title,
			NewValue: target.Title,
			Message:  fmt.Sprintf("title changed from %q to %q", source.Title, target.Title),
		})
	}

	if source.Version != target.Version {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".version",
			Type:     ChangeTypeModified,
			Category: CategoryInfo,
			Severity: SeverityInfo,
			OldValue: source.Version,
			NewValue: target.Version,
			Message:  fmt.Sprintf("API version changed from %q to %q", source.Version, target.Version),
		})
	}

	if source.Description != target.Description && source.Description != "" {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryInfo,
			Severity: SeverityInfo,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "description changed",
		})
	}

	// Compare Info extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffServersBreaking compares Server slices (OAS 3.x)
func (d *Differ) diffServersBreaking(source, target []*parser.Server, path string, result *DiffResult) {
	sourceMap := make(map[string]*parser.Server)
	for _, srv := range source {
		sourceMap[srv.URL] = srv
	}

	targetMap := make(map[string]*parser.Server)
	for _, srv := range target {
		targetMap[srv.URL] = srv
	}

	// Removed servers - warning
	for url, sourceSrv := range sourceMap {
		targetSrv, exists := targetMap[url]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, url),
				Type:     ChangeTypeRemoved,
				Category: CategoryServer,
				Severity: SeverityWarning,
				OldValue: url,
				Message:  fmt.Sprintf("server %q removed", url),
			})
			continue
		}

		// Compare server details if both exist
		d.diffServerBreaking(sourceSrv, targetSrv, fmt.Sprintf("%s[%s]", path, url), result)
	}

	// Added servers - info
	for url := range targetMap {
		if _, exists := sourceMap[url]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, url),
				Type:     ChangeTypeAdded,
				Category: CategoryServer,
				Severity: SeverityInfo,
				NewValue: url,
				Message:  fmt.Sprintf("server %q added", url),
			})
		}
	}
}

// diffServerBreaking compares individual Server objects
func (d *Differ) diffServerBreaking(source, target *parser.Server, path string, result *DiffResult) {
	if source.Description != target.Description && source.Description != "" {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryServer,
			Severity: SeverityInfo,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "server description changed",
		})
	}

	// Compare Server extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffPathsBreaking compares Paths with breaking change detection
func (d *Differ) diffPathsBreaking(source, target parser.Paths, path string, result *DiffResult) {
	// Removed paths - CRITICAL breaking change
	for pathName := range source {
		if _, exists := target[pathName]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, pathName),
				Type:     ChangeTypeRemoved,
				Category: CategoryEndpoint,
				Severity: SeverityCritical,
				Message:  fmt.Sprintf("endpoint %q removed", pathName),
			})
		}
	}

	// Added paths - info (non-breaking)
	for pathName := range target {
		if _, exists := source[pathName]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, pathName),
				Type:     ChangeTypeAdded,
				Category: CategoryEndpoint,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("endpoint %q added", pathName),
			})
		}
	}

	// Compare common paths
	for pathName := range source {
		if targetItem, exists := target[pathName]; exists {
			d.diffPathItemBreaking(source[pathName], targetItem, fmt.Sprintf("%s.%s", path, pathName), result)
		}
	}
}

// diffPathItemBreaking compares PathItem objects with breaking change detection
func (d *Differ) diffPathItemBreaking(source, target *parser.PathItem, path string, result *DiffResult) {
	operations := map[string]struct {
		source *parser.Operation
		target *parser.Operation
	}{
		"get":     {source.Get, target.Get},
		"put":     {source.Put, target.Put},
		"post":    {source.Post, target.Post},
		"delete":  {source.Delete, target.Delete},
		"options": {source.Options, target.Options},
		"head":    {source.Head, target.Head},
		"patch":   {source.Patch, target.Patch},
		"trace":   {source.Trace, target.Trace},
	}

	for method, ops := range operations {
		opPath := fmt.Sprintf("%s.%s", path, method)

		if ops.source == nil && ops.target == nil {
			continue
		}

		// Removed operation - CRITICAL
		if ops.source != nil && ops.target == nil {
			result.Changes = append(result.Changes, Change{
				Path:     opPath,
				Type:     ChangeTypeRemoved,
				Category: CategoryOperation,
				Severity: SeverityCritical,
				Message:  fmt.Sprintf("operation %s removed", method),
			})
			continue
		}

		// Added operation - info
		if ops.source == nil && ops.target != nil {
			result.Changes = append(result.Changes, Change{
				Path:     opPath,
				Type:     ChangeTypeAdded,
				Category: CategoryOperation,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("operation %s added", method),
			})
			continue
		}

		// Compare operations
		d.diffOperationBreaking(ops.source, ops.target, opPath, result)
	}

	// Compare PathItem extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffOperationBreaking compares Operation objects with breaking change detection
func (d *Differ) diffOperationBreaking(source, target *parser.Operation, path string, result *DiffResult) {
	// Deprecated flag change
	if !source.Deprecated && target.Deprecated {
		// Marking as deprecated is a warning
		result.Changes = append(result.Changes, Change{
			Path:     path + ".deprecated",
			Type:     ChangeTypeModified,
			Category: CategoryOperation,
			Severity: SeverityWarning,
			OldValue: false,
			NewValue: true,
			Message:  "operation marked as deprecated",
		})
	} else if source.Deprecated && !target.Deprecated {
		// Un-deprecating is info
		result.Changes = append(result.Changes, Change{
			Path:     path + ".deprecated",
			Type:     ChangeTypeModified,
			Category: CategoryOperation,
			Severity: SeverityInfo,
			OldValue: true,
			NewValue: false,
			Message:  "operation no longer deprecated",
		})
	}

	// Compare parameters
	d.diffParametersBreaking(source.Parameters, target.Parameters, path+".parameters", result)

	// Compare responses
	d.diffResponsesBreaking(source.Responses, target.Responses, path+".responses", result)

	// Compare request body (OAS 3.x)
	if source.RequestBody != nil || target.RequestBody != nil {
		d.diffRequestBodyBreaking(source.RequestBody, target.RequestBody, path+".requestBody", result)
	}

	// Compare Operation extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffParametersBreaking compares Parameter slices with breaking change detection
func (d *Differ) diffParametersBreaking(source, target []*parser.Parameter, path string, result *DiffResult) {
	sourceMap := make(map[string]*parser.Parameter)
	for _, param := range source {
		key := param.Name + ":" + param.In
		sourceMap[key] = param
	}

	targetMap := make(map[string]*parser.Parameter)
	for _, param := range target {
		key := param.Name + ":" + param.In
		targetMap[key] = param
	}

	// Removed parameters
	for key, sourceParam := range sourceMap {
		if _, exists := targetMap[key]; !exists {
			severity := SeverityWarning
			if sourceParam.Required {
				// Removing required parameter is CRITICAL breaking change
				severity = SeverityCritical
			}
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, key),
				Type:     ChangeTypeRemoved,
				Category: CategoryParameter,
				Severity: severity,
				Message:  fmt.Sprintf("parameter %q in %s removed (required: %v)", sourceParam.Name, sourceParam.In, sourceParam.Required),
			})
		}
	}

	// Added parameters
	for key, targetParam := range targetMap {
		if _, exists := sourceMap[key]; !exists {
			severity := SeverityInfo
			if targetParam.Required {
				// Adding required parameter is a WARNING (clients must update)
				severity = SeverityWarning
			}
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, key),
				Type:     ChangeTypeAdded,
				Category: CategoryParameter,
				Severity: severity,
				Message:  fmt.Sprintf("parameter %q in %s added (required: %v)", targetParam.Name, targetParam.In, targetParam.Required),
			})
		}
	}

	// Compare common parameters
	for key := range sourceMap {
		if targetParam, exists := targetMap[key]; exists {
			d.diffParameterBreaking(sourceMap[key], targetParam, fmt.Sprintf("%s[%s]", path, key), result)
		}
	}
}

// diffParameterBreaking compares individual Parameter objects with breaking change detection
func (d *Differ) diffParameterBreaking(source, target *parser.Parameter, path string, result *DiffResult) {
	// Required changed
	if !source.Required && target.Required {
		// Making optional parameter required - BREAKING
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			Severity: SeverityError,
			OldValue: false,
			NewValue: true,
			Message:  "parameter changed from optional to required",
		})
	} else if source.Required && !target.Required {
		// Making required parameter optional - INFO (relaxing constraint)
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			Severity: SeverityInfo,
			OldValue: true,
			NewValue: false,
			Message:  "parameter changed from required to optional",
		})
	}

	// Type changed
	if source.Type != target.Type {
		// Type change is generally breaking unless it's a compatible widening
		severity := SeverityError
		if isCompatibleTypeChange(source.Type, target.Type) {
			severity = SeverityWarning
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".type",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			Severity: severity,
			OldValue: source.Type,
			NewValue: target.Type,
			Message:  fmt.Sprintf("type changed from %q to %q", source.Type, target.Type),
		})
	}

	// Format changed
	if source.Format != target.Format && source.Format != "" {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".format",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			Severity: SeverityWarning,
			OldValue: source.Format,
			NewValue: target.Format,
			Message:  fmt.Sprintf("format changed from %q to %q", source.Format, target.Format),
		})
	}

	// Enum constraints
	d.diffEnumBreaking(source.Enum, target.Enum, path+".enum", result)

	// Compare Parameter extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffRequestBodyBreaking compares RequestBody objects with breaking change detection
func (d *Differ) diffRequestBodyBreaking(source, target *parser.RequestBody, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	// Request body added
	if source == nil && target != nil {
		severity := SeverityInfo
		if target.Required {
			severity = SeverityWarning
		}
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategoryRequestBody,
			Severity: severity,
			Message:  fmt.Sprintf("request body added (required: %v)", target.Required),
		})
		return
	}

	// Request body removed - BREAKING
	if source != nil && target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategoryRequestBody,
			Severity: SeverityError,
			Message:  "request body removed",
		})
		return
	}

	// Required changed
	if !source.Required && target.Required {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryRequestBody,
			Severity: SeverityError,
			OldValue: false,
			NewValue: true,
			Message:  "request body changed from optional to required",
		})
	} else if source.Required && !target.Required {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryRequestBody,
			Severity: SeverityInfo,
			OldValue: true,
			NewValue: false,
			Message:  "request body changed from required to optional",
		})
	}

	// Compare RequestBody extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffResponsesBreaking compares Responses with breaking change detection
func (d *Differ) diffResponsesBreaking(source, target *parser.Responses, path string, result *DiffResult) {
	if source == nil || target == nil {
		return
	}

	// Compare response codes
	for code, sourceResp := range source.Codes {
		targetResp, exists := target.Codes[code]
		if !exists {
			// Removed response code
			severity := SeverityWarning
			if isSuccessCode(code) {
				// Removing success response is BREAKING
				severity = SeverityError
			}
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, code),
				Type:     ChangeTypeRemoved,
				Category: CategoryResponse,
				Severity: severity,
				OldValue: sourceResp,
				Message:  fmt.Sprintf("response code %s removed", code),
			})
			continue
		}

		// Compare response details
		d.diffResponseBreaking(sourceResp, targetResp, fmt.Sprintf("%s[%s]", path, code), result)
	}

	// Added response codes - generally INFO
	for code, targetResp := range target.Codes {
		if _, exists := source.Codes[code]; !exists {
			severity := SeverityInfo
			if isErrorCode(code) {
				// New error codes might indicate new failure modes
				severity = SeverityWarning
			}
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, code),
				Type:     ChangeTypeAdded,
				Category: CategoryResponse,
				Severity: severity,
				NewValue: targetResp,
				Message:  fmt.Sprintf("response code %s added", code),
			})
		}
	}
}

// diffResponseBreaking compares individual Response objects
func (d *Differ) diffResponseBreaking(source, target *parser.Response, path string, result *DiffResult) {
	if source.Description != target.Description && source.Description != "" {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryResponse,
			Severity: SeverityInfo,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "response description changed",
		})
	}

	// Compare Response headers
	d.diffResponseHeadersBreaking(source.Headers, target.Headers, path, result)

	// Compare Response content
	d.diffResponseContentBreaking(source.Content, target.Content, path, result)

	// Compare Response links
	d.diffResponseLinksBreaking(source.Links, target.Links, path, result)

	// Compare Response examples
	d.diffResponseExamplesBreaking(source.Examples, target.Examples, path, result)

	// Compare Response extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffResponseHeadersBreaking compares header maps with breaking change detection
func (d *Differ) diffResponseHeadersBreaking(source, target map[string]*parser.Header, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Removed headers - WARNING (removing a response header is informational)
	for name := range source {
		if _, exists := target[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.headers.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategoryResponse,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("response header %q removed", name),
			})
		}
	}

	// Added headers - INFO
	for name, targetHeader := range target {
		sourceHeader, exists := source[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.headers.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategoryResponse,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("response header %q added", name),
			})
			continue
		}

		// Compare header details
		d.diffHeaderBreaking(sourceHeader, targetHeader, fmt.Sprintf("%s.headers.%s", path, name), result)
	}
}

// diffHeaderBreaking compares individual Header objects with breaking change detection
func (d *Differ) diffHeaderBreaking(source, target *parser.Header, path string, result *DiffResult) {
	// Required changed - WARNING/ERROR
	if source.Required != target.Required {
		severity := SeverityWarning
		if !source.Required && target.Required {
			// Making a header required is an ERROR
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryResponse,
			Severity: severity,
			OldValue: source.Required,
			NewValue: target.Required,
			Message:  fmt.Sprintf("required changed from %v to %v", source.Required, target.Required),
		})
	}

	// Type changed - WARNING
	if source.Type != target.Type {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".type",
			Type:     ChangeTypeModified,
			Category: CategoryResponse,
			Severity: SeverityWarning,
			OldValue: source.Type,
			NewValue: target.Type,
			Message:  fmt.Sprintf("type changed from %q to %q", source.Type, target.Type),
		})
	}

	// Compare Header extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffResponseContentBreaking compares response content maps with breaking change detection
func (d *Differ) diffResponseContentBreaking(source, target map[string]*parser.MediaType, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Removed media types - WARNING
	for mediaType := range source {
		if _, exists := target[mediaType]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.content.%s", path, mediaType),
				Type:     ChangeTypeRemoved,
				Category: CategoryResponse,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("response media type %q removed", mediaType),
			})
		}
	}

	// Added or modified media types - INFO/changes
	for mediaType, targetMedia := range target {
		sourceMedia, exists := source[mediaType]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.content.%s", path, mediaType),
				Type:     ChangeTypeAdded,
				Category: CategoryResponse,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("response media type %q added", mediaType),
			})
			continue
		}

		// Compare media type details
		d.diffMediaTypeBreaking(sourceMedia, targetMedia, fmt.Sprintf("%s.content.%s", path, mediaType), result)
	}
}

// diffMediaTypeBreaking compares individual MediaType objects with breaking change detection
func (d *Differ) diffMediaTypeBreaking(source, target *parser.MediaType, path string, result *DiffResult) {
	// Compare schemas if present
	if source.Schema != nil && target.Schema != nil {
		d.diffSchemaBreaking(source.Schema, target.Schema, path+".schema", result)
	} else if source.Schema != nil && target.Schema == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".schema",
			Type:     ChangeTypeRemoved,
			Category: CategoryResponse,
			Severity: SeverityWarning,
			Message:  "schema removed",
		})
	} else if source.Schema == nil && target.Schema != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".schema",
			Type:     ChangeTypeAdded,
			Category: CategoryResponse,
			Severity: SeverityInfo,
			Message:  "schema added",
		})
	}

	// Compare MediaType extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffResponseLinksBreaking compares response link maps with breaking change detection
func (d *Differ) diffResponseLinksBreaking(source, target map[string]*parser.Link, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Removed links - WARNING
	for name := range source {
		if _, exists := target[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.links.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategoryResponse,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("response link %q removed", name),
			})
		}
	}

	// Added or modified links - INFO/changes
	for name, targetLink := range target {
		sourceLink, exists := source[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.links.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategoryResponse,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("response link %q added", name),
			})
			continue
		}

		// Compare link details
		d.diffLinkBreaking(sourceLink, targetLink, fmt.Sprintf("%s.links.%s", path, name), result)
	}
}

// diffLinkBreaking compares individual Link objects with breaking change detection
func (d *Differ) diffLinkBreaking(source, target *parser.Link, path string, result *DiffResult) {
	// OperationRef changed - WARNING
	if source.OperationRef != target.OperationRef {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".operationRef",
			Type:     ChangeTypeModified,
			Category: CategoryResponse,
			Severity: SeverityWarning,
			OldValue: source.OperationRef,
			NewValue: target.OperationRef,
			Message:  "operationRef changed",
		})
	}

	// OperationID changed - WARNING
	if source.OperationID != target.OperationID {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".operationId",
			Type:     ChangeTypeModified,
			Category: CategoryResponse,
			Severity: SeverityWarning,
			OldValue: source.OperationID,
			NewValue: target.OperationID,
			Message:  "operationId changed",
		})
	}

	// Compare Link extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffResponseExamplesBreaking compares response example maps with breaking changes
func (d *Differ) diffResponseExamplesBreaking(source, target map[string]any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Removed examples - INFO
	for name := range source {
		if _, exists := target[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.examples.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategoryResponse,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("response example %q removed", name),
			})
		}
	}

	// Added examples - INFO (we don't deep-compare example values)
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.examples.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategoryResponse,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("response example %q added", name),
			})
		}
	}
}

// diffSchemasBreaking compares schema maps with breaking change detection
func (d *Differ) diffSchemasBreaking(source, target map[string]*parser.Schema, path string, result *DiffResult) {
	// Removed schemas - ERROR
	for name, sourceSchema := range source {
		targetSchema, exists := target[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityError,
				Message:  fmt.Sprintf("schema %q removed", name),
			})
			continue
		}

		// Compare schema extensions
		d.diffSchemaBreaking(sourceSchema, targetSchema, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Added schemas - INFO
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("schema %q added", name),
			})
		}
	}
}

// diffSchemaBreaking compares individual Schema objects
func (d *Differ) diffSchemaBreaking(source, target *parser.Schema, path string, result *DiffResult) {
	// Use recursive diffing with cycle detection
	visited := newSchemaVisited()
	d.diffSchemaRecursiveBreaking(source, target, path, visited, result)
}

// diffSchemaMetadata compares schema metadata fields
func (d *Differ) diffSchemaMetadata(source, target *parser.Schema, path string, result *DiffResult) {
	if source.Title != target.Title {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".title",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityInfo,
			OldValue: source.Title,
			NewValue: target.Title,
			Message:  "schema title changed",
		})
	}

	if source.Description != target.Description {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityInfo,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "schema description changed",
		})
	}
}

// diffSchemaType compares schema type and format fields
func (d *Differ) diffSchemaType(source, target *parser.Schema, path string, result *DiffResult) {
	// Type can be string or []string in OAS 3.1+
	sourceTypeStr := fmt.Sprintf("%v", source.Type)
	targetTypeStr := fmt.Sprintf("%v", target.Type)
	if sourceTypeStr != targetTypeStr {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".type",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityError,
			OldValue: source.Type,
			NewValue: target.Type,
			Message:  "schema type changed",
		})
	}

	if source.Format != target.Format {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".format",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source.Format,
			NewValue: target.Format,
			Message:  "schema format changed",
		})
	}
}

// diffSchemaNumericConstraints compares numeric validation constraints
func (d *Differ) diffSchemaNumericConstraints(source, target *parser.Schema, path string, result *DiffResult) {
	if source.MultipleOf != nil && target.MultipleOf != nil && *source.MultipleOf != *target.MultipleOf {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".multipleOf",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: *source.MultipleOf,
			NewValue: *target.MultipleOf,
			Message:  "multipleOf constraint changed",
		})
	}

	// Maximum constraint
	if source.Maximum != nil && target.Maximum != nil && *source.Maximum != *target.Maximum {
		severity := SeverityWarning
		if *target.Maximum < *source.Maximum {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maximum",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.Maximum,
			NewValue: *target.Maximum,
			Message:  "maximum constraint changed",
		})
	} else if source.Maximum == nil && target.Maximum != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maximum",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.Maximum,
			Message:  "maximum constraint added",
		})
	}

	// Minimum constraint
	if source.Minimum != nil && target.Minimum != nil && *source.Minimum != *target.Minimum {
		severity := SeverityWarning
		if *target.Minimum > *source.Minimum {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minimum",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.Minimum,
			NewValue: *target.Minimum,
			Message:  "minimum constraint changed",
		})
	} else if source.Minimum == nil && target.Minimum != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minimum",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.Minimum,
			Message:  "minimum constraint added",
		})
	}
}

// diffSchemaStringConstraints compares string validation constraints
func (d *Differ) diffSchemaStringConstraints(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxLength constraint
	if source.MaxLength != nil && target.MaxLength != nil && *source.MaxLength != *target.MaxLength {
		severity := SeverityWarning
		if *target.MaxLength < *source.MaxLength {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maxLength",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.MaxLength,
			NewValue: *target.MaxLength,
			Message:  "maxLength constraint changed",
		})
	} else if source.MaxLength == nil && target.MaxLength != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maxLength",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.MaxLength,
			Message:  "maxLength constraint added",
		})
	}

	// MinLength constraint
	if source.MinLength != nil && target.MinLength != nil && *source.MinLength != *target.MinLength {
		severity := SeverityWarning
		if *target.MinLength > *source.MinLength {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minLength",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.MinLength,
			NewValue: *target.MinLength,
			Message:  "minLength constraint changed",
		})
	} else if source.MinLength == nil && target.MinLength != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minLength",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.MinLength,
			Message:  "minLength constraint added",
		})
	}

	// Pattern constraint
	if source.Pattern != target.Pattern {
		severity := SeverityWarning
		if source.Pattern == "" && target.Pattern != "" {
			severity = SeverityError
		}
		if source.Pattern != "" || target.Pattern != "" {
			result.Changes = append(result.Changes, Change{
				Path:     path + ".pattern",
				Type:     ChangeTypeModified,
				Category: CategorySchema,
				Severity: severity,
				OldValue: source.Pattern,
				NewValue: target.Pattern,
				Message:  "pattern constraint changed",
			})
		}
	}
}

// diffSchemaArrayConstraints compares array validation constraints
func (d *Differ) diffSchemaArrayConstraints(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxItems constraint
	if source.MaxItems != nil && target.MaxItems != nil && *source.MaxItems != *target.MaxItems {
		severity := SeverityWarning
		if *target.MaxItems < *source.MaxItems {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maxItems",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.MaxItems,
			NewValue: *target.MaxItems,
			Message:  "maxItems constraint changed",
		})
	} else if source.MaxItems == nil && target.MaxItems != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maxItems",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.MaxItems,
			Message:  "maxItems constraint added",
		})
	}

	// MinItems constraint
	if source.MinItems != nil && target.MinItems != nil && *source.MinItems != *target.MinItems {
		severity := SeverityWarning
		if *target.MinItems > *source.MinItems {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minItems",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.MinItems,
			NewValue: *target.MinItems,
			Message:  "minItems constraint changed",
		})
	} else if source.MinItems == nil && target.MinItems != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minItems",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.MinItems,
			Message:  "minItems constraint added",
		})
	}

	// UniqueItems constraint
	if source.UniqueItems != target.UniqueItems {
		severity := SeverityWarning
		if !source.UniqueItems && target.UniqueItems {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".uniqueItems",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: source.UniqueItems,
			NewValue: target.UniqueItems,
			Message:  "uniqueItems constraint changed",
		})
	}
}

// diffSchemaObjectConstraints compares object validation constraints
func (d *Differ) diffSchemaObjectConstraints(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxProperties constraint
	if source.MaxProperties != nil && target.MaxProperties != nil && *source.MaxProperties != *target.MaxProperties {
		severity := SeverityWarning
		if *target.MaxProperties < *source.MaxProperties {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maxProperties",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.MaxProperties,
			NewValue: *target.MaxProperties,
			Message:  "maxProperties constraint changed",
		})
	} else if source.MaxProperties == nil && target.MaxProperties != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".maxProperties",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.MaxProperties,
			Message:  "maxProperties constraint added",
		})
	}

	// MinProperties constraint
	if source.MinProperties != nil && target.MinProperties != nil && *source.MinProperties != *target.MinProperties {
		severity := SeverityWarning
		if *target.MinProperties > *source.MinProperties {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minProperties",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: *source.MinProperties,
			NewValue: *target.MinProperties,
			Message:  "minProperties constraint changed",
		})
	} else if source.MinProperties == nil && target.MinProperties != nil {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".minProperties",
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError,
			NewValue: *target.MinProperties,
			Message:  "minProperties constraint added",
		})
	}
}

// diffSchemaRequiredFields compares required field lists
func (d *Differ) diffSchemaRequiredFields(source, target *parser.Schema, path string, result *DiffResult) {
	sourceRequired := make(map[string]bool)
	for _, req := range source.Required {
		sourceRequired[req] = true
	}
	targetRequired := make(map[string]bool)
	for _, req := range target.Required {
		targetRequired[req] = true
	}

	// Removed required fields - INFO (relaxing)
	for req := range sourceRequired {
		if !targetRequired[req] {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.required[%s]", path, req),
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("required field %q removed", req),
			})
		}
	}

	// Added required fields - ERROR (stricter)
	for req := range targetRequired {
		if !sourceRequired[req] {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.required[%s]", path, req),
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Severity: SeverityError,
				Message:  fmt.Sprintf("required field %q added", req),
			})
		}
	}
}

// diffSchemaOASFields compares OAS-specific schema fields
func (d *Differ) diffSchemaOASFields(source, target *parser.Schema, path string, result *DiffResult) {
	if source.Nullable != target.Nullable {
		severity := SeverityWarning
		if source.Nullable && !target.Nullable {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".nullable",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: source.Nullable,
			NewValue: target.Nullable,
			Message:  "nullable changed",
		})
	}

	if source.ReadOnly != target.ReadOnly {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".readOnly",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source.ReadOnly,
			NewValue: target.ReadOnly,
			Message:  "readOnly changed",
		})
	}

	if source.WriteOnly != target.WriteOnly {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".writeOnly",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source.WriteOnly,
			NewValue: target.WriteOnly,
			Message:  "writeOnly changed",
		})
	}

	if source.Deprecated != target.Deprecated {
		severity := SeverityInfo
		if !source.Deprecated && target.Deprecated {
			severity = SeverityWarning
		}
		result.Changes = append(result.Changes, Change{
			Path:     path + ".deprecated",
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: source.Deprecated,
			NewValue: target.Deprecated,
			Message:  "deprecated status changed",
		})
	}
}

// diffSecuritySchemesBreaking compares security schemes with breaking change detection
func (d *Differ) diffSecuritySchemesBreaking(source, target map[string]*parser.SecurityScheme, path string, result *DiffResult) {
	// Removed security schemes - ERROR
	for name, sourceScheme := range source {
		targetScheme, exists := target[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategorySecurity,
				Severity: SeverityError,
				Message:  fmt.Sprintf("security scheme %q removed", name),
			})
			continue
		}

		// Compare security scheme details
		d.diffSecuritySchemeBreaking(sourceScheme, targetScheme, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Added security schemes - WARNING (clients may need to handle new auth)
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategorySecurity,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("security scheme %q added", name),
			})
		}
	}
}

// diffSecuritySchemeBreaking compares individual SecurityScheme objects
func (d *Differ) diffSecuritySchemeBreaking(source, target *parser.SecurityScheme, path string, result *DiffResult) {
	if source.Type != target.Type {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".type",
			Type:     ChangeTypeModified,
			Category: CategorySecurity,
			Severity: SeverityError,
			OldValue: source.Type,
			NewValue: target.Type,
			Message:  fmt.Sprintf("security scheme type changed from %q to %q", source.Type, target.Type),
		})
	}

	// Compare SecurityScheme extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffComponentsBreaking compares Components with breaking change detection
func (d *Differ) diffComponentsBreaking(source, target *parser.Components, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityInfo,
			Message:  "components added",
		})
		return
	}

	if target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategorySchema,
			Severity: SeverityError,
			Message:  "components removed",
		})
		return
	}

	// Compare schemas
	d.diffSchemasBreaking(source.Schemas, target.Schemas, path+".schemas", result)

	// Compare security schemes
	d.diffSecuritySchemesBreaking(source.SecuritySchemes, target.SecuritySchemes, path+".securitySchemes", result)

	// Compare Components extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffWebhooksBreaking compares webhooks with breaking change detection
func (d *Differ) diffWebhooksBreaking(source, target map[string]*parser.PathItem, path string, result *DiffResult) {
	// Removed webhooks - ERROR
	for name := range source {
		if _, exists := target[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategoryEndpoint,
				Severity: SeverityError,
				Message:  fmt.Sprintf("webhook %q removed", name),
			})
		}
	}

	// Added webhooks - INFO
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategoryEndpoint,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("webhook %q added", name),
			})
		}
	}
}

// diffStringSlicesBreaking compares string slices with severity classification
func (d *Differ) diffStringSlicesBreaking(source, target []string, path string, category ChangeCategory, itemName string, removeSeverity Severity, result *DiffResult) {
	sourceMap := make(map[string]struct{})
	for _, item := range source {
		sourceMap[item] = struct{}{}
	}

	targetMap := make(map[string]struct{})
	for _, item := range target {
		targetMap[item] = struct{}{}
	}

	// Removed items
	for item := range sourceMap {
		if _, ok := targetMap[item]; !ok {
			result.Changes = append(result.Changes, Change{
				Path:     path,
				Type:     ChangeTypeRemoved,
				Category: category,
				Severity: removeSeverity,
				OldValue: item,
				Message:  fmt.Sprintf("%s %q removed", itemName, item),
			})
		}
	}

	// Added items - INFO
	for item := range targetMap {
		if _, ok := sourceMap[item]; !ok {
			result.Changes = append(result.Changes, Change{
				Path:     path,
				Type:     ChangeTypeAdded,
				Category: category,
				Severity: SeverityInfo,
				NewValue: item,
				Message:  fmt.Sprintf("%s %q added", itemName, item),
			})
		}
	}
}

// diffEnumBreaking compares enum values with breaking change detection
func (d *Differ) diffEnumBreaking(source, target []any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	sourceMap := make(map[string]struct{})
	for _, val := range source {
		sourceMap[anyToString(val)] = struct{}{}
	}

	targetMap := make(map[string]struct{})
	for _, val := range target {
		targetMap[anyToString(val)] = struct{}{}
	}

	// Removed enum values - ERROR (restricts valid values)
	for val := range sourceMap {
		if _, ok := targetMap[val]; !ok {
			result.Changes = append(result.Changes, Change{
				Path:     path,
				Type:     ChangeTypeRemoved,
				Category: CategoryParameter,
				Severity: SeverityError,
				Message:  fmt.Sprintf("enum value %q removed", val),
			})
		}
	}

	// Added enum values - INFO (expands valid values)
	for val := range targetMap {
		if _, ok := sourceMap[val]; !ok {
			result.Changes = append(result.Changes, Change{
				Path:     path,
				Type:     ChangeTypeAdded,
				Category: CategoryParameter,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("enum value %q added", val),
			})
		}
	}
}

// diffExtrasBreaking compares Extra maps with severity classification
func (d *Differ) diffExtrasBreaking(source, target map[string]any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed extensions - INFO (non-breaking, extensions are typically optional)
	for key, sourceValue := range source {
		targetValue, exists := target[key]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, key),
				Type:     ChangeTypeRemoved,
				Category: CategoryExtension,
				Severity: SeverityInfo,
				OldValue: sourceValue,
				Message:  fmt.Sprintf("extension %q removed", key),
			})
			continue
		}

		// Check if value changed - INFO (extensions are non-normative)
		if !reflect.DeepEqual(sourceValue, targetValue) {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, key),
				Type:     ChangeTypeModified,
				Category: CategoryExtension,
				Severity: SeverityInfo,
				OldValue: sourceValue,
				NewValue: targetValue,
				Message:  fmt.Sprintf("extension %q modified", key),
			})
		}
	}

	// Find added extensions - INFO (non-breaking)
	for key, targetValue := range target {
		if _, exists := source[key]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, key),
				Type:     ChangeTypeAdded,
				Category: CategoryExtension,
				Severity: SeverityInfo,
				NewValue: targetValue,
				Message:  fmt.Sprintf("extension %q added", key),
			})
		}
	}
}

// Helper functions

// isCompatibleTypeChange determines if a type change is compatible (widening)
func isCompatibleTypeChange(oldType, newType string) bool {
	// integer -> number is a widening conversion (compatible)
	if oldType == "integer" && newType == "number" {
		return true
	}
	return false
}

// isSuccessCode checks if a status code is a success code (2xx)
func isSuccessCode(code string) bool {
	if strings.HasPrefix(code, "2") {
		return true
	}
	// Check if it's a numeric 2xx code
	if codeNum, err := strconv.Atoi(code); err == nil {
		return codeNum >= 200 && codeNum < 300
	}
	return false
}

// isErrorCode checks if a status code is an error code (4xx or 5xx)
func isErrorCode(code string) bool {
	if strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") {
		return true
	}
	// Check if it's a numeric 4xx or 5xx code
	if codeNum, err := strconv.Atoi(code); err == nil {
		return codeNum >= 400
	}
	return false
}

func anyToString(v any) string {
	switch val := v.(type) {
	case string:
		return val // Direct string return is most efficient
	case int:
		return strconv.Itoa(val) // Optimized for integers
	case int64:
		return strconv.FormatInt(val, 10) // Optimized for int64
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64) // Use strconv for floats as well
	case fmt.Stringer:
		return val.String() // Use the String() method if the type implements it
	default:
		return fmt.Sprint(val) // Fallback for all other types (uses reflection)
	}
}

// diffSchemaRecursiveBreaking performs comprehensive recursive schema comparison with cycle detection
func (d *Differ) diffSchemaRecursiveBreaking(
	source, target *parser.Schema,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	// Nil handling
	if source == nil && target == nil {
		return
	}
	if source == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityInfo,
			NewValue: target,
			Message:  "schema added",
		})
		return
	}
	if target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategorySchema,
			Severity: SeverityError,
			OldValue: source,
			Message:  "schema removed",
		})
		return
	}

	// Cycle detection - track both source and target to prevent infinite loops
	if visited.enter(source, target, path) {
		// Already visiting this schema pair - circular reference
		// Don't report as a change, just skip further traversal
		return
	}
	defer visited.leave(source, target)

	// Compare all existing fields (already implemented)
	d.diffSchemaMetadata(source, target, path, result)
	d.diffSchemaType(source, target, path, result)
	d.diffSchemaNumericConstraints(source, target, path, result)
	d.diffSchemaStringConstraints(source, target, path, result)
	d.diffSchemaArrayConstraints(source, target, path, result)
	d.diffSchemaObjectConstraints(source, target, path, result)
	d.diffSchemaRequiredFields(source, target, path, result)
	d.diffSchemaOASFields(source, target, path, result)

	// NEW: Compare recursive/complex fields
	d.diffSchemaPropertiesBreaking(source.Properties, target.Properties, source.Required, target.Required, path, visited, result)
	d.diffSchemaItemsBreaking(source.Items, target.Items, path, visited, result)
	d.diffSchemaAdditionalPropertiesBreaking(source.AdditionalProperties, target.AdditionalProperties, path, visited, result)

	// Compare composition fields
	d.diffSchemaAllOfBreaking(source.AllOf, target.AllOf, path, visited, result)
	d.diffSchemaAnyOfBreaking(source.AnyOf, target.AnyOf, path, visited, result)
	d.diffSchemaOneOfBreaking(source.OneOf, target.OneOf, path, visited, result)
	d.diffSchemaNotBreaking(source.Not, target.Not, path, visited, result)

	// Compare conditional schemas
	d.diffSchemaConditionalBreaking(source.If, source.Then, source.Else, target.If, target.Then, target.Else, path, visited, result)

	// All known fields addressed, now address extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
}

// diffSchemaPropertiesBreaking compares schema properties maps
func (d *Differ) diffSchemaPropertiesBreaking(
	source, target map[string]*parser.Schema,
	sourceRequired, targetRequired []string,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed properties
	for name, sourceSchema := range source {
		propPath := fmt.Sprintf("%s.properties.%s", path, name)
		if targetSchema, exists := target[name]; !exists {
			// Removed property
			// Severity depends on whether it was required in the parent schema
			severity := SeverityWarning
			if isPropertyRequired(name, sourceRequired) {
				severity = SeverityError
			}
			result.Changes = append(result.Changes, Change{
				Path:     propPath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: severity,
				OldValue: sourceSchema,
				Message:  fmt.Sprintf("property %q removed", name),
			})
		} else {
			// Property exists in both - recursive comparison
			d.diffSchemaRecursiveBreaking(sourceSchema, targetSchema, propPath, visited, result)
		}
	}

	// Find added properties
	for name, targetSchema := range target {
		if _, exists := source[name]; !exists {
			propPath := fmt.Sprintf("%s.properties.%s", path, name)
			// Added property
			// Severity depends on whether it's required in the parent schema
			severity := SeverityInfo
			if isPropertyRequired(name, targetRequired) {
				severity = SeverityWarning
			}
			result.Changes = append(result.Changes, Change{
				Path:     propPath,
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Severity: severity,
				NewValue: targetSchema,
				Message:  fmt.Sprintf("property %q added", name),
			})
		}
	}
}

// diffSchemaItemsBreaking compares schema Items field (can be *Schema or bool)
func (d *Differ) diffSchemaItemsBreaking(
	source, target any,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	sourceType := getSchemaItemsType(source)
	targetType := getSchemaItemsType(target)

	itemsPath := path + ".items"

	// Check for unknown types (spec violation)
	// If both have unknown type, skip comparison (can't diff unknown structures)
	if sourceType == schemaItemsTypeUnknown && targetType == schemaItemsTypeUnknown {
		return
	}
	if sourceType == schemaItemsTypeUnknown {
		result.Changes = append(result.Changes, Change{
			Path:     itemsPath,
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source,
			Message:  fmt.Sprintf("items has unexpected type in source: %T (should be Schema or bool)", source),
		})
		return
	}
	if targetType == schemaItemsTypeUnknown {
		result.Changes = append(result.Changes, Change{
			Path:     itemsPath,
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			NewValue: target,
			Message:  fmt.Sprintf("items has unexpected type in target: %T (should be Schema or bool)", target),
		})
		return
	}

	// Both nil
	if sourceType == schemaItemsTypeNil && targetType == schemaItemsTypeNil {
		return
	}

	// Items added
	if sourceType == schemaItemsTypeNil && targetType != schemaItemsTypeNil {
		result.Changes = append(result.Changes, Change{
			Path:     itemsPath,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityWarning,
			NewValue: target,
			Message:  "items schema added",
		})
		return
	}

	// Items removed
	if sourceType != schemaItemsTypeNil && targetType == schemaItemsTypeNil {
		result.Changes = append(result.Changes, Change{
			Path:     itemsPath,
			Type:     ChangeTypeRemoved,
			Category: CategorySchema,
			Severity: SeverityError,
			OldValue: source,
			Message:  "items schema removed",
		})
		return
	}

	// Type changed
	if sourceType != targetType {
		severity := SeverityError
		if sourceType == schemaItemsTypeBool && targetType == schemaItemsTypeSchema {
			// bool -> schema might be relaxing or tightening depending on bool value
			severity = SeverityWarning
		}
		result.Changes = append(result.Changes, Change{
			Path:     itemsPath,
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: severity,
			OldValue: source,
			NewValue: target,
			Message:  "items type changed",
		})
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaItemsTypeSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveBreaking(sourceSchema, targetSchema, itemsPath, visited, result)

	case schemaItemsTypeBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityWarning
			if sourceBool && !targetBool {
				// true -> false: was allowing any, now disallowing
				severity = SeverityError
			}
			result.Changes = append(result.Changes, Change{
				Path:     itemsPath,
				Type:     ChangeTypeModified,
				Category: CategorySchema,
				Severity: severity,
				OldValue: sourceBool,
				NewValue: targetBool,
				Message:  fmt.Sprintf("items changed from %v to %v", sourceBool, targetBool),
			})
		}
	}
}

// diffSchemaAdditionalPropertiesBreaking compares additionalProperties field (can be *Schema or bool)
func (d *Differ) diffSchemaAdditionalPropertiesBreaking(
	source, target any,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)

	addPropsPath := path + ".additionalProperties"

	// Check for unknown types (spec violation)
	// If both have unknown type, skip comparison (can't diff unknown structures)
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		result.Changes = append(result.Changes, Change{
			Path:     addPropsPath,
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source,
			Message:  fmt.Sprintf("additionalProperties has unexpected type in source: %T (should be Schema or bool)", source),
		})
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		result.Changes = append(result.Changes, Change{
			Path:     addPropsPath,
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			NewValue: target,
			Message:  fmt.Sprintf("additionalProperties has unexpected type in target: %T (should be Schema or bool)", target),
		})
		return
	}

	// Both nil (means true in JSON Schema)
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// additionalProperties added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		// If target is false, this restricts what was previously allowed
		if targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		result.Changes = append(result.Changes, Change{
			Path:     addPropsPath,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: severity,
			NewValue: target,
			Message:  "additionalProperties constraint added",
		})
		return
	}

	// additionalProperties removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		severity := SeverityWarning
		// If source was false, removing it relaxes constraint
		if sourceType == schemaAdditionalPropsTypeBool && !source.(bool) {
			severity = SeverityInfo
		}
		result.Changes = append(result.Changes, Change{
			Path:     addPropsPath,
			Type:     ChangeTypeRemoved,
			Category: CategorySchema,
			Severity: severity,
			OldValue: source,
			Message:  "additionalProperties constraint removed",
		})
		return
	}

	// Type changed
	if sourceType != targetType {
		result.Changes = append(result.Changes, Change{
			Path:     addPropsPath,
			Type:     ChangeTypeModified,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source,
			NewValue: target,
			Message:  "additionalProperties type changed",
		})
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveBreaking(sourceSchema, targetSchema, addPropsPath, visited, result)

	case schemaAdditionalPropsTypeBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityInfo
			if sourceBool && !targetBool {
				// true -> false: was allowing additional properties, now disallowing
				severity = SeverityError
			}
			// false -> true: was disallowing, now allowing (relaxing) - Info severity
			result.Changes = append(result.Changes, Change{
				Path:     addPropsPath,
				Type:     ChangeTypeModified,
				Category: CategorySchema,
				Severity: severity,
				OldValue: sourceBool,
				NewValue: targetBool,
				Message:  fmt.Sprintf("additionalProperties changed from %v to %v", sourceBool, targetBool),
			})
		}
	}
}

// diffSchemaAllOfBreaking compares allOf composition schemas
// allOf requires ALL subschemas to validate. Changes affect strictness:
// - Add subschema: Error for requests (stricter), Info for responses
// - Remove subschema: Info for requests (relaxed), Error for responses
func (d *Differ) diffSchemaAllOfBreaking(
	source, target []*parser.Schema,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	allOfPath := path + ".allOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Track which schemas have been matched
	matched := make(map[int]bool)

	// Compare schemas by index (order matters for validation)
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", allOfPath, i)

		if i < len(target) {
			// Schema at same index in both
			targetSchema := target[i]
			matched[i] = true
			d.diffSchemaRecursiveBreaking(sourceSchema, targetSchema, schemaPath, visited, result)
		} else {
			// Schema removed from target
			// Removing an allOf constraint relaxes validation (Info for requests)
			// But for responses, removing requirements is breaking (Error)
			result.Changes = append(result.Changes, Change{
				Path:     schemaPath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityInfo, // Default to Info (request context)
				OldValue: sourceSchema,
				Message:  fmt.Sprintf("allOf schema at index %d removed", i),
			})
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", allOfPath, i)
		targetSchema := target[i]

		// Adding an allOf constraint makes validation stricter (Error for requests)
		// For responses, adding more requirements is informational (Info)
		result.Changes = append(result.Changes, Change{
			Path:     schemaPath,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityError, // Default to Error (request context)
			NewValue: targetSchema,
			Message:  fmt.Sprintf("allOf schema at index %d added", i),
		})
	}
}

// diffSchemaAnyOfBreaking compares anyOf composition schemas
// anyOf requires AT LEAST ONE subschema to validate:
// - Add subschema: Info (more options) for requests, Warning for responses
// - Remove subschema: Warning (fewer options) for both contexts
func (d *Differ) diffSchemaAnyOfBreaking(
	source, target []*parser.Schema,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	anyOfPath := path + ".anyOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare schemas by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", anyOfPath, i)

		if i < len(target) {
			// Schema at same index in both
			targetSchema := target[i]
			d.diffSchemaRecursiveBreaking(sourceSchema, targetSchema, schemaPath, visited, result)
		} else {
			// Schema removed from target
			// Removing an anyOf option reduces choices (Warning)
			result.Changes = append(result.Changes, Change{
				Path:     schemaPath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityWarning,
				OldValue: sourceSchema,
				Message:  fmt.Sprintf("anyOf schema at index %d removed", i),
			})
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", anyOfPath, i)
		targetSchema := target[i]

		// Adding an anyOf option provides more choices (Info for requests, Warning for responses)
		result.Changes = append(result.Changes, Change{
			Path:     schemaPath,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityInfo,
			NewValue: targetSchema,
			Message:  fmt.Sprintf("anyOf schema at index %d added", i),
		})
	}
}

// diffSchemaOneOfBreaking compares oneOf composition schemas
// oneOf requires EXACTLY ONE subschema to validate:
// - Add subschema: Warning (changes validation logic)
// - Remove subschema: Warning (changes validation logic)
func (d *Differ) diffSchemaOneOfBreaking(
	source, target []*parser.Schema,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	oneOfPath := path + ".oneOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare schemas by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", oneOfPath, i)

		if i < len(target) {
			// Schema at same index in both
			targetSchema := target[i]
			d.diffSchemaRecursiveBreaking(sourceSchema, targetSchema, schemaPath, visited, result)
		} else {
			// Schema removed from target
			// Removing a oneOf option changes exclusive validation (Warning)
			result.Changes = append(result.Changes, Change{
				Path:     schemaPath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityWarning,
				OldValue: sourceSchema,
				Message:  fmt.Sprintf("oneOf schema at index %d removed", i),
			})
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", oneOfPath, i)
		targetSchema := target[i]

		// Adding a oneOf option changes exclusive validation (Warning)
		result.Changes = append(result.Changes, Change{
			Path:     schemaPath,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityWarning,
			NewValue: targetSchema,
			Message:  fmt.Sprintf("oneOf schema at index %d added", i),
		})
	}
}

// diffSchemaNotBreaking compares not schemas
// not negates a schema - changes affect validation logic
func (d *Differ) diffSchemaNotBreaking(
	source, target *parser.Schema,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	notPath := path + ".not"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		// not added - changes what's rejected
		result.Changes = append(result.Changes, Change{
			Path:     notPath,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Severity: SeverityWarning,
			NewValue: target,
			Message:  "not schema added",
		})
		return
	}

	if target == nil {
		// not removed - changes what's rejected
		result.Changes = append(result.Changes, Change{
			Path:     notPath,
			Type:     ChangeTypeRemoved,
			Category: CategorySchema,
			Severity: SeverityWarning,
			OldValue: source,
			Message:  "not schema removed",
		})
		return
	}

	// Both exist - compare recursively
	d.diffSchemaRecursiveBreaking(source, target, notPath, visited, result)
}

// diffSchemaConditionalBreaking compares conditional schemas (if/then/else)
// Conditional schemas affect validation based on conditions
func (d *Differ) diffSchemaConditionalBreaking(
	sourceIf, sourceThen, sourceElse *parser.Schema,
	targetIf, targetThen, targetElse *parser.Schema,
	path string,
	visited *schemaVisited,
	result *DiffResult,
) {
	// Compare if condition
	if sourceIf != nil || targetIf != nil {
		ifPath := path + ".if"
		if sourceIf == nil {
			result.Changes = append(result.Changes, Change{
				Path:     ifPath,
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Severity: SeverityWarning,
				NewValue: targetIf,
				Message:  "conditional if schema added",
			})
		} else if targetIf == nil {
			result.Changes = append(result.Changes, Change{
				Path:     ifPath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityWarning,
				OldValue: sourceIf,
				Message:  "conditional if schema removed",
			})
		} else {
			d.diffSchemaRecursiveBreaking(sourceIf, targetIf, ifPath, visited, result)
		}
	}

	// Compare then branch
	if sourceThen != nil || targetThen != nil {
		thenPath := path + ".then"
		if sourceThen == nil {
			result.Changes = append(result.Changes, Change{
				Path:     thenPath,
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Severity: SeverityWarning,
				NewValue: targetThen,
				Message:  "conditional then schema added",
			})
		} else if targetThen == nil {
			result.Changes = append(result.Changes, Change{
				Path:     thenPath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityWarning,
				OldValue: sourceThen,
				Message:  "conditional then schema removed",
			})
		} else {
			d.diffSchemaRecursiveBreaking(sourceThen, targetThen, thenPath, visited, result)
		}
	}

	// Compare else branch
	if sourceElse != nil || targetElse != nil {
		elsePath := path + ".else"
		if sourceElse == nil {
			result.Changes = append(result.Changes, Change{
				Path:     elsePath,
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Severity: SeverityWarning,
				NewValue: targetElse,
				Message:  "conditional else schema added",
			})
		} else if targetElse == nil {
			result.Changes = append(result.Changes, Change{
				Path:     elsePath,
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Severity: SeverityWarning,
				OldValue: sourceElse,
				Message:  "conditional else schema removed",
			})
		} else {
			d.diffSchemaRecursiveBreaking(sourceElse, targetElse, elsePath, visited, result)
		}
	}
}
