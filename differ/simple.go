package differ

import (
	"fmt"
	"reflect"

	"github.com/erraggy/oastools/parser"
)

// diffSimple performs a simple semantic diff that reports all differences
// without categorizing them as breaking or non-breaking
func (d *Differ) diffSimple(source, target parser.ParseResult, result *DiffResult) {
	// Compare based on OAS version
	switch {
	case source.OASVersion == parser.OASVersion20 && target.OASVersion == parser.OASVersion20:
		d.diffOAS2Simple(source.Document.(*parser.OAS2Document), target.Document.(*parser.OAS2Document), result)
	case source.OASVersion >= parser.OASVersion300 && target.OASVersion >= parser.OASVersion300:
		d.diffOAS3Simple(source.Document.(*parser.OAS3Document), target.Document.(*parser.OAS3Document), result)
	default:
		// Cross-version comparison - convert both to OAS3 for comparison
		d.diffCrossVersionSimple(source, target, result)
	}
}

// diffOAS2Simple compares two OAS 2.0 documents
func (d *Differ) diffOAS2Simple(source, target *parser.OAS2Document, result *DiffResult) {
	basePath := "document"

	// Compare Info
	d.diffInfo(source.Info, target.Info, basePath+".info", result)

	// Compare Host, BasePath, Schemes
	if source.Host != target.Host {
		result.Changes = append(result.Changes, Change{
			Path:     basePath + ".host",
			Type:     ChangeTypeModified,
			Category: CategoryServer,
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
			OldValue: source.BasePath,
			NewValue: target.BasePath,
			Message:  fmt.Sprintf("basePath changed from %q to %q", source.BasePath, target.BasePath),
		})
	}

	d.diffStringSlices(source.Schemes, target.Schemes, basePath+".schemes", CategoryServer, "scheme", result)

	// Compare Consumes/Produces
	d.diffStringSlices(source.Consumes, target.Consumes, basePath+".consumes", CategoryOperation, "consumes media type", result)
	d.diffStringSlices(source.Produces, target.Produces, basePath+".produces", CategoryOperation, "produces media type", result)

	// Compare Paths
	d.diffPaths(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Definitions
	d.diffSchemas(source.Definitions, target.Definitions, basePath+".definitions", result)

	// Compare Security Definitions
	d.diffSecuritySchemes(source.SecurityDefinitions, target.SecurityDefinitions, basePath+".securityDefinitions", result)

	// Compare Tags
	d.diffTags(source.Tags, target.Tags, basePath+".tags", result)

	// Compare Extensions
	d.diffExtras(source.Extra, target.Extra, basePath, result)
}

// diffOAS3Simple compares two OAS 3.x documents
func (d *Differ) diffOAS3Simple(source, target *parser.OAS3Document, result *DiffResult) {
	basePath := "document"

	// Compare Info
	d.diffInfo(source.Info, target.Info, basePath+".info", result)

	// Compare Servers
	d.diffServers(source.Servers, target.Servers, basePath+".servers", result)

	// Compare Paths
	d.diffPaths(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Webhooks (OAS 3.1+)
	if source.Webhooks != nil || target.Webhooks != nil {
		d.diffWebhooks(source.Webhooks, target.Webhooks, basePath+".webhooks", result)
	}

	// Compare Components
	if source.Components != nil || target.Components != nil {
		d.diffComponents(source.Components, target.Components, basePath+".components", result)
	}

	// Compare Tags
	d.diffTags(source.Tags, target.Tags, basePath+".tags", result)

	// Compare Extensions
	d.diffExtras(source.Extra, target.Extra, basePath, result)
}

// diffCrossVersionSimple compares documents of different OAS versions
func (d *Differ) diffCrossVersionSimple(source, target parser.ParseResult, result *DiffResult) {
	// For simplicity, just note that cross-version comparison is limited
	result.Changes = append(result.Changes, Change{
		Path:     "document",
		Type:     ChangeTypeModified,
		Category: CategoryInfo,
		OldValue: source.Version,
		NewValue: target.Version,
		Message:  fmt.Sprintf("OAS version changed from %s to %s (cross-version diff has limitations)", source.Version, target.Version),
	})

	// Compare what we can (Info and Paths exist in both)
	var sourceInfo, targetInfo *parser.Info
	var sourcePaths, targetPaths parser.Paths

	switch source.OASVersion {
	case parser.OASVersion20:
		doc := source.Document.(*parser.OAS2Document)
		sourceInfo = doc.Info
		sourcePaths = doc.Paths
	default:
		doc := source.Document.(*parser.OAS3Document)
		sourceInfo = doc.Info
		sourcePaths = doc.Paths
	}

	switch target.OASVersion {
	case parser.OASVersion20:
		doc := target.Document.(*parser.OAS2Document)
		targetInfo = doc.Info
		targetPaths = doc.Paths
	default:
		doc := target.Document.(*parser.OAS3Document)
		targetInfo = doc.Info
		targetPaths = doc.Paths
	}

	d.diffInfo(sourceInfo, targetInfo, "document.info", result)
	d.diffPaths(sourcePaths, targetPaths, "document.paths", result)
}

// diffInfo compares Info objects
func (d *Differ) diffInfo(source, target *parser.Info, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategoryInfo,
			NewValue: target,
			Message:  "info object added",
		})
		return
	}

	if target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategoryInfo,
			OldValue: source,
			Message:  "info object removed",
		})
		return
	}

	// Compare fields
	if source.Title != target.Title {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".title",
			Type:     ChangeTypeModified,
			Category: CategoryInfo,
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
			OldValue: source.Version,
			NewValue: target.Version,
			Message:  fmt.Sprintf("API version changed from %q to %q", source.Version, target.Version),
		})
	}

	if source.Description != target.Description {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryInfo,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "description changed",
		})
	}

	// Compare Info extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffServers compares Server slices (OAS 3.x)
