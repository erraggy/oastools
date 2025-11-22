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

	// Compare Response extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
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
	// Note: Full schema comparison is complex.
	// Currently we only compare extensions.
	// Deep schema field comparison would require significant additional logic.

	// Compare Schema extensions
	d.diffExtrasBreaking(source.Extra, target.Extra, path, result)
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
	sourceMap := make(map[string]bool)
	for _, item := range source {
		sourceMap[item] = true
	}

	targetMap := make(map[string]bool)
	for _, item := range target {
		targetMap[item] = true
	}

	// Removed items
	for item := range sourceMap {
		if !targetMap[item] {
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
		if !sourceMap[item] {
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

	sourceMap := make(map[string]bool)
	for _, val := range source {
		sourceMap[fmt.Sprint(val)] = true
	}

	targetMap := make(map[string]bool)
	for _, val := range target {
		targetMap[fmt.Sprint(val)] = true
	}

	// Removed enum values - ERROR (restricts valid values)
	for val := range sourceMap {
		if !targetMap[val] {
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
		if !sourceMap[val] {
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