func (d *Differ) diffServers(source, target []*parser.Server, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Build maps by URL for easier comparison
	sourceMap := make(map[string]*parser.Server)
	for _, srv := range source {
		sourceMap[srv.URL] = srv
	}

	targetMap := make(map[string]*parser.Server)
	for _, srv := range target {
		targetMap[srv.URL] = srv
	}

	// Find removed servers
	for url, sourceSrv := range sourceMap {
		if _, exists := targetMap[url]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, url),
				Type:     ChangeTypeRemoved,
				Category: CategoryServer,
				OldValue: url,
				Message:  fmt.Sprintf("server %q removed", url),
			})
			continue
		}

		// Compare server details if both exist
		d.diffServer(sourceSrv, targetMap[url], fmt.Sprintf("%s[%s]", path, url), result)
	}

	// Find added servers
	for url := range targetMap {
		if _, exists := sourceMap[url]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, url),
				Type:     ChangeTypeAdded,
				Category: CategoryServer,
				NewValue: url,
				Message:  fmt.Sprintf("server %q added", url),
			})
		}
	}
}

// diffServer compares individual Server objects
func (d *Differ) diffServer(source, target *parser.Server, path string, result *DiffResult) {
	if source.Description != target.Description {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryServer,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "server description changed",
		})
	}

	// Compare Server extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffPaths compares Paths objects
func (d *Differ) diffPaths(source, target parser.Paths, path string, result *DiffResult) {
	// Find removed paths
	for pathName, sourceItem := range source {
		targetItem, exists := target[pathName]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, pathName),
				Type:     ChangeTypeRemoved,
				Category: CategoryEndpoint,
				OldValue: sourceItem,
				Message:  fmt.Sprintf("endpoint %q removed", pathName),
			})
			continue
		}

		// Compare path items
		d.diffPathItem(sourceItem, targetItem, fmt.Sprintf("%s.%s", path, pathName), result)
	}

	// Find added paths
	for pathName, targetItem := range target {
		if _, exists := source[pathName]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, pathName),
				Type:     ChangeTypeAdded,
				Category: CategoryEndpoint,
				NewValue: targetItem,
				Message:  fmt.Sprintf("endpoint %q added", pathName),
			})
		}
	}
}

// diffPathItem compares PathItem objects
func (d *Differ) diffPathItem(source, target *parser.PathItem, path string, result *DiffResult) {
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

		if ops.source == nil && ops.target != nil {
			result.Changes = append(result.Changes, Change{
				Path:     opPath,
				Type:     ChangeTypeAdded,
				Category: CategoryOperation,
				NewValue: ops.target,
				Message:  fmt.Sprintf("operation %s added", method),
			})
			continue
		}

		if ops.source != nil && ops.target == nil {
			result.Changes = append(result.Changes, Change{
				Path:     opPath,
				Type:     ChangeTypeRemoved,
				Category: CategoryOperation,
				OldValue: ops.source,
				Message:  fmt.Sprintf("operation %s removed", method),
			})
			continue
		}

		// Compare operations
		d.diffOperation(ops.source, ops.target, opPath, result)
	}

	// Compare PathItem extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffOperation compares Operation objects
func (d *Differ) diffOperation(source, target *parser.Operation, path string, result *DiffResult) {
	// Compare deprecated flag
	if source.Deprecated != target.Deprecated {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".deprecated",
			Type:     ChangeTypeModified,
			Category: CategoryOperation,
			OldValue: source.Deprecated,
			NewValue: target.Deprecated,
			Message:  fmt.Sprintf("deprecated changed from %v to %v", source.Deprecated, target.Deprecated),
		})
	}

	// Compare parameters
	d.diffParameters(source.Parameters, target.Parameters, path+".parameters", result)

	// Compare responses
	d.diffResponses(source.Responses, target.Responses, path+".responses", result)

	// Compare request body (OAS 3.x)
	if source.RequestBody != nil || target.RequestBody != nil {
		d.diffRequestBody(source.RequestBody, target.RequestBody, path+".requestBody", result)
	}

	// Compare Operation extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffParameters compares Parameter slices
func (d *Differ) diffParameters(source, target []*parser.Parameter, path string, result *DiffResult) {
	// Build maps by name+in for easier comparison
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

	// Find removed parameters
	for key, sourceParam := range sourceMap {
		if _, exists := targetMap[key]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, key),
				Type:     ChangeTypeRemoved,
				Category: CategoryParameter,
				OldValue: sourceParam,
				Message:  fmt.Sprintf("parameter %q in %s removed", sourceParam.Name, sourceParam.In),
			})
		}
	}

	// Find added or modified parameters
	for key, targetParam := range targetMap {
		sourceParam, exists := sourceMap[key]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, key),
				Type:     ChangeTypeAdded,
				Category: CategoryParameter,
				NewValue: targetParam,
				Message:  fmt.Sprintf("parameter %q in %s added", targetParam.Name, targetParam.In),
			})
			continue
		}

		// Compare parameter details
		d.diffParameter(sourceParam, targetParam, fmt.Sprintf("%s[%s]", path, key), result)
	}
}

// diffParameter compares individual Parameter objects
func (d *Differ) diffParameter(source, target *parser.Parameter, path string, result *DiffResult) {
	if source.Required != target.Required {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			OldValue: source.Required,
			NewValue: target.Required,
			Message:  fmt.Sprintf("required changed from %v to %v", source.Required, target.Required),
		})
	}

	if source.Type != target.Type {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".type",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			OldValue: source.Type,
			NewValue: target.Type,
			Message:  fmt.Sprintf("type changed from %q to %q", source.Type, target.Type),
		})
	}

	if source.Format != target.Format {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".format",
			Type:     ChangeTypeModified,
			Category: CategoryParameter,
			OldValue: source.Format,
			NewValue: target.Format,
			Message:  fmt.Sprintf("format changed from %q to %q", source.Format, target.Format),
		})
	}

	// Compare Parameter extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffRequestBody compares RequestBody objects (OAS 3.x)
func (d *Differ) diffRequestBody(source, target *parser.RequestBody, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategoryRequestBody,
			NewValue: target,
			Message:  "request body added",
		})
		return
	}

	if target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategoryRequestBody,
			OldValue: source,
			Message:  "request body removed",
		})
		return
	}

	if source.Required != target.Required {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".required",
			Type:     ChangeTypeModified,
			Category: CategoryRequestBody,
			OldValue: source.Required,
			NewValue: target.Required,
			Message:  fmt.Sprintf("required changed from %v to %v", source.Required, target.Required),
		})
	}

	// Compare RequestBody extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffResponses compares Responses objects
func (d *Differ) diffResponses(source, target *parser.Responses, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategoryResponse,
			NewValue: target,
			Message:  "responses added",
		})
		return
	}

	if target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategoryResponse,
			OldValue: source,
			Message:  "responses removed",
		})
		return
	}

	// Compare individual response codes
	for code, sourceResp := range source.Codes {
		targetResp, exists := target.Codes[code]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, code),
				Type:     ChangeTypeRemoved,
				Category: CategoryResponse,
				OldValue: sourceResp,
				Message:  fmt.Sprintf("response code %s removed", code),
			})
			continue
		}

		// Compare response details
		d.diffResponse(sourceResp, targetResp, fmt.Sprintf("%s[%s]", path, code), result)
	}

	// Find added response codes
	for code, targetResp := range target.Codes {
		if _, exists := source.Codes[code]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, code),
				Type:     ChangeTypeAdded,
				Category: CategoryResponse,
				NewValue: targetResp,
				Message:  fmt.Sprintf("response code %s added", code),
			})
		}
	}
}

// diffResponse compares individual Response objects
func (d *Differ) diffResponse(source, target *parser.Response, path string, result *DiffResult) {
	if source.Description != target.Description {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryResponse,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "response description changed",
		})
	}

	// Compare Response extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffSchemas compares schema maps
func (d *Differ) diffSchemas(source, target map[string]*parser.Schema, path string, result *DiffResult) {
	// Find removed schemas
	for name, sourceSchema := range source {
		targetSchema, exists := target[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategorySchema,
				Message:  fmt.Sprintf("schema %q removed", name),
			})
			continue
		}

		// Compare schema extensions
		d.diffSchema(sourceSchema, targetSchema, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Find added schemas
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategorySchema,
				Message:  fmt.Sprintf("schema %q added", name),
			})
		}
	}
}

// diffSchema compares individual Schema objects
func (d *Differ) diffSchema(source, target *parser.Schema, path string, result *DiffResult) {
	// Note: Full schema comparison is complex.
	// Currently we only compare extensions.
	// Deep schema field comparison would require significant additional logic.

	// Compare Schema extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffSecuritySchemes compares security scheme maps
func (d *Differ) diffSecuritySchemes(source, target map[string]*parser.SecurityScheme, path string, result *DiffResult) {
	// Find removed schemes
	for name, sourceScheme := range source {
		targetScheme, exists := target[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategorySecurity,
				Message:  fmt.Sprintf("security scheme %q removed", name),
			})
			continue
		}

		// Compare security scheme details
		d.diffSecurityScheme(sourceScheme, targetScheme, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Find added schemes
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategorySecurity,
				Message:  fmt.Sprintf("security scheme %q added", name),
			})
		}
	}
}

// diffSecurityScheme compares individual SecurityScheme objects
func (d *Differ) diffSecurityScheme(source, target *parser.SecurityScheme, path string, result *DiffResult) {
	if source.Type != target.Type {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".type",
			Type:     ChangeTypeModified,
			Category: CategorySecurity,
			OldValue: source.Type,
			NewValue: target.Type,
			Message:  fmt.Sprintf("security scheme type changed from %q to %q", source.Type, target.Type),
		})
	}

	// Compare SecurityScheme extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffTags compares Tag slices
func (d *Differ) diffTags(source, target []*parser.Tag, path string, result *DiffResult) {
	sourceMap := make(map[string]*parser.Tag)
	for _, tag := range source {
		sourceMap[tag.Name] = tag
	}

	targetMap := make(map[string]*parser.Tag)
	for _, tag := range target {
		targetMap[tag.Name] = tag
	}

	// Find removed tags
	for name, sourceTag := range sourceMap {
		targetTag, exists := targetMap[name]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategoryInfo,
				Message:  fmt.Sprintf("tag %q removed", name),
			})
			continue
		}

		// Compare tag details
		d.diffTag(sourceTag, targetTag, fmt.Sprintf("%s[%s]", path, name), result)
	}

	// Find added tags
	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s[%s]", path, name),
				Type:     ChangeTypeAdded,
				Category: CategoryInfo,
				Message:  fmt.Sprintf("tag %q added", name),
			})
		}
	}
}

// diffTag compares individual Tag objects
func (d *Differ) diffTag(source, target *parser.Tag, path string, result *DiffResult) {
	if source.Description != target.Description {
		result.Changes = append(result.Changes, Change{
			Path:     path + ".description",
			Type:     ChangeTypeModified,
			Category: CategoryInfo,
			OldValue: source.Description,
			NewValue: target.Description,
			Message:  "tag description changed",
		})
	}

	// Compare Tag extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffComponents compares Components objects (OAS 3.x)
func (d *Differ) diffComponents(source, target *parser.Components, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeAdded,
			Category: CategorySchema,
			Message:  "components added",
		})
		return
	}

	if target == nil {
		result.Changes = append(result.Changes, Change{
			Path:     path,
			Type:     ChangeTypeRemoved,
			Category: CategorySchema,
			Message:  "components removed",
		})
		return
	}

	// Compare schemas
	d.diffSchemas(source.Schemas, target.Schemas, path+".schemas", result)

	// Compare security schemes
	d.diffSecuritySchemes(source.SecuritySchemes, target.SecuritySchemes, path+".securitySchemes", result)

	// Compare Components extensions
	d.diffExtras(source.Extra, target.Extra, path, result)
}

// diffWebhooks compares webhook maps (OAS 3.1+)
func (d *Differ) diffWebhooks(source, target map[string]*parser.PathItem, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed webhooks
	for name := range source {
		if _, exists := target[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeRemoved,
				Category: CategoryEndpoint,
				Message:  fmt.Sprintf("webhook %q removed", name),
			})
		}
	}

	// Find added webhooks
	for name := range target {
		if _, exists := source[name]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, name),
				Type:     ChangeTypeAdded,
				Category: CategoryEndpoint,
				Message:  fmt.Sprintf("webhook %q added", name),
			})
		}
	}
}

// diffStringSlices compares string slices and reports differences
func (d *Differ) diffStringSlices(source, target []string, path string, category ChangeCategory, itemName string, result *DiffResult) {
	sourceMap := make(map[string]bool)
	for _, item := range source {
		sourceMap[item] = true
	}

	targetMap := make(map[string]bool)
	for _, item := range target {
		targetMap[item] = true
	}

	// Find removed items
	for item := range sourceMap {
		if !targetMap[item] {
			result.Changes = append(result.Changes, Change{
				Path:     path,
				Type:     ChangeTypeRemoved,
				Category: category,
				OldValue: item,
				Message:  fmt.Sprintf("%s %q removed", itemName, item),
			})
		}
	}

	// Find added items
	for item := range targetMap {
		if !sourceMap[item] {
			result.Changes = append(result.Changes, Change{
				Path:     path,
				Type:     ChangeTypeAdded,
				Category: category,
				NewValue: item,
				Message:  fmt.Sprintf("%s %q added", itemName, item),
			})
		}
	}
}

// diffExtras compares Extra maps (specification extensions with x- prefix)
func (d *Differ) diffExtras(source, target map[string]any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed extensions
	for key, sourceValue := range source {
		targetValue, exists := target[key]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, key),
				Type:     ChangeTypeRemoved,
				Category: CategoryExtension,
				OldValue: sourceValue,
				Message:  fmt.Sprintf("extension %q removed", key),
			})
			continue
		}

		// Check if value changed
		if !reflect.DeepEqual(sourceValue, targetValue) {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, key),
				Type:     ChangeTypeModified,
				Category: CategoryExtension,
				OldValue: sourceValue,
				NewValue: targetValue,
				Message:  fmt.Sprintf("extension %q modified", key),
			})
		}
	}

	// Find added extensions
	for key, targetValue := range target {
		if _, exists := source[key]; !exists {
			result.Changes = append(result.Changes, Change{
				Path:     fmt.Sprintf("%s.%s", path, key),
				Type:     ChangeTypeAdded,
				Category: CategoryExtension,
				NewValue: targetValue,
				Message:  fmt.Sprintf("extension %q added", key),
			})
		}
	}
}
